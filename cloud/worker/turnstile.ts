import { z } from "zod";
// Server-side check of a Turnstile token (siteverify). Better Auth's captcha
// plugin covers the e-mail code; this covers the routes it can't reach.
export async function turnstileOk(secret: string, token: string | undefined, ip: string | undefined): Promise<boolean> {
	if (!token) return false;
	const form = new FormData();
	form.append("secret", secret);
	form.append("response", token);
	if (ip) form.append("remoteip", ip);
	try {
	const r = await fetch("https://challenges.cloudflare.com/turnstile/v0/siteverify", { method: "POST", body: form,signal:AbortSignal.timeout(5000),redirect:"error" });
	if (!r.ok) return false;
	const data = z.object({success:z.literal(true)}).safeParse(await r.json());
	return data.success;
	}catch{return false;}
}
