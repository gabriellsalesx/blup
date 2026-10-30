import { redirect } from "@sveltejs/kit";
import { authClient } from "#lib/auth.ts";

// Only for whoever is signed in.
export async function load() {
	const { data } = await authClient.getSession();
	if (!data) redirect(307, "/entrar");
}
