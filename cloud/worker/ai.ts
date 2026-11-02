import { accountAllowed } from "./lifecycle";
import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, eq, gte, sum } from "drizzle-orm";
import { auth, db } from "./auth";
import { aiUsage } from "./db/schema";
import { planOf } from "./account";
import { z } from "zod";
import { reserve, release, settle } from "./quota";
import { validate } from "./validation";
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
const VOICE_MODEL = "microsoft/mai-voice-2.1";
const PER_CHAR_MICROS = 22;
const VOICE = /^(pt-BR|en-US|es-MX|es-ES)-[A-Za-z]+:MAI-Voice-2\.1$/;

// The other voice models: any of OpenRouter's that answer MP3, at what
// OpenRouter charges (a dearer one uses the allowance faster). The list
// is read from OpenRouter at most every 6 hours per isolate; without it,
// only the recommended model.
const catalogSchema = z.object({ data: z.array(z.object({ id: z.string().min(1).max(200), pricing: z.object({prompt:z.string().max(32)}), supported_voices:z.array(z.string().max(100)).max(1000).optional() })).max(5000) });
const costSchema = z.number().finite().min(0).max(Number.MAX_SAFE_INTEGER / 1e6);
const usageSchema = z.object({usage:z.object({cost:costSchema}).optional()});
const outputSchema = z.object({choices:z.array(z.object({message:z.object({content:z.string().max(256_000).nullable().optional()}).optional(),finish_reason:z.string().nullable().optional()})).max(16).optional()});

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
			const list = catalogSchema.parse(await res.json());
			for (const m of list.data ?? []) {
				const perChar = Number(m.pricing?.prompt ?? 0) * 1e6;
				if (!m.id || NOT_OFFERED.test(m.id) || !(perChar > 0) || !Number.isFinite(perChar)) continue;
				models.set(m.id, { perCharMicros: perChar, voices: (m.supported_voices ?? []).filter((v) => typeof v === "string") });
			}
		}
	} catch {}
	if (!models.size) models.set(VOICE_MODEL,{perCharMicros:PER_CHAR_MICROS,voices:[]});
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
const TRANSLATE_MODEL = "anthropic/claude-haiku-5.5";
const LANG_NAMES: Record<string, string> = { pt: "Brazilian Portuguese", en: "English", es: "Spanish (neutral Latin American)" };
// The Studio sends a note in parts of at most 12000 bytes.
const MAX_TRANSLATE = 16000;
// What a character costs to translate, at most, before the real cost is
// known: Haiku's US$ 0.10 in and US$ 0.50 out per million tokens, about
// three characters a token, the answer as long as the text.


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
	if (!s || !await accountAllowed(c.env.DB,s.user.id)) return c.json(fail("Entre de novo no Blup Cloud.","UNAUTHORIZED"),401);
	c.set("userId", s.user.id);
	await next();
});

