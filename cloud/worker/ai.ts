import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, eq, gte, sum } from "drizzle-orm";
import { auth, db } from "./auth";
import { aiUsage } from "./db/schema";
import { planOf } from "./account";
import type { Plan } from "./plans";

// AI through Blup Cloud (narration and translation), paid by the plan instead of the
// person's own OpenRouter key. No credits: each request costs what the model
// costs, and the plan has a monthly allowance (25% of what the plan leaves
// after Stripe's fee) that no single week can use more than half of. The
// Studio sees a percentage.
//
// /audio/speech speaks OpenRouter's API, so the Studio's narration client
// talks to it unchanged, with its Blup session as the key.

type App = { Bindings: Cloudflare.Env; Variables: { userId: string } };

// The voice model offered, and what it costs: MAI-Voice-2.1 on OpenRouter,
// US$ 22 per million characters.
export const VOICE_MODEL = "microsoft/mai-voice-2.1";
export const PER_CHAR_MICROS = 22;
const VOICE = /^(pt-BR|en-US|es-MX|es-ES)-[A-Za-z]+:MAI-Voice-2\.1$/;

// The other voice models: any of OpenRouter's that answer MP3, at what
// OpenRouter charges (a dearer one uses the allowance faster). The list
// is read from OpenRouter at most every 6 hours per isolate; without it,
// only the recommended model.
type SpeechModel = { perCharMicros: number; voices: string[] };
let speechCache: { at: number; models: Map<string, SpeechModel> } | null = null;
// Models that answer only raw audio, or are run on the computer instead.
const NOT_OFFERED = /^(google\/gemini|heygen\/)|kokoro|seed/i;
async function speechModels(env: Cloudflare.Env): Promise<Map<string, SpeechModel>> {
	if (speechCache && Date.now() - speechCache.at < 6 * 3600 * 1000) return speechCache.models;
	const models = new Map<string, SpeechModel>();
	try {
		const res = await fetch(`${env.OPENROUTER_URL || "https://openrouter.ai/api/v1"}/models?output_modalities=speech`);
		if (res.ok) {
			const list = (await res.json()) as { data?: { id?: string; pricing?: { prompt?: string }; supported_voices?: string[] }[] };
			for (const m of list.data ?? []) {
				const perChar = Number(m.pricing?.prompt ?? 0) * 1e6;
				if (!m.id || NOT_OFFERED.test(m.id) || !(perChar > 0)) continue;
				models.set(m.id, { perCharMicros: perChar, voices: (m.supported_voices ?? []).filter((v) => typeof v === "string") });
			}
		}
	} catch {}
	if (!models.size) return new Map([[VOICE_MODEL, { perCharMicros: PER_CHAR_MICROS, voices: [] }]]);
	speechCache = { at: Date.now(), models };
	return models;
}

// What a narration request costs per character, or null when the model
// or the voice isn't offered.
export async function voicePrice(env: Cloudflare.Env, model: string, voice: string): Promise<number | null> {
	if (model === VOICE_MODEL) return VOICE.test(voice) ? PER_CHAR_MICROS : null;
	const m = (await speechModels(env)).get(model);
	if (!m || voice.length > 100) return null;
	if (m.voices.length ? !m.voices.includes(voice) : voice !== "") return null;
	return m.perCharMicros;
}

// The translation model (the same as the Studio's internal/translate) and
// its languages. The request is built here: Blup Cloud translates notes,
// it is not a general chat.
export const TRANSLATE_MODEL = "anthropic/claude-haiku-5.5";
const LANG_NAMES: Record<string, string> = { pt: "Brazilian Portuguese", en: "English", es: "Spanish (neutral Latin American)" };
// The Studio sends a note in parts of at most 12000 bytes.
const MAX_TRANSLATE = 16000;
// What a character costs to translate, at most, before the real cost is
// known: Haiku's US$ 0.10 in and US$ 0.50 out per million tokens, about
// three characters a token, the answer as long as the text.
const TRANSLATE_CHAR_MICROS = 0.2;

