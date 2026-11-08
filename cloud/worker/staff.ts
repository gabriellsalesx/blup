import { queueAccountPurge } from "./site-cache-purge";
import { Hono, type Context } from "hono";
import { createMiddleware } from "hono/factory";
import { and, asc, count, desc, eq, gt, gte, inArray, lt, sql } from "drizzle-orm";
import { createRemoteJWKSet, jwtVerify } from "jose";

import { z } from "zod";
import { idParam,opaqueId,safeInt,validate } from "./validation";
import { softDelete,restoreAccount,cancelDeletedBilling } from "./lifecycle";
import { db } from "./auth";
import { billingEvent, passkey, session, staff, staffLog, subscription, user, resourceQuota } from "./db/schema";
import { planNamed } from "./plans";
import { GIFT_PLAN, isOwner } from "./account";
import { hasEmail } from "./auth-options";
import { usageOf } from "./usage";
import { addonUsageOf } from "./addon-usage";
import { aiUsageOf, aiFeatures, aiFeatureNames, modelId, modelList, modelListFor, settingFor, chatModels, speechCatalog } from "./ai";

// The staff area (support and administration). In production Cloudflare
// Access stands in front of /staff and /api/staff; here the Worker checks
// Access's token again (it could be reached some other way) and that the
// e-mail is on the staff. The staff never reads notes (they are encrypted
// on the person's computer) and never signs in as a user.

type Role = "support" | "admin";
// owner: the e-mail in STAFF_OWNER_EMAIL (the Worker's secrets, .dev.vars
// here), the only one who sees and changes the AI settings.
type Staff = { email: string; role: Role; owner: boolean };
type App = { Bindings: Cloudflare.Env; Variables: { staff: Staff } };

// The e-mail in STAFF_OWNER_EMAIL, as Access gives e-mails (lowercase).
const ownerEmail=(env:Cloudflare.Env)=>env.STAFF_OWNER_EMAIL?.trim().toLowerCase()||null;

let jwks: ReturnType<typeof createRemoteJWKSet> | undefined;

async function accessEmail(c: { req: { header: (n: string) => string | undefined; url: string }; env: Cloudflare.Env }): Promise<string | null> {
	const host = new URL(c.req.url).hostname;
	// Only on this computer, with STAFF_DEV_EMAIL in .dev.vars.
	if (c.env.STAFF_DEV_EMAIL && (host === "localhost" || host === "127.0.0.1" || host.endsWith(".localhost"))) return c.env.STAFF_DEV_EMAIL;
	const token = c.req.header("cf-access-jwt-assertion");
	if (!token || !c.env.ACCESS_TEAM_DOMAIN || !c.env.ACCESS_AUD) return null;
	jwks ??= createRemoteJWKSet(new URL(`${c.env.ACCESS_TEAM_DOMAIN}/cdn-cgi/access/certs`));
	try {
		const { payload } = await jwtVerify(token, jwks, { issuer: c.env.ACCESS_TEAM_DOMAIN, audience: c.env.ACCESS_AUD, algorithms: ["RS256"] });
		return typeof payload.email === "string" ? payload.email.toLowerCase() : null;
	} catch {
		return null;
	}
}

const onStaff = createMiddleware<App>(async (c, next) => {
	const email = await accessEmail(c);
	if (!email) return c.json({ code: "FORBIDDEN" }, 403);
	let role: Role | null = null;
	const owner = email === ownerEmail(c.env);
	if (owner) role = "admin";
	else {
		const [s] = await db.select({ role: staff.role }).from(staff).where(eq(staff.email, email)).limit(1);
		role = s?.role ?? null;
	}
	if (!role) return c.json({ code: "FORBIDDEN" }, 403);
	c.set("staff", { email, role, owner });
	await next();
});

const adminOnly = createMiddleware<App>(async (c, next) => {
	if (c.get("staff").role !== "admin") return c.json({ code: "ADMIN_ONLY" }, 403);
	await next();
});

const ownerOnly = createMiddleware<App>(async (c, next) => {
	if (!c.get("staff").owner) return c.json({ code: "OWNER_ONLY" }, 403);
	await next();
});

const log = (email: string, action: string, userId: string | null, detail?: string) =>
	db.insert(staffLog).values({ staffEmail: email, action, userId, detail: detail ?? null });
