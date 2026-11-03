import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

// The counts and the first page of accounts, together.
export async function load({ parent }) {
	const [stats, first] = await Promise.all([loaded(staffApi.stats(), parent, null, false), loaded(staffApi.users("", null), parent, null, "Não deu para buscar.")]);
	return { stats, first };
}
