import { accountAllowed } from "./lifecycle";
import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, eq, gte, sum } from "drizzle-orm";
import { auth, db } from "./auth";
import { aiUsage } from "./db/schema";
import { planOf } from "./account";
import { z } from "zod";
import { reserve, release, settle } from "./quota";
import { boundedJson, validate } from "./validation";
// The most OpenRouter's model catalog and one chat answer may weigh.
const CATALOG_MAX = 8 * 1024 * 1024, ANSWER_MAX = 1024 * 1024;
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
// Read model by model: one OpenRouter lists oddly (null voices, a field
// too long) is left out instead of losing the whole list.
const catalogSchema = z.object({ data: z.array(z.unknown()).max(5000) });
const speechSchema = z.object({ id: z.string().min(1).max(200), name: z.string().max(200).nullish(), pricing: z.object({prompt:z.string().max(32)}), supported_voices:z.array(z.string().max(100)).max(5000).nullish() });
const costSchema = z.number().finite().min(0).max(Number.MAX_SAFE_INTEGER / 1e6);
const usageSchema = z.object({usage:z.object({cost:costSchema}).optional()});
// OpenRouter answers a failure after the provider took the request with a
// 200 whose body is only {"error":...} (no choices): that one is not an
// answer, so without a usage.cost it is not charged at the estimate.
const errorBody = z.object({ error: z.unknown().refine((e) => e !== undefined && e !== null) });
const answered = (res: Response, raw: unknown) => res.ok && !errorBody.safeParse(raw).success;
const outputSchema = z.object({choices:z.array(z.object({message:z.object({content:z.string().max(256_000).nullable().optional()}).optional(),finish_reason:z.string().nullable().optional()})).max(16).optional()});

export type SpeechModel = { name: string; perCharMicros: number; voices: string[] };
let speechCache: { at: number; url: string; models: Map<string, SpeechModel>; live: boolean } | null = null;
const openRouter = (env: Cloudflare.Env) => env.OPENROUTER_URL || "https://openrouter.ai/api/v1";
// Models that answer only raw audio, or are run on the computer instead.
const NOT_OFFERED = /^(google\/gemini|heygen\/)|kokoro|seed/i;
// live: read from OpenRouter now (else only the recommended model, from
// here, and the staff can't check the IDs).
export async function speechCatalog(env: Cloudflare.Env): Promise<{ models: Map<string, SpeechModel>; live: boolean }> {
	if (speechCache && speechCache.url === openRouter(env) && Date.now() - speechCache.at < 6 * 3600 * 1000) return speechCache;
	const models = new Map<string, SpeechModel>();
	try {
		const res = await fetch(`${openRouter(env)}/models?output_modalities=speech`, { signal: AbortSignal.timeout(15_000) });
		if (res.ok) {
			const list = catalogSchema.parse(await boundedJson(res, CATALOG_MAX));
			for (const item of list.data) {
				const parsed = speechSchema.safeParse(item);
				if (!parsed.success) continue;
				const m = parsed.data;
				const perChar = Number(m.pricing?.prompt ?? 0) * 1e6;
				if (!m.id || NOT_OFFERED.test(m.id) || !(perChar > 0) || !Number.isFinite(perChar)) continue;
				models.set(m.id, { name: m.name ?? m.id, perCharMicros: perChar, voices: (m.supported_voices ?? []).filter((v) => typeof v === "string") });
			}
		}
	} catch {}
	const live = models.size > 0;
	if (!live) models.set(VOICE_MODEL,{name:"Microsoft AI: MAI-Voice-2.1",perCharMicros:PER_CHAR_MICROS,voices:[]});
	speechCache = { at: Date.now(), url: openRouter(env), models, live };
	return speechCache;
}

