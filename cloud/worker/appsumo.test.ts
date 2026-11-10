import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { TestBucket, TestD1, seedUser } from "./security-test-support";
import * as schema from "./db/schema";

// AppSumo's lifetime codes, end to end on real SQLite: redemption, levels,
// refunds, the lockout, the staff's roles, the purge, and the plan every
// reader picks when Stripe's row and the lifetime row are both there.
const state = vi.hoisted(() => ({ db: undefined as unknown as ReturnType<typeof drizzle>, env: {} as Cloudflare.Env }));
vi.mock("cloudflare:workers", () => ({ env: state.env, DurableObject: class {}, WorkerEntrypoint: class {}, waitUntil: () => {} }));
vi.mock("./auth", () => ({
	db: new Proxy({}, { get: (_, k) => { const v = Reflect.get(state.db, k); return typeof v === "function" ? v.bind(state.db) : v; } }),
	auth: { api: { getSession: async ({ headers }: { headers: Headers }) => {
		const id = headers.get("authorization")?.replace("Bearer audit-", "") ?? "";
		return ["alice", "bob"].includes(id) ? { user: { id, email: `${id}@audit.invalid`, emailVerified: true }, session: { id: `session-${id}`, createdAt: new Date() } } : null;
	} }, handler: async () => Response.json({ ok: true }) },
}));
vi.mock("./email", () => ({ sendDowngrade: async () => true, sendReminder: async () => true }));

import worker from "./index";
import { staffApi } from "./staff";
import { importCodes, normalizeCode, codeHasher, refundCodes, settleRevoked } from "./appsumo";
import { accountAccess, activeSubscription } from "./account";
import { accountPlan, softDelete, cancelDeletedBilling, purgeAccounts, DAY } from "./lifecycle";
import { usageOf } from "./usage";
import { pulseAccess } from "./pulse/limits";
import { dealsAccess } from "./deals/entitlement";
import { forgetSiteHeads, serveSite } from "./serve-site";
import { byPlanOrder, planOrder } from "../shared/pulse-plans";

const PEPPER = "audit-pepper-not-a-secret-0123456789", OWNER = "owner@audit.invalid", ADMIN = "admin@audit.invalid", SUPPORT = "support@audit.invalid";
const CODES = ["BLUPABCDEFGH2222", "BLUPABCDEFGH3333", "BLUPABCDEFGH4444", "BLUPABCDEFGH5555", "BLUPABCDEFGH6666", "BLUPABCDEFGH7777", "BLUPABCDEFGH8888"];
const code = (i: number) => CODES[i] ?? "";
let d1: TestD1, pending: Promise<unknown>[], limit: ReturnType<typeof vi.fn>;
const ctx = { waitUntil: (p: Promise<unknown>) => pending.push(p), passThroughOnException: () => {} } as unknown as ExecutionContext;

beforeEach(async () => {
	vi.useFakeTimers({ toFake: ["Date"] }); vi.setSystemTime(new Date("2026-11-15T12:00:00Z")); forgetSiteHeads();
	d1 = new TestD1(); pending = [];
	state.db = drizzle(d1.asD1(), { schema });
	limit = vi.fn(async () => ({ success: true }));
	Object.assign(state.env, {
		DB: d1.asD1(), NOTES: new TestBucket().asR2(), SITES: new TestBucket().asR2(), APP_URL: "https://cloud.audit.invalid", STAFF_HOST: "", SITES_DOMAIN: "getblup.site",
		OTP_LIMIT: { limit: async () => ({ success: true }) }, ABUSE_LIMIT: { limit: async () => ({ success: true }) }, REDEEM_LIMIT: { limit }, BULK_LIMIT: { limit: async () => ({ success: true }) },
		STAFF_OWNER_EMAIL: OWNER, STAFF_DEV_EMAIL: "", APPSUMO_CODE_PEPPER: PEPPER, STRIPE_SECRET_KEY: "sk_test_fixture",
		SYNC_LIVE: { getByName: () => ({ notify: async () => {}, revoke: async () => {} }) },
	});
	seedUser(d1, "alice"); seedUser(d1, "bob");
	d1.sqlite.prepare("INSERT INTO staff(email,role,added_by) VALUES (?,'admin','fixture'),(?,'support','fixture')").run(ADMIN, SUPPORT);
	vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("External network forbidden in audit tests"); }));
	vi.stubGlobal("caches", { default: { match: async () => undefined, put: async () => {} } });
	await importCodes(state.env, PEPPER, CODES, OWNER);
});
afterEach(async () => { await Promise.allSettled(pending); d1.close(); vi.unstubAllGlobals(); vi.useRealTimers(); vi.restoreAllMocks(); });

