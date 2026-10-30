import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, desc, eq, inArray, like, or } from "drizzle-orm";
import { createRemoteJWKSet, jwtVerify } from "jose";
import Stripe from "stripe";
import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import { db } from "./auth";
import { passkey, session, staff, staffLog, subscription, user } from "./db/schema";
import { planNamed } from "./plans";

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
	if (c.env.STAFF_DEV_EMAIL && (host === "localhost" || host === "127.0.0.1")) return c.env.STAFF_DEV_EMAIL;
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

// A JSON body checked by a Zod schema; a bad one answers 400 with the
// fields that failed (not Zod's whole report).
const body = <T extends z.ZodType>(schema: T) =>
	zValidator("json", schema, (r, c) => {
		if (!r.success) return c.json({ code: "INVALID", fields: r.error.issues.map((i) => i.path.join(".")) }, 400);
	});

export const staffApi = new Hono<App>()
	.use(onStaff)
	.get("/me", (c) => c.json(c.get("staff")))

	// Accounts: by e-mail (its start) or by ID.
	.get("/users", async (c) => {
		const q = (c.req.query("q") ?? "").trim().toLowerCase();
		const rows = await db
			.select({ id: user.id, email: user.email, createdAt: user.createdAt, banned: user.banned })
			.from(user)
			.where(q ? or(eq(user.id, q), like(user.email, `${q.replace(/[%_]/g, "")}%`)) : undefined)
			.orderBy(desc(user.createdAt))
			.limit(25);
		const subs = rows.length
			? await db.select({ ref: subscription.referenceId, plan: subscription.plan }).from(subscription).where(and(inArray(subscription.referenceId, rows.map((r) => r.id)), inArray(subscription.status, ACTIVE)))
			: [];
		return c.json(rows.map((r) => ({ ...r, plan: planNamed(subs.find((s) => s.ref === r.id)?.plan).label })));
	})

	.get("/users/:id", async (c) => {
		const id = c.req.param("id");
		const [u] = await db.select().from(user).where(eq(user.id, id)).limit(1);
		if (!u) return c.json({ code: "NOT_FOUND" }, 404);
		const [sessions, keys, subs, history] = await Promise.all([
			db.select({ id: session.id, createdAt: session.createdAt, expiresAt: session.expiresAt, ipAddress: session.ipAddress, userAgent: session.userAgent }).from(session).where(eq(session.userId, id)).orderBy(desc(session.createdAt)),
			db.select({ id: passkey.id, name: passkey.name, createdAt: passkey.createdAt, deviceType: passkey.deviceType }).from(passkey).where(eq(passkey.userId, id)),
			db.select().from(subscription).where(eq(subscription.referenceId, id)),
			db.select().from(staffLog).where(eq(staffLog.userId, id)).orderBy(desc(staffLog.createdAt)).limit(20),
		]);
		const active = subs.find((s) => ACTIVE.includes(s.status));
		const stripeMode = c.env.STRIPE_SECRET_KEY.startsWith("sk_live_") ? "" : "test/";
		return c.json({
			user: { id: u.id, email: u.email, emailVerified: u.emailVerified, createdAt: u.createdAt, banned: u.banned, banReason: u.banReason },
			plan: { label: planNamed(active?.plan).label, status: active?.status ?? null, periodEnd: active?.periodEnd ?? null, cancelAtPeriodEnd: !!active?.cancelAtPeriodEnd },
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
		await log(c.get("staff").email, "user.delete", id, u.email);
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

	.get("/log", adminOnly, async (c) => c.json(await db.select().from(staffLog).orderBy(desc(staffLog.createdAt)).limit(100)));
