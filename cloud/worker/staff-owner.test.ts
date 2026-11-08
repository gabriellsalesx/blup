import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { generateKeyPair, exportJWK, SignJWT, type CryptoKey } from "jose";
import { TestD1, seedUser } from "./security-test-support";
import * as schema from "./db/schema";

// The staff's sensitive actions belong to the owner (STAFF_OWNER_EMAIL),
// told by the e-mail in Cloudflare Access's signed token: making or
// unmaking admins, and anything against the owner's own account.
const state = vi.hoisted(() => ({ db: undefined as unknown as ReturnType<typeof drizzle>, env: {} as Cloudflare.Env }));
vi.mock("cloudflare:workers", () => ({ env: state.env, DurableObject: class {}, waitUntil: () => {} }));
vi.mock("./auth", () => ({
	db: new Proxy({}, { get: (_, k) => { const v = Reflect.get(state.db, k); return typeof v === "function" ? v.bind(state.db) : v; } }),
	auth: { api: { getSession: async () => null }, handler: async () => Response.json({ ok: true }) },
}));

import { staffApi } from "./staff";

const OWNER = "owner@audit.invalid", ADMIN = "admin@audit.invalid", SUPPORT = "support@audit.invalid";
let d1: TestD1;
let privateKey: CryptoKey;
let jwk: Record<string, unknown>;
const ctx = { waitUntil: () => {}, passThroughOnException: () => {} } as unknown as ExecutionContext;

beforeAll(async () => {
	const pair = await generateKeyPair("RS256", { extractable: true });
	privateKey = pair.privateKey;
	jwk = { ...await exportJWK(pair.publicKey), kid: "audit-key", alg: "RS256" };
});
beforeEach(() => {
	d1 = new TestD1();
	state.db = drizzle(d1.asD1(), { schema });
	Object.assign(state.env, {
		DB: d1.asD1(), STAFF_OWNER_EMAIL: OWNER, STAFF_DEV_EMAIL: "",
		ACCESS_TEAM_DOMAIN: "https://access.audit.invalid", ACCESS_AUD: "audit-audience",
		SYNC_LIVE: { getByName: () => ({ revoke: async () => {} }) },
	});
	vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === "https://access.audit.invalid/cdn-cgi/access/certs") return Response.json({ keys: [jwk] });
		throw new Error("External network forbidden");
	}));
	seedUser(d1, "alice");
	d1.sqlite.prepare("INSERT INTO user (id,name,email,email_verified) VALUES ('boss','boss',?,1)").run(OWNER);
	d1.sqlite.prepare("INSERT INTO staff(email,role,added_by) VALUES (?,'admin','fixture'),(?,'support','fixture'),('other-admin@audit.invalid','admin','fixture')").run(ADMIN, SUPPORT);
	d1.sqlite.exec("INSERT INTO passkey(id,name,public_key,user_id,credential_id,counter,device_type,backed_up) VALUES ('pk-boss','Notebook','k','boss','c',0,'singleDevice',0)");
});
afterEach(() => { d1.close(); vi.unstubAllGlobals(); });