const request = (path: string, method = "GET", body?: unknown, user = "alice", headers: Record<string, string> = {}) =>
	worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`, { method, headers: { ...(user ? { authorization: `Bearer audit-${user}` } : {}), ...(body !== undefined ? { "content-type": "application/json" } : {}), ...headers }, body: body === undefined ? undefined : JSON.stringify(body) }), state.env, ctx);
const redeem = (value: string, user = "alice") => request("/redeem", "POST", { code: value }, user);
type State = { level: number; max: number; plan: string; codes: { id: string; last4: string; redeemedAt: number }[]; stripeActive: boolean; available: boolean; redeemed?: { last4: string } };
const json = async <T,>(res: Response) => await res.json() as T;
const codeOf = async (res: Response) => ({ status: res.status, code: (await json<{ code?: string }>(res)).code });
const lifetime = (user = "alice") => d1.sqlite.prepare("SELECT plan,status,period_start AS start,period_end AS end,stripe_subscription_id AS stripe FROM subscription WHERE id=?").get(`appsumo-${user}`);
const staffAs = (email: string, path: string, method = "GET", body?: unknown) => {
	Object.assign(state.env, { STAFF_DEV_EMAIL: email });
	return staffApi.request(`http://localhost${path}`, { method, headers: body === undefined ? {} : { "content-type": "application/json" }, body: body === undefined ? undefined : JSON.stringify(body) }, state.env, ctx);
};

