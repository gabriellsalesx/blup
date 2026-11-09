import { redirect } from "@sveltejs/kit";
import { resolve } from "$app/paths";
import { DetailedError } from "hono/client";
import { api } from "#lib/api.ts";
import { nextParam } from "../../../shared/panel-paths";

// The panel is only for whoever is signed in; every page gets the account
// and its plan (loaded again with refreshAll after a change). One request:
// /api/me answers 401 without a session (or for a blocked or deleted
// account), which is the same as signed out.
export async function load({ url }) {
	try {
		return { me: await api.me() };
	} catch (error) {
		// Signed out: to sign-in, coming back here afterwards (?next=).
		if (error instanceof DetailedError && error.statusCode === 401) redirect(307, url.pathname === resolve("/(panel)/account") ? resolve("/(auth)/sign-in") : `${resolve("/(auth)/sign-in")}?${new URLSearchParams({ [nextParam]: url.pathname + url.search })}`);
		throw error;
	}
}