// What a narration request costs per character, or null when the model
// isn't one the staff offers on the plans (studio_voice) or the voice
// isn't one of the model's.
export async function voicePrice(env: Cloudflare.Env, model: string, voice: string): Promise<number | null> {
	if (!(await modelsFor(env.DB, "studio_voice")).includes(model)) return null;
	// The recommended model at its own price: its voices of the four
	// locales, and any other OpenRouter lists for it (the Studio offers
	// pt-PT, en-GB... from that same list).
	if (model === VOICE_MODEL && VOICE.test(voice)) return PER_CHAR_MICROS;
	const m = (await speechCatalog(env)).models.get(model);
	if (model === VOICE_MODEL) return m?.voices.includes(voice) ? PER_CHAR_MICROS : null;
	if (!m || voice.length > 100) return null;
	if (m.voices.length ? !m.voices.includes(voice) : voice !== "") return null;
	return m.perCharMicros;
}

// The translation's languages (its models are the owner's list,
// studio_translate). The request is built here: Blup Cloud translates
// notes, it is not a general chat.
const LANG_NAMES: Record<string, string> = { pt: "Brazilian Portuguese", en: "English", es: "Spanish (neutral Latin American)" };
// The Studio sends a note in parts of at most 12000 bytes.
const MAX_TRANSLATE = 16000;

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