// An action and its log entry go in one D1 batch (a transaction): no change
// without its entry, and no entry for a change that did not happen (the
// entry is written only for a row that exists).
const logFor = (d1: D1Database, email: string, action: string, from: "user" | "passkey", where: string, binds: string[], detail: string | null) =>
	d1.prepare(`INSERT INTO staff_log(staff_email,action,user_id,detail) SELECT ?,?,${from === "user" ? "id" : "user_id"},${from === "passkey" ? "name" : "?"} FROM ${from} WHERE ${where}`).bind(email, action, ...(from === "passkey" ? [] : [detail]), ...binds);

// The owner's own account (its e-mail is STAFF_OWNER_EMAIL): nobody on the
// staff acts against it, so a stolen admin seat cannot lock the owner out.
async function isOwnerAccount(env:Cloudflare.Env,id:string){
	const owner=ownerEmail(env);
	if(!owner)return false;
	return !!await env.DB.prepare("SELECT id FROM user WHERE id=? AND lower(email)=?").bind(id,owner).first();
}
// A refused sensitive action: 403 with its code, and an entry in the log.
type Refusal="OWNER_ONLY"|"OWNER_ACCOUNT";
async function refuse(c:Context<App>,action:string,userId:string|null,code:Refusal,detail?:string){
	await log(c.get("staff").email,`denied.${action}`,userId,detail?`${code} ${detail}`:code);
	return c.json({code},403);
}
// Blocking or deleting the owner's account: refused even to the owner (no
// way to lock oneself out). Ending its sessions or removing its passkeys:
// the owner only.
async function guardOwnerAccount(c:Context<App>,action:string,id:string,ownerToo:boolean){
	if(!await isOwnerAccount(c.env,id))return null;
	if(!ownerToo&&c.get("staff").owner)return null;
	return refuse(c,action,id,"OWNER_ACCOUNT");
}

const statsSchema=z.object({total:z.number().int().min(0),recent:z.number().int().min(0),banned:z.number().int().min(0),plans:z.array(z.object({name:z.string(),label:z.string(),count:z.number().int().min(0)}))});
const ACTIVE = ["active", "trialing", "past_due"];

// Pages by cursor, not OFFSET: D1 bills every row a query reads, and an
// OFFSET reads (and bills) all the rows it skips. The cursor is the last
// item's key: UUIDv7 account IDs and the log's integer IDs both follow
// creation order, so "older than the cursor" is a range on the primary key.
const PAGE = 25;
const pageLimit=safeInt.pipe(z.number().min(1).max(100)).default(PAGE);
const usersQuery=z.object({q:z.string().trim().toLowerCase().max(254).default(""),cursor:z.string().min(1).max(254).optional(),limit:pageLimit});
const logQuery=z.object({cursor:safeInt.pipe(z.number().positive()).optional(),limit:pageLimit});

// A JSON body checked by a Zod schema; a bad one answers 400 with the
// fields that failed (not Zod's whole report).
const body=<T extends z.ZodType>(schema:T)=>validate("json",schema);

