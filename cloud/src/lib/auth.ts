import {language} from "./i18n.svelte";
import {errorMessage} from "./i18n/errors";
import { createAuthClient } from "better-auth/svelte";
import { emailOTPClient, deviceAuthorizationClient } from "better-auth/client/plugins";
import { passkeyClient } from "@better-auth/passkey/client";
import { stripeClient } from "@better-auth/stripe/client";

// Same origin as the Worker: /api/auth.
export const authClient = createAuthClient({
	basePath: "/api/auth",
	fetchOptions:{onRequest:ctx=>{ctx.headers.set("x-blup-language",language());}},
	plugins: [emailOTPClient(), passkeyClient(), stripeClient({ subscription: true }), deviceAuthorizationClient()],
});

// Better Auth's errors, in the panel's words.
export const message=errorMessage;

// The person closed the passkey prompt: not an error to show.
export const cancelled = (err: object) => "code" in err && err.code === "AUTH_CANCELLED";

// Where to go after signing in: the page that sent the person to /entrar
// (?volta=), only if it is a page of this panel.
export function afterSignIn(): string {
	const back = new URLSearchParams(location.search).get("volta") ?? "";
	if (!back.startsWith("/") || back.startsWith("//") || /[\u0000-\u001f\u007f\\]/.test(back)) return "/conta";
	try {
		const url = new URL(back, location.origin);
		return url.origin === location.origin ? url.pathname + url.search + url.hash : "/conta";
	} catch { return "/conta"; }
}
