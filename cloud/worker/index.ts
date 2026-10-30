import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, eq, inArray } from "drizzle-orm";
import { auth, db, type Session } from "./auth";
import { subscription } from "./db/schema";
import { PLANS, planNamed } from "./plans";
import { staffApi } from "./staff";
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
	.get("/plans", (c) => c.json(PLANS.map(({ name, label, priceCents, limits, perks }) => ({ name, label, priceCents, limits, perks }))))
	.get("/me", session, signedIn, async (c) => {
		const { user } = c.get("session")!;
		// The plan of the subscription that is paid (or on trial); else free.
		const [sub] = await db
			.select({ plan: subscription.plan, status: subscription.status, periodEnd: subscription.periodEnd, cancelAtPeriodEnd: subscription.cancelAtPeriodEnd })
			.from(subscription)
			.where(and(eq(subscription.referenceId, user.id), inArray(subscription.status, ["active", "trialing", "past_due"])))
			.limit(1);
		const plan = planNamed(sub?.plan);
		return c.json({
			id: user.id,
			email: hasEmail(user.email) ? user.email : null,
			name: user.name,
			createdAt: user.createdAt,
			plan: { name: plan.name, label: plan.label, limits: plan.limits, status: sub?.status ?? null, renewsAt: sub?.periodEnd ?? null, cancelsAtPeriodEnd: !!sub?.cancelAtPeriodEnd },
		});
	});

app.route("/staff", staffApi);

app.notFound((c) => c.json({ code: "NOT_FOUND" }, 404));
app.onError((err, c) => {
	console.error("[blup]", err);
	return c.json({ code: "INTERNAL" }, 500);
});

export type AppType = typeof routes;
export default app;
