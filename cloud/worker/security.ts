// Browser cookies, including Cloudflare Access cookies, are ambient
// authority. SameSite alone does not protect against another subdomain.
// Studio uses Bearer without an Origin; Stripe authenticates its webhook
// with its signature in the Worker.
export function mutationOriginAllowed(req: Request,trustedOrigins:readonly string[]=[]): boolean {
	const socket = req.headers.get("upgrade")?.toLowerCase() === "websocket";
	if (["GET", "HEAD", "OPTIONS"].includes(req.method) && !socket) return true;
	const url = new URL(req.url);
	if (url.pathname === "/api/auth/stripe/webhook") return true;
	const origin = req.headers.get("origin");
	if (origin !== null) return origin === url.origin || trustedOrigins.includes(origin);
	const site = req.headers.get("sec-fetch-site");
	if (site && site !== "same-origin" && site !== "none") return false;
	if (req.headers.has("cookie") && !req.headers.get("authorization")?.startsWith("Bearer ") && site !== "same-origin") return false;
	return true;
}

// What the Pulse panel (another subdomain) may call with its cookies: its
// own API, who is signed in, and the sign-in itself (code or passkey).
const pulseCorsPaths=/^\/api\/(?:pulse(?:\/|$)|me(?:\/language)?$|config$|auth\/(?:get-session|sign-out|email-otp\/send-verification-otp|sign-in\/email-otp|passkey\/(?:generate-authenticate-options|verify-authentication|generate-register-options|verify-registration))$)/;
export const pulseCorsPath=(pathname:string)=>pulseCorsPaths.test(pathname);
