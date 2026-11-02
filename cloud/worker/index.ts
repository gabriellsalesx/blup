import { flushSitePurges } from "./site-cache-purge";
import { expireTransfers } from "./sync-transfers";
import { SyncError,drainSyncGarbage } from "./sync-storage";
import { expireDomains } from "./domain-downgrade";
import { z } from "zod";
import { HTTPException } from "hono/http-exception";
import { boundedStream,idParam,safeInt,validate,opaqueId } from "./validation";
import { accountAllowed,softDelete,purgeAccounts,cancelDeletedBilling,downgradeNotices } from "./lifecycle";
import { approveDevice,approvalBody } from "./devices";
import { expireReservations } from "./quota";
import { cleanSites } from "./sites-cleanup";
import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, desc, eq, gt } from "drizzle-orm";
import { auth, db, type Session } from "./auth";
import { activeSubscription, planOf, isOwner, GIFT_PLAN } from "./account";
import { PLANS, SUGGESTED, planNamed } from "./plans";
import { staffApi } from "./staff";
import { syncApi } from "./sync";
import { sitesApi } from "./sites";
import { forgetOld, remindersApi, sendDue } from "./reminders";
import { cleanUp } from "./cleanup";
import { aiApi, aiUsageOf } from "./ai";
import { devicesOf, isStudio, usageOf } from "./usage";
import { session as sessionTable } from "./db/schema";
import { SEND_CODE_PATH, hasEmail } from "./auth-options";
import { turnstileOk } from "./turnstile";
import { mutationOriginAllowed } from "./security";

type App = { Bindings: Cloudflare.Env; Variables: { session: Session } };

const app = new Hono<App>().basePath("/api");

const headersSchema=z.object({authorization:z.string().max(8192).optional(),cookie:z.string().max(16384).optional(),origin:z.url().max(2048).optional(),"cf-connecting-ip":z.union([z.ipv4(),z.ipv6()]).optional(),"user-agent":z.string().max(1024).optional(),"cf-access-jwt-assertion":z.string().max(16384).optional(),"x-captcha-response":z.string().max(4096).optional(),"x-blup-upload-ticket":opaqueId.optional(),"content-length":safeInt.optional(),upgrade:z.enum(["websocket"]).optional()});
app.use("*",async(c,next)=>{
 const checked=headersSchema.safeParse(Object.fromEntries(c.req.raw.headers));
 if(!checked.success)return c.json({code:"INVALID"},400);
 if(c.req.raw.body && !/\/(?:files|blocks)\/[0-9a-f]{64}$/.test(c.req.path)){
  const max=/\/sync\/vaults\/[^/]+\/(?:versions|prepare)$/.test(c.req.path)?34*1024*1024:/\/sites\/[^/]+\/deploys$/.test(c.req.path)?8*1024*1024:96*1024;
  const bounded=boundedStream(c.req.raw.body,max,checked.data["content-length"]);
  const init={body:bounded,duplex:"half"};c.req.raw=new Request(c.req.raw,init);
 }
 await next();
});

app.use("*", async (c, next) => {
	if (!mutationOriginAllowed(c.req.raw)) return c.json({ code: "ORIGIN_NOT_ALLOWED" }, 403);
	await next();
	// A WebSocket (101) goes as it is: copying the response would drop the socket.
	if (c.res.status !== 101 && !c.res.headers.has("cache-control")) c.header("cache-control", "no-store");
});

// Codes by e-mail cost a send each: a few per minute per address.
app.post(`/auth${SEND_CODE_PATH}`, async (c, next) => {
	const ip = c.req.header("cf-connecting-ip") ?? "local";
	const { success } = await c.env.OTP_LIMIT.limit({ key: `otp:${ip}` });
	if (!success) return c.json({ code: "TOO_MANY_REQUESTS", message: "Muitos pedidos. Espere um minuto." }, 429);
	await next();
});

