import {language} from "./i18n.svelte";
import {errorMessage} from "./i18n/errors";
import {cloudPaths,nextParam} from "../../shared/panel-paths";
import { createAuthClient } from "better-auth/svelte";
import { emailOTPClient, deviceAuthorizationClient } from "better-auth/client/plugins";
import { passkeyClient } from "@better-auth/passkey/client";

// The Blup sign-in, from the Cloud panel (same origin as the Worker) or
// from the Pulse panel (another subdomain: the Cloud's address, with its
// cookies).
export function createBlupAuthClient(baseURL?: string) {
	return createAuthClient({
		...(baseURL ? { baseURL } : {}),
		basePath: "/api/auth",
		fetchOptions: { ...(baseURL ? { credentials: "include" as const } : {}), onRequest: (ctx) => { ctx.headers.set("x-blup-language", language()); } },
		plugins: [emailOTPClient(), passkeyClient(), deviceAuthorizationClient()],
	});
}
export type BlupAuthClient = ReturnType<typeof createBlupAuthClient>;
export const authClient = createBlupAuthClient();

// Better Auth's errors, in the panel's words.
export const message=errorMessage;

// The person closed the passkey prompt: not an error to show.
export const cancelled = (err: object) => "code" in err && err.code === "AUTH_CANCELLED";

// Where to go after signing in: the page that sent the person to the
// sign-in (?next=), only if it is a page of this same panel, else `home`.
export function returnPath(search: string, home: string): string {
	const back = new URLSearchParams(search).get(nextParam) ?? "";
	if (!back.startsWith("/") || back.startsWith("//") || /[\u0000-\u001f\u007f\\]/.test(back)) return home;
	try {
		const url = new URL(back, location.origin);
		return url.origin === location.origin ? url.pathname + url.search + url.hash : home;
	} catch { return home; }
}
export const afterSignIn = (): string => returnPath(location.search, cloudPaths.account);
