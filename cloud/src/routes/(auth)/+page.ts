import { redirect } from "@sveltejs/kit";
import { resolve } from "$app/paths";
import { authClient } from "#lib/auth.ts";

// The start: the account when signed in, else the sign-in page, decided
// before anything draws.
export async function load() {
	const { data } = await authClient.getSession().catch(() => ({ data: null }));
	redirect(307, data ? resolve("/(panel)/account") : resolve("/(auth)/sign-in"));
}
