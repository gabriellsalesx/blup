import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

// AppSumo's codes: the counts for the whole staff; importing (the owner)
// and refunds (admins) are on the page, and the API checks the same.
export async function load({ parent }) {
	return { stats: await loaded(staffApi.appsumo(), parent, null, "Não deu para carregar o AppSumo.") };
}
