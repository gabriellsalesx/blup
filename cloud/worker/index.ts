import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { auth, type Session } from "./auth";
import { activeSubscription, planOf } from "./account";
import { PLANS, SUGGESTED, planNamed } from "./plans";
import { staffApi } from "./staff";
import { syncApi } from "./sync";
import { devicesOf, usageOf } from "./usage";
import { SEND_CODE_PATH, hasEmail } from "./auth-options";
import { turnstileOk } from "./turnstile";

type App = { Bindings: Cloudflare.Env; Variables: { session: Session | null } };

const app = new Hono<App>().basePath("/api");

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

// Each plan lets a number of Blup Studio installs in at once.
app.post("/auth/device/approve", async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (s) {
		const [plan, devices] = await Promise.all([planOf(s.user.id), devicesOf(s.user.id)]);
		if (plan.limits.devices >= 0 && devices >= plan.limits.devices) return c.json({ code: "DEVICES_LIMIT", limit: plan.limits.devices }, 403);
	}
	await next();
});

app.on(["GET", "POST"], "/auth/*", (c) => auth.handler(c.req.raw));

const session = createMiddleware<App>(async (c, next) => {
	c.set("session", await auth.api.getSession({ headers: c.req.raw.headers }));
	await next();
});

const signedIn = createMiddleware<App>(async (c, next) => {
	if (!c.get("session")) return c.json({ code: "UNAUTHORIZED" }, 401);
	await next();
});

const routes = app
	// What the panel needs before signing in (public keys only).
	.get("/config", (c) => c.json({ turnstileSiteKey: c.env.TURNSTILE_SITE_KEY }))
	// The plans, for the panel and the Studio (no Stripe IDs).
	.get("/plans", (c) => c.json(PLANS.map(({ name, label, priceCents, yearlyCents, currency, limits, perks }) => ({ name, label, priceCents, yearlyCents, currency, limits, perks, suggested: name === SUGGESTED }))))
	.get("/me", session, signedIn, async (c) => {
		const { user } = c.get("session")!;
		// The portal cancels by setting cancel_at, not cancel_at_period_end.
		const sub = await activeSubscription(user.id);
		const plan = planNamed(sub?.plan);
		const usage = await usageOf(user.id);
		return c.json({
			usage,
			id: user.id,
			email: hasEmail(user.email) ? user.email : null,
			name: user.name,
			createdAt: user.createdAt,
			plan: { name: plan.name, label: plan.label, limits: plan.limits, status: sub?.status ?? null, renewsAt: sub?.cancelAt ?? sub?.periodEnd ?? null, cancelsAtPeriodEnd: !!(sub?.cancelAtPeriodEnd || sub?.cancelAt) },
		});
	});

app.route("/staff", staffApi);
app.route("/sync", syncApi);

app.notFound((c) => c.json({ code: "NOT_FOUND" }, 404));
app.onError((err, c) => {
	console.error("[blup]", err);
	return c.json({ code: "INTERNAL" }, 500);
});

export type AppType = typeof routes;

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
	const out = new Response(res.body, res);
	for (const [k, v] of Object.entries(SECURITY_HEADERS)) if (!out.headers.has(k)) out.headers.set(k, v);
	return out;
}

// One Worker, two addresses. With STAFF_HOST set (production), the staff
// lives only on its own host (behind Cloudflare Access) and the account
// panel only on the other: each answers "not found" for the other's pages
// and API. Without it (this computer), both live on localhost.
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
} satisfies ExportedHandler<Cloudflare.Env>;
