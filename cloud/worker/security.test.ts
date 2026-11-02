import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { TestBucket, TestD1, seedUser } from "./security-test-support";
import * as schema from "./db/schema";

const state = vi.hoisted(() => ({ db: undefined as unknown as ReturnType<typeof drizzle>, env: {} as Cloudflare.Env, mailFailures:0,mailCalls:0,mails: [] as string[] }));
vi.mock("cloudflare:workers", () => ({ env: state.env, DurableObject: class {}, waitUntil: () => {} }));
vi.mock("./auth", () => ({
	db: new Proxy({}, { get: (_, k) => { const v = Reflect.get(state.db, k); return typeof v === "function" ? v.bind(state.db) : v; } }),
	auth: { api: { getSession: async ({headers}: {headers:Headers}) => {
		const id = headers.get("authorization")?.replace("Bearer audit-", "") ?? (headers.has("cookie") ? "alice" : "");
		return ["alice", "bob", "free"].includes(id) ? {user:{id,email:`${id}@audit.invalid`,emailVerified:true},session:{id:`session-${id}`,createdAt:new Date()}} : null;
	} }, handler: async () => Response.json({ok:true}) },
}));
vi.mock("./email", () => ({ sendDowngrade: async()=>true,sendReminder: async (_:unknown, email:string) => {state.mailCalls++;if(state.mailFailures-->0)return false;state.mails.push(email);return true;} }));

import worker from "./index";
import { sendDue, forgetOld } from "./reminders";
import { staffApi } from "./staff";
import { generateKeyPair, exportJWK, SignJWT } from "jose";
import { voicePrice } from "./ai";
import { cleanUp } from "./cleanup";
import { cleanSites } from "./sites-cleanup";
import { DAY,softDelete,restoreAccount,purgeAccounts,cancelDeletedBilling,observePlan } from "./lifecycle";
import { serveSite } from "./serve-site";

let d1: TestD1;
let notes: TestBucket;
let sites: TestBucket;
let pending: Promise<unknown>[];
const HASH = "a".repeat(64);
const ctx = {waitUntil:(p:Promise<unknown>) => pending.push(p),passThroughOnException:()=>{}} as unknown as ExecutionContext;

beforeEach(() => {
	d1 = new TestD1(); notes = new TestBucket(); sites = new TestBucket(); pending = []; state.mails = [];state.mailFailures=0;state.mailCalls=0;
	state.db = drizzle(d1.asD1(), {schema});
	Object.assign(state.env, {DB:d1.asD1(),NOTES:notes.asR2(),SITES:sites.asR2(),APP_URL:"https://cloud.audit.invalid",STAFF_HOST:"staff.audit.invalid",SITES_DOMAIN:"getblup.site",OTP_LIMIT:{limit:async()=>({success:true})},ABUSE_LIMIT:{limit:async()=>({success:true})},BULK_LIMIT:{limit:async()=>({success:true})},TURNSTILE_SITE_KEY:"audit-sitekey"});
	Object.assign(state.env, {STAFF_OWNER_EMAIL:"",STAFF_DEV_EMAIL:""});
	Object.assign(state.env, {SYNC_LIVE:{getByName:()=>({notify:async()=>{},revoke:async()=>{}})}});
	seedUser(d1,"alice","max"); seedUser(d1,"bob","max"); seedUser(d1,"free");
	d1.sqlite.exec("INSERT INTO sync_account(user_id,wrapped_key) VALUES ('alice','{}'),('bob','{}'),('free','{}'); INSERT INTO vault(id,user_id,name) VALUES ('vault-a','alice','YQ=='),('vault-b','bob','Yg=='); INSERT INTO site(id,user_id,slug,title) VALUES ('site-a','alice','alice-site','A'),('site-b','bob','bob-site','B')");
	vi.stubGlobal("fetch", vi.fn(async () => {throw new Error("External network forbidden in audit tests");}));
});
afterEach(async () => { await Promise.allSettled(pending); d1.close(); vi.unstubAllGlobals(); });