describe("redeeming AppSumo codes", () => {
	it("moves up one level per code, to the top, with the subscription plans' names", async () => {
		const first = Date.now();
		const plans = ["pro", "max", "max_pulse", "ultra", "ultra_pulse_deals"];
		for (const [i, plan] of plans.entries()) {
			vi.setSystemTime(first + i * 1000);
			const res = await redeem(code(i));
			expect(res.status).toBe(200);
			const body = await json<State>(res);
			expect(body).toMatchObject({ level: i + 1, max: 5, plan, stripeActive: false, available: true, redeemed: { last4: code(i).slice(-4) } });
			expect(body.codes).toHaveLength(i + 1);
			expect(body.codes[i]?.id).toMatch(/^[0-9a-f]{16}$/);
			expect(lifetime()).toEqual({ plan, status: "active", start: first, end: null, stripe: null });
		}
		expect(await codeOf(await redeem(code(5)))).toEqual({ status: 409, code: "MAX_LEVEL" });
		// The sixth code is still there for someone else.
		expect((await redeem(code(5), "bob")).status).toBe(200);
		const me = await json<{ plan: { billingName: string; lifetime: boolean; renewsAt: unknown } }>(await request("/me"));
		expect(me.plan).toMatchObject({ billingName: "ultra_pulse_deals", lifetime: true, renewsAt: null });
	});
	it("accepts any case, spaces and hyphens, and never stores the code", async () => {
		expect((await redeem(" blup-abcd efgh-2222 ")).status).toBe(200);
		expect(normalizeCode("ab-cd")).toBeNull();
		expect(normalizeCode("x".repeat(65))).toBeNull();
		const rows = d1.sqlite.prepare("SELECT * FROM appsumo_code").all();
		expect(JSON.stringify(rows)).not.toContain("ABCDEFGH");
		expect(rows.find((r) => r.status === "redeemed")?.code_hash).toBe(await (await codeHasher(PEPPER))(code(0)));
		expect((await redeem("bad")).status).toBe(400);
	});
	it("tells a reused code apart only for its own account", async () => {
		expect((await redeem(code(0))).status).toBe(200);
		expect(await codeOf(await redeem(code(0)))).toEqual({ status: 409, code: "ALREADY_REDEEMED" });
		expect(await codeOf(await redeem(code(0), "bob"))).toEqual({ status: 400, code: "INVALID_CODE" });
		expect(await codeOf(await redeem("BLUPZZZZZZZZZZZZ", "bob"))).toEqual({ status: 400, code: "INVALID_CODE" });
		expect(lifetime("bob")).toBeUndefined();
	});
	it("refuses a revoked code like an unknown one", async () => {
		expect((await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(1)] })).status).toBe(200);
		expect(await codeOf(await redeem(code(1)))).toEqual({ status: 400, code: "INVALID_CODE" });
	});
	it("locks the account out after 10 wrong codes in an hour; a success clears it", async () => {
		for (let i = 0; i < 10; i++) expect((await redeem(`BLUPWRONG${i}AAAAAA`)).status).toBe(400);
		const locked = await redeem(code(0));
		expect(await codeOf(locked)).toEqual({ status: 429, code: "TOO_MANY_REQUESTS" });
		expect(Number(locked.headers.get("retry-after"))).toBeGreaterThan(3000);
		expect((await redeem(code(0), "bob")).status).toBe(200);
		vi.setSystemTime(Date.now() + 3_600_001);
		expect((await redeem(code(1))).status).toBe(200);
		expect(d1.sqlite.prepare("SELECT * FROM appsumo_attempt WHERE user_id='alice'").get()).toBeUndefined();
	});
	it("stops at 10 failures even when requests passed the early check together", async () => {
		const results = await Promise.all(Array.from({ length: 12 }, (_, i) => redeem(`BLUPWRONG${String(i).padStart(2, "0")}AAAAA`)));
		const statuses = results.map((r) => r.status);
		expect(statuses.filter((s) => s === 400)).toHaveLength(10);
		expect(statuses.filter((s) => s === 429)).toHaveLength(2);
		// The 11th failure locks even with no lockout yet when it was checked.
		d1.sqlite.exec("UPDATE appsumo_attempt SET failures=10 WHERE user_id='alice'");
		d1.sqlite.exec("DELETE FROM appsumo_attempt WHERE user_id='bob'");
		d1.sqlite.exec(`INSERT INTO appsumo_attempt(user_id,window_start,failures) VALUES('bob',${Date.now()},9)`);
		expect((await redeem("BLUPWRONGBOB1AAAA", "bob")).status).toBe(400);
		const locked = await redeem("BLUPWRONGBOB2AAAA", "bob");
		expect(await codeOf(locked)).toEqual({ status: 429, code: "TOO_MANY_REQUESTS" });
		expect(Number(locked.headers.get("retry-after"))).toBeGreaterThan(3000);
	});
	it("leaves the plan row alone when the claim takes no code", async () => {
		expect((await redeem(code(0))).status).toBe(200);
		d1.sqlite.exec("UPDATE subscription SET period_start=1,period_end=5 WHERE id='appsumo-alice'");
		vi.setSystemTime(Date.now() + 1000);
		expect((await redeem("BLUPZZZZZZZZZZZZ")).status).toBe(400);
		expect(await codeOf(await redeem(code(0)))).toEqual({ status: 409, code: "ALREADY_REDEEMED" });
		expect(lifetime()).toMatchObject({ plan: "pro", start: 1, end: 5 });
	});
	it("uses the Workers rate limit per account and per address", async () => {
		await request("/redeem", "POST", { code: code(0) }, "alice", { "cf-connecting-ip": "192.0.2.7" });
		// Its own binding (REDEEM_LIMIT), not the general ABUSE_LIMIT.
		expect(limit.mock.calls.map(([v]) => (v as { key: string }).key).sort()).toEqual(["redeem-ip:192.0.2.7", "redeem:alice"]);
		limit.mockImplementation(async ({ key }: { key: string }) => ({ success: !key.startsWith("redeem-ip:") }));
		const res = await redeem(code(1));
		expect(res.status).toBe(429); expect(res.headers.get("retry-after")).toBe("60");
	});
	it("answers UNAVAILABLE without the pepper, and says so on GET", async () => {
		Object.assign(state.env, { APPSUMO_CODE_PEPPER: undefined });
		expect(await codeOf(await redeem(code(0)))).toEqual({ status: 503, code: "UNAVAILABLE" });
		expect(await json<State>(await request("/redeem"))).toMatchObject({ level: 0, plan: "free", available: false });
		expect(await codeOf(await staffAs(OWNER, "/appsumo/codes", "POST", { codes: [code(0)] }))).toEqual({ status: 503, code: "UNAVAILABLE" });
		expect(await codeOf(await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(0)] }))).toEqual({ status: 503, code: "UNAVAILABLE" });
	});
	it("takes a pepper under 32 characters as not set", async () => {
		Object.assign(state.env, { APPSUMO_CODE_PEPPER: "x".repeat(31) });
		expect(await codeOf(await redeem(code(0)))).toEqual({ status: 503, code: "UNAVAILABLE" });
		expect(await json<State>(await request("/redeem"))).toMatchObject({ available: false });
		expect(await json(await staffAs(SUPPORT, "/appsumo"))).toMatchObject({ available: false });
		expect(await codeOf(await staffAs(OWNER, "/appsumo/codes", "POST", { codes: [code(0)] }))).toEqual({ status: 503, code: "UNAVAILABLE" });
		expect(await codeOf(await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(0)] }))).toEqual({ status: 503, code: "UNAVAILABLE" });
	});
	it("reports a Stripe subscription that still renews (Option A: nothing is cancelled)", async () => {
		d1.sqlite.exec("INSERT INTO subscription(id,plan,reference_id,status,stripe_subscription_id,period_end) VALUES ('sub-stripe','max','alice','active','sub_fixture',1900000000000)");
		const body = await json<State>(await redeem(code(0)));
		expect(body).toMatchObject({ level: 1, plan: "pro", stripeActive: true });
		// The account is on the higher of the two: Max (Stripe) over Pro.
		expect((await accountAccess(d1.asD1(), "alice")).plan.name).toBe("max");
		d1.sqlite.exec("UPDATE subscription SET cancel_at_period_end=1 WHERE id='sub-stripe'");
		expect((await json<State>(await request("/redeem"))).stripeActive).toBe(false);
	});
});