// The instruction, the same as internal/translate.Prompt.
const translatePrompt = (from: string, to: string) => `You translate Markdown notes of a website from ${LANG_NAMES[from]} to ${LANG_NAMES[to]}.
Rules:
- Output only the translation, nothing before or after it.
- Keep the Markdown exactly: headings, lists, tables, quotes, callouts ("> [!tip] Title": translate the title, never the type in brackets), bold, italics, line breaks and blank lines.
- Never change code blocks, inline code, HTML tags, URLs, the targets of links (the part in parentheses) or of [[wikilinks]] (translate only the text after |), image paths, file names, {.attributes} or math between $.
- Keep names of people, products and brands.
- Write naturally, as a native speaker would, keeping the tone.`;
// Pro's monthly allowance, in millionths of a dollar (1x); the other plans
// are multiples (aiUsage).
const PRO_MONTH_MICROS = 660_000;
// The Studio sends a post in parts of at most 2500 bytes.
const MAX_INPUT = 4000;

const signedIn = createMiddleware<App>(async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (!s) return c.json({ error: { message: "Entre de novo no Blup Cloud." } }, 401);
	c.set("userId", s.user.id);
	await next();
});

export type AIUsage = {
	monthMicros: number; // the plan's allowance this month
	weekCapMicros: number; // the most a week may use
	usedMonthMicros: number;
	usedWeekMicros: number; // the last 7 days
	perCharMicros: number;
	percent: number; // of the month, 0 to 100
	resetsAt: string; // the first day of next month (UTC)
};

const monthStart = (now: Date) => new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));

async function spent(userId: string, since: Date) {
	const [row] = await db
		.select({ total: sum(aiUsage.costMicros) })
		.from(aiUsage)
		.where(and(eq(aiUsage.userId, userId), gte(aiUsage.createdAt, since)));
	return Number(row?.total ?? 0);
}

export async function aiUsageOf(userId: string, plan: Plan): Promise<AIUsage> {
	const now = new Date();
	const month = monthStart(now);
	const week = new Date(now.getTime() - 7 * 86_400_000);
	const [usedMonth, usedWeek] = await Promise.all([spent(userId, month), spent(userId, week)]);
	const monthMicros = plan.limits.aiUsage * PRO_MONTH_MICROS;
	return {
		monthMicros,
		weekCapMicros: Math.floor(monthMicros / 2),
		usedMonthMicros: usedMonth,
		usedWeekMicros: usedWeek,
		perCharMicros: PER_CHAR_MICROS,
		percent: monthMicros ? Math.min(100, Math.round((usedMonth / monthMicros) * 100)) : 0,
		resetsAt: new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1)).toISOString(),
	};
}

// OpenRouter's error shape, so the Studio's client reads it as usual.
const fail = (message: string, code: string) => ({ error: { message, code } });

// Whether the plan's allowance leaves room for a cost now (an error to
// answer otherwise).
async function allowance(userId: string, cost: number) {
	const plan = await planOf(userId);
	if (!plan.limits.aiUsage) return { status: 402 as const, body: fail("O seu plano não inclui IA pelo Blup Cloud.", "NO_AI") };
	const use = await aiUsageOf(userId, plan);
	if (use.usedMonthMicros + cost > use.monthMicros) return { status: 402 as const, body: fail("O uso de IA deste mês acabou. Ele renova no dia 1.", "AI_MONTH_LIMIT") };
	if (use.usedWeekMicros + cost > use.weekCapMicros) return { status: 402 as const, body: fail("Você já usou metade do uso de IA do mês nesta semana. Tente de novo em alguns dias.", "AI_WEEK_LIMIT") };
	return null;
}

