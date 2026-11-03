import { api } from "#lib/api.ts";
import { loaded } from "#lib/load.ts";

export async function load({ parent }) {
	return { sites: await loaded(api.sites(), parent, null) };
}