describe("refunds and revocation", () => {
	it("lowers the account to the codes it keeps and deletes the plan at 0", async () => {
		for (let i = 0; i < 3; i++) expect((await redeem(code(i))).status).toBe(200);
		expect((await redeem(code(3), "bob")).status).toBe(200);
		const res = await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(2).toLowerCase(), code(3), code(6), "BLUPNOTKNOWN2345", "x"] });
		expect(await json(res)).toEqual({ revoked: 3, notFound: 1, users: 2 });
		await Promise.allSettled(pending);
		expect(lifetime()).toMatchObject({ plan: "max" });
		expect(lifetime("bob")).toBeUndefined();
		const logs = d1.sqlite.prepare("SELECT action,user_id AS userId,detail FROM staff_log WHERE action='appsumo.refund' AND user_id IS NOT NULL ORDER BY user_id").all();
		expect(logs).toEqual([{ action: "appsumo.refund", userId: "alice", detail: `…${code(2).slice(-4)}` }, { action: "appsumo.refund", userId: "bob", detail: `…${code(3).slice(-4)}` }]);
		// And one entry for the whole refund, counts only (code(6) was never
		// redeemed: only this one records it).
		expect(d1.sqlite.prepare("SELECT user_id AS userId,detail FROM staff_log WHERE action='appsumo.refund' AND user_id IS NULL").all()).toEqual([{ userId: null, detail: "3 revogados, 1 não encontrados ou já revogados, 2 contas" }]);
		expect(JSON.stringify(d1.sqlite.prepare("SELECT * FROM staff_log").all())).not.toContain("ABCDEFGH");
		// Refunding the same list again changes nothing.
		expect(await json(await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(2)] }))).toEqual({ revoked: 0, notFound: 1, users: 0 });
		await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(0), code(1)] });
		expect(lifetime()).toBeUndefined();
		expect((await accountAccess(d1.asD1(), "alice")).plan.name).toBe("free");
	});
	it("logs a refund of codes nobody redeemed", async () => {
		expect(await json(await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(5), code(6)] }))).toEqual({ revoked: 2, notFound: 0, users: 0 });
		expect(d1.sqlite.prepare("SELECT staff_email AS email,user_id AS userId,detail FROM staff_log WHERE action='appsumo.refund'").all()).toEqual([{ email: ADMIN, userId: null, detail: "2 revogados, 0 não encontrados ou já revogados, 0 contas" }]);
	});
	it("settles later, from the daily cron, an account whose refund did not settle", async () => {
		await redeem(code(0)); await redeem(code(1));
		const lastPlan = () => d1.sqlite.prepare("SELECT last_plan AS plan FROM resource_quota WHERE user_id='alice'").get();
		expect(lastPlan()).toEqual({ plan: "max" });
		// The refund alone, without the settling the route starts after it.
		await refundCodes(state.env, PEPPER, [code(1)], ADMIN);
		expect(lastPlan()).toEqual({ plan: "max" });
		await settleRevoked(state.env);
		expect(lastPlan()).toEqual({ plan: "pro" });
	});
	it("lets an admin revoke one code of an account, by its opaque id, with its log", async () => {
		await redeem(code(0)); await redeem(code(1));
		const detail = await json<{ appsumo: { id: string; last4: string; status: string }[]; plan: { lifetime: boolean } }>(await staffAs(SUPPORT, "/users/alice"));
		expect(detail.plan.lifetime).toBe(true);
		expect(detail.appsumo.map((c) => c.status)).toEqual(["redeemed", "redeemed"]);
		const id = detail.appsumo[1]?.id ?? "";
		expect(await codeOf(await staffAs(SUPPORT, `/users/alice/appsumo/${id}/revoke`, "POST"))).toEqual({ status: 403, code: "ADMIN_ONLY" });
		expect((await staffAs(ADMIN, `/users/alice/appsumo/${id}/revoke`, "POST")).status).toBe(200);
		expect((await staffAs(ADMIN, `/users/bob/appsumo/${id}/revoke`, "POST")).status).toBe(404);
		expect(lifetime()).toMatchObject({ plan: "pro" });
		expect(d1.sqlite.prepare("SELECT detail FROM staff_log WHERE action='appsumo.revoke'").get()).toEqual({ detail: `…${code(1).slice(-4)}` });
		expect(await codeOf(await redeem(code(1)))).toEqual({ status: 400, code: "INVALID_CODE" });
	});
});

