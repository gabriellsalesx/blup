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

// What the Pulse and Deals panels (other subdomains) may call with their
// cookies: their own API, who is signed in, and the sign-in itself (code
// or passkey).
const sharedCorsPaths=String.raw`me(?:\/language)?$|config$|auth\/(?:get-session|sign-out|email-otp\/send-verification-otp|sign-in\/email-otp|passkey\/(?:generate-authenticate-options|verify-authentication|generate-register-options|verify-registration))$`;
const dealsCorsPaths=new RegExp(String.raw`^\/api\/(?:deals(?:\/|$)|`+sharedCorsPaths+")");
export const dealsCorsPath=(pathname:string)=>dealsCorsPaths.test(pathname);
const pulseCorsPaths=new RegExp(String.raw`^\/api\/(?:pulse(?:\/|$)|`+sharedCorsPaths+")");
export const pulseCorsPath=(pathname:string)=>pulseCorsPaths.test(pathname);
// The panel allowed to call this path from this origin (with cookies), if
// any: each panel only reaches its own API and the shared sign-in.
export function panelCorsOrigin(env:{PULSE_APP_URL?:string;DEALS_APP_URL?:string},origin:string|null,pathname:string):string|null{
 if(!origin)return null;
 if(env.PULSE_APP_URL&&origin===env.PULSE_APP_URL&&pulseCorsPath(pathname))return origin;
 if(env.DEALS_APP_URL&&origin===env.DEALS_APP_URL&&dealsCorsPath(pathname))return origin;
 return null;
}
