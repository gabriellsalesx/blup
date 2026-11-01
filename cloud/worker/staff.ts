import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, asc, count, desc, eq, gt, gte, inArray, lt, sql } from "drizzle-orm";
import { createRemoteJWKSet, jwtVerify } from "jose";
import Stripe from "stripe";
import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import { db } from "./auth";
import { billingEvent, passkey, session, staff, staffLog, subscription, user } from "./db/schema";
import { planNamed } from "./plans";
import { hasEmail } from "./auth-options";
import { usageOf } from "./usage";
import { aiUsageOf } from "./ai";

// The staff area (support and administration). In production Cloudflare
// Access stands in front of /staff and /api/staff; here the Worker checks
// Access's token again (it could be reached some other way) and that the
// e-mail is on the staff. The staff never reads notes (they are encrypted
// on the person's computer) and never signs in as a user.

type Role = "support" | "admin";
type Staff = { email: string; role: Role };
type App = { Bindings: Cloudflare.Env; Variables: { staff: Staff } };

let jwks: ReturnType<typeof createRemoteJWKSet> | undefined;

async function accessEmail(c: { req: { header: (n: string) => string | undefined; url: string }; env: Cloudflare.Env }): Promise<string | null> {
	const host = new URL(c.req.url).hostname;
	// Only on this computer, with STAFF_DEV_EMAIL in .dev.vars.
	if (c.env.STAFF_DEV_EMAIL && (host === "localhost" || host === "127.0.0.1" || host.endsWith(".localhost"))) return c.env.STAFF_DEV_EMAIL;
	const token = c.req.header("cf-access-jwt-assertion");
	if (!token || !c.env.ACCESS_TEAM_DOMAIN || !c.env.ACCESS_AUD) return null;
	jwks ??= createRemoteJWKSet(new URL(`${c.env.ACCESS_TEAM_DOMAIN}/cdn-cgi/access/certs`));
	try {
		const { payload } = await jwtVerify(token, jwks, { issuer: c.env.ACCESS_TEAM_DOMAIN, audience: c.env.ACCESS_AUD });
		return typeof payload.email === "string" ? payload.email.toLowerCase() : null;
	} catch {
		return null;
	}
}

const onStaff = createMiddleware<App>(async (c, next) => {
	const email = await accessEmail(c);
	if (!email) return c.json({ code: "FORBIDDEN" }, 403);
	let role: Role | null = null;
	if (c.env.STAFF_OWNER_EMAIL && email === c.env.STAFF_OWNER_EMAIL.toLowerCase()) role = "admin";
	else {
		const [s] = await db.select({ role: staff.role }).from(staff).where(eq(staff.email, email)).limit(1);
		role = s?.role ?? null;
	}
	if (!role) return c.json({ code: "FORBIDDEN" }, 403);
	c.set("staff", { email, role });
	await next();
});

const adminOnly = createMiddleware<App>(async (c, next) => {
	if (c.get("staff").role !== "admin") return c.json({ code: "ADMIN_ONLY" }, 403);
	await next();
});

const log = (email: string, action: string, userId: string | null, detail?: string) =>
	db.insert(staffLog).values({ staffEmail: email, action, userId, detail: detail ?? null });

const ACTIVE = ["active", "trialing", "past_due"];

// Pages by cursor, not OFFSET: D1 bills every row a query reads, and an
// OFFSET reads (and bills) all the rows it skips. The cursor is the last
// item's key: UUIDv7 account IDs and the log's integer IDs both follow
// creation order, so "older than the cursor" is a range on the primary key.
const PAGE = 25;
const pageSize = (v: string | undefined) => Math.min(100, Math.max(1, Number(v) || PAGE));

// A JSON body checked by a Zod schema; a bad one answers 400 with the
// fields that failed (not Zod's whole report).
const body = <T extends z.ZodType>(schema: T) =>
	zValidator("json", schema, (r, c) => {
		if (!r.success) return c.json({ code: "INVALID", fields: r.error.issues.map((i) => i.path.join(".")) }, 400);
	});