describe("the staff's roles", () => {
	it("lets only the owner import, and only admins refund", async () => {
		expect(await codeOf(await staffAs(ADMIN, "/appsumo/codes", "POST", { codes: ["BLUPNEWCODE22222"] }))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect(await codeOf(await staffAs(SUPPORT, "/appsumo/codes", "POST", { codes: ["BLUPNEWCODE22222"] }))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect(await json(await staffAs(OWNER, "/appsumo/codes", "POST", { codes: ["blup-newcode-22222", "BLUPNEWCODE22222", code(0), "no"] }))).toEqual({ added: 1, skipped: 3 });
		expect(d1.sqlite.prepare("SELECT detail FROM staff_log WHERE action='appsumo.import' ORDER BY id DESC").get()).toEqual({ detail: "2 códigos, 2 inválidos ou repetidos" });
		expect(await codeOf(await staffAs(SUPPORT, "/appsumo/refunds", "POST", { codes: [code(0)] }))).toEqual({ status: 403, code: "ADMIN_ONLY" });
		expect((await staffAs(OWNER, "/appsumo/codes", "POST", { codes: Array.from({ length: 2501 }, () => code(0)) })).status).toBe(400);
	});
	it("counts codes by status and accounts by level", async () => {
		await redeem(code(0)); await redeem(code(1)); await redeem(code(2), "bob");
		await staffAs(ADMIN, "/appsumo/refunds", "POST", { codes: [code(6)] });
		const stats = await json<{ codes: Record<string, number>; levels: { level: number; plan: string; users: number }[] }>(await staffAs(SUPPORT, "/appsumo"));
		expect(stats.codes).toEqual({ available: 3, redeemed: 3, revoked: 1 });
		expect(stats.levels.slice(0, 2)).toEqual([{ level: 1, plan: "pro", users: 1 }, { level: 2, plan: "max", users: 1 }]);
	});
});

describe("account deletion", () => {
	it("keeps a used code used after the purge, without its account", async () => {
		await redeem(code(0));
		const now = Date.now();
		await softDelete(state.env, "alice", now); await cancelDeletedBilling(state.env);
		// The lifetime row is not a Stripe subscription: it is not cancelled.
		expect(lifetime()).toMatchObject({ status: "active" });
		await purgeAccounts(state.env, now + 31 * DAY);
		expect(d1.sqlite.prepare("SELECT id FROM user WHERE id='alice'").get()).toBeUndefined();
		expect(d1.sqlite.prepare("SELECT status,user_id AS userId FROM appsumo_code WHERE status='redeemed'").all()).toEqual([{ status: "redeemed", userId: null }]);
		expect(lifetime()).toBeUndefined();
		expect(await codeOf(await redeem(code(0), "bob"))).toEqual({ status: 400, code: "INVALID_CODE" });
	});
});

// The Stripe row first: without an order, SQLite returns it first and the
// old LIMIT 1 picked Pro.
describe("the highest plan wins with two subscriptions", () => {
	const both = (stripePlan: string, lifetimePlan: string) => {
		d1.sqlite.exec("DELETE FROM subscription");
		d1.sqlite.prepare("INSERT INTO subscription(id,plan,reference_id,status,stripe_subscription_id,period_end) VALUES ('a-stripe',?,'alice','active','sub_fixture',1900000000000)").run(stripePlan);
		d1.sqlite.prepare("INSERT INTO subscription(id,plan,reference_id,status,period_start) VALUES ('appsumo-alice',?,'alice','active',1)").run(lifetimePlan);
	};
	it("orders by plan, then the row without an end, in SQL and in JS", () => {
		both("pro", "ultra");
		expect(d1.sqlite.prepare(`SELECT plan FROM subscription WHERE reference_id='alice' ORDER BY ${planOrder()}`).all().map((r) => r.plan)).toEqual(["ultra", "pro"]);
		expect([{ plan: "pro", periodEnd: 5, id: "b" }, { plan: "ultra", periodEnd: null, id: "c" }, { plan: "pro", periodEnd: null, id: "a" }].sort(byPlanOrder).map((r) => r.id)).toEqual(["c", "a", "b"]);
	});
	it("in the account, the public Worker, usage and /me", async () => {
		both("pro", "ultra");
		expect((await accountAccess(d1.asD1(), "alice")).plan.name).toBe("ultra");
		expect((await accountPlan(state.env, "alice"))?.name).toBe("ultra");
		expect(await activeSubscription("alice")).toMatchObject({ id: "appsumo-alice", plan: "ultra" });
		expect((await usageOf("alice")).storageLimitBytes).toBe(10240 * 1024 * 1024);
		const me = await json<{ plan: { billingName: string; lifetime: boolean } }>(await request("/me"));
		expect(me.plan).toEqual(expect.objectContaining({ billingName: "ultra", lifetime: true }));
	});
	it("in Pulse and Deals (period_end of the lifetime row is no end)", async () => {
		both("max_pulse", "ultra_pulse_deals");
		expect((await pulseAccess(state.env, "alice")).plan).toBe("ultra_pulse");
		expect(d1.sqlite.prepare("SELECT last_plan AS plan,paid_until AS until FROM pulse_account WHERE user_id='alice'").get()).toEqual({ plan: "ultra_pulse", until: null });
		expect((await dealsAccess(state.env, "alice")).plan).toBe("ultra_pulse_deals");
	});
	it("in the published sites", async () => {
		both("pro", "ultra");
		d1.sqlite.exec("INSERT INTO resource_quota(user_id) VALUES ('alice'); INSERT INTO site(id,user_id,slug,title) VALUES ('site-1','alice','alice-one','A'),('site-2','alice','alice-two','B')");
		await serveSite(new Request("https://alice-one.getblup.site/"), state.env, ctx, "alice-one");
		expect(d1.sqlite.prepare("SELECT last_plan AS plan,grace_until AS grace FROM resource_quota WHERE user_id='alice'").get()).toEqual({ plan: "ultra", grace: null });
	});
	it("in the staff's list, detail and counts (each account once)", async () => {
		both("pro", "ultra");
		d1.sqlite.exec("INSERT INTO subscription(id,plan,reference_id,status) VALUES ('b-stripe','max','bob','active')");
		const list = await json<{ items: { id: string; plan: string }[] }>(await staffAs(SUPPORT, "/users"));
		expect(list.items.find((u) => u.id === "alice")?.plan).toBe("Ultra");
		const detail = await json<{ plan: { label: string; lifetime: boolean } }>(await staffAs(SUPPORT, "/users/alice"));
		expect(detail.plan).toMatchObject({ label: "Ultra", lifetime: true });
		const stats = await json<{ total: number; plans: { name: string; count: number }[] }>(await staffAs(SUPPORT, "/stats"));
		expect(stats.plans).toEqual(expect.arrayContaining([{ name: "free", label: "Grátis", count: 0 }, { name: "ultra", label: "Ultra", count: 1 }, { name: "max", label: "Max", count: 1 }]));
		expect(stats.plans.find((p) => p.name === "pro")).toBeUndefined();
	});
});