// A new account by passkey (no session yet): the same guards as the e-mail
// code, a few per minute per address and Turnstile.
app.post("/auth/passkey/verify-registration", async (c, next) => {
	if (await auth.api.getSession({ headers: c.req.raw.headers })) return next();
	const ip = c.req.header("cf-connecting-ip");
	const { success } = await c.env.OTP_LIMIT.limit({ key: `passkey:${ip ?? "local"}` });
	if (!success) return c.json({ code: "TOO_MANY_REQUESTS", message: "Muitos pedidos. Espere um minuto." }, 429);
	if (!(await turnstileOk(c.env.TURNSTILE_SECRET_KEY, c.req.header("x-captcha-response"), ip))) return c.json({ code: "VERIFICATION_FAILED", message: "Captcha" }, 403);
	await next();
});

// Linking an e-mail is for accounts made with a passkey. Changing a real
// e-mail goes through support for now (it would need the current one
// confirmed first).
app.on("POST", ["/auth/email-otp/request-email-change", "/auth/email-otp/change-email"], async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (!s) return c.json({ code: "UNAUTHORIZED" }, 401);
	if (hasEmail(s.user.email)) return c.json({ code: "EMAIL_CHANGE_SUPPORT", message: "Para trocar o e-mail, fale com o suporte." }, 403);
	if (c.req.path.endsWith("request-email-change")) {
		const { success } = await c.env.OTP_LIMIT.limit({ key: `otp:${c.req.header("cf-connecting-ip") ?? "local"}` });
		if (!success) return c.json({ code: "TOO_MANY_REQUESTS", message: "Muitos pedidos. Espere um minuto." }, 429);
	}
	await next();
});

// Paying needs a real e-mail (receipts, and a way back into the account).
app.post("/auth/subscription/upgrade", async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (s && !hasEmail(s.user.email)) return c.json({ code: "EMAIL_REQUIRED", message: "Vincule um e-mail antes de assinar." }, 403);
	await next();
});

// Approval and the pending slot are committed in one D1 transaction.
app.post("/auth/device/approve",validate("json",approvalBody),async c=>{
 const s=await auth.api.getSession({headers:c.req.raw.headers});
 if(!s||!await accountAllowed(c.env.DB,s.user.id))return c.json({code:"UNAUTHORIZED"},401);
 const result=await approveDevice(c.env.DB,s.user.id,c.req.valid("json").userCode,(await planOf(s.user.id)).limits.devices);
 return result==="OK"?c.json({success:true}):c.json({code:result},result==="DEVICES_LIMIT"?403:400);
});
app.post("/auth/delete-user",validate("json",z.object({}).strict()),async c=>{
 const s=await auth.api.getSession({headers:c.req.raw.headers});
 if(!s||!await accountAllowed(c.env.DB,s.user.id))return c.json({code:"UNAUTHORIZED"},401);
 if(!Number.isFinite(new Date(s.session.createdAt).getTime())||Date.now()-new Date(s.session.createdAt).getTime()>3600_000)return c.json({code:"SESSION_EXPIRED"},403);
 await softDelete(c.env,s.user.id);
 c.executionCtx.waitUntil(cancelDeletedBilling(c.env));
 return c.json({success:true,message:"Conta excluída."});
});

