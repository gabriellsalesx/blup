import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { auth, type Session } from "./auth";
import { SEND_CODE_PATH } from "./auth-options";

type App = { Bindings: Cloudflare.Env; Variables: { session: Session | null } };

const app = new Hono<App>().basePath("/api");

// Codes by e-mail cost a send each: a few per minute per address.
app.post(`/auth${SEND_CODE_PATH}`, async (c, next) => {
	const ip = c.req.header("cf-connecting-ip") ?? "local";
	const { success } = await c.env.OTP_LIMIT.limit({ key: `otp:${ip}` });
	if (!success) return c.json({ code: "TOO_MANY_REQUESTS", message: "Muitos pedidos. Espere um minuto." }, 429);
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
	.get("/me", session, signedIn, (c) => {
		const { user } = c.get("session")!;
		return c.json({ id: user.id, email: user.email, name: user.name, createdAt: user.createdAt });
	});

app.notFound((c) => c.json({ code: "NOT_FOUND" }, 404));
app.onError((err, c) => {
	console.error("[blup]", err);
	return c.json({ code: "INTERNAL" }, 500);
});

export type AppType = typeof routes;
export default app;