export const aiApi = new Hono<App>()
	.use(signedIn)
	.get("/usage", async (c) => {
		const userId = c.get("userId");
		return c.json(await aiUsageOf(userId, await planOf(userId)));
	})
	.post("/audio/speech", async (c) => {
		const userId = c.get("userId");
		const body = await c.req.json<{ model?: string; input?: string; voice?: string }>().catch(() => null);
		const input = typeof body?.input === "string" ? body.input : "";
		const chars = [...input].length;
		const model = typeof body?.model === "string" ? body.model : "";
		const voice = typeof body?.voice === "string" ? body.voice : "";
		const perChar = model ? await voicePrice(c.env, model, voice) : null;
		if (perChar === null || !chars || chars > MAX_INPUT) {
			return c.json(fail("Pedido de narração inválido.", "BAD_REQUEST"), 400);
		}
		const plan = await planOf(userId);
		if (!plan.limits.aiUsage) return c.json(fail("O seu plano não inclui IA pelo Blup Cloud.", "NO_AI"), 402);
		const use = await aiUsageOf(userId, plan);
		const cost = Math.ceil(chars * perChar);
		if (use.usedMonthMicros + cost > use.monthMicros) return c.json(fail("O uso de IA deste mês acabou. Ele renova no dia 1.", "AI_MONTH_LIMIT"), 402);
		if (use.usedWeekMicros + cost > use.weekCapMicros) return c.json(fail("Você já usou metade do uso de IA do mês nesta semana. Tente de novo em alguns dias.", "AI_WEEK_LIMIT"), 402);
		if (!c.env.OPENROUTER_API_KEY) return c.json(fail("A narração pelo Blup Cloud está indisponível agora.", "UNAVAILABLE"), 503);

		const res = await fetch(`${c.env.OPENROUTER_URL || "https://openrouter.ai/api/v1"}/audio/speech`, {
			method: "POST",
			headers: {
				Authorization: `Bearer ${c.env.OPENROUTER_API_KEY}`,
				"Content-Type": "application/json",
				"HTTP-Referer": "https://getblup.io",
				"X-Title": "Blup Cloud",
			},
			body: JSON.stringify({ model, input, ...(voice ? { voice } : {}), response_format: "mp3" }),
		});
		if (!res.ok) {
			console.error("[blup] narração falhou", res.status);
			// Busy or failing upstream: the Studio tries again on 429/5xx.
			return c.json(fail("A narração falhou agora. Tente de novo.", "UPSTREAM"), res.status === 429 ? 429 : 502);
		}
		const audio = await res.arrayBuffer();
		await db.insert(aiUsage).values({ userId, kind: "narration", units: chars, costMicros: cost });
		return new Response(audio, { headers: { "content-type": "audio/mpeg" } });
	})
	.post("/translate", async (c) => {
		const userId = c.get("userId");
		const body = await c.req.json<{ text?: string; from?: string; to?: string }>().catch(() => null);
		const text = typeof body?.text === "string" ? body.text : "";
		const from = typeof body?.from === "string" ? body.from : "", to = typeof body?.to === "string" ? body.to : "";
		if (!text.trim() || text.length > MAX_TRANSLATE || !Object.hasOwn(LANG_NAMES, from) || !Object.hasOwn(LANG_NAMES, to) || from === to) {
			return c.json(fail("Pedido de tradução inválido.", "BAD_REQUEST"), 400);
		}
		const estimate = Math.ceil([...text].length * TRANSLATE_CHAR_MICROS);
		const denied = await allowance(userId, estimate);
		if (denied) return c.json(denied.body, denied.status);
		if (!c.env.OPENROUTER_API_KEY) return c.json(fail("A tradução pelo Blup Cloud está indisponível agora.", "UNAVAILABLE"), 503);

		const res = await fetch(`${c.env.OPENROUTER_URL || "https://openrouter.ai/api/v1"}/chat/completions`, {
			method: "POST",
			headers: {
				Authorization: `Bearer ${c.env.OPENROUTER_API_KEY}`,
				"Content-Type": "application/json",
				"HTTP-Referer": "https://getblup.io",
				"X-Title": "Blup Cloud",
			},
			body: JSON.stringify({
				model: TRANSLATE_MODEL,
				temperature: 0.2,
				usage: { include: true },
				messages: [
					{ role: "system", content: translatePrompt(from, to) },
					{ role: "user", content: text },
				],
			}),
		});
		if (!res.ok) {
			console.error("[blup] tradução falhou", res.status);
			return c.json(fail("A tradução falhou agora. Tente de novo.", "UPSTREAM"), res.status === 429 ? 429 : 502);
		}
		const out = await res.json<{ choices?: { message?: { content?: string }; finish_reason?: string }[]; usage?: { cost?: number } }>();
		const answer = out.choices?.[0]?.message?.content ?? "";
		if (!answer || out.choices?.[0]?.finish_reason === "length") {
			return c.json(fail("A tradução voltou incompleta. Tente de novo.", "UPSTREAM"), 502);
		}
		// What OpenRouter charged (in dollars), else the estimate.
		const cost = typeof out.usage?.cost === "number" ? Math.ceil(out.usage.cost * 1_000_000) : estimate;
		await db.insert(aiUsage).values({ userId, kind: "translation", units: [...text].length, costMicros: cost });
		return c.json({ text: answer });
	});