export const staffApi = new Hono<App>()
	.use(onStaff)
	.get("/me", (c) => c.json(c.get("staff")))

	// How many accounts, per plan. Counting reads every account, so the
	// answer is kept 5 minutes in the Worker's cache (free, per data center).
	.get("/stats", async (c) => {
		c.header("Cache-Control","no-store");
		const key = new Request(new URL("/api/staff/stats?cache", c.req.url));
		const hit = await caches.default.match(key);
		if (hit) return c.json(statsSchema.parse(await hit.json()),200);
		const week = new Date(Date.now() - 7 * 24 * 3600 * 1000);
		// COUNT always answers one row (the defaults only satisfy the types).
		const [[{ total } = { total: 0 }], [{ recent } = { recent: 0 }], [{ banned } = { banned: 0 }], paid] = await Promise.all([
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
		const cached=res.clone();cached.headers.set("Cache-Control","max-age=300");
		c.executionCtx.waitUntil(caches.default.put(key,cached));
		return res;
	})

	// Accounts, a page at a time, always through an index (checked with
	// EXPLAIN QUERY PLAN): without a search, newest first by ID; an ID finds
	// that account; anything else is the start of the e-mail, as a range on
	// the e-mail index (Better Auth stores e-mails in lowercase), A to Z.
	.get("/users", validate("query",usersQuery), async (c) => {
		const {q,cursor,limit:size}=c.req.valid("query");
		const cols = { id: user.id, email: user.email, emailVerified: user.emailVerified, createdAt: user.createdAt, banned: user.banned };
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
		const last = rows.at(-1);
		const nextCursor = found.length > size && last ? key(last) : null;
		const subs: { ref: string; plan: string }[] = [];
		// The statuses consume three of D1's 100 bindings.
		for (let i = 0; i < rows.length; i += 97) {
			subs.push(...await db.select({ ref: subscription.referenceId, plan: subscription.plan }).from(subscription)
				.where(and(inArray(subscription.referenceId, rows.slice(i, i + 97).map((r) => r.id)), inArray(subscription.status, ACTIVE))));
		}
		return c.json({ items: rows.map(({ emailVerified, ...r }) => ({ ...r, email: hasEmail(r.email) ? r.email : null, plan: isOwner(r.email, emailVerified) ? `${planNamed(GIFT_PLAN).label} (cortesia)` : planNamed(subs.find((s) => s.ref === r.id)?.plan).label })), nextCursor });
	})

	.get("/users/:id", validate("param",idParam), async (c) => {
		const { id } = c.req.valid("param");
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
		const gift = isOwner(u.email, u.emailVerified);
		const plan = planNamed(gift ? GIFT_PLAN : active?.plan);
		const [usage, ai, addons] = await Promise.all([usageOf(id), aiUsageOf(id, plan), addonUsageOf(c.env, id, plan)]);
		const [state]=await db.select({deletedAt:resourceQuota.deletedAt,purgeAfter:resourceQuota.purgeAfter,purgeStarted:resourceQuota.purgeStarted}).from(resourceQuota).where(eq(resourceQuota.userId,id)).limit(1);
		const stripeMode = /^[rs]k_live_/.test(c.env.STRIPE_SECRET_KEY) ? "" : "test/";
		const owner=ownerEmail(c.env);
		return c.json({
			ownerAccount:!!owner&&u.email.toLowerCase()===owner,
			deletion:{requestedAt:state?.deletedAt?new Date(state.deletedAt).toISOString():null,purgeAfter:state?.purgeAfter?new Date(state.purgeAfter).toISOString():null,restorable:!!state?.deletedAt&&!!state.purgeAfter&&state.purgeAfter>Date.now()&&!state.purgeStarted},
			user: { id: u.id, email: hasEmail(u.email) ? u.email : null, emailVerified: u.emailVerified, createdAt: u.createdAt, banned: u.banned, banReason: u.banReason },
			plan: { label: gift ? `${plan.label} (cortesia)` : plan.label, gift, limits: plan.limits, status: active?.status ?? null, periodEnd: active?.cancelAt ?? active?.periodEnd ?? null, cancelAtPeriodEnd: !!(active?.cancelAtPeriodEnd || active?.cancelAt) },
			usage,
			ai: { percent: ai.percent, resetsAt: ai.resetsAt, included: plan.limits.aiUsage > 0 },
			// Blup Pulse and Blup Deals, when the plan includes them.
			addons,
			billing,
			stripeUrl: u.stripeCustomerId ? `https://dashboard.stripe.com/${stripeMode}customers/${u.stripeCustomerId}` : null,
			sessions,
			passkeys: keys,
			history,
		});
	})

	// "I lost my laptop": every session ends; the person signs in again.
	.post("/users/:id/sessions/revoke", validate("param",idParam), async (c) => {
		const { id } = c.req.valid("param");
		const denied=await guardOwnerAccount(c,"sessions.revoke",id,false);
		if(denied)return denied;
		const [, gone] = await c.env.DB.batch<{ id: string }>([
			c.env.DB.prepare("INSERT INTO staff_log(staff_email,action,user_id,detail) VALUES(?,'sessions.revoke',?,(SELECT count(*) || ' sessões' FROM session WHERE user_id=?))").bind(c.get("staff").email, id, id),
			c.env.DB.prepare("DELETE FROM session WHERE user_id=? RETURNING id").bind(id),
		]);
		await c.env.SYNC_LIVE.getByName(id).revoke();
		return c.json({ revoked: gone?.results.length ?? 0 });
	})

	// A lost passkey: the person signs in by e-mail code and adds another.
	.delete("/users/:id/passkeys/:pid", validate("param",idParam.extend({pid:opaqueId})), async (c) => {
		const { id, pid } = c.req.valid("param");
		const denied=await guardOwnerAccount(c,"passkey.remove",id,false);
		if(denied)return denied;
		const [, gone] = await c.env.DB.batch([
			logFor(c.env.DB, c.get("staff").email, "passkey.remove", "passkey", "id=? AND user_id=?", [pid, id], null),
			c.env.DB.prepare("DELETE FROM passkey WHERE id=? AND user_id=? RETURNING id").bind(pid, id),
		]);
		if (!gone?.results.length) return c.json({ code: "NOT_FOUND" }, 404);
		return c.json({ ok: true });
	})

	.post("/users/:id/ban", adminOnly, validate("param",idParam), body(z.strictObject({ reason: z.string().trim().min(3).max(300) })), async (c) => {
		const { id } = c.req.valid("param");
		const { reason } = c.req.valid("json");
		const denied=await guardOwnerAccount(c,"user.ban",id,true);
		if(denied)return denied;
		const [found] = await c.env.DB.batch([
			c.env.DB.prepare("UPDATE user SET banned=1,ban_reason=? WHERE id=? RETURNING id").bind(reason, id),
			c.env.DB.prepare("DELETE FROM session WHERE user_id=?").bind(id),
			logFor(c.env.DB, c.get("staff").email, "user.ban", "user", "id=?", [id], reason),
		]);
		if (!found?.results.length) return c.json({ code: "NOT_FOUND" }, 404);
		await queueAccountPurge(c.env.DB,id).run();
		await c.env.SYNC_LIVE.getByName(id).revoke();
		return c.json({ ok: true });
	})

	.post("/users/:id/unban", adminOnly, validate("param",idParam), async (c) => {
		const { id } = c.req.valid("param");
		const [found] = await c.env.DB.batch([
			c.env.DB.prepare("UPDATE user SET banned=0,ban_reason=NULL WHERE id=? RETURNING id").bind(id),
			logFor(c.env.DB, c.get("staff").email, "user.unban", "user", "id=?", [id], null),
		]);
		if (!found?.results.length) return c.json({ code: "NOT_FOUND" }, 404);
		await queueAccountPurge(c.env.DB,id).run();
		return c.json({ ok: true });
	})

	// Recoverable deletion: no impersonation and no subscription restarted.
	.delete("/users/:id",adminOnly,validate("param",idParam),async c=>{
	 const {id}=c.req.valid("param");
	 const denied=await guardOwnerAccount(c,"user.delete",id,true);
	 if(denied)return denied;
	 if(!await c.env.DB.prepare("SELECT id FROM user WHERE id=?").bind(id).first())return c.json({code:"NOT_FOUND"},404);
	 await softDelete(c.env,id,Date.now(),[logFor(c.env.DB,c.get("staff").email,"user.delete","user","id=?",[id],null)]);c.executionCtx.waitUntil(cancelDeletedBilling(c.env));
	 return c.json({ok:true});
	})
	.post("/users/:id/restore",validate("param",idParam),async c=>{
	 const {id}=c.req.valid("param");
	 if(!await restoreAccount(c.env.DB,id,Date.now(),c.get("staff").email))return c.json({code:"NOT_FOUND"},404);
	 return c.json({ok:true});
	})

	// The team (admin only). Making, unmaking or removing an admin: the
	// owner only (checked by the e-mail in Access's token, never the body).
	.get("/team", adminOnly, async (c) => {
		const rows = await db.select().from(staff).orderBy(staff.email);
		return c.json({ owner: ownerEmail(c.env), staff: rows });
	})
	.post("/team", adminOnly, body(z.strictObject({ email: z.email().max(254).transform((e) => e.toLowerCase()), role: z.enum(["support", "admin"]) })), async (c) => {
		const body = c.req.valid("json");
		const me = c.get("staff");
		if (!me.owner) {
			if (body.role === "admin") return refuse(c, "team.set", null, "OWNER_ONLY", `${body.email} ${body.role}`);
			const [current] = await db.select({ role: staff.role }).from(staff).where(eq(staff.email, body.email)).limit(1);
			if (current?.role === "admin") return refuse(c, "team.set", null, "OWNER_ONLY", `${body.email} ${body.role}`);
		}
		await db.batch([
			// Not the owner: an admin made in the meantime keeps the role.
			db.insert(staff).values({ ...body, addedBy: me.email }).onConflictDoUpdate({ target: staff.email, set: { role: body.role }, ...(me.owner ? {} : { setWhere: eq(staff.role, "support") }) }),
			log(c.get("staff").email, "team.set", null, `${body.email} ${body.role}`),
		]);
		return c.json({ ok: true });
	})
	.delete("/team/:email", adminOnly, validate("param",z.object({email:z.email().max(254).toLowerCase()})), async (c) => {
		const { email } = c.req.valid("param");
		const me = c.get("staff");
		if (email === me.email) return c.json({ code: "NOT_YOURSELF" }, 400);
		if (!me.owner) {
			const [current] = await db.select({ role: staff.role }).from(staff).where(eq(staff.email, email)).limit(1);
			if (current?.role === "admin") return refuse(c, "team.remove", null, "OWNER_ONLY", email);
		}
		// Not the owner: only a support seat goes, even if it changed meanwhile.
		await db.batch([db.delete(staff).where(me.owner ? eq(staff.email, email) : and(eq(staff.email, email), eq(staff.role, "support"))), log(me.email, "team.remove", null, email)]);
		return c.json({ ok: true });
	})

	// The AI models of each feature, in order (the owner only): a text
	// feature's first model answers and the others take over when it fails;
	// a voice list is what Blup Studio offers. Each must be in OpenRouter's
	// catalog (when it can be read); the price is shown.
	.get("/ai", ownerOnly, async (c) => {
		const [chat, speech] = await Promise.all([chatModels(c.env), speechCatalog(c.env)]);
		const features = await Promise.all(aiFeatureNames.map(async (feature) => {
			const spec = aiFeatures[feature], { models, recommended } = await settingFor(c.env.DB, feature);
			return {
				feature, kind: spec.kind, max: spec.max, recommended,
				models: models.map((id) => {
					if (spec.kind === "speech") { const m = speech.live ? speech.models.get(id) : undefined; return { id, name: m?.name ?? null, prompt: null, completion: null, perThousand: m ? m.perCharMicros / 1000 : null }; }
					const m = chat.get(id);
					return { id, name: m?.name ?? null, prompt: m ? m.prompt * 1e6 : null, completion: m ? m.completion * 1e6 : null, perThousand: null };
				}),
			};
		}));
		return c.json({ features, catalog: chat.size > 0, speechCatalog: speech.live });
	})
	.put("/ai/:feature", ownerOnly, validate("param", z.object({ feature: z.enum(aiFeatureNames) })), body(z.strictObject({ models: modelList, recommended: modelId.nullable().default(null) })), async (c) => {
		const { feature } = c.req.valid("param"), { models, recommended } = c.req.valid("json"), spec = aiFeatures[feature];
		if (!modelListFor(feature).safeParse(models).success) return c.json({ code: "TOO_MANY", max: spec.max }, 400);
		// The marked one must be in the list.
		if (recommended !== null && !models.includes(recommended)) return c.json({ code: "NOT_IN_LIST" }, 400);
		if (spec.kind === "speech") {
			const speech = await speechCatalog(c.env);
			const unknown = speech.live ? models.filter((m) => !speech.models.has(m)) : [];
			if (unknown.length) return c.json({ code: "UNKNOWN_MODEL", models: unknown }, 400);
		} else {
			const catalog = await chatModels(c.env);
			const unknown = catalog.size ? models.filter((m) => !catalog.has(m)) : [];
			if (unknown.length) return c.json({ code: "UNKNOWN_MODEL", models: unknown }, 400);
			// Each feature runs with reasoning off, which these models refuse.
			const thinking = models.filter((m) => catalog.get(m)?.thinks);
			if (thinking.length) return c.json({ code: "REASONING_MODEL", models: thinking }, 400);
		}
		await c.env.DB.batch([
			c.env.DB.prepare("INSERT INTO ai_model_setting(feature,models,recommended,updated_by,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(feature) DO UPDATE SET models=excluded.models,recommended=excluded.recommended,updated_by=excluded.updated_by,updated_at=excluded.updated_at").bind(feature, JSON.stringify(models), recommended, c.get("staff").email, Date.now()),
			c.env.DB.prepare("INSERT INTO staff_log(staff_email,action,detail) VALUES(?,'ai.models',?)").bind(c.get("staff").email, `${feature}: ${models.join(", ")}${recommended ? ` (marcado: ${recommended})` : ""}`),
		]);
		return c.json({ ok: true });
	})

	.get("/log", adminOnly, validate("query",logQuery), async (c) => {
		const {cursor,limit:size}=c.req.valid("query");
		// Each entry with its account's e-mail (null once the account is gone),
		// by the account's primary key.
		const found = await db
			.select({ log: staffLog, email: user.email })
			.from(staffLog)
			.leftJoin(user, eq(user.id, staffLog.userId))
			.where(cursor ? lt(staffLog.id, cursor) : undefined)
			.orderBy(desc(staffLog.id))
			.limit(size + 1);
		const rows = found.slice(0, size), last = rows.at(-1);
		return c.json({
			items: rows.map((r) => ({ ...r.log, account: r.log.userId ? (r.email === null ? { gone: true, email: null } : { gone: false, email: hasEmail(r.email) ? r.email : null }) : null })),
			nextCursor: found.length > size && last ? String(last.log.id) : null,
		});
	});
