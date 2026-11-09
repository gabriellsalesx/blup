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

// The key of a rate limit for a client address. An IPv6 user usually holds
// a whole /64 (trillions of addresses): counted one by one, a new address a
// request would never hit any limit. So IPv6 counts by its /64; IPv4 as is.
export function clientKey(ip: string | null | undefined): string {
	if (!ip) return "local";
	if (!ip.includes(":")) return ip;
	const [head = "", tail] = ip.toLowerCase().split("::");
	const left = head ? head.split(":") : [];
	const right = tail ? tail.split(":") : [];
	const groups = tail === undefined ? left : [...left, ...Array<string>(Math.max(0, 8 - left.length - right.length)).fill("0"), ...right];
	return `${groups.slice(0, 4).map((g) => g.replace(/^0+(?=.)/, "")).join(":")}::/64`;
}
