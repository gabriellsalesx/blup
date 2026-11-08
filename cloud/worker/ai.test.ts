import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { TestD1, TestBucket, seedUser } from "./security-test-support";
import * as schema from "./db/schema";

// Blup Cloud's AI against a simulated OpenRouter: what goes out (no key
// back to the person, no obsolete parameters) and what is charged.
const state = vi.hoisted(() => ({ db: undefined as unknown as ReturnType<typeof drizzle>, env: {} as Cloudflare.Env }));
vi.mock("cloudflare:workers", () => ({ env: state.env, DurableObject: class {}, waitUntil: () => {} }));
vi.mock("./auth", () => ({
	db: new Proxy({}, { get: (_, k) => { const v = Reflect.get(state.db, k); return typeof v === "function" ? v.bind(state.db) : v; } }),
	auth: { api: { getSession: async ({ headers }: { headers: Headers }) => {
		const id = headers.get("authorization")?.replace("Bearer audit-", "") ?? "";
		return ["alice", "free"].includes(id) ? { user: { id, email: `${id}@audit.invalid`, emailVerified: true }, session: { id: `session-${id}`, createdAt: new Date() } } : null;
	} }, handler: async () => Response.json({ ok: true }) },
}));
vi.mock("./email", () => ({ sendDowngrade: async () => true, sendReminder: async () => true }));

import worker from "./index";
import { aiText } from "./ai";

let d1: TestD1;
const KEY = "sk-or-audit-secret";
const ctx = { waitUntil: () => {}, passThroughOnException: () => {} } as unknown as ExecutionContext;
type Call = { url: string; headers: Headers; body: Record<string, unknown> };
let calls: Call[];

function upstream(answer: (url: string) => Response) {
	vi.stubGlobal("fetch", vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
		const u = String(url);
		calls.push({ url: u, headers: new Headers(init?.headers), body: init?.body ? JSON.parse(String(init.body)) : {} });
		return answer(u);
	}));
}
const request = (path: string, body: unknown, user = "alice") => worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`, { method: "POST", headers: { authorization: `Bearer audit-${user}`, "content-type": "application/json" }, body: JSON.stringify(body) }), state.env, ctx);
const charged = () => Number(d1.sqlite.prepare("SELECT coalesce(SUM(cost_micros),0) AS c FROM ai_usage").get()?.c);
const held = () => Number(d1.sqlite.prepare("SELECT coalesce(SUM(ai),0) AS c FROM resource_quota").get()?.c);

beforeEach(() => {
	vi.useFakeTimers({ toFake: ["Date"] }); vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
	d1 = new TestD1(); calls = [];
	state.db = drizzle(d1.asD1(), { schema });
	// A different address per test: the catalogs are cached per address.
	Object.assign(state.env, { DB: d1.asD1(), NOTES: new TestBucket().asR2(), SITES: new TestBucket().asR2(), APP_URL: "https://cloud.audit.invalid", STAFF_HOST: "staff.audit.invalid", SITES_DOMAIN: "getblup.site", OTP_LIMIT: { limit: async () => ({ success: true }) }, ABUSE_LIMIT: { limit: async () => ({ success: true }) }, BULK_LIMIT: { limit: async () => ({ success: true }) }, STAFF_OWNER_EMAIL: "", STAFF_DEV_EMAIL: "", OPENROUTER_API_KEY: KEY, OPENROUTER_URL: `https://or-${crypto.randomUUID()}.audit.invalid` });
	seedUser(d1, "alice", "max"); seedUser(d1, "free");
	for (const [feature, models] of Object.entries({ studio_voice: ["microsoft/mai-voice-2.1"], studio_translate: ["anthropic/claude-haiku-5.5"], deals_describe: ["deepseek/deepseek-v4.1-flash"] }))
		d1.sqlite.prepare("INSERT INTO ai_model_setting(feature,models,updated_by,updated_at) VALUES(?,?,'owner@audit.invalid',1)").run(feature, JSON.stringify(models));
});
afterEach(() => { d1.close(); vi.unstubAllGlobals(); vi.useRealTimers(); });

