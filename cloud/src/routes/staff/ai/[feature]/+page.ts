import { error } from "@sveltejs/kit";
import { loaded } from "#lib/load.ts";
import { featureInfo, readAI } from "#lib/staff-ai.ts";

// One AI list, to change it: the owner (STAFF_OWNER_EMAIL) only.
export async function load({ parent, params }) {
	const request = readAI();
	const { me } = await parent();
	if (!me?.owner) {
		request.catch(() => {});
		error(403, "Só o dono do Blup vê as configurações de IA.");
	}
	if (!Object.hasOwn(featureInfo, params.feature)) {
		request.catch(() => {});
		error(404, "Essa lista de IA não existe.");
	}
	return { ai: await loaded(request, parent, null, "Não deu para carregar os modelos de IA.") };
}
