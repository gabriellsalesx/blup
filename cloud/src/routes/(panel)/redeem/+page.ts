import { api } from "#lib/api.ts";
import { loaded } from "#lib/load.ts";

// AppSumo's redemption page (cloud.getblup.io/redeem?code=...): the level
// of the account and its codes. A signed-out visit goes to sign-in first
// and comes back here with the same ?code= (the layout's ?next=).
export async function load({ url, parent }) {
	const state = await loaded(api.redeemState(), parent, null);
	return { state, code: url.searchParams.get("code")?.trim() ?? "" };
}