type AIUsage = {
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
const fail = (message: string, code: string) => ({ code,error: { message, code } });

const speechInput = z.object({model:z.string().min(1).max(200),input:z.string().min(1).max(MAX_INPUT*2).refine(s=>[...s].length<=MAX_INPUT),voice:z.string().max(100).default(""),response_format:z.literal("mp3").optional()}).strict();
const translationInput = z.object({text:z.string().trim().min(1).max(MAX_TRANSLATE),from:z.enum(["pt","en","es"]),to:z.enum(["pt","en","es"])}).strict().refine(b=>b.from!==b.to);
async function aiReservation(env:Cloudflare.Env,userId:string,cost:number) {
 const plan=await planOf(userId);
 if(!plan.limits.aiUsage) return {error:fail("O seu plano não inclui IA pelo Blup Cloud.","NO_AI")};
 const month=plan.limits.aiUsage*PRO_MONTH_MICROS;
 const hold=await reserve(env.DB,userId,"ai",cost,month,Math.floor(month/2));
 if(hold) return {hold};
 const use=await aiUsageOf(userId,plan);
 return {error:fail("O uso de IA disponível acabou. Tente novamente depois.",use.usedMonthMicros+cost>month?"AI_MONTH_LIMIT":"AI_WEEK_LIMIT")};
}
function providerCost(value:unknown,fallback:number) {
 const usage=usageSchema.safeParse(value);
 return usage.success && usage.data.usage?.cost!==undefined ? Math.ceil(usage.data.usage.cost*1e6) : fallback;
}
export const aiApi = new Hono<App>()
 .use(signedIn)
 .get("/usage",async c=>c.json(await aiUsageOf(c.get("userId"),await planOf(c.get("userId")))))
 .post("/audio/speech",validate("json",speechInput),async c=>{
  const userId=c.get("userId"),{model,input,voice}=c.req.valid("json");
  const perChar=await voicePrice(c.env,model,voice),chars=[...input].length;
  if(perChar===null) return c.json(fail("Pedido de narração inválido.","BAD_REQUEST"),400);
  const cost=Math.ceil(chars*perChar),admission=await aiReservation(c.env,userId,cost);
  if(!admission.hold) return c.json(admission.error,402);
  const hold=admission.hold;
  try {
   if(!c.env.OPENROUTER_API_KEY) return c.json(fail("A narração está indisponível agora.","UNAVAILABLE"),503);
   const res=await fetch(`${c.env.OPENROUTER_URL||"https://openrouter.ai/api/v1"}/audio/speech`,{method:"POST",signal:AbortSignal.timeout(120_000),redirect:"error",headers:{Authorization:`Bearer ${c.env.OPENROUTER_API_KEY}`,"Content-Type":"application/json","HTTP-Referer":"https://getblup.io","X-Title":"Blup Cloud"},body:JSON.stringify({model,input,...(voice?{voice}:{}),response_format:"mp3"})});
   // Audio responses may expose the provider cost as a numeric header.
   const headerCost=z.coerce.number().pipe(costSchema).safeParse(res.headers.get("x-openrouter-cost")??res.headers.get("x-usage-cost")??undefined);
   let actual=headerCost.success?Math.ceil(headerCost.data*1e6):cost;
   if(!res.ok||res.headers.get("content-type")?.includes("application/json")){
    const errorBody:unknown=await res.json().catch(()=>null);
    actual=providerCost(errorBody,headerCost.success?actual:0);
    if(actual) await settle(c.env.DB,hold,[c.env.DB.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros) VALUES (?,'narration',?,?)").bind(userId,chars,actual)]);
    return c.json(fail("A narração falhou agora. Tente de novo.","UPSTREAM"),res.status===429?429:502);
   }
   await settle(c.env.DB,hold,[c.env.DB.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros) VALUES (?,'narration',?,?)").bind(userId,chars,actual)]);
   return new Response(res.body,{headers:{"content-type":"audio/mpeg"}});
  }catch{ return c.json(fail("A narração falhou agora. Tente de novo.","UPSTREAM"),502); }
  finally{await release(c.env.DB,hold);}
 })
 .post("/translate",validate("json",translationInput),async c=>{
  const userId=c.get("userId"),{text,from,to}=c.req.valid("json"),chars=[...text].length;
  const inputTokens=Math.ceil(new TextEncoder().encode(text).length/3);
  const maxTokens=Math.max(32,inputTokens*2);
  // Reserve a conservative input bound and the entire allowed output.
  const estimate=Math.ceil((new TextEncoder().encode(text+translatePrompt(from,to)).length)*0.1+maxTokens*0.5);
  const admission=await aiReservation(c.env,userId,estimate);
  if(!admission.hold) return c.json(admission.error,402);
  const hold=admission.hold;
  try{
   if(!c.env.OPENROUTER_API_KEY) return c.json(fail("A tradução está indisponível agora.","UNAVAILABLE"),503);
   const res=await fetch(`${c.env.OPENROUTER_URL||"https://openrouter.ai/api/v1"}/chat/completions`,{method:"POST",signal:AbortSignal.timeout(120_000),redirect:"error",headers:{Authorization:`Bearer ${c.env.OPENROUTER_API_KEY}`,"Content-Type":"application/json","HTTP-Referer":"https://getblup.io","X-Title":"Blup Cloud"},body:JSON.stringify({model:TRANSLATE_MODEL,temperature:0.2,max_tokens:maxTokens,usage:{include:true},messages:[{role:"system",content:translatePrompt(from,to)},{role:"user",content:text}]})});
   const raw:unknown=await res.json().catch(()=>null);
   const cost=providerCost(raw,res.ok?estimate:0);
   // Charge before validating content: truncated/malformed answers can cost.
   if(res.ok||usageSchema.safeParse(raw).success) await settle(c.env.DB,hold,[c.env.DB.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros) VALUES (?,'translation',?,?)").bind(userId,chars,cost)]);
   const out=outputSchema.safeParse(raw),choice=out.success?out.data.choices?.[0]:undefined;
   if(!res.ok || !choice?.message?.content || choice.finish_reason==="length") return c.json(fail("A tradução voltou incompleta. Tente de novo.","UPSTREAM"),res.status===429?429:502);
   return c.json({text:choice.message.content});
  }catch{return c.json(fail("A tradução falhou agora. Tente de novo.","UPSTREAM"),502);}
  finally{await release(c.env.DB,hold);}
 });
