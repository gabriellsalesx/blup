import { redirect } from "@sveltejs/kit";
import { resolve } from "$app/paths";
import { authClient } from "#lib/auth.ts";
import { api } from "#lib/api.ts";
import { nextParam } from "../../../shared/panel-paths";

// The panel is only for whoever is signed in; every page gets the account
// and its plan (loaded again with refreshAll after a change).
export async function load({ url }) {
	// Both requests leave together; the account only counts with a session.
	const account = api.me().then((me) => ({ me }), (error: unknown) => ({ error }));
	const { data } = await authClient.getSession();
	// Signed out: to sign-in, coming back here afterwards (?next=).
	if (!data) redirect(307, url.pathname === resolve("/(panel)/account") ? resolve("/(auth)/sign-in") : `${resolve("/(auth)/sign-in")}?${new URLSearchParams({ [nextParam]: url.pathname + url.search })}`);
	const result = await account;
	if ("error" in result) throw result.error;
	return { me: result.me };
}