async function request(path:string, method="GET", body?:unknown, user="alice", headers:Record<string,string>={}) {
	return worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`,{method,headers:{...(user?{authorization:`Bearer audit-${user}`} : {}),...(body!==undefined?{"content-type":"application/json"}:{}),...headers},body:body===undefined?undefined:JSON.stringify(body)}),state.env,ctx);
}

describe("local attacks on actual Cloud routes", () => {
 it("saves only the authenticated account's validated language",async()=>{
  expect((await request("/me/language","PUT",{language:"es"})).status).toBe(200);
  expect(d1.sqlite.prepare("SELECT language FROM user WHERE id='alice'").get()?.language).toBe("es");
  expect(d1.sqlite.prepare("SELECT language FROM user WHERE id='bob'").get()?.language).toBe("pt");
  expect((await request("/me/language","PUT",{language:"en",id:"bob"})).status).toBe(400);
  expect((await request("/me/language","PUT",{language:"de"})).status).toBe(400);
  expect((await request("/me/language","PUT",{language:"es"},"")).status).toBe(401);
 });

	it.each([{plan:"max",storageMB:5120},{plan:"ultra",storageMB:10240}])("enforces the revised $plan storage boundary on block uploads",async({plan,storageMB})=>{
		d1.sqlite.prepare("UPDATE subscription SET plan=? WHERE reference_id='alice'").run(plan);
		const cap=storageMB*1024*1024;
		d1.sqlite.prepare("UPDATE sync_account SET storage_bytes=? WHERE user_id='alice'").run(cap-1);
		const upload=(hash:string)=>worker.fetch(new Request(`https://cloud.audit.invalid/api/sync/blocks/${hash}`,{method:"PUT",headers:{authorization:"Bearer audit-alice"},body:new Uint8Array([1])}),state.env,ctx);
		expect((await upload(HASH)).status).toBe(201);
		expect((await upload("b".repeat(64))).status).toBe(413);
		expect(d1.sqlite.prepare("SELECT storage_bytes FROM sync_account WHERE user_id='alice'").get()?.storage_bytes).toBe(cap);
	});
	it("keeps concurrent Max site creation within three sites",async()=>{
		const responses=await Promise.all(["second","third","fourth"].map(name=>request("/sites","POST",{slug:`max-${name}`,title:name})));
		expect(responses.map(r=>r.status).sort()).toEqual([201,201,403]);
		expect(d1.sqlite.prepare("SELECT count(*) AS n FROM site WHERE user_id='alice'").get()?.n).toBe(3);
	});
	it.each(["ban","sessions/revoke"])("closes existing live connections on staff %s",async action=>{
		const revoke=vi.fn(async()=>{});
		Object.assign(state.env,{STAFF_DEV_EMAIL:"owner@audit.invalid",STAFF_OWNER_EMAIL:"owner@audit.invalid",SYNC_LIVE:{getByName:()=>({revoke})}});
		const res=await staffApi.fetch(new Request(`http://localhost:18848/users/alice/${action}`,{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({reason:"local test"})}),state.env,ctx);
		expect(res.status).toBe(200);expect(revoke).toHaveBeenCalledOnce();
	});
	it("keeps staff stats out of browser caches while retaining the authorized internal cache",async()=>{
		const cache=new Map<string,Response>();
		vi.stubGlobal("caches",{default:{match:async(key:Request)=>cache.get(key.url)?.clone(),put:async(key:Request,res:Response)=>{cache.set(key.url,res);}}});
		Object.assign(state.env,{STAFF_DEV_EMAIL:"owner@audit.invalid",STAFF_OWNER_EMAIL:"owner@audit.invalid"});
		const req=()=>new Request("http://localhost:18848/stats");
		const first=await staffApi.fetch(req(),state.env,ctx);expect(first.headers.get("cache-control")).toBe("no-store");
		await Promise.all(pending);expect([...cache.values()][0]?.headers.get("cache-control")).toBe("max-age=300");
		const hit=await staffApi.fetch(req(),state.env,ctx);expect(hit.headers.get("cache-control")).toBe("no-store");
	});
	it("returns from Checkout locally without letting the plugin write subscription state",async()=>{
		const result=await request("/auth/subscription/success?callbackURL="+encodeURIComponent("https://cloud.audit.invalid/conta/plano?assinatura=ok"),"GET",undefined,"");
		expect(result.status).toBe(303);expect(result.headers.get("location")).toBe("https://cloud.audit.invalid/conta/plano?assinatura=ok");
		expect((await request("/auth/subscription/success?callbackURL="+encodeURIComponent("https://evil.invalid/"))).status).toBe(400);
	});
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
		d1.sqlite.prepare("INSERT INTO reminder(user_id,id,send_at,sent_at) VALUES ('alice','old',?,NULL)").run(now-3*86400000);
		await forgetOld(state.env,now);
		expect(d1.sqlite.prepare("SELECT * FROM reminder").all()).toHaveLength(0);
	});
	it("prunes old sync versions within D1's 100-parameter limit",async()=>{
		const old=Date.now()-400*86400000;
		d1.sqlite.exec("UPDATE vault SET head=100 WHERE id='vault-a'");
		for(let n=1;n<=100;n++){d1.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-a',?,?,1,?)").run(n,`old-${n}`,old);await notes.put(`old-${n}/r`,"");await notes.put(`old-${n}/m`,"x");}
		d1.sqlite.exec("UPDATE sync_account SET storage_bytes=100 WHERE user_id='alice'");
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

// Regressions from the first audit: normal tests after round-two fixes.
describe("atomic resource admission and publish regressions",()=>{
	it("two simultaneous creations must not exceed the paid site's count",async()=>{
		d1.sqlite.exec("DELETE FROM site WHERE id='site-a'; UPDATE subscription SET plan='pro' WHERE reference_id='alice'");
		const responses=await Promise.all([request("/sites","POST",{slug:"race-one",title:"A"}),request("/sites","POST",{slug:"race-two",title:"B"})]);
		expect(responses.filter(r=>r.status===201)).toHaveLength(1);
	});
	it("parallel blocks must not exceed Free storage",async()=>{
		d1.sqlite.prepare("UPDATE sync_account SET storage_bytes=? WHERE user_id='free'").run(50*1024*1024-1);
		await Promise.all(["c","d"].map(c=>worker.fetch(new Request(`https://cloud.audit.invalid/api/sync/blocks/${c.repeat(64)}`,{method:"PUT",headers:{authorization:"Bearer audit-free"},body:new Uint8Array([1])}),state.env,ctx)));
		const row=d1.sqlite.prepare("SELECT storage_bytes FROM sync_account WHERE user_id='free'").get()!;
		expect(row.storage_bytes).toBeLessThanOrEqual(50*1024*1024);
	});
	it("parallel AI calls must not exceed the weekly allowance",async()=>{
		d1.sqlite.exec("UPDATE subscription SET plan='pro' WHERE reference_id='alice'");
		d1.sqlite.prepare("INSERT INTO ai_usage(user_id,kind,units,cost_micros,created_at) VALUES ('alice','narration',1,329978,?)").run(Date.now());
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>new Response(new Uint8Array([1,2,3]))));
		const body={model:"microsoft/mai-voice-2.1",input:"A",voice:"en-US-Emma:MAI-Voice-2.1"};
		const responses=await Promise.all([request("/ai/audio/speech","POST",body),request("/ai/audio/speech","POST",body)]);
		expect(responses.filter(r=>r.status===200)).toHaveLength(1);
	});
	it("parallel AI calls must not exceed the monthly allowance",async()=>{
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
	it("an incomplete translation still has a provider cost to account for",async()=>{
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({choices:[{message:{content:"partial"},finish_reason:"length"}],usage:{cost:0.1}})));
		expect((await request("/ai/translate","POST",{text:"Test",from:"en",to:"pt"})).status).toBe(502);
		expect(d1.sqlite.prepare("SELECT SUM(cost_micros) AS cost FROM ai_usage").get()?.cost).toBe(100000);
	});
	it("concurrent deploy cleanup must not remove the live deploy's files",async()=>{
		d1.sqlite.prepare("INSERT INTO site_blob(site_id,hash,size) VALUES ('site-a',?,1),('site-a',?,1)").run(HASH,"b".repeat(64));
		await sites.put(`s/site-a/f/${HASH}`,"A");await sites.put(`s/site-a/f/${"b".repeat(64)}`,"B");
		await Promise.all([HASH,"b".repeat(64)].map(hash=>request("/sites/site-a/deploys","POST",{files:[{path:"/index.html",hash,type:"text/html",gz:false}]})));
		await Promise.all(pending);
		const {deploy}=d1.sqlite.prepare("SELECT deploy FROM site WHERE id='site-a'").get() as {deploy:string};
		const manifest=JSON.parse(new TextDecoder().decode(sites.objects.get(`s/site-a/d/${deploy}`)));
		expect(sites.objects.has(`s/site-a/f/${manifest["/index.html"][0]}`)).toBe(true);
	});
});

describe("round two: malicious inputs and resource failures",()=>{
	it("refunds storage on an R2 failure",async()=>{
		const put=vi.spyOn(notes,"put").mockRejectedValueOnce(new Error("R2 unavailable"));
		const res=await worker.fetch(new Request(`https://cloud.audit.invalid/api/sync/blocks/${HASH}`,{method:"PUT",headers:{authorization:"Bearer audit-free"},body:new Uint8Array([1])}),state.env,ctx);
		expect(res.status).toBe(500);expect(d1.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='free'").get()?.storage).toBe(0);put.mockRestore();
	});
	it("limits the translation output and charges malformed paid responses",async()=>{
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		const mock=vi.fn(async(_url:RequestInfo|URL,_init?:RequestInit)=>Response.json({choices:"invalid",usage:{cost:0.002}}));vi.stubGlobal("fetch",mock);
		expect((await request("/ai/translate","POST",{text:"some text",from:"en",to:"pt"})).status).toBe(502);
		const init=mock.mock.calls[0]?.[1];
		expect(init&&JSON.parse(String(init.body)).max_tokens).toBeGreaterThan(0);
		expect(d1.sqlite.prepare("SELECT cost_micros FROM ai_usage").get()?.cost_micros).toBe(2000);
	});
	it("uses an upstream narration cost when supplied",async()=>{
		Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
		vi.stubGlobal("fetch",vi.fn(async()=>new Response(new Uint8Array([1]),{headers:{"x-openrouter-cost":"0.0001"}})));
		expect((await request("/ai/audio/speech","POST",{model:"microsoft/mai-voice-2.1",input:"A",voice:"en-US-Emma:MAI-Voice-2.1"})).status).toBe(200);
		expect(d1.sqlite.prepare("SELECT cost_micros FROM ai_usage").get()?.cost_micros).toBe(100);
	});
	it.each([undefined,"", "1.5","Infinity","9007199254740992"])("rejects unsafe multipart parent %s",async parent=>{
		const form=new FormData();if(parent!==undefined)form.set("parent",parent);form.set("blocks","");form.set("manifest",new File(["sealed"],"m"));
		const res=await worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/vault-a/versions",{method:"POST",headers:{authorization:"Bearer audit-alice"},body:form}),state.env,ctx);
		expect(res.status).toBe(400);expect(notes.objects.size).toBe(0);
	});
	it("returns a controlled 400 for malformed multipart and JSON",async()=>{
		for(const [path,ct,body] of [["/sync/vaults/vault-a/versions","multipart/form-data; boundary=absent","bad"],["/sites","application/json","{"]]){
			const res=await worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`,{method:"POST",headers:{authorization:"Bearer audit-alice","content-type":ct},body}),state.env,ctx);expect(res.status).toBe(400);expect(res.headers.get("content-type")).toMatch(/application\/json/);const value=await res.json() as {code:string};expect(value.code).toBe("INVALID");
		}
	});
	it("rejects an oversized announced upload before consuming its body",async()=>{
		let read=0;
		const res=await worker.fetch(new Request(`https://cloud.audit.invalid/api/sites/site-a/files/${HASH}`,{method:"PUT",headers:{authorization:"Bearer audit-alice","content-length":String(50*1024*1024+1)},body:new ReadableStream({pull(c){read++;c.enqueue(new Uint8Array([1]));c.close();}}),duplex:"half"} as RequestInit),state.env,ctx);
		expect(res.status).toBe(413);expect(read).toBeLessThanOrEqual(1);expect(sites.objects.size).toBe(0);
	});
	it("rate-limits both account and IP on expensive calls",async()=>{
		const limit=vi.fn(async({key}:{key:string})=>({success:!key.startsWith("ip:")}));Object.assign(state.env,{ABUSE_LIMIT:{limit}});
		expect((await request("/ai/translate","POST",{text:"test",from:"en",to:"pt"},"alice",{"cf-connecting-ip":"192.0.2.1"})).status).toBe(429);
		expect(limit.mock.calls.map(([v])=>v.key).sort()).toEqual(["account:alice","ip:192.0.2.1"]);expect(fetch).not.toHaveBeenCalled();
	});
	it.each(["limit=1.5","limit=101","cursor=Infinity"])("validates staff page inputs %s",async query=>{
		Object.assign(state.env,{STAFF_DEV_EMAIL:"staff@audit.invalid"});d1.sqlite.exec("INSERT INTO staff(email,role,added_by) VALUES ('staff@audit.invalid','admin','fixture')");
		const res=await staffApi.request(`http://localhost/log?${query}`,undefined,state.env,ctx);expect(res.status).toBe(400);
	});
});

describe("round two: publication and deletion lifecycle",()=>{
	it("returns DEPLOY_BUSY while another publication holds the lease",async()=>{
		d1.sqlite.prepare("INSERT INTO site_lease(site_id,token,expires_at) VALUES ('site-a','held',?)").run(Date.now()+120_000);
		const res=await request("/sites/site-a/deploys","POST",{files:[{path:"/index.html",hash:HASH,type:"text/html",gz:false}]});
		expect(res.status).toBe(409);expect(await res.json()).toEqual({code:"DEPLOY_BUSY"});
	});
	it("keeps retired files ten minutes then deletes only unused blobs",async()=>{
		const other="b".repeat(64),now=Date.now();
		d1.sqlite.prepare("INSERT INTO site_blob(site_id,hash,size,created_at) VALUES ('site-a',?,1,?),('site-a',?,1,?)").run(HASH,now-700_000,other,now-700_000);
		d1.sqlite.exec("UPDATE site SET bytes=2 WHERE id='site-a'");
		await sites.put(`s/site-a/f/${HASH}`,"old");await sites.put(`s/site-a/f/${other}`,"live");
		const post=(hash:string)=>request("/sites/site-a/deploys","POST",{files:[{path:"/index.html",hash,type:"text/html",gz:false}]});
		const first=await(await post(HASH)).json() as {deploy:string};expect((await post(other)).status).toBe(201);
		await cleanSites(state.env,now+599_000);expect(sites.objects.has(`s/site-a/d/${first.deploy}`)).toBe(true);expect(sites.objects.has(`s/site-a/f/${HASH}`)).toBe(true);
		await cleanSites(state.env,now+601_000);expect(sites.objects.has(`s/site-a/f/${HASH}`)).toBe(false);expect(sites.objects.has(`s/site-a/f/${other}`)).toBe(true);
	});
	it("revokes immediately, allows restoration, and purges only after thirty days",async()=>{
		const now=Date.now();await notes.put("u/alice/b/fixture","sealed");await sites.put("s/site-a/f/fixture","public");
		await softDelete(state.env,"alice",now);expect((await request("/me")).status).toBe(401);expect(await restoreAccount(state.env.DB,"alice",now+DAY)).toBe(true);
		expect((await request("/me")).status).toBe(200);
		await softDelete(state.env,"alice",now);await cancelDeletedBilling(state.env);await purgeAccounts(state.env,now+29*DAY);
		expect(notes.objects.size).toBe(1);await purgeAccounts(state.env,now+30*DAY+1);
		expect(notes.objects.size).toBe(0);expect(sites.objects.size).toBe(0);expect(d1.sqlite.prepare("SELECT id FROM user WHERE id='alice'").get()).toBeUndefined();
	});
	it("keeps a Free downgrade online for fourteen days and paying restores it",async()=>{
		vi.stubGlobal("caches",{default:{match:async()=>undefined,put:async()=>{}}});
		const now=Date.now();d1.sqlite.exec("UPDATE site SET deploy='fixture-deploy' WHERE id='site-a';UPDATE subscription SET status='canceled' WHERE reference_id='alice'");
		await sites.put("s/site-a/d/fixture-deploy",JSON.stringify({"/index.html":[HASH,"text/plain",0]}));await sites.put(`s/site-a/f/${HASH}`,"public");
		await observePlan(state.env.DB,"alice","free",now);
		const serve=()=>serveSite(new Request("https://alice-site.getblup.site/"),state.env,ctx,"alice-site");
		expect((await serve()).status).toBe(200);
		d1.sqlite.prepare("UPDATE resource_quota SET grace_until=? WHERE user_id='alice'").run(now-1);expect((await serve()).status).toBe(404);
		d1.sqlite.exec("UPDATE subscription SET status='active' WHERE reference_id='alice'");expect((await serve()).status).toBe(200);
	});
});

describe("round two: custom domains with mocked Cloudflare",()=>{
	function configured(){Object.assign(state.env,{CF_ZONE_ID:"f".repeat(32),CF_API_TOKEN:"audit-only-placeholder",SITES_CNAME_TARGET:"sites.audit.invalid"});}
	function body(hostname="domain.audit.example"){return {hostname};}
	it.each(["https://evil.example","a.example/path","a.example:443","127.0.0.1","localhost","a.local","a.internal","getblup.io","x.getblup.site","a.example@evil.example"])("rejects hostname %s",async hostname=>{
		configured();expect((await request("/sites/site-a/domain","POST",body(hostname))).status).toBe(400);expect(fetch).not.toHaveBeenCalled();
	});
	it("requires a fresh TXT proof before provisioning or serving",async()=>{
		configured();const network=vi.fn(async()=>Response.json({Status:0,Answer:[]}));vi.stubGlobal("fetch",network);
		const response=await request("/sites/site-a/domain","POST",body());expect(response.status).toBe(200);
		const info=await response.json() as {status:string;records:{type:string;name:string;value:string}[]};expect(info.status).toBe("pending_ownership");expect(info.records[0]?.name).toBe("_blup.domain.audit.example");
		expect(network.mock.calls).toHaveLength(1);expect(d1.sqlite.prepare("SELECT provider_id FROM site_domain").get()?.provider_id).toBeNull();
		expect((await request("/sites/site-b/domain","POST",body(),"bob")).status).toBe(200); // Unproved claims cannot squat another person's hostname.
		expect((await request("/sites/site-a/domain","GET",undefined,"bob")).status).toBe(404);
	});
	it("provisions only after DNS proof, tracks SSL and removes the provider hostname",async()=>{
		configured();let proof="";const called:string[]=[];
		vi.stubGlobal("fetch",vi.fn(async(url:RequestInfo|URL,init?:RequestInit)=>{
			called.push(`${init?.method??"GET"} ${String(url)}`);
			if(String(url).startsWith("https://cloudflare-dns.com/"))return Response.json({Status:0,Answer:proof?[{name:"_blup.domain.audit.example.",type:16,data:`"${proof}"`}]:[]});
			if(init?.method==="DELETE")return Response.json({success:true});
			return Response.json({success:true,result:{id:"provider-fixture",hostname:"domain.audit.example",status:"active",ssl:{status:"active"}}});
		}));
		await request("/sites/site-a/domain","POST",body());proof=String(d1.sqlite.prepare("SELECT proof FROM site_domain").get()?.proof);d1.sqlite.exec("UPDATE site_domain SET checked_at=0");
		const verified=await request("/sites/site-a/domain");expect(verified.status).toBe(200);expect(await verified.json()).toMatchObject({status:"active",ssl:"active"});
			expect((await request("/sites/site-b/domain","POST",body(),"bob")).status).toBe(409);
		d1.sqlite.exec("UPDATE subscription SET plan='pro' WHERE reference_id='alice'");expect((await request("/sites/site-a/domain","POST",body())).status).toBe(403);
		expect((await request("/sites/site-a/domain")).status).toBe(200);expect((await request("/sites/site-a/domain","DELETE")).status).toBe(200);
		expect(called.some(s=>s.startsWith("DELETE https://api.cloudflare.com/client/v4/"))).toBe(true);expect(await(await request("/sites/site-a/domain")).json()).toBeNull();
	});
	it("purges the external hostname before deleting its D1 identifier",async()=>{
		configured();d1.sqlite.exec("INSERT INTO site_domain(site_id,hostname,proof,provider_id) VALUES ('site-a','domain.audit.example','proof','provider-fixture')");
		const mock=vi.fn(async()=>Response.json({success:true}));vi.stubGlobal("fetch",mock);
		const now=Date.now();await softDelete(state.env,"alice",now);await cancelDeletedBilling(state.env);await purgeAccounts(state.env,now+31*DAY);
		expect(mock).toHaveBeenCalled();expect(d1.sqlite.prepare("SELECT * FROM site_domain").all()).toHaveLength(0);
	});
});

describe("round two: reminder retries and query plans",()=>{
 it("charges only successful mail and retries failed sends in later rounds",async()=>{
  const now=Date.now();state.mailFailures=2;
  d1.sqlite.prepare("INSERT INTO reminder(user_id,id,send_at) VALUES ('alice','retry',?)").run(now-1000);
  await sendDue(state.env,now);
  expect(d1.sqlite.prepare("SELECT sent,reserved FROM reminder_month").get()).toEqual({sent:0,reserved:0});
  await sendDue(state.env,now+1000);expect(state.mailCalls).toBe(1);
  await sendDue(state.env,now+300_000);await sendDue(state.env,now+600_000);
  expect(state.mailCalls).toBe(3);expect(state.mails).toHaveLength(1);
  expect(d1.sqlite.prepare("SELECT sent,reserved FROM reminder_month").get()).toEqual({sent:1,reserved:0});
 });
 it("stops after three failed attempts even with overlapping crons",async()=>{
  const now=Date.now();state.mailFailures=99;
  d1.sqlite.prepare("INSERT INTO reminder(user_id,id,send_at) VALUES ('alice','retry',?)").run(now-1000);
  for(let i=0;i<4;i++)await Promise.all([sendDue(state.env,now+i*300_000),sendDue(state.env,now+i*300_000)]);
  expect(state.mailCalls).toBe(3);expect(d1.sqlite.prepare("SELECT sent_at,attempts FROM reminder").get()).toEqual({sent_at:0,attempts:3});
  expect(d1.sqlite.prepare("SELECT sent FROM reminder_month").get()?.sent).toBe(0);
 });
 it("uses range indexes for daily expiry instead of table scans",()=>{
  const reminderPlan=d1.sqlite.prepare("EXPLAIN QUERY PLAN SELECT rowid FROM reminder WHERE send_at<? AND claim IS NULL ORDER BY send_at LIMIT 100").all(Date.now());
  const monthPlan=d1.sqlite.prepare("EXPLAIN QUERY PLAN SELECT rowid FROM reminder_month WHERE month<? AND reserved=0 ORDER BY month LIMIT 100").all("2026-10");
  expect(JSON.stringify(reminderPlan)).toMatch(/reminder_cleanup_idx/);expect(JSON.stringify(monthPlan)).toMatch(/reminder_month_cleanup_idx/);
 });
});

it("rejects an invalid unban id after checking the admin role",async()=>{
 Object.assign(state.env,{STAFF_DEV_EMAIL:"staff@audit.invalid"});d1.sqlite.exec("INSERT INTO staff(email,role,added_by) VALUES ('staff@audit.invalid','admin','fixture')");
 const res=await staffApi.request("http://localhost/users/invalid%20id/unban",{method:"POST"},state.env,ctx);expect(res.status).toBe(400);
});

it("keeps bulk transfers on their separate limiter with a ten-second retry",async()=>{
 const costly=vi.fn(async()=>({success:false})),bulk=vi.fn(async()=>({success:false}));Object.assign(state.env,{ABUSE_LIMIT:{limit:costly},BULK_LIMIT:{limit:bulk}});
 const r=await request("/sync/blocks/missing","POST",{ids:[HASH]});expect(r.status).toBe(429);expect(r.headers.get("retry-after")).toBe("10");expect(costly).not.toHaveBeenCalled();expect(bulk).toHaveBeenCalledTimes(1);
});

it("preserves encrypted history during the thirty-day recovery window",async()=>{
 const now=Date.now();
 d1.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-a',1,'u/alice/v/vault-a/1-fixture',1,?)").run(now-400*DAY);
 await notes.put("u/alice/v/vault-a/1-fixture/m","sealed");await softDelete(state.env,"alice",now);await cancelDeletedBilling(state.env);await cleanUp(state.env);
 expect(notes.objects.has("u/alice/v/vault-a/1-fixture/m")).toBe(true);expect(d1.sqlite.prepare("SELECT n FROM vault_version WHERE vault_id='vault-a'").all()).toHaveLength(1);
});

it("recovers a lost Cloudflare ID before forgetting a proved domain",async()=>{
 Object.assign(state.env,{CF_ZONE_ID:"f".repeat(32),CF_API_TOKEN:"audit-only-placeholder",SITES_CNAME_TARGET:"sites.audit.invalid"});
 d1.sqlite.prepare("INSERT INTO site_domain(site_id,hostname,proof,verified_at) VALUES ('site-a','lost.audit.example','proof',?)").run(Date.now());
 const calls:string[]=[];vi.stubGlobal("fetch",vi.fn(async(url:RequestInfo|URL,init?:RequestInit)=>{
  calls.push(`${init?.method??"GET"} ${String(url)}`);
  return Response.json(init?.method==="DELETE"?{success:true}:{success:true,result:[{id:"lost-fixture",hostname:"lost.audit.example",status:"pending",ssl:{status:"pending"}}]});
 }));
 expect((await request("/sites/site-a/domain","DELETE")).status).toBe(200);
 expect(calls).toHaveLength(2);expect(calls[1]).toContain("DELETE https://api.cloudflare.com/client/v4/");
});
it("keeps recovery metadata when external domain deletion fails",async()=>{
 Object.assign(state.env,{CF_ZONE_ID:"f".repeat(32),CF_API_TOKEN:"audit-only-placeholder",SITES_CNAME_TARGET:"sites.audit.invalid"});
 d1.sqlite.exec("INSERT INTO site_domain(site_id,hostname,proof,provider_id) VALUES ('site-a','held.audit.example','proof','fixture-id')");vi.stubGlobal("fetch",vi.fn(async()=>new Response(null,{status:503})));
 expect((await request("/sites/site-a/domain","DELETE")).status).toBe(502);expect(d1.sqlite.prepare("SELECT provider_id FROM site_domain").get()?.provider_id).toBe("fixture-id");
});
it("support sees the recovery deadline and can restore without a new subscription",async()=>{
 Object.assign(state.env,{STAFF_DEV_EMAIL:"support@audit.invalid"});d1.sqlite.exec("INSERT INTO staff(email,role,added_by) VALUES ('support@audit.invalid','support','fixture')");
 await softDelete(state.env,"alice");await cancelDeletedBilling(state.env);
 const detail=await staffApi.request("http://localhost/users/alice",undefined,state.env,ctx);expect(await detail.json()).toMatchObject({deletion:{restorable:true}});
 expect((await staffApi.request("http://localhost/users/alice/restore",{method:"POST"},state.env,ctx)).status).toBe(200);expect(d1.sqlite.prepare("SELECT status FROM subscription WHERE reference_id='alice'").get()?.status).toBe("canceled");
});

it("does not refund the same history bytes from overlapping cleanups",async()=>{
 d1.sqlite.exec("INSERT INTO vault(id,user_id,name,head) VALUES ('free-vault','free','YQ==',1);INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('free-vault',1,'u/free/v/free-vault/1-old',1,1);UPDATE sync_account SET storage_bytes=1 WHERE user_id='free'");
 await notes.put("u/free/v/free-vault/1-old/r","");await notes.put("u/free/v/free-vault/1-old/m","x");
 let finish!:()=>void;const barrier=new Promise<void>(yes=>{finish=yes;});let deletions=0;
 const original=notes.delete.bind(notes);vi.spyOn(notes,"delete").mockImplementation(async keys=>{if(Array.isArray(keys)&&keys.includes("u/free/v/free-vault/1-old/m")){deletions++;await barrier;}return original(keys);});
 const post=async(parent:number)=>{
  const form=new FormData();form.set("parent",String(parent));form.set("blocks","");form.set("manifest",new File(["x"],"manifest"));
  return worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/free-vault/versions",{method:"POST",headers:{authorization:"Bearer audit-free"},body:form}),state.env,ctx);
 };
 const first=post(1);
 try{await vi.waitFor(()=>expect(deletions).toBe(1));const second=await post(2);expect(second.status).toBe(409);expect(await second.json()).toMatchObject({code:"SYNC_BUSY"});}
 finally{finish();expect((await first).status).toBe(201);await Promise.all(pending);}
 expect(d1.sqlite.prepare("SELECT storage_bytes FROM sync_account WHERE user_id='free'").get()?.storage_bytes).toBe(1);
});

it("does not refund history twice when daily cleanups overlap",async()=>{
 const now=Date.now();d1.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-a',1,'u/alice/v/old',1,?),('vault-a',2,'u/alice/v/current',1,?)").run(now-400*DAY,now);
 d1.sqlite.exec("UPDATE vault SET head=2 WHERE id='vault-a';UPDATE sync_account SET storage_bytes=2 WHERE user_id='alice'");
 await notes.put("u/alice/v/current/r","");await notes.put("u/alice/v/old/r","");
 let finish!:()=>void;const barrier=new Promise<void>(yes=>{finish=yes;});let deletions=0;
 const original=notes.delete.bind(notes);vi.spyOn(notes,"delete").mockImplementation(async keys=>{deletions++;await barrier;return original(keys);});
 const tasks=[cleanUp(state.env),cleanUp(state.env)];
 try{await vi.waitFor(()=>expect(deletions).toBe(1));}finally{finish();await Promise.all(tasks);}
 expect(d1.sqlite.prepare("SELECT storage_bytes FROM sync_account WHERE user_id='alice'").get()?.storage_bytes).toBe(1);
});

it("normalizes malformed AI JSON while preserving the voice error envelope",async()=>{
 const r=await worker.fetch(new Request("https://cloud.audit.invalid/api/ai/translate",{method:"POST",headers:{authorization:"Bearer audit-alice","content-type":"application/json"},body:"{"}),state.env,ctx);
 expect(r.status).toBe(400);expect(r.headers.get("content-type")).toMatch(/application\/json/);expect(await r.json()).toMatchObject({code:"INVALID",error:{code:"INVALID"}});
});

it("keeps the AI error envelope for unauthenticated requests and invalid headers",async()=>{
 const unauthorized=await worker.fetch(new Request("https://cloud.audit.invalid/api/ai/usage"),state.env,ctx);
 expect(unauthorized.status).toBe(401);expect(await unauthorized.json()).toMatchObject({code:"UNAUTHORIZED",error:{code:"UNAUTHORIZED"}});
 const invalid=await worker.fetch(new Request("https://cloud.audit.invalid/api/ai/usage",{headers:{authorization:"Bearer audit-alice",origin:"invalid-origin"}}),state.env,ctx);
 expect(invalid.status).toBe(400);expect(await invalid.json()).toMatchObject({code:"INVALID",error:{code:"INVALID"}});
 const origin=await request("/ai/translate","POST",{text:"fixture",from:"pt",to:"en"},"alice",{origin:"https://evil.audit.invalid"});
 expect(origin.status).toBe(403);expect(await origin.json()).toMatchObject({code:"ORIGIN_NOT_ALLOWED",error:{code:"ORIGIN_NOT_ALLOWED"}});
 Object.assign(state.env,{ABUSE_LIMIT:{limit:async()=>({success:false})}});
 const limited=await request("/ai/translate","POST",{text:"fixture",from:"pt",to:"en"});
 expect(limited.status).toBe(429);expect(await limited.json()).toMatchObject({code:"TOO_MANY_REQUESTS",error:{code:"TOO_MANY_REQUESTS"}});
});

it("refuses checkout when no verified display price exists",async()=>{
 Object.assign(state.env,{STRIPE_SECRET_KEY:"sk_test_audit_no_verified_catalog"});
 vi.stubGlobal("fetch",vi.fn(async()=>{throw new Error("fixture Stripe outage");}));
 const res=await request("/auth/subscription/upgrade","POST",{plan:"pro",annual:true});
 expect(res.status).toBe(503);expect(await res.json()).toMatchObject({code:"PRICING_UNAVAILABLE"});
 const foreign=await request("/auth/subscription/upgrade","POST",{plan:"pro",referenceId:"bob"});expect(foreign.status).toBe(403);
});

it("does not retain an obsolete password wrapping after a key update",async()=>{
 d1.sqlite.exec("DELETE FROM sync_account WHERE user_id='alice'");
 const wrapped={params:{time:3,memory_kib:65536,threads:1,salt:"YQ=="},by_password:"b2xk",by_recovery:"cmVjb3Zlcnk="};
 expect((await request("/sync/key","PUT",{wrapped})).status).toBe(201);
 expect((await request("/sync/key","PUT",{wrapped:{...wrapped,by_password:"bmV3"}})).status).toBe(200);
 expect(JSON.stringify(d1.sqlite.prepare("SELECT * FROM sync_account WHERE user_id='alice'").get())).not.toContain("b2xk");
});

it("reports a conflict instead of accepting two different initial sync keys",async()=>{
 d1.sqlite.exec("DELETE FROM sync_account WHERE user_id='alice'");
 const wrapped={params:{time:3,memory_kib:65536,threads:1,salt:"YQ=="},by_password:"b25l",by_recovery:"cmVjb3Zlcnk="};
 const responses=await Promise.all([request("/sync/key","PUT",{wrapped}),request("/sync/key","PUT",{wrapped:{...wrapped,by_password:"dHdv"}})]);
 expect(responses.map(r=>r.status).sort()).toEqual([201,409]);
 const conflict=responses.find(r=>r.status===409);expect(await conflict?.json()).toMatchObject({code:"KEY_EXISTS"});
});

it("skips passkey-only mail reminders without spending quota or attempts",async()=>{
 d1.sqlite.exec("UPDATE user SET email='fixture@sem-email.blup.invalid' WHERE id='alice';UPDATE subscription SET plan='max' WHERE reference_id='alice'");
 d1.sqlite.prepare("INSERT INTO reminder(user_id,id,send_at) VALUES ('alice','no-email',?)").run(Date.now()-1);
 await sendDue(state.env);await sendDue(state.env);
 expect(d1.sqlite.prepare("SELECT sent_at,attempts FROM reminder WHERE id='no-email'").get()).toMatchObject({sent_at:0,attempts:0});
 expect(d1.sqlite.prepare("SELECT count(*) AS n FROM reminder_month WHERE user_id='alice'").get()?.n).toBe(0);expect(state.mailCalls).toBe(0);
 expect(await(await request("/reminders")).json()).toMatchObject({emailMissing:true});
});

it("records a narration usage.cost from an invalid JSON audio response",async()=>{
 Object.assign(state.env,{OPENROUTER_API_KEY:"audit-only-placeholder",OPENROUTER_URL:"https://upstream.audit.invalid"});
 vi.stubGlobal("fetch",vi.fn(async()=>Response.json({usage:{cost:0.002},error:{message:"no audio"}})));
 const r=await request("/ai/audio/speech","POST",{model:"microsoft/mai-voice-2.1",input:"A",voice:"en-US-Emma:MAI-Voice-2.1"});
 expect(r.status).toBe(502);expect(d1.sqlite.prepare("SELECT cost_micros FROM ai_usage").get()?.cost_micros).toBe(2000);
});