describe("translation through Blup Cloud", () => {
	it("sends the current OpenRouter request and never returns the key", async () => {
		upstream((u) => u.endsWith("/models") ? Response.json({ data: [] }) : Response.json({ model: "anthropic/claude-haiku-5.5", choices: [{ message: { content: "Hello" }, finish_reason: "stop" }], usage: { cost: 0.0003 } }));
		const res = await request("/ai/translate", { text: "Olá", from: "pt", to: "en" });
		expect(res.status).toBe(200);
		const text = await res.text();
		expect(JSON.parse(text)).toEqual({ text: "Hello" });
		expect(text).not.toContain(KEY);
		const chat = calls.find((c) => c.url.endsWith("/chat/completions"));
		expect(chat?.headers.get("authorization")).toBe(`Bearer ${KEY}`);
		expect(chat?.headers.get("x-openrouter-title")).toBe("Blup Cloud");
		// usage.include is obsolete (usage always comes back); reasoning off.
		expect(chat?.body.usage).toBeUndefined();
		expect(chat?.body.reasoning).toEqual({ effort: "none" });
		expect(chat?.body.models).toEqual(["anthropic/claude-haiku-5.5"]);
		expect(charged()).toBe(300);
		expect(held()).toBe(0);
	});
	it("does not charge the estimate for a 200 that carries only an error", async () => {
		upstream((u) => u.endsWith("/models") ? Response.json({ data: [] }) : Response.json({ error: { code: 502, message: "provider failed" } }));
		const res = await request("/ai/translate", { text: "Olá", from: "pt", to: "en" });
		expect(res.status).toBe(502);
		expect(charged()).toBe(0);
		expect(held()).toBe(0);
	});
	it.each([402, 429, 500])("answers %i from OpenRouter without its message and releases the hold", async (status) => {
		upstream((u) => u.endsWith("/models") ? Response.json({ data: [] }) : Response.json({ error: { code: status, message: `upstream ${KEY}` } }, { status }));
		const res = await request("/ai/translate", { text: "Olá", from: "pt", to: "en" });
		expect(res.status).toBe(status === 429 ? 429 : 502);
		const body = await res.text();
		expect(body).not.toContain(KEY);
		expect(JSON.parse(body).error.code).toBe("UPSTREAM");
		expect(charged()).toBe(0);
		expect(held()).toBe(0);
	});
	it("answers a timeout with 502 and releases the hold", async () => {
		upstream((u) => { if (u.endsWith("/models")) return Response.json({ data: [] }); throw new DOMException("timed out", "TimeoutError"); });
		expect((await request("/ai/translate", { text: "Olá", from: "pt", to: "en" })).status).toBe(502);
		expect(held()).toBe(0);
	});
	it("refuses a plan without AI before anything goes out", async () => {
		upstream(() => { throw new Error("no call expected"); });
		expect((await request("/ai/translate", { text: "Olá", from: "pt", to: "en" }, "free")).status).toBe(402);
		expect(calls).toHaveLength(0);
	});
});

describe("narration through Blup Cloud", () => {
	const catalog = { data: [{ id: "microsoft/mai-voice-2.1", name: "MAI", pricing: { prompt: "0.000022" }, supported_voices: ["pt-BR-Luana:MAI-Voice-2.1", "pt-PT-Raquel:MAI-Voice-2.1"] }] };
	it("takes the recommended model's voices OpenRouter lists beyond the four locales", async () => {
		upstream((u) => u.includes("/models") ? Response.json(catalog) : new Response(new Uint8Array([1, 2]), { headers: { "content-type": "audio/mpeg" } }));
		const ok = await request("/ai/audio/speech", { model: "microsoft/mai-voice-2.1", input: "Olá", voice: "pt-PT-Raquel:MAI-Voice-2.1" });
		expect(ok.status).toBe(200);
		const speech = calls.find((c) => c.url.endsWith("/audio/speech"));
		expect(speech?.headers.get("x-openrouter-title")).toBe("Blup Cloud");
		expect(speech?.body).toEqual({ model: "microsoft/mai-voice-2.1", input: "Olá", voice: "pt-PT-Raquel:MAI-Voice-2.1", response_format: "mp3" });
		expect(charged()).toBe(3 * 22);
		expect((await request("/ai/audio/speech", { model: "microsoft/mai-voice-2.1", input: "Olá", voice: "pt-PT-Forged:MAI-Voice-2.1" })).status).toBe(400);
	});
});

describe("aiText (Deals)", () => {
	it("sends no obsolete usage parameter and does not charge a 200 error", async () => {
		upstream((u) => u.endsWith("/models") ? Response.json({ data: [] }) : Response.json({ error: { code: 502, message: "provider failed" } }));
		expect(await aiText(state.env, "alice", "deals_describe", { system: "s", user: "u", maxTokens: 160 })).toEqual({ code: "UPSTREAM", status: 502 });
		const chat = calls.find((c) => c.url.endsWith("/chat/completions"));
		expect(chat?.body.usage).toBeUndefined();
		expect(chat?.body.reasoning).toEqual({ effort: "none" });
		expect(charged()).toBe(0);
		expect(held()).toBe(0);
	});
});