const speechInput = z.strictObject({model:z.string().min(1).max(200),input:z.string().min(1).max(MAX_INPUT*2).refine(s=>[...s].length<=MAX_INPUT),voice:z.string().max(100).default(""),response_format:z.literal("mp3").optional()});
const translationInput = z.strictObject({text:z.string().trim().min(1).max(MAX_TRANSLATE),from:z.enum(["pt","en","es"]),to:z.enum(["pt","en","es"])}).refine(b=>b.from!==b.to);
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
// The features that write text with a list of models the staff sets
// (/staff/ai): each one's default list, used until an admin changes it.
// OpenRouter takes the list as models: the first answers, the next ones
// take over when it fails, and the one that answered is the one charged.
// deals_describe: the "write with AI" button and the hunts' descriptions;
// deals_curate: a store hunt's finds sorted (short names, what fits the
// search, the category); deals_extract: a mirrored post's coupon and
// details; deals_suggest: words for a store hunt; deals_translate: a post
// for a channel in another language.
// Blup Studio's lists: studio_translate and studio_voice are the models
// the plans pay for (Blup Cloud translates and narrates with them);
// studio_translate_own and studio_voice_own are what the Studio offers to
// someone with their own OpenRouter key. A voice list is the models shown
// to choose from; a text list is the order OpenRouter tries them in. Every
// list starts empty: the owner adds each model (an empty one turns its
// feature off).
type FeatureSpec = { kind: "chat" | "speech"; max: number };
export const aiFeatures = {
	deals_describe: { kind: "chat", max: 5 },
	deals_curate: { kind: "chat", max: 5 },
	deals_extract: { kind: "chat", max: 5 },
	deals_suggest: { kind: "chat", max: 5 },
	deals_translate: { kind: "chat", max: 5 },
	studio_translate: { kind: "chat", max: 5 },
	studio_translate_own: { kind: "chat", max: 5 },
	studio_voice: { kind: "speech", max: 10 },
	studio_voice_own: { kind: "speech", max: 10 },
} as const satisfies Record<string, FeatureSpec>;
export type AIFeature = keyof typeof aiFeatures;
export const aiFeatureNames = Object.keys(aiFeatures) as [AIFeature, ...AIFeature[]];
export const modelId = z.string().regex(/^[a-z0-9][a-z0-9._-]{0,63}\/[a-z0-9][a-z0-9._:-]{0,99}$/);
const listOf = (max: number) => z.array(modelId).max(max).refine((l) => new Set(l).size === l.length);
// The longest list any feature takes; each one's own limit is checked too.
export const modelList = listOf(Math.max(...Object.values(aiFeatures).map((f) => f.max)));
export const modelListFor = (feature: AIFeature) => listOf(aiFeatures[feature].max);
// A feature's list and the model the owner marked in it (a voice list's
// recommended one, a text list's main one), null until one is marked.
export type AISetting = { models: string[]; recommended: string | null };
export async function settingFor(db: D1Database, feature: AIFeature): Promise<AISetting> {
	const row = await db.prepare("SELECT models,recommended FROM ai_model_setting WHERE feature=?").bind(feature).first<{ models: string; recommended: string | null }>();
	try {
		const parsed = modelListFor(feature).safeParse(JSON.parse(row?.models ?? "null"));
		if (parsed.success) return { models: parsed.data, recommended: row?.recommended && parsed.data.includes(row.recommended) ? row.recommended : null };
	} catch {}
	return { models: [], recommended: null };
}
// The order the models are tried in: the marked one first, then the list.
export async function modelsFor(db: D1Database, feature: AIFeature): Promise<string[]> {
	const { models, recommended } = await settingFor(db, feature);
	return recommended ? [recommended, ...models.filter((m) => m !== recommended)] : models;
}
// OpenRouter's text models and their prices (US$ per token), read at most
// every 6 hours per isolate, to hold enough of the allowance before a
// request (the dearest of the list) and to show the staff what each costs.
// thinks: the model always reasons before answering (OpenRouter's
// reasoning.mandatory) and refuses reasoning turned off.
const chatCatalog = z.object({ data: z.array(z.object({ id: z.string().min(1).max(200), name: z.string().max(200).optional(), pricing: z.object({ prompt: z.string().max(32), completion: z.string().max(32) }), reasoning: z.object({ mandatory: z.boolean().optional() }).nullish().catch(null) })).max(10000) });
export type ChatModel = { id: string; name: string; prompt: number; completion: number; thinks: boolean };
let chatCache: { at: number; url: string; models: Map<string, ChatModel> } | null = null;
export async function chatModels(env: Cloudflare.Env): Promise<Map<string, ChatModel>> {
	if (chatCache && chatCache.url === openRouter(env) && Date.now() - chatCache.at < 6 * 3600 * 1000) return chatCache.models;
	const models = new Map<string, ChatModel>();
	try {
		const res = await fetch(`${env.OPENROUTER_URL || "https://openrouter.ai/api/v1"}/models`, { signal: AbortSignal.timeout(15_000) });
		if (res.ok) for (const m of chatCatalog.parse(await boundedJson(res, CATALOG_MAX)).data) {
			const prompt = Number(m.pricing.prompt), completion = Number(m.pricing.completion);
			if (Number.isFinite(prompt) && Number.isFinite(completion) && prompt >= 0 && completion >= 0) models.set(m.id, { id: m.id, name: m.name ?? m.id, prompt, completion, thinks: m.reasoning?.mandatory === true });
		}
	} catch {}
	if (models.size) chatCache = { at: Date.now(), url: openRouter(env), models };
	return models;
}
// A short text written by the AI for a feature, paid by the person's plan
// (the same allowance as narration and translation). The hold is the
// dearest model of the list at the most it may write; the charge, what
// OpenRouter says it cost.
export type AIText = { text: string; model: string } | { code: "NO_AI" | "AI_MONTH_LIMIT" | "AI_WEEK_LIMIT" | "UNAVAILABLE" | "UPSTREAM"; status: 402 | 429 | 502 | 503 };
// json: the answer is a JSON object (OpenRouter's response_format).
// Reasoning is off by default: these are short answers, and a model that
// reasons first spends the max_tokens on it and answers nothing (the staff
// list only models that can turn it off). think: a little of it, for a
// task the person waits for and that comes out poor without it (room for
// it goes in maxTokens).
export async function aiText(env: Cloudflare.Env, userId: string, feature: AIFeature, input: { system: string; user: string; maxTokens: number; json?: boolean; temperature?: number; think?: boolean }): Promise<AIText> {
	if (!env.OPENROUTER_API_KEY) return { code: "UNAVAILABLE", status: 503 };
	const models = await modelsFor(env.DB, feature);
	if (!models.length) return { code: "UNAVAILABLE", status: 503 }; // none added yet
	const catalog = await chatModels(env);
	// Without the catalog, a generous price per token (US$ 20 per million).
	const price = (id: string) => catalog.get(id) ?? { prompt: 0.00002, completion: 0.00002 };
	const inTokens = Math.ceil(new TextEncoder().encode(input.system + input.user).length / 3) + 16;
	const estimate = Math.ceil(Math.max(...models.map((m) => inTokens * price(m).prompt + input.maxTokens * price(m).completion)) * 1e6) + 1;
	const admission = await aiReservation(env, userId, estimate);
	if (!admission.hold) return { code: admission.error.code as "NO_AI" | "AI_MONTH_LIMIT" | "AI_WEEK_LIMIT", status: 402 };
	const hold = admission.hold;
	try {
		const res = await fetch(`${env.OPENROUTER_URL || "https://openrouter.ai/api/v1"}/chat/completions`, { method: "POST", signal: AbortSignal.timeout(60_000), redirect: "manual", headers: { Authorization: `Bearer ${env.OPENROUTER_API_KEY}`, "Content-Type": "application/json", "HTTP-Referer": "https://getblup.io", "X-OpenRouter-Title": "Blup Cloud" }, body: JSON.stringify({ models, temperature: input.temperature ?? 0.7, max_tokens: input.maxTokens, reasoning: { effort: input.think ? "low" : "none" }, ...(input.json ? { response_format: { type: "json_object" } } : {}), messages: [{ role: "system", content: input.system }, { role: "user", content: input.user }] }) });
		const raw: unknown = await boundedJson(res, ANSWER_MAX).catch(() => null);
		const cost = providerCost(raw, answered(res, raw) ? estimate : 0);
		if (res.ok || usageSchema.safeParse(raw).success) await settle(env.DB, hold, [env.DB.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros) VALUES (?,?,?,?)").bind(userId, feature, input.maxTokens, cost)]);
		const out = outputSchema.safeParse(raw), choice = out.success ? out.data.choices?.[0] : undefined;
		const model = z.object({ model: z.string().max(200) }).safeParse(raw);
		if (!res.ok || !choice?.message?.content?.trim() || (input.json && choice.finish_reason === "length")) return { code: "UPSTREAM", status: res.status === 429 ? 429 : 502 };
		return { text: choice.message.content.trim(), model: model.success ? model.data.model : models[0] ?? "" };
	} catch { return { code: "UPSTREAM", status: 502 }; }
	finally { await release(env.DB, hold); }
}

// What Blup Studio offers, read by it without signing in (someone with
// only an OpenRouter key sees the staff's choice too): the voice models
// and the translation models, for the plans and for one's own key. Only
// model IDs, which aren't secret.
export const studioAIApi = new Hono<{ Bindings: Cloudflare.Env }>()
 .get("/",async c=>{
  // Voices keep the owner's order and say which one is recommended (none
  // until marked); translation is the order to try, the main one first.
  const [voicePlan,voiceOwn]=await Promise.all((["studio_voice","studio_voice_own"] as const).map(f=>settingFor(c.env.DB,f)));
  const [translatePlan,translateOwn]=await Promise.all((["studio_translate","studio_translate_own"] as const).map(f=>modelsFor(c.env.DB,f)));
  c.header("Cache-Control","public, max-age=300");
  return c.json({voice:{plan:voicePlan,own:voiceOwn},translate:{plan:translatePlan,own:translateOwn}});
 });

export const aiApi = new Hono<App>()
 .use(signedIn)
 .get("/usage",async c=>c.json(await aiUsageOf(c.get("userId"),await planOf(c.get("userId")))))
 .post("/audio/speech",validate("json",speechInput),async c=>{
  const userId=c.get("userId"),{model,input,voice}=c.req.valid("json");
  // No AI on the plan: refused first; then a list the owner left empty.
  if(!(await planOf(userId)).limits.aiUsage) return c.json(fail("O seu plano não inclui IA pelo Blup Cloud.","NO_AI"),402);
  if(!(await modelsFor(c.env.DB,"studio_voice")).length) return c.json(fail("A narração pelo Blup Cloud está sem modelo agora.","UNAVAILABLE"),503);
  const perChar=await voicePrice(c.env,model,voice),chars=[...input].length;
  if(perChar===null) return c.json(fail("Pedido de narração inválido.","BAD_REQUEST"),400);
  const cost=Math.ceil(chars*perChar),admission=await aiReservation(c.env,userId,cost);
  if(!admission.hold) return c.json(admission.error,402);
  const hold=admission.hold;
  try {
   if(!c.env.OPENROUTER_API_KEY) return c.json(fail("A narração está indisponível agora.","UNAVAILABLE"),503);
   const res=await fetch(`${c.env.OPENROUTER_URL||"https://openrouter.ai/api/v1"}/audio/speech`,{method:"POST",signal:AbortSignal.timeout(120_000),redirect:"manual",headers:{Authorization:`Bearer ${c.env.OPENROUTER_API_KEY}`,"Content-Type":"application/json","HTTP-Referer":"https://getblup.io","X-OpenRouter-Title":"Blup Cloud"},body:JSON.stringify({model,input,...(voice?{voice}:{}),response_format:"mp3"})});
   // Audio responses may expose the provider cost as a numeric header.
   const headerCost=z.coerce.number().pipe(costSchema).safeParse(res.headers.get("x-openrouter-cost")??res.headers.get("x-usage-cost")??undefined);
   let actual=headerCost.success?Math.ceil(headerCost.data*1e6):cost;
   if(!res.ok||res.headers.get("content-type")?.includes("application/json")){
    const errorBody:unknown=await boundedJson(res,ANSWER_MAX).catch(()=>null);
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
  // No AI on the plan: refused before anything goes out (the catalog too).
  if(!(await planOf(userId)).limits.aiUsage) return c.json(fail("O seu plano não inclui IA pelo Blup Cloud.","NO_AI"),402);
  // The staff's list (studio_translate), in order: the hold is the dearest
  // of them at a conservative input bound and the entire allowed output.
  const models=await modelsFor(c.env.DB,"studio_translate");
  if(!models.length) return c.json(fail("A tradução pelo Blup Cloud está sem modelo agora.","UNAVAILABLE"),503);
  const catalog=await chatModels(c.env);
  const price=(id:string)=>catalog.get(id)??{prompt:0.00002,completion:0.00002};
  const inTokens=new TextEncoder().encode(text+translatePrompt(from,to)).length;
  const estimate=Math.ceil(Math.max(...models.map(m=>inTokens*price(m).prompt+maxTokens*price(m).completion))*1e6)+1;
  const admission=await aiReservation(c.env,userId,estimate);
  if(!admission.hold) return c.json(admission.error,402);
  const hold=admission.hold;
  try{
   if(!c.env.OPENROUTER_API_KEY) return c.json(fail("A tradução está indisponível agora.","UNAVAILABLE"),503);
   const res=await fetch(`${c.env.OPENROUTER_URL||"https://openrouter.ai/api/v1"}/chat/completions`,{method:"POST",signal:AbortSignal.timeout(120_000),redirect:"manual",headers:{Authorization:`Bearer ${c.env.OPENROUTER_API_KEY}`,"Content-Type":"application/json","HTTP-Referer":"https://getblup.io","X-OpenRouter-Title":"Blup Cloud"},body:JSON.stringify({models,temperature:0.2,max_tokens:maxTokens,reasoning:{effort:"none"},messages:[{role:"system",content:translatePrompt(from,to)},{role:"user",content:text}]})});
   const raw:unknown=await boundedJson(res,ANSWER_MAX).catch(()=>null);
   const cost=providerCost(raw,answered(res,raw)?estimate:0);
   // Charge before validating content: truncated/malformed answers can cost.
   if(res.ok||usageSchema.safeParse(raw).success) await settle(c.env.DB,hold,[c.env.DB.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros) VALUES (?,'translation',?,?)").bind(userId,chars,cost)]);
   const out=outputSchema.safeParse(raw),choice=out.success?out.data.choices?.[0]:undefined;
   if(!res.ok || !choice?.message?.content || choice.finish_reason==="length") return c.json(fail("A tradução voltou incompleta. Tente de novo.","UPSTREAM"),res.status===429?429:502);
   return c.json({text:choice.message.content});
  }catch{return c.json(fail("A tradução falhou agora. Tente de novo.","UPSTREAM"),502);}
  finally{await release(c.env.DB,hold);}
 });
