import { staffApi, type AIFeature, type AIModels } from "#lib/staff.ts";

// The staff's AI pages: the tabs of /staff/ai, the lists in each, and what
// each list is called and shows (its mark: Blup Cloud's blup, OpenRouter's
// logo, or Deals' coral blup).
export const aiTabs = ["voice", "translate", "deals"] as const;
export type AITab = (typeof aiTabs)[number];
export type AIMarkKind = "blup" | "openrouter" | "deals";

export const tabLabels: Record<AITab, string> = { voice: "Narração", translate: "Tradução", deals: "Blup Deals" };

export const tabAbout: Record<AITab, string> = {
	voice: "As vozes que o Blup Studio mostra na hora de narrar um post. Abra uma lista para adicionar modelos e marcar o recomendado.",
	translate: "Quem traduz as notas e os posts do Blup Studio. Abra uma lista para adicionar modelos e marcar o principal.",
	deals: "Os textos que a IA escreve no Blup Deals. Abra uma função para adicionar modelos e marcar o principal.",
};

export const tabFeatures: Record<AITab, AIFeature[]> = {
	voice: ["studio_voice", "studio_voice_own"],
	translate: ["studio_translate", "studio_translate_own"],
	deals: ["deals_describe", "deals_curate", "deals_extract", "deals_suggest", "deals_translate"],
};

export const tabOf = (feature: AIFeature): AITab => aiTabs.find((t) => tabFeatures[t].includes(feature)) ?? "voice";

export const featureInfo: Record<AIFeature, { title: string; lead: string; mark: AIMarkKind }> = {
	studio_voice: { title: "Blup Cloud", lead: "Vozes que aparecem em \"Blup Cloud, incluso no plano\". O uso sai da cota de IA do plano da pessoa.", mark: "blup" },
	studio_voice_own: { title: "OpenRouter, chave própria", lead: "Vozes que aparecem para quem conectou a própria chave do OpenRouter. A pessoa paga direto ao OpenRouter.", mark: "openrouter" },
	studio_translate: { title: "Blup Cloud", lead: "A tradução feita pelo Blup Cloud. O uso sai da cota de IA do plano da pessoa.", mark: "blup" },
	studio_translate_own: { title: "OpenRouter, chave própria", lead: "A tradução de quem conectou a própria chave do OpenRouter. A pessoa paga direto ao OpenRouter.", mark: "openrouter" },
	deals_describe: { title: "Descrição do post", lead: "O botão \"Escrever com IA\" do Colar link e do Vídeo novo, e a descrição automática nas caçadas. Texto curto: um modelo barato basta.", mark: "deals" },
	deals_curate: { title: "Triagem da caçada de lojas", lead: "Nome curto, se combina com a busca e categoria, para até 10 ofertas de uma vez. Responde em JSON.", mark: "deals" },
	deals_extract: { title: "Leitura do espelho", lead: "Cupom, descrição com palavras próprias, nome curto e categoria de um post espelhado. Responde em JSON.", mark: "deals" },
	deals_suggest: { title: "Sugestão de buscas", lead: "Oito buscas a partir do assunto do canal. Responde em JSON.", mark: "deals" },
	deals_translate: { title: "Tradução do post", lead: "A frase, a descrição e o texto extra para um canal em outro idioma. Responde em JSON.", mark: "deals" },
};

// What the marked model is called in a list: a voice list's recommended
// one, a text list's main one.
export const markName = (kind: string) => (kind === "speech" ? "Recomendado" : "Principal");

// The lists, read again once after a short wait when the first read fails
// (the Cloud restarting, a dropped connection): only a second failure is
// shown as an error.
export async function readAI(): Promise<AIModels> {
	try {
		return await staffApi.ai();
	} catch {
		await new Promise((done) => setTimeout(done, 2500));
		return staffApi.ai();
	}
}
