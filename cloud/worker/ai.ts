import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, eq, gte, sum } from "drizzle-orm";
import { auth, db } from "./auth";
import { aiUsage } from "./db/schema";
import { planOf } from "./account";
import type { Plan } from "./plans";

// AI through Blup Cloud (narration for now), paid by the plan instead of the
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
const VOICE = /^(pt-BR|en-US)-[A-Za-z]+:MAI-Voice-2\.1$/;
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
		if (body?.model !== VOICE_MODEL || !VOICE.test(body.voice ?? "") || !chars || chars > MAX_INPUT) {
			return c.json(fail("Pedido de narração inválido.", "BAD_REQUEST"), 400);
		}
		const plan = await planOf(userId);
		if (!plan.limits.aiUsage) return c.json(fail("O seu plano não inclui IA pelo Blup Cloud.", "NO_AI"), 402);
		const use = await aiUsageOf(userId, plan);
		const cost = chars * PER_CHAR_MICROS;
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
			body: JSON.stringify({ model: VOICE_MODEL, input, voice: body.voice, response_format: "mp3" }),
		});
		if (!res.ok) {
			console.error("[blup] narração falhou", res.status, (await res.text()).slice(0, 300));
			// Busy or failing upstream: the Studio tries again on 429/5xx.
			return c.json(fail("A narração falhou agora. Tente de novo.", "UPSTREAM"), res.status === 429 ? 429 : 502);
		}
		const audio = await res.arrayBuffer();
		await db.insert(aiUsage).values({ userId, kind: "narration", units: chars, costMicros: cost });
		return new Response(audio, { headers: { "content-type": "audio/mpeg" } });
	});
