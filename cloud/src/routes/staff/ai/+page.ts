import { error } from "@sveltejs/kit";
import { loaded } from "#lib/load.ts";
import { aiTabs, readAI, type AITab } from "#lib/staff-ai.ts";

// Only the owner (STAFF_OWNER_EMAIL) sees the AI settings; the API refuses
// anyone else too. The tab to open comes from the address (?tab=translate).
export async function load({ parent, url }) {
	const request = readAI();
	const { me } = await parent();
	if (!me?.owner) {
		request.catch(() => {});
		error(403, "Só o dono do Blup vê as configurações de IA.");
	}
	const asked = url.searchParams.get("tab");
	const tab: AITab = aiTabs.find((t) => t === asked) ?? "voice";
	return { tab, ai: await loaded(request, parent, null, "Não deu para carregar os modelos de IA.") };
}
