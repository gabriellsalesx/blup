import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { TestBucket, TestD1, seedUser } from "./security-test-support";
import * as schema from "./db/schema";

const state = vi.hoisted(() => ({ db: undefined as unknown as ReturnType<typeof drizzle>, env: {} as Cloudflare.Env, mails: [] as string[] }));
vi.mock("cloudflare:workers", () => ({ env: state.env, DurableObject: class {}, waitUntil: () => {} }));
vi.mock("./auth", () => ({
	db: new Proxy({}, { get: (_, k) => { const v = Reflect.get(state.db, k); return typeof v === "function" ? v.bind(state.db) : v; } }),
	auth: { api: { getSession: async ({headers}: {headers:Headers}) => {
		const id = headers.get("authorization")?.replace("Bearer audit-", "") ?? (headers.has("cookie") ? "alice" : "");
		return ["alice", "bob", "free"].includes(id) ? {user:{id,email:`${id}@audit.invalid`,emailVerified:true},session:{id:`session-${id}`}} : null;
	} }, handler: async () => Response.json({ok:true}) },
}));
vi.mock("./email", () => ({ sendReminder: async (_:unknown, email:string) => {state.mails.push(email);} }));

import worker from "./index";
import { sendDue, forgetOld } from "./reminders";
import { staffApi } from "./staff";
import { recordBillingEvent } from "./billing";
import type Stripe from "stripe";
import { generateKeyPair, exportJWK, SignJWT } from "jose";
import { voicePrice } from "./ai";

let d1: TestD1;
let notes: TestBucket;
let sites: TestBucket;
let pending: Promise<unknown>[];
const HASH = "a".repeat(64);
const ctx = {waitUntil:(p:Promise<unknown>) => pending.push(p),passThroughOnException:()=>{}} as unknown as ExecutionContext;

beforeEach(() => {
	d1 = new TestD1(); notes = new TestBucket(); sites = new TestBucket(); pending = []; state.mails = [];
	state.db = drizzle(d1.asD1(), {schema});
	Object.assign(state.env, {DB:d1.asD1(),NOTES:notes.asR2(),SITES:sites.asR2(),APP_URL:"https://cloud.audit.invalid",STAFF_HOST:"staff.audit.invalid",SITES_DOMAIN:"getblup.site",OTP_LIMIT:{limit:async()=>({success:true})},TURNSTILE_SITE_KEY:"audit-sitekey"});
	Object.assign(state.env, {STAFF_OWNER_EMAIL:"",STAFF_DEV_EMAIL:""});
	Object.assign(state.env, {SYNC_LIVE:{getByName:()=>({notify:async()=>{}})}});
	seedUser(d1,"alice","max"); seedUser(d1,"bob","max"); seedUser(d1,"free");
	d1.sqlite.exec("INSERT INTO sync_account(user_id,wrapped_key) VALUES ('alice','{}'),('bob','{}'),('free','{}'); INSERT INTO vault(id,user_id,name) VALUES ('vault-a','alice','YQ=='),('vault-b','bob','Yg=='); INSERT INTO site(id,user_id,slug,title) VALUES ('site-a','alice','alice-site','A'),('site-b','bob','bob-site','B')");
	vi.stubGlobal("fetch", vi.fn(async () => {throw new Error("External network forbidden in audit tests");}));
});
afterEach(async () => { await Promise.allSettled(pending); d1.close(); vi.unstubAllGlobals(); });

