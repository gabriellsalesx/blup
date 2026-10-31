import { redirect } from "@sveltejs/kit";
import { authClient } from "#lib/auth.ts";
import { api } from "#lib/api.ts";

// The panel is only for whoever is signed in; every page gets the account
// and its plan (loaded again with invalidateAll after a change).
export async function load({ url }) {
	const { data } = await authClient.getSession();
	if (!data) redirect(307, url.pathname === "/conta" ? "/entrar" : `/entrar?volta=${encodeURIComponent(url.pathname + url.search)}`);
	return { me: await api.me() };
}
