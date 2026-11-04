import { staffApi } from "#lib/staff.ts";
import { loaded } from "#lib/load.ts";

export async function load({ parent }) {
	return { ai: await loaded(staffApi.ai(), parent, null, "Não deu para carregar os modelos de IA.") };
}