async function request(path:string, method="GET", body?:unknown, user="alice", headers:Record<string,string>={}) {
	return worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`,{method,headers:{...(user?{authorization:`Bearer audit-${user}`} : {}),...(body!==undefined?{"content-type":"application/json"}:{}),...headers},body:body===undefined?undefined:JSON.stringify(body)}),state.env,ctx);
}

describe("local attacks on actual Cloud routes", () => {
	it.each(["/me","/devices","/sync/key","/sync/vaults","/sync/blocks/"+HASH,"/sites","/sites/check?slug=alice-site","/reminders","/ai/usage"])("requires a session at %s",async path => {
		expect((await request(path,"GET",undefined,"")).status).toBe(401);
	});
	it.each([
		["GET","/sync/vaults/vault-b"], ["GET","/sync/vaults/vault-b/versions"], ["GET","/sync/vaults/vault-b/versions/1"],
		["DELETE","/sites/site-b"], ["POST","/sites/site-b/missing"], ["PUT",`/sites/site-b/files/${HASH}`], ["POST","/sites/site-b/deploys"],
	])("rejects IDOR %s %s",async(method,path) => {
		const body=path.endsWith("missing")?{hashes:[]}:path.endsWith("deploys")?{files:[{path:"/index.html",hash:HASH,type:"text/html",gz:false}]}:undefined;
		expect((await request(path,method,body)).status).toBe(404);
		expect(d1.sqlite.prepare("SELECT id FROM site WHERE id='site-b'").get()).toBeTruthy();
	});
	it("never returns another user's encrypted block",async()=>{
		await notes.put(`u/bob/b/${HASH}`,"private fixture");
		expect((await request(`/sync/blocks/${HASH}`)).status).toBe(404);
	});
	it("ignores user IDs injected in reminder payloads",async()=>{
		const res=await request("/reminders","PUT",{userId:"bob",reminders:[{id:"a".repeat(32),at:Date.now()+1000}]});
		expect(res.status).toBe(200);
		expect(d1.sqlite.prepare("SELECT user_id FROM reminder").all()).toEqual([{user_id:"alice"}]);
	});
	it("Free cannot create sites, invoke AI or schedule email",async()=>{
		expect((await request("/sites","POST",{slug:"free-site",title:"Free"},"free")).status).toBe(403);
		expect((await request("/ai/audio/speech","POST",{model:"microsoft/mai-voice-2.1",input:"Test",voice:"en-US-Emma:MAI-Voice-2.1"},"free")).status).toBe(402);
		expect((await request("/ai/translate","POST",{text:"Test",from:"en",to:"pt"},"free")).status).toBe(402);
		await request("/reminders","PUT",{reminders:[{id:"f".repeat(32),at:Date.now()+1000}]},"free");
		expect(d1.sqlite.prepare("SELECT * FROM reminder WHERE user_id='free'").all()).toHaveLength(0);
		expect(fetch).not.toHaveBeenCalled();
	});
	it("staff rejects forged Access identity headers",async()=>{
		const res=await worker.fetch(new Request("https://staff.audit.invalid/api/staff/team",{headers:{"cf-access-authenticated-user-email":"owner@audit.invalid","cf-access-jwt-assertion":"forged"}}),state.env,ctx);
		expect(res.status).toBe(403);
	});
	it("protects cookie-authenticated mutations against cross-origin forms",async()=>{
		const res=await request("/sites/site-a","DELETE",undefined,"",{cookie:"audit-cookie",origin:"https://evil.audit.invalid","sec-fetch-site":"same-site"});
		expect(res.status).toBe(403);
		expect(d1.sqlite.prepare("SELECT id FROM site WHERE id='site-a'").get()).toBeTruthy();
	});
	it("rejects an actual simple multipart CSRF against sync",async()=>{
		const form=new FormData(); form.set("parent","0"); form.set("blocks",""); form.set("manifest",new File(["sealed fixture"],"manifest.bin"));
		const res=await worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/vault-a/versions",{method:"POST",headers:{cookie:"audit-cookie",origin:"https://evil.audit.invalid","sec-fetch-site":"same-site"},body:form}),state.env,ctx);
		expect(res.status).toBe(403);
		expect(notes.objects.size).toBe(0);
	});
	it("rejects prototype names used as translation languages",async()=>{
		expect((await request("/ai/translate","POST",{text:"Test",from:"constructor",to:"pt"})).status).toBe(400);
		expect(fetch).not.toHaveBeenCalled();
	});
	it("does not send the same reminder twice across overlapping cron runs",async()=>{
		const now=Date.now();
		d1.sqlite.prepare("INSERT INTO reminder(user_id,id,send_at) VALUES ('alice','due',?)").run(now-1000);
		await Promise.all([sendDue(state.env,now),sendDue(state.env,now)]);
		expect(state.mails).toHaveLength(1);
		expect(d1.sqlite.prepare("SELECT sent FROM reminder_month").get()).toEqual({sent:1});
	});
	it.each([50,100])("keeps the atomic monthly email cap at %s",async limit=>{
		const now=Date.now();
		if(limit===100)d1.sqlite.exec("UPDATE subscription SET plan='ultra' WHERE reference_id='alice'");
		for(let i=0;i<120;i++) d1.sqlite.prepare("INSERT INTO reminder(user_id,id,send_at) VALUES ('alice',?,?)").run(`due-${i}`,now-1000);
		await sendDue(state.env,now);
		expect(state.mails).toHaveLength(limit);
		expect(d1.sqlite.prepare("SELECT sent FROM reminder_month").get()).toEqual({sent:limit});
		await sendDue(state.env,now);
		expect(state.mails).toHaveLength(limit);
	});
	it("cleans up old reminder rows",async()=>{
		const now=Date.now();
		d1.sqlite.prepare("INSERT INTO reminder VALUES ('alice','old',?,NULL)").run(now-3*86400000);
		await forgetOld(state.env,now);
		expect(d1.sqlite.prepare("SELECT * FROM reminder").all()).toHaveLength(0);
	});
	it("prunes old sync versions within D1's 100-parameter limit",async()=>{
		const old=Date.now()-400*86400000;
		d1.sqlite.exec("UPDATE vault SET head=100 WHERE id='vault-a'");
		for(let n=1;n<=100;n++)d1.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-a',?,?,1,?)").run(n,`old-${n}`,old);
		const form=new FormData(); form.set("parent","100");form.set("blocks","");form.set("manifest",new File(["sealed"],"m"));
		const res=await worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/vault-a/versions",{method:"POST",headers:{authorization:"Bearer audit-alice"},body:form}),state.env,ctx);
		expect(res.status).toBe(201);
		const tasks=await Promise.allSettled(pending);
		expect(tasks.every(t=>t.status==="fulfilled")).toBe(true);
		expect(d1.statements.every(s=>s.bindings<=100)).toBe(true);
	});
	it("keeps staff's largest supported page below 100 D1 bindings",async()=>{
		for(let i=0;i<110;i++)seedUser(d1,`staff-fixture-${i}`);
		Object.assign(state.env,{STAFF_DEV_EMAIL:"owner@audit.invalid",STAFF_OWNER_EMAIL:"owner@audit.invalid"});
		const res=await staffApi.fetch(new Request("http://localhost:18848/users?limit=100"),state.env,ctx);
		expect(res.status).toBe(200);
		expect((await res.json() as {items:unknown[]}).items).toHaveLength(100);
	});
	it("records a retried Stripe payment only once",async()=>{
		d1.sqlite.exec("UPDATE user SET stripe_customer_id='cus_audit' WHERE id='alice'");
		const event={id:"evt_audit",type:"invoice.paid",data:{object:{customer:"cus_audit",amount_paid:1490,currency:"brl"}}} as unknown as Stripe.Event;
		await recordBillingEvent(event,{} as Stripe);await recordBillingEvent(event,{} as Stripe);
		expect(d1.sqlite.prepare("SELECT user_id,amount FROM billing_event").all()).toEqual([{user_id:"alice",amount:1490}]);
	});
	it("does not cache account data in a browser or shared proxy by default",async()=>{
		expect((await request("/me")).headers.get("cache-control")).toBe("no-store");
	});
	it("does not write upstream text into narration error logs",async()=>{
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>new Response("private note fixture",{status:500})));
		const log=vi.spyOn(console,"error").mockImplementation(()=>{});
		try {
			expect((await request("/ai/audio/speech","POST",{model:"microsoft/mai-voice-2.1",input:"A",voice:"en-US-Emma:MAI-Voice-2.1"})).status).toBe(502);
			expect(JSON.stringify(log.mock.calls)).not.toContain("private note fixture");
		} finally {log.mockRestore();}
	});
	it("rejects cookie-authenticated WebSocket hijacking from a sibling origin",async()=>{
		const fetchRoom=vi.fn(async()=>new Response("local room fixture"));
		Object.assign(state.env,{SYNC_LIVE:{getByName:()=>({fetch:fetchRoom})}});
		const res=await request("/sync/live","GET",undefined,"",{cookie:"audit-cookie",origin:"https://evil.audit.invalid",upgrade:"websocket"});
		expect(res.status).toBe(403);expect(fetchRoom).not.toHaveBeenCalled();
	});
	it("validates Access signature, issuer, audience, expiry and staff role",async()=>{
		const {privateKey,publicKey}=await generateKeyPair("RS256",{extractable:true});
		const jwk={...await exportJWK(publicKey),kid:"audit-key",alg:"RS256"};
		Object.assign(state.env,{ACCESS_TEAM_DOMAIN:"https://access.audit.invalid",ACCESS_AUD:"audit-audience",STAFF_OWNER_EMAIL:"owner@audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({keys:[jwk]})));
		d1.sqlite.exec("INSERT INTO staff(email,role,added_by) VALUES ('support@audit.invalid','support','audit')");
		const token=(email:string,issuer="https://access.audit.invalid",audience="audit-audience",expires="5m")=>new SignJWT({email}).setProtectedHeader({alg:"RS256",kid:"audit-key"}).setIssuer(issuer).setAudience(audience).setExpirationTime(expires).sign(privateKey);
		const access=(jwt:string,path="/team")=>worker.fetch(new Request("https://staff.audit.invalid/api/staff"+path,{headers:{"cf-access-jwt-assertion":jwt}}),state.env,ctx);
		const owner=await token("owner@audit.invalid");expect((await access(owner)).status).toBe(200);
		expect((await access(await token("support@audit.invalid"),"/me")).status).toBe(200);
		expect((await access(await token("support@audit.invalid"))).status).toBe(403);
		for(const jwt of [await token("owner@audit.invalid","https://evil.audit.invalid"),await token("owner@audit.invalid",undefined,"wrong-audience"),await token("owner@audit.invalid",undefined,undefined,"-1m")])expect((await access(jwt)).status).toBe(403);
		const parts=owner.split(".");parts[2]=(parts[2][0]==="A"?"B":"A")+parts[2].slice(1);expect((await access(parts.join("."))).status).toBe(403);
	});
	it("uses the selected voice model's price and rejects unavailable voices",async()=>{
		Object.assign(state.env,{OPENROUTER_URL:"https://catalog.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({data:[{id:"vendor/audit-voice",pricing:{prompt:"0.000015"},supported_voices:["demo"]}]})));
		expect(await voicePrice(state.env,"vendor/audit-voice","demo")).toBe(15);
		expect(await voicePrice(state.env,"vendor/audit-voice","forged")).toBeNull();
		expect(await voicePrice(state.env,"vendor/absent","")).toBeNull();
	});
});

// Business/cost decisions are intentionally not changed in this audit.
// Expected failures keep these reproducible blockers visible without a red CI.
describe("owner decisions: non-atomic resource admission",()=>{
	it.fails("two simultaneous creations must not exceed the paid site's count",async()=>{
		d1.sqlite.exec("DELETE FROM site WHERE id='site-a'; UPDATE subscription SET plan='pro' WHERE reference_id='alice'");
		const responses=await Promise.all([request("/sites","POST",{slug:"race-one",title:"A"}),request("/sites","POST",{slug:"race-two",title:"B"})]);
		expect(responses.filter(r=>r.status===201)).toHaveLength(1);
	});
	it.fails("parallel blocks must not exceed Free storage",async()=>{
		d1.sqlite.prepare("UPDATE sync_account SET storage_bytes=? WHERE user_id='free'").run(50*1024*1024-1);
		await Promise.all(["c","d"].map(c=>worker.fetch(new Request(`https://cloud.audit.invalid/api/sync/blocks/${c.repeat(64)}`,{method:"PUT",headers:{authorization:"Bearer audit-free"},body:new Uint8Array([1])}),state.env,ctx)));
		const row=d1.sqlite.prepare("SELECT storage_bytes FROM sync_account WHERE user_id='free'").get()!;
		expect(row.storage_bytes).toBeLessThanOrEqual(50*1024*1024);
	});
	it.fails("parallel AI calls must not exceed the weekly allowance",async()=>{
		d1.sqlite.exec("UPDATE subscription SET plan='pro' WHERE reference_id='alice'");
		d1.sqlite.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros,created_at) VALUES ('alice','narration',1,329978,?)").run(Date.now());
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>new Response(new Uint8Array([1,2,3]))));
		const body={model:"microsoft/mai-voice-2.1",input:"A",voice:"en-US-Emma:MAI-Voice-2.1"};
		const responses=await Promise.all([request("/ai/audio/speech","POST",body),request("/ai/audio/speech","POST",body)]);
		expect(responses.filter(r=>r.status===200)).toHaveLength(1);
	});
	it.fails("parallel AI calls must not exceed the monthly allowance",async()=>{
		vi.useFakeTimers();vi.setSystemTime(new Date("2026-11-20T12:00:00Z"));
		try {
			d1.sqlite.exec("UPDATE subscription SET plan='pro' WHERE reference_id='alice'");
			d1.sqlite.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros,created_at) VALUES ('alice','narration',1,659978,?)").run(Date.parse("2026-11-02T12:00:00Z"));
			Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
			vi.stubGlobal("fetch",vi.fn(async()=>new Response(new Uint8Array([1]))));
			const body={model:"microsoft/mai-voice-2.1",input:"A",voice:"en-US-Emma:MAI-Voice-2.1"};
			const responses=await Promise.all([request("/ai/audio/speech","POST",body),request("/ai/audio/speech","POST",body)]);
			expect(responses.filter(r=>r.status===200)).toHaveLength(1);
		} finally {vi.useRealTimers();}
	});
	it.fails("an incomplete translation still has a provider cost to account for",async()=>{
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({choices:[{message:{content:"partial"},finish_reason:"length"}],usage:{cost:0.1}})));
		expect((await request("/ai/translate","POST",{text:"Test",from:"en",to:"pt"})).status).toBe(502);
		expect(d1.sqlite.prepare("SELECT SUM(cost_micros) AS cost FROM ai_usage").get()?.cost).toBe(100000);
	});
	it.fails("concurrent deploy cleanup must not remove the live deploy's files",async()=>{
		d1.sqlite.prepare("INSERT INTO site_blob(site_id,hash,size) VALUES ('site-a',?,1),('site-a',?,1)").run(HASH,"b".repeat(64));
		await sites.put(`s/site-a/f/${HASH}`,"A");await sites.put(`s/site-a/f/${"b".repeat(64)}`,"B");
		await Promise.all([HASH,"b".repeat(64)].map(hash=>request("/sites/site-a/deploys","POST",{files:[{path:"/index.html",hash,type:"text/html",gz:false}]})));
		await Promise.all(pending);
		const {deploy}=d1.sqlite.prepare("SELECT deploy FROM site WHERE id='site-a'").get() as {deploy:string};
		const manifest=JSON.parse(new TextDecoder().decode(sites.objects.get(`s/site-a/d/${deploy}`)));
		expect(sites.objects.has(`s/site-a/f/${manifest["/index.html"][0]}`)).toBe(true);
	});
});