export const staffApi = new Hono<App>()
	.use(onStaff)
	.get("/me", (c) => c.json(c.get("staff")))

	// How many accounts, per plan. Counting reads every account, so the
	// answer is kept 5 minutes in the Worker's cache (free, per data center).
	.get("/stats", async (c) => {
		const key = new Request(new URL("/api/staff/stats?cache", c.req.url));
		const hit = await caches.default.match(key);
		if (hit) return new Response(hit.body, hit);
		const week = new Date(Date.now() - 7 * 24 * 3600 * 1000);
		const [[{ total }], [{ recent }], [{ banned }], paid] = await Promise.all([
			db.select({ total: count() }).from(user),
			db.select({ recent: count() }).from(user).where(gte(user.createdAt, week)),
			db.select({ banned: count() }).from(user).where(eq(user.banned, true)),
			db.select({ plan: subscription.plan, n: sql<number>`count(distinct ${subscription.referenceId})` }).from(subscription).where(inArray(subscription.status, ACTIVE)).groupBy(subscription.plan),
		]);
		const paying = paid.reduce((sum, p) => sum + p.n, 0);
		const res = c.json({
			total,
			recent,
			banned,
			plans: [{ name: "free", label: planNamed("free").label, count: total - paying }, ...paid.map((p) => ({ name: p.plan, label: planNamed(p.plan).label, count: p.n }))],
		});
		res.headers.set("Cache-Control", "max-age=300");
		c.executionCtx.waitUntil(caches.default.put(key, res.clone()));
		return res;
	})

	// Accounts, a page at a time, always through an index (checked with
	// EXPLAIN QUERY PLAN): without a search, newest first by ID; an ID finds
	// that account; anything else is the start of the e-mail, as a range on
	// the e-mail index (Better Auth stores e-mails in lowercase), A to Z.
	.get("/users", async (c) => {
		const q = (c.req.query("q") ?? "").trim().toLowerCase();
		const cursor = c.req.query("cursor");
		const size = pageSize(c.req.query("limit"));
		const cols = { id: user.id, email: user.email, createdAt: user.createdAt, banned: user.banned };
		let found;
		let key: (r: { id: string; email: string }) => string = (r) => r.id;
		if (!q) {
			found = await db.select(cols).from(user).where(cursor ? lt(user.id, cursor) : undefined).orderBy(desc(user.id)).limit(size + 1);
		} else if (/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(q)) {
			found = await db.select(cols).from(user).where(eq(user.id, q)).limit(1);
		} else {
			key = (r) => r.email;
			found = await db
				.select(cols)
				.from(user)
				.where(and(gte(user.email, q), lt(user.email, `${q}\u{10FFFF}`), cursor ? gt(user.email, cursor) : undefined))
				.orderBy(asc(user.email))
				.limit(size + 1);
		}
		const rows = found.slice(0, size);
		const nextCursor = found.length > size ? key(rows[rows.length - 1]) : null;
		const subs = rows.length
			? await db.select({ ref: subscription.referenceId, plan: subscription.plan }).from(subscription).where(and(inArray(subscription.referenceId, rows.map((r) => r.id)), inArray(subscription.status, ACTIVE)))
			: [];
		return c.json({ items: rows.map((r) => ({ ...r, email: hasEmail(r.email) ? r.email : null, plan: planNamed(subs.find((s) => s.ref === r.id)?.plan).label })), nextCursor });
	})

	.get("/users/:id", async (c) => {
		const id = c.req.param("id");
		const [u] = await db.select().from(user).where(eq(user.id, id)).limit(1);
		if (!u) return c.json({ code: "NOT_FOUND" }, 404);
		const [sessions, keys, subs, history, billing] = await Promise.all([
			db.select({ id: session.id, createdAt: session.createdAt, expiresAt: session.expiresAt, ipAddress: session.ipAddress, userAgent: session.userAgent }).from(session).where(eq(session.userId, id)).orderBy(desc(session.createdAt)),
			db.select({ id: passkey.id, name: passkey.name, createdAt: passkey.createdAt, deviceType: passkey.deviceType }).from(passkey).where(eq(passkey.userId, id)),
			db.select().from(subscription).where(eq(subscription.referenceId, id)),
			db.select().from(staffLog).where(eq(staffLog.userId, id)).orderBy(desc(staffLog.createdAt)).limit(20),
			db.select({ id: billingEvent.id, type: billingEvent.type, amount: billingEvent.amount, currency: billingEvent.currency, createdAt: billingEvent.createdAt }).from(billingEvent).where(eq(billingEvent.userId, id)).orderBy(desc(billingEvent.id)).limit(20),
		]);
		const active = subs.find((s) => ACTIVE.includes(s.status));
		const plan = planNamed(active?.plan);
		const [usage, ai] = await Promise.all([usageOf(id), aiUsageOf(id, plan)]);
		const stripeMode = /^[rs]k_live_/.test(c.env.STRIPE_SECRET_KEY) ? "" : "test/";
		return c.json({
			user: { id: u.id, email: hasEmail(u.email) ? u.email : null, emailVerified: u.emailVerified, createdAt: u.createdAt, banned: u.banned, banReason: u.banReason },
			plan: { label: plan.label, limits: plan.limits, status: active?.status ?? null, periodEnd: active?.cancelAt ?? active?.periodEnd ?? null, cancelAtPeriodEnd: !!(active?.cancelAtPeriodEnd || active?.cancelAt) },
			usage,
			ai: { percent: ai.percent, resetsAt: ai.resetsAt, included: plan.limits.aiUsage > 0 },
			billing,
			stripeUrl: u.stripeCustomerId ? `https://dashboard.stripe.com/${stripeMode}customers/${u.stripeCustomerId}` : null,
			sessions,
			passkeys: keys,
			history,
		});
	})

	// "I lost my laptop": every session ends; the person signs in again.
	.post("/users/:id/sessions/revoke", async (c) => {
		const id = c.req.param("id");
		const gone = await db.delete(session).where(eq(session.userId, id)).returning({ id: session.id });
		await log(c.get("staff").email, "sessions.revoke", id, `${gone.length} sessões`);
		return c.json({ revoked: gone.length });
	})

	// A lost passkey: the person signs in by e-mail code and adds another.
	.delete("/users/:id/passkeys/:pid", async (c) => {
		const { id, pid } = c.req.param();
		const gone = await db.delete(passkey).where(and(eq(passkey.id, pid), eq(passkey.userId, id))).returning({ name: passkey.name });
		if (!gone.length) return c.json({ code: "NOT_FOUND" }, 404);
		await log(c.get("staff").email, "passkey.remove", id, gone[0].name ?? undefined);
		return c.json({ ok: true });
	})

	.post("/users/:id/ban", adminOnly, body(z.object({ reason: z.string().trim().min(3).max(300) })), async (c) => {
		const id = c.req.param("id");
		const { reason } = c.req.valid("json");
		await db.update(user).set({ banned: true, banReason: reason }).where(eq(user.id, id));
		await db.delete(session).where(eq(session.userId, id));
		await log(c.get("staff").email, "user.ban", id, reason);
		return c.json({ ok: true });
	})

	.post("/users/:id/unban", adminOnly, async (c) => {
		const id = c.req.param("id");
		await db.update(user).set({ banned: false, banReason: null }).where(eq(user.id, id));
		await log(c.get("staff").email, "user.unban", id);
		return c.json({ ok: true });
	})

	// Deleting an account (LGPD): the subscription ends in Stripe, then the
	// account and everything tied to it (sessions, passkeys) goes.
	.delete("/users/:id", adminOnly, async (c) => {
		const id = c.req.param("id");
		const [u] = await db.select({ email: user.email }).from(user).where(eq(user.id, id)).limit(1);
		if (!u) return c.json({ code: "NOT_FOUND" }, 404);
		const subs = await db.select({ sid: subscription.stripeSubscriptionId }).from(subscription).where(and(eq(subscription.referenceId, id), inArray(subscription.status, ACTIVE)));
		const stripe = new Stripe(c.env.STRIPE_SECRET_KEY, { httpClient: Stripe.createFetchHttpClient() });
		for (const s of subs) if (s.sid) await stripe.subscriptions.cancel(s.sid);
		await db.delete(subscription).where(eq(subscription.referenceId, id));
		await db.delete(user).where(eq(user.id, id));
		// The account's e-mail stays in the log only as the action's detail.
		await log(c.get("staff").email, "user.delete", id, hasEmail(u.email) ? u.email : "sem e-mail (só passkey)");
		return c.json({ ok: true });
	})

	// The team (admin only).
	.get("/team", adminOnly, async (c) => {
		const rows = await db.select().from(staff).orderBy(staff.email);
		return c.json({ owner: c.env.STAFF_OWNER_EMAIL?.toLowerCase() ?? null, staff: rows });
	})
	.post("/team", adminOnly, body(z.object({ email: z.email().transform((e) => e.toLowerCase()), role: z.enum(["support", "admin"]) })), async (c) => {
		const body = c.req.valid("json");
		await db.insert(staff).values({ ...body, addedBy: c.get("staff").email }).onConflictDoUpdate({ target: staff.email, set: { role: body.role } });
		await log(c.get("staff").email, "team.set", null, `${body.email} ${body.role}`);
		return c.json({ ok: true });
	})
	.delete("/team/:email", adminOnly, async (c) => {
		const email = c.req.param("email").toLowerCase();
		if (email === c.get("staff").email) return c.json({ code: "NOT_YOURSELF" }, 400);
		await db.delete(staff).where(eq(staff.email, email));
		await log(c.get("staff").email, "team.remove", null, email);
		return c.json({ ok: true });
	})

	.get("/log", adminOnly, async (c) => {
		const cursor = Number(c.req.query("cursor")) || 0;
		const size = pageSize(c.req.query("limit"));
		// Each entry with its account's e-mail (null once the account is gone),
		// by the account's primary key.
		const found = await db
			.select({ log: staffLog, email: user.email })
			.from(staffLog)
			.leftJoin(user, eq(user.id, staffLog.userId))
			.where(cursor ? lt(staffLog.id, cursor) : undefined)
			.orderBy(desc(staffLog.id))
			.limit(size + 1);
		const rows = found.slice(0, size);
		return c.json({
			items: rows.map((r) => ({ ...r.log, account: r.log.userId ? (r.email === null ? { gone: true, email: null } : { gone: false, email: hasEmail(r.email) ? r.email : null }) : null })),
			nextCursor: found.length > size ? String(rows[rows.length - 1].log.id) : null,
		});
	});
