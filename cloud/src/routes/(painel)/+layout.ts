import { redirect } from "@sveltejs/kit";
import { authClient } from "#lib/auth.ts";
import { api } from "#lib/api.ts";

// The panel is only for whoever is signed in; every page gets the account
// and its plan (loaded again with invalidateAll after a change).
export async function load() {
	const { data } = await authClient.getSession();
	if (!data) redirect(307, "/entrar");
	return { me: await api.me() };
}