// Costly control operations use 120/minute per account and IP. Data
// transfer has storage quotas and a separate 500/10s per-account fuse.
app.use("*",async(c,next)=>{
 const path=c.req.path,method=c.req.method;
 const costly=(method==="POST"&&/^\/api\/(?:ai\/(?:audio\/speech|translate)|sites\/?|sync\/vaults)$/.test(path)) || /\/api\/sites\/[^/]+\/domain$/.test(path) || method==="GET"&&path==="/api/sites/check" || method==="PUT"&&/^\/api\/reminders\/?$/.test(path) || /\/api\/sync\/vaults\/[^/]+\/(?:prepare|history)$/.test(path) || method==="PUT"&&path==="/api/sites/retained" || method==="DELETE"&&/^\/api\/sites\/[^/]+$/.test(path);
 const bulk=!costly&&(/^\/api\/sync\//.test(path)||/^\/api\/sites\/[^/]+\/(?:files|missing|deploys)(?:\/|$)/.test(path));
 if(!costly&&!bulk)return next();
 const s=await auth.api.getSession({headers:c.req.raw.headers});if(!s)return next();
 const binding=costly?c.env.ABUSE_LIMIT:c.env.BULK_LIMIT;
 const keys=costly?[`account:${s.user.id}`,`ip:${c.req.header("cf-connecting-ip")??"local"}`]:[`account:${s.user.id}`];
 const hits=await Promise.all(keys.map(key=>binding.limit({key})));
 if(hits.some(r=>!r.success)){c.header("Retry-After",costly?"60":"10");return c.json({code:"TOO_MANY_REQUESTS"},429);}
 await next();
});

app.on(["GET", "POST"], "/auth/*", (c) => auth.handler(c.req.raw));

const session = createMiddleware<App>(async(c,next)=>{
 const s=await auth.api.getSession({headers:c.req.raw.headers});
 if(!s||!await accountAllowed(c.env.DB,s.user.id))return c.json({code:"UNAUTHORIZED"},401);
 c.set("session",s);await next();
});

const routes = app
	// What the panel needs before signing in (public keys only).
	.get("/config", (c) => c.json({ turnstileSiteKey: c.env.TURNSTILE_SITE_KEY }))
	// The plans, for the panel and the Studio (no Stripe IDs).
	.get("/plans", (c) => c.json(PLANS.map(({ name, label, priceCents, yearlyCents, currency, limits, perks }) => ({ name, label, priceCents, yearlyCents, currency, limits, perks, suggested: name === SUGGESTED }))))
	.get("/me", session, async (c) => {
		const { user } = c.get("session");
		// The portal cancels by setting cancel_at, not cancel_at_period_end.
		const sub = await activeSubscription(user.id);
		const gift = isOwner(user.email, user.emailVerified);
		const plan = planNamed(gift ? GIFT_PLAN : sub?.plan);
		const [usage, ai] = await Promise.all([usageOf(user.id), aiUsageOf(user.id, plan)]);
		return c.json({
			usage,
			ai: { percent: ai.percent, resetsAt: ai.resetsAt, included: plan.limits.aiUsage > 0 },
			id: user.id,
			email: hasEmail(user.email) ? user.email : null,
			name: user.name,
			createdAt: user.createdAt,
			plan: { name: plan.name, label: plan.label, limits: plan.limits, gift, status: sub?.status ?? null, renewsAt: sub?.cancelAt ?? sub?.periodEnd ?? null, cancelsAtPeriodEnd: !!(sub?.cancelAtPeriodEnd || sub?.cancelAt) },
		});
	})
	// The Blup Studio installs signed in (each is a session), to see and
	// disconnect them.
	.get("/devices", session, async (c) => {
		const { user } = c.get("session");
		const rows = await db
			.select({ id: sessionTable.id, agent: sessionTable.userAgent, createdAt: sessionTable.createdAt, lastSeen: sessionTable.updatedAt })
			.from(sessionTable)
			.where(and(eq(sessionTable.userId, user.id), isStudio, gt(sessionTable.expiresAt, new Date())))
			.orderBy(desc(sessionTable.updatedAt));
		// "Blup Studio (Linux; my-pc)" -> "my-pc · Linux".
		const name = (agent: string | null) => {
			const m = /\((.+?);\s*(.+)\)/.exec(agent ?? "");
			return m ? `${m[2]} · ${m[1]}` : "Blup Studio";
		};
		return c.json({ devices: rows.map((r) => ({ id: r.id, name: name(r.agent), createdAt: r.createdAt, lastSeen: r.lastSeen })) });
	})
	.delete("/devices/:id", session, validate("param",idParam), async (c) => {
		const { user } = c.get("session");
		await db.delete(sessionTable).where(and(eq(sessionTable.id, c.req.param("id")), eq(sessionTable.userId, user.id), isStudio));
		return c.json({ ok: true });
	})
	.route("/staff",staffApi)
	.route("/sync",syncApi)
	.route("/sites",sitesApi)
	.route("/reminders",remindersApi)
	.route("/ai",aiApi);

app.notFound((c) => c.json({ code: "NOT_FOUND" }, 404));
app.onError(async (err, c) => {
	if(err instanceof SyncError)return c.json({code:err.code,...(err.head===undefined?{}:{head:err.head})},err.status);
	if(err instanceof HTTPException){
  const response=err.getResponse();
  if(response.headers.get("content-type")?.includes("application/json"))return response;
  return c.req.path.startsWith("/api/ai/")?c.json({code:"INVALID",error:{code:"INVALID",message:"Pedido inválido."}},err.status):c.json({code:err.status===400?"INVALID":err.status===413?"TOO_LARGE":"REQUEST_FAILED"},err.status);
 }
	// Database/provider errors may contain query bindings or private text.
	console.error("[blup] request_failed", err.name);
	return c.json({ code: "INTERNAL" }, 500);
});

export type AppType = typeof routes;

export { SyncLive } from "./live";

// The security headers of static/_headers, on what the Worker itself
// serves (Cloudflare applies _headers only to assets it serves directly).
const SECURITY_HEADERS: Record<string, string> = {
	"Content-Security-Policy": "frame-ancestors 'none'",
	"X-Frame-Options": "DENY",
	"X-Content-Type-Options": "nosniff",
	"Referrer-Policy": "strict-origin-when-cross-origin",
	"Permissions-Policy": "camera=(), microphone=(), geolocation=(), payment=()",
	"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
};

function secure(res: Response): Response {
	if (res.status === 101) return res; // a WebSocket: copying it would drop the socket
	const out = new Response(res.body, res);
	for (const [k, v] of Object.entries(SECURITY_HEADERS)) if (!out.headers.has(k)) out.headers.set(k, v);
	return out;
}

// One Worker, two addresses. With STAFF_HOST set (production), the staff
// lives only on its own host (behind Cloudflare Access) and the account
// panel only on the other: each answers "not found" for the other's pages
// and API. Without it (this computer), both live on localhost.
// The daily cron (wrangler.jsonc); the other one is the reminders'.
const DAILY = "0 4 * * *";

export default {
	async fetch(req: Request, env: Cloudflare.Env, ctx: ExecutionContext) {
		const url = new URL(req.url);
		const staffApi = url.pathname.startsWith("/api/staff");
		const staffPage = url.pathname === "/staff" || url.pathname.startsWith("/staff/");
		if (env.STAFF_HOST) {
			const onStaff = url.hostname === env.STAFF_HOST;
			const api = url.pathname.startsWith("/api/");
			if (onStaff ? api && !staffApi : staffApi || staffPage) return secure(Response.json({ code: "NOT_FOUND" }, { status: 404 }));
		}
		if (url.pathname.startsWith("/api/")) return secure(await app.fetch(req, env, ctx));
		return secure(await env.ASSETS.fetch(req));
	},
	// Every 5 minutes, the calendar's reminders by e-mail; once a day, the
	// clean-up of old versions and unused blocks.
	async scheduled(controller: ScheduledController, env: Cloudflare.Env, ctx: ExecutionContext) {
		if (controller.cron === DAILY) ctx.waitUntil(Promise.all([cleanUp(env), forgetOld(env),cancelDeletedBilling(env).then(()=>purgeAccounts(env))]));
		else ctx.waitUntil(Promise.all([sendDue(env),expireReservations(env.DB),cleanSites(env),cancelDeletedBilling(env),downgradeNotices(env),expireDomains(env),expireTransfers(env),drainSyncGarbage(env)]).then(()=>flushSitePurges(env)));
	},
} satisfies ExportedHandler<Cloudflare.Env>;
