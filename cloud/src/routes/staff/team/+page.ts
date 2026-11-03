import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

export async function load({ parent }) {
	return { team: await loaded(staffApi.team(), parent, null, "Não deu para carregar a equipe.") };
}
