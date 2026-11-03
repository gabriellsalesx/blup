import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

export async function load({ parent }) {
	return { first: await loaded(staffApi.log(null), parent, null, "Não deu para carregar o registro.") };
}