const token = (email: string) => new SignJWT({ email }).setProtectedHeader({ alg: "RS256", kid: "audit-key" }).setIssuer("https://access.audit.invalid").setAudience("audit-audience").setExpirationTime("5m").sign(privateKey);
async function as(email: string, path: string, method = "GET", body?: unknown, extra: Record<string, string> = {}) {
	const headers: Record<string, string> = { "cf-access-jwt-assertion": await token(email), ...extra, ...(body === undefined ? {} : { "content-type": "application/json" }) };
	return staffApi.fetch(new Request(`https://staff.audit.invalid${path}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) }), state.env, ctx);
}
const roleOf = (email: string) => d1.sqlite.prepare("SELECT role FROM staff WHERE email=?").get(email)?.role ?? null;
const denials = () => d1.sqlite.prepare("SELECT staff_email AS staffEmail,action,user_id AS userId,detail FROM staff_log WHERE action LIKE 'denied.%' ORDER BY id").all();
const codeOf = async (res: Response) => ({ status: res.status, code: (await res.json() as { code?: string }).code });

describe("staff: actions only the owner may take", () => {
	it("refuses promoting, demoting or removing an admin to another admin, and logs each try", async () => {
		expect(await codeOf(await as(ADMIN, "/team", "POST", { email: SUPPORT, role: "admin" }))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect(await codeOf(await as(ADMIN, "/team", "POST", { email: "new@audit.invalid", role: "admin" }))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect(await codeOf(await as(ADMIN, "/team", "POST", { email: "other-admin@audit.invalid", role: "support" }))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect(await codeOf(await as(ADMIN, "/team/other-admin%40audit.invalid", "DELETE"))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect([roleOf(SUPPORT), roleOf("new@audit.invalid"), roleOf("other-admin@audit.invalid")]).toEqual(["support", null, "admin"]);
		expect(denials()).toEqual([
			{ staffEmail: ADMIN, action: "denied.team.set", userId: null, detail: `OWNER_ONLY ${SUPPORT} admin` },
			{ staffEmail: ADMIN, action: "denied.team.set", userId: null, detail: "OWNER_ONLY new@audit.invalid admin" },
			{ staffEmail: ADMIN, action: "denied.team.set", userId: null, detail: "OWNER_ONLY other-admin@audit.invalid support" },
			{ staffEmail: ADMIN, action: "denied.team.remove", userId: null, detail: "OWNER_ONLY other-admin@audit.invalid" },
		]);
	});

	it("keeps the other admin actions: adding and removing support", async () => {
		expect((await as(ADMIN, "/team", "POST", { email: "helper@audit.invalid", role: "support" })).status).toBe(200);
		expect(roleOf("helper@audit.invalid")).toBe("support");
		expect((await as(ADMIN, `/team/${encodeURIComponent(SUPPORT)}`, "DELETE")).status).toBe(200);
		expect(roleOf(SUPPORT)).toBeNull();
		expect((await as(ADMIN, "/users/alice/ban", "POST", { reason: "local test" })).status).toBe(200);
		expect(denials()).toEqual([]);
	});

	it("lets the owner promote, demote and remove admins", async () => {
		expect((await as(OWNER, "/team", "POST", { email: SUPPORT, role: "admin" })).status).toBe(200);
		expect(roleOf(SUPPORT)).toBe("admin");
		expect((await as(OWNER, "/team", "POST", { email: ADMIN, role: "support" })).status).toBe(200);
		expect(roleOf(ADMIN)).toBe("support");
		expect((await as(OWNER, "/team/other-admin%40audit.invalid", "DELETE")).status).toBe(200);
		expect(roleOf("other-admin@audit.invalid")).toBeNull();
		expect(denials()).toEqual([]);
	});

	it("trusts only the e-mail in Access's token, not the body or other headers", async () => {
		// The body cannot name who asks (strict schema), and a forged header is ignored.
		expect((await as(ADMIN, "/team", "POST", { email: SUPPORT, role: "admin", by: OWNER })).status).toBe(400);
		expect(await codeOf(await as(ADMIN, "/team", "POST", { email: SUPPORT, role: "admin" }, { "cf-access-authenticated-user-email": OWNER }))).toEqual({ status: 403, code: "OWNER_ONLY" });
		expect(roleOf(SUPPORT)).toBe("support");
	});

	it("never lets anyone block or delete the owner's account, not even the owner", async () => {
		for (const who of [ADMIN, OWNER]) {
			expect(await codeOf(await as(who, "/users/boss/ban", "POST", { reason: "local test" }))).toEqual({ status: 403, code: "OWNER_ACCOUNT" });
			expect(await codeOf(await as(who, "/users/boss", "DELETE"))).toEqual({ status: 403, code: "OWNER_ACCOUNT" });
		}
		expect(d1.sqlite.prepare("SELECT banned FROM user WHERE id='boss'").get()?.banned).toBe(0);
		expect(d1.sqlite.prepare("SELECT count(*) AS n FROM resource_quota WHERE user_id='boss' AND deleted_at IS NOT NULL").get()?.n).toBe(0);
		expect(denials()).toEqual([
			{ staffEmail: ADMIN, action: "denied.user.ban", userId: "boss", detail: "OWNER_ACCOUNT" },
			{ staffEmail: ADMIN, action: "denied.user.delete", userId: "boss", detail: "OWNER_ACCOUNT" },
			{ staffEmail: OWNER, action: "denied.user.ban", userId: "boss", detail: "OWNER_ACCOUNT" },
			{ staffEmail: OWNER, action: "denied.user.delete", userId: "boss", detail: "OWNER_ACCOUNT" },
		]);
	});

	it("lets only the owner end the owner's sessions or remove the owner's passkeys", async () => {
		d1.sqlite.exec("INSERT INTO session(id,expires_at,token,created_at,updated_at,user_id) VALUES ('s-boss',9999999999999,'t-boss',1,1,'boss')");
		for (const who of [ADMIN, SUPPORT]) {
			expect(await codeOf(await as(who, "/users/boss/sessions/revoke", "POST"))).toEqual({ status: 403, code: "OWNER_ACCOUNT" });
			expect(await codeOf(await as(who, "/users/boss/passkeys/pk-boss", "DELETE"))).toEqual({ status: 403, code: "OWNER_ACCOUNT" });
		}
		expect(d1.sqlite.prepare("SELECT count(*) AS n FROM session WHERE user_id='boss'").get()?.n).toBe(1);
		expect(denials().map((d) => `${d.staffEmail} ${d.action}`)).toEqual([`${ADMIN} denied.sessions.revoke`, `${ADMIN} denied.passkey.remove`, `${SUPPORT} denied.sessions.revoke`, `${SUPPORT} denied.passkey.remove`]);
		expect((await as(OWNER, "/users/boss/sessions/revoke", "POST")).status).toBe(200);
		expect(d1.sqlite.prepare("SELECT count(*) AS n FROM session WHERE user_id='boss'").get()?.n).toBe(0);
		expect((await as(OWNER, "/users/boss/passkeys/pk-boss", "DELETE")).status).toBe(200);
	});

	it("tells the panel which account is the owner's", async () => {
		expect(await (await as(ADMIN, "/users/boss")).json()).toMatchObject({ ownerAccount: true });
		expect(await (await as(ADMIN, "/users/alice")).json()).toMatchObject({ ownerAccount: false });
	});
});
