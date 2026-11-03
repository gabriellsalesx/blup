import { redirect } from "@sveltejs/kit";
import { authClient } from "#lib/auth.ts";
import { api } from "#lib/api.ts";

// The panel is only for whoever is signed in; every page gets the account
// and its plan (loaded again with refreshAll after a change).
export async function load({ url }) {
	// Both requests leave together; the account only counts with a session.
	const account = api.me().then((me) => ({ me }), (error: unknown) => ({ error }));
	const { data } = await authClient.getSession();
	if (!data) redirect(307, url.pathname === "/conta" ? "/entrar" : `/entrar?volta=${encodeURIComponent(url.pathname + url.search)}`);
	const result = await account;
	if ("error" in result) throw result.error;
	return { me: result.me };
}
