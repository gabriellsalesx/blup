import { beforeEach,afterEach,describe,it,expect,vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { TestD1,TestBucket,seedUser } from "./security-test-support";
import * as schema from "./db/schema";
const runtime=vi.hoisted(()=>({env:{} as Cloudflare.Env,db:undefined as unknown as ReturnType<typeof drizzle>}));
vi.mock("cloudflare:workers",()=>({env:runtime.env,DurableObject:class{},WorkerEntrypoint:class{},waitUntil:()=>{}}));
vi.mock("./auth",()=>({db:new Proxy({},{get:(_,key)=>{const v=Reflect.get(runtime.db,key);return typeof v==="function"?v.bind(runtime.db):v;}}),auth:{api:{getSession:async({headers}:{headers:Headers})=>{
	const id=headers.get("authorization")?.replace("Bearer audit-","");return id==="owner"||id==="other"?{user:{id,email:`${id}@audit.invalid`,emailVerified:true,createdAt:new Date(),name:id},session:{id:`session-${id}`,createdAt:new Date()}}:null;
}},handler:async()=>Response.json({ok:true})}}));
import worker from "./index";
import sitesEntry from "./sites-entry";
import { accountPlan,observePlan,DAY,downgradeNotices } from "./lifecycle";
import { expireDomains,disableExpiredDomain } from "./domain-downgrade";
import { expireTransfers } from "./sync-transfers";
import { drainSyncGarbage } from "./sync-storage";
let db:TestD1,notes:TestBucket,sites:TestBucket,pending:Promise<unknown>[],mail:{text:string;html:string}[];
const ctx={waitUntil:(p:Promise<unknown>)=>pending.push(p)} as unknown as ExecutionContext;
const H="e".repeat(64),MB=1024*1024;
beforeEach(()=>{
	vi.useFakeTimers();vi.setSystemTime(new Date("2026-11-02T12:00:00Z"));
	db=new TestD1();notes=new TestBucket();sites=new TestBucket();pending=[];mail=[];
	runtime.db=drizzle(db.asD1(),{schema});
	Object.assign(runtime.env,{DB:db.asD1(),NOTES:notes.asR2(),SITES:sites.asR2(),APP_URL:"https://cloud.audit.invalid",SITES_DOMAIN:"getblup.site",STAFF_HOST:"",STAFF_OWNER_EMAIL:"",STAFF_DEV_EMAIL:"",CF_ZONE_ID:"f".repeat(32),CF_API_TOKEN:"audit-placeholder",SITES_CNAME_TARGET:"sites.audit.invalid",EMAIL_MODE:"send",MAIL_FROM:"Blup <fixture@audit.invalid>",EMAIL:{send:async(m:{text:string;html:string})=>{mail.push(m);}},ABUSE_LIMIT:{limit:async()=>({success:true})},BULK_LIMIT:{limit:async()=>({success:true})},OTP_LIMIT:{limit:async()=>({success:true})},SYNC_LIVE:{getByName:()=>({notify:async()=>{},revoke:async()=>{}})}});
	seedUser(db,"owner","max");seedUser(db,"other","max");
	db.sqlite.exec("INSERT INTO site(id,user_id,slug,title,created_at) VALUES ('s-1','owner','first-site','First',1),('s-2','owner','second-site','Second',1),('s-other','other','other-site','Other',1);INSERT INTO vault(id,user_id,name) VALUES ('vault-1','owner','YQ=='),('vault-other','other','Yg==');INSERT INTO sync_account(user_id,wrapped_key) VALUES ('owner','{}'),('other','{}')");
	vi.stubGlobal("fetch",vi.fn(async()=>{throw new Error("External network forbidden");}));
	vi.stubGlobal("caches",{default:{match:async()=>undefined,put:async()=>{}}});
});
afterEach(async()=>{await Promise.allSettled(pending);db.close();vi.useRealTimers();vi.unstubAllGlobals();});
async function request(path:string,method="GET",body?:unknown,user="owner"){
	return worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`,{method,headers:{...(user?{authorization:`Bearer audit-${user}`}:{ }),...(body===undefined?{}:{"content-type":"application/json"})},body:body===undefined?undefined:JSON.stringify(body)}),runtime.env,ctx);
}
async function downgrade(plan:string){await observePlan(runtime.env.DB,"owner","max");db.sqlite.prepare("UPDATE subscription SET plan=? WHERE reference_id='owner'").run(plan);await observePlan(runtime.env.DB,"owner",plan);}
async function content(id:string){db.sqlite.prepare("UPDATE site SET deploy=? WHERE id=?").run(`deploy-${id}`,id);await sites.put(`s/${id}/d/deploy-${id}`,JSON.stringify({"/index.html":[H,"text/plain",0]}));await sites.put(`s/${id}/f/${H}`,id);}
async function domain(){await content("s-1");db.sqlite.prepare("INSERT INTO site_domain(site_id,hostname,proof,provider_id,verified_at,status,ssl,checked_at) VALUES ('s-1','owned.audit.example','proof','cf-fixture',1,'active','active',?)").run(Date.now());}
function provider(){const calls:string[]=[];vi.stubGlobal("fetch",vi.fn(async(url:RequestInfo|URL,init?:RequestInit)=>{calls.push(`${init?.method??"GET"} ${String(url)}`);return Response.json(init?.method==="DELETE"?{success:true}:{success:true,result:{id:"cf-fixture",hostname:"owned.audit.example",status:"active",ssl:{status:"active"}}});}));return calls;}
const serve=(slug:string,custom=false)=>sitesEntry.fetch(new Request(`https://${custom?"owned.audit.example":`${slug}.getblup.site`}/`),runtime.env,ctx);

describe("downgrade: hostname grace and explicit site choice",()=>{
	it("serves the custom domain for fourteen days, then removes the provider hostname",async()=>{
		await domain();const calls=provider();await downgrade("pro");const start=Date.now();
		const info=await(await request("/sites/s-1/domain")).json();expect(info).toMatchObject({status:"active",ssl:"active",disableAt:new Date(start+14*DAY).toISOString()});expect((await serve("first-site",true)).status).toBe(200);
		vi.setSystemTime(start+14*DAY+1);expect((await serve("first-site",true)).status).toBe(404);
		const ended=await(await request("/sites/s-1/domain")).json();expect(ended).toMatchObject({status:"disabled",ssl:"pending",records:[],disableAt:new Date(start+14*DAY).toISOString()});
		expect((await serve("first-site")).status).toBe(200);expect(calls.filter(x=>x.startsWith("DELETE"))).toHaveLength(1);
		await expireDomains(runtime.env);expect(calls.filter(x=>x.startsWith("DELETE"))).toHaveLength(1);
	});
	it("cancels shutdown when Max returns before the deadline",async()=>{
		await domain();const calls=provider();await downgrade("pro");vi.setSystemTime(Date.now()+13*DAY);
		db.sqlite.exec("UPDATE subscription SET plan='max' WHERE reference_id='owner'");
		expect(await(await request("/sites/s-1/domain")).json()).toMatchObject({status:"active",disableAt:null});
		vi.setSystemTime(Date.now()+2*DAY);await expireDomains(runtime.env);expect(calls.some(x=>x.startsWith("DELETE"))).toBe(false);expect((await serve("first-site",true)).status).toBe(200);
	});
	it("keeps expiry metadata when Cloudflare fails and retries deletion",async()=>{
		await domain();await downgrade("pro");vi.setSystemTime(Date.now()+15*DAY);
		vi.stubGlobal("fetch",vi.fn(async()=>new Response(null,{status:503})));await expireDomains(runtime.env);
		expect(db.sqlite.prepare("SELECT provider_id,status FROM site_domain").get()).toMatchObject({provider_id:"cf-fixture",status:"disabled"});expect((await serve("first-site",true)).status).toBe(404);
		const calls=provider();await expireDomains(runtime.env);expect(calls.some(x=>x.startsWith("DELETE"))).toBe(true);expect(db.sqlite.prepare("SELECT provider_id FROM site_domain").get()?.provider_id).toBeNull();
	});
	it("does not remove the provider after an upgrade wins the conditional shutdown claim",async()=>{
		await domain();const calls=provider();await downgrade("pro");vi.setSystemTime(Date.now()+15*DAY);
		const beforeClaim=Promise.withResolvers<void>(),resume=Promise.withResolvers<void>(),original=db.prepare.bind(db);
		vi.spyOn(db,"prepare").mockImplementation(sql=>{
			const statement=original(sql);if(!sql.startsWith("UPDATE site_domain SET status='disabled'"))return statement;
			const bind=statement.bind.bind(statement);vi.spyOn(statement,"bind").mockImplementation((...values)=>{
				const prepared=bind(...values),first=prepared.first.bind(prepared);
				vi.spyOn(prepared,"first").mockImplementation(async column=>{beforeClaim.resolve();await resume.promise;return first(column);});return prepared;
			});return statement;
		});
		const expiry=disableExpiredDomain(runtime.env,"s-1");await beforeClaim.promise;
		db.sqlite.exec("UPDATE subscription SET plan='max' WHERE reference_id='owner'");await observePlan(runtime.env.DB,"owner","max");resume.resolve();
		expect(await expiry).toBe(false);expect(calls.some(x=>x.startsWith("DELETE"))).toBe(false);expect(db.sqlite.prepare("SELECT status,disable_at FROM site_domain").get()).toMatchObject({status:"active",disable_at:null});
	});
	it("allows changing the retained site until expiry, with owner-only access",async()=>{
		await content("s-1");await content("s-2");await downgrade("pro");
		expect(await(await request("/sites")).json()).toMatchObject({downgrade:{selectedSiteId:null,retainedSiteIds:["s-1"],selectionOpen:true}});
		expect((await request("/sites/retained","PUT",{siteId:"s-other"})).status).toBe(404);
		expect((await request("/sites/retained","PUT",{siteId:"s-2"},"")).status).toBe(401);
		for(const siteId of ["s-2","s-1","s-2"])expect((await request("/sites/retained","PUT",{siteId})).status).toBe(200);
		vi.setSystemTime(Date.now()+15*DAY);expect((await request("/sites/retained","PUT",{siteId:"s-1"})).status).toBe(409);
		expect((await serve("first-site")).status).toBe(404);expect((await serve("second-site")).status).toBe(200);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site WHERE user_id='owner'").get()?.n).toBe(2);
	});
	it("falls back by creation date then ID, and Free retains no native site",async()=>{
		await content("s-1");await content("s-2");await domain();provider();await downgrade("free");const deadline=Date.now()+14*DAY;
		expect((await serve("first-site",true)).status).toBe(200);vi.setSystemTime(deadline+1);
		expect((await serve("first-site")).status).toBe(404);expect((await serve("second-site")).status).toBe(404);
		expect((await request("/sites/retained","PUT",{siteId:"s-1"})).status).toBe(409);
	});
	it("warns with the exact cutoff and asks for a site choice",async()=>{
		await domain();await downgrade("pro");await downgradeNotices(runtime.env);expect(mail).toHaveLength(1);expect(mail[0]?.text).toMatch(/Escolha.*qual site continua/);expect(mail[0]?.text).toContain("16 de novembro de 2026");expect(mail[0]?.text).toContain("domínio próprio");expect(mail[0]?.html).toContain("/email/blup.png");
		await downgradeNotices(runtime.env);expect(mail).toHaveLength(1);
	});
});

async function storedVault(count=26){
	const ids=Array.from({length:count},(_,i)=>(i+1).toString(16).padStart(64,"0")),refs=ids.join("\n"),size=new TextEncoder().encode(refs).length+1;
	db.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-1',1,'u/owner/v/current',?,?)").run(size,Date.now());db.sqlite.exec("UPDATE vault SET head=1 WHERE id='vault-1'");
	for(const id of ids){db.sqlite.prepare("INSERT INTO sync_block(user_id,id,size,created_at) VALUES ('owner',?,?,?)").run(id,2*MB,Date.now()-2*DAY);await notes.put(`u/owner/b/${id}`,"sealed fixture");}
	await notes.put("u/owner/v/current/m","x");await notes.put("u/owner/v/current/r",refs);db.sqlite.prepare("UPDATE sync_account SET storage_bytes=? WHERE user_id='owner'").run(ids.length*2*MB+size);return ids;
}
async function prepare(blocks:{id:string;size:number}[],parent=1,user="owner",manifest="new"){
	const form=new FormData();form.set("parent",String(parent));form.set("blocks",JSON.stringify(blocks));form.set("manifest",new File([manifest],"manifest"));
	return worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/vault-1/prepare",{method:"POST",headers:{authorization:`Bearer audit-${user}`},body:form}),runtime.env,ctx);
}
async function put(id:string,body:Uint8Array<ArrayBuffer>,ticket?:string){return worker.fetch(new Request(`https://cloud.audit.invalid/api/sync/blocks/${id}`,{method:"PUT",headers:{authorization:"Bearer audit-owner",...(ticket?{"x-blup-upload-ticket":ticket}:{})},body}),runtime.env,ctx);}
describe("storage grace and net reservations",()=>{
	it("accepts the Studio's four parallel uploads and excludes preparation until they finish",async()=>{
		await storedVault(1);const waiting=Promise.withResolvers<void>(),resume=Promise.withResolvers<void>(),original=notes.put.bind(notes);let started=0;
		vi.spyOn(notes,"put").mockImplementation(async(key,body)=>{
			if(key.startsWith("u/owner/b/")){if(++started===4)waiting.resolve();await resume.promise;}await original(key,body);
		});
		const uploads=["a","b","c","d"].map(x=>put(x.repeat(64),new Uint8Array([1])));await waiting.promise;
		expect((await prepare([])).status).toBe(409);expect((await request("/sync/vaults/vault-1/history","DELETE")).status).toBe(409);
		resume.resolve();expect((await Promise.all(uploads)).map(r=>r.status)).toEqual([201,201,201,201]);
		const result=await prepare([]);expect(result.status).toBe(200);const {ticket}=await result.json() as {ticket:string};
		expect((await put(H,new Uint8Array([1]))).status).toBe(409);await request(`/sync/preparations/${ticket}`,"DELETE");
	});
	it("does not extend grace or raise the previous cap to already excessive usage",async()=>{
		await observePlan(runtime.env.DB,"owner","max");
		db.sqlite.prepare("UPDATE sync_account SET storage_bytes=? WHERE user_id='owner'").run(11*1024*MB);
		await downgrade("pro");const first=await observePlan(runtime.env.DB,"owner","pro");
		expect(first?.storageGraceLimit).toBe(10*1024*MB);
		vi.setSystemTime(Date.now()+DAY);await downgrade("free");const later=await observePlan(runtime.env.DB,"owner","free");
		expect(later?.graceUntil).toBe(first?.graceUntil);expect(later?.storageGraceLimit).toBe(first?.storageGraceLimit);
		await downgradeNotices(runtime.env);expect(mail[0]?.text).toContain("armazenamento");expect(mail[0]?.text).toContain("16 de novembro de 2026");
	});
	it("uses the previous cap during grace and permits deleting after expiry",async()=>{
		await storedVault();await downgrade("free");
		expect((await put("b".repeat(64),new Uint8Array([1]))).status).toBe(201);
		const me=await(await request("/me")).json() as {usage:{storageLimitBytes:number;storageGraceLimitBytes:number;storageGraceUntil:string}};
		expect(me.usage.storageLimitBytes).toBe(50*MB);expect(me.usage.storageGraceLimitBytes).toBe(10240*MB);
		vi.setSystemTime(Date.parse(me.usage.storageGraceUntil)+1);expect((await put("c".repeat(64),new Uint8Array([2]))).status).toBe(413);
		const form=new FormData();form.set("parent","1");form.set("blocks","");form.set("manifest",new File(["longer"],"manifest"));
		expect((await worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/vault-1/versions",{method:"POST",headers:{authorization:"Bearer audit-owner"},body:form}),runtime.env,ctx)).status).toBe(201);
		expect(db.sqlite.prepare("SELECT storage_bytes FROM sync_account WHERE user_id='owner'").get()?.storage_bytes).toBeLessThan(50*MB);
		expect((await put("d".repeat(64),new Uint8Array([3]))).status).toBe(201);
	});
	it("reserves net replacement credit once and rejects parallel reuse",async()=>{
		const ids=await storedVault();await downgrade("free");vi.setSystemTime(Date.now()+15*DAY);
		const newId="b".repeat(64),list=[...ids.slice(2).map(id=>({id,size:0})),{id:newId,size:6}];
		const response=await prepare(list);expect(response.status).toBe(200);const ready=await response.json() as {ticket:string;missing:string[]};expect(ready.missing).toEqual([newId]);
		expect((await prepare(list)).status).toBe(409);expect((await put(newId,new Uint8Array(6))).status).toBe(409);
		expect((await put(newId,new Uint8Array(7),ready.ticket)).status).toBe(400);expect((await put(newId,new Uint8Array(6),ready.ticket)).status).toBe(200);
		const commit=await request("/sync/vaults/vault-1/commit","POST",{ticket:ready.ticket});expect(commit.status).toBe(200);expect(await commit.json()).toEqual({n:2});
		expect(await(await request("/sync/vaults/vault-1/commit","POST",{ticket:ready.ticket})).json()).toEqual({n:2});
		expect((await request(`/sync/blocks/${newId}`)).status).toBe(200);expect(db.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='owner'").get()?.storage).toBe(0);
	});
	it("never overwrites committed bytes when a duplicate PUT acquires its lease after commit",async()=>{
		await storedVault(1);const {ticket}=await(await prepare([{id:H,size:1}])).json() as {ticket:string};
		const firstWrite=Promise.withResolvers<void>(),finishFirst=Promise.withResolvers<void>(),secondLease=Promise.withResolvers<void>(),resumeSecond=Promise.withResolvers<void>();
		const originalPut=notes.put.bind(notes);let writes=0;
		vi.spyOn(notes,"put").mockImplementation(async(key,body)=>{await originalPut(key,body);if(key.includes(`/t/${ticket}/b/`)&&++writes===1){firstWrite.resolve();await finishFirst.promise;}});
		const original=db.prepare.bind(db);let attempts=0;
		vi.spyOn(db,"prepare").mockImplementation(sql=>{
			const statement=original(sql);if(!sql.startsWith("INSERT INTO site_lease"))return statement;
			const bind=statement.bind.bind(statement);vi.spyOn(statement,"bind").mockImplementation((...values)=>{
				const prepared=bind(...values);if(values[0]===`sync-block:${ticket}:${H}`&&++attempts===2){const first=prepared.first.bind(prepared);vi.spyOn(prepared,"first").mockImplementation(async column=>{secondLease.resolve();await resumeSecond.promise;return first(column);});}return prepared;
			});return statement;
		});
		const first=put(H,new Uint8Array([1]),ticket);await firstWrite.promise;
		const second=put(H,new Uint8Array([2]),ticket);await secondLease.promise;finishFirst.resolve();expect((await first).status).toBe(200);
		expect((await request("/sync/vaults/vault-1/commit","POST",{ticket})).status).toBe(200);
		const key=db.sqlite.prepare("SELECT object_key FROM sync_block WHERE id=? AND user_id='owner'").get(H)?.object_key;expect(typeof key).toBe("string");
		resumeSecond.resolve();expect((await second).status).toBe(200);expect(writes).toBe(1);
		if(typeof key!=="string")throw new Error("missing physical key");expect([...notes.objects.get(key)??[]]).toEqual([1]);
	});
	it("does not erase adopted bytes when D1 loses its acknowledgement",async()=>{
		await storedVault(1);const {ticket}=await(await prepare([{id:H,size:1}])).json() as {ticket:string};
		const original=db.batch.bind(db);let lost=false;
		vi.spyOn(db,"batch").mockImplementation(async statements=>{
			const result=await original(statements);
			if(!lost&&statements.some(s=>s.sql.startsWith("UPDATE sync_transfer_block SET ready=1"))){lost=true;throw new Error("fixture acknowledgement lost");}return result;
		});
		expect((await put(H,new Uint8Array([7]),ticket)).status).toBe(500);
		expect((await request("/sync/vaults/vault-1/commit","POST",{ticket})).status).toBe(200);
		const key=db.sqlite.prepare("SELECT object_key FROM sync_block WHERE id=? AND user_id='owner'").get(H)?.object_key;if(typeof key!=="string")throw new Error("missing key");expect([...notes.objects.get(key)??[]]).toEqual([7]);
	});
	it("never credits a block still referenced by another vault",async()=>{
		const ids=await storedVault();db.sqlite.exec("INSERT INTO vault(id,user_id,name,head) VALUES ('vault-shared','owner','YQ==',1)");
		await notes.put("u/owner/v/shared/r",ids[0]??"");db.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-shared',1,'u/owner/v/shared',1,?)").run(Date.now());
		await downgrade("free");vi.setSystemTime(Date.now()+15*DAY);
		const response=await prepare([...ids.slice(1).map(id=>({id,size:0})),{id:"b".repeat(64),size:4096}]);expect(response.status).toBe(413);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM sync_block WHERE user_id='owner'").get()?.n).toBe(26);
	});
	it("rejects duplicate IDs and unknown zero-sized blocks",async()=>{
		await storedVault(1);expect((await prepare([{id:H,size:0}])).status).toBe(400);expect((await prepare([{id:H,size:1},{id:H,size:1}])).status).toBe(400);
	});
	it("rejects duplicate IDs in the legacy protocol before creating a transfer",async()=>{
		const ids=await storedVault(1),id=ids[0];const form=new FormData();form.set("parent","1");form.set("blocks",`${id}\n${id}`);form.set("manifest",new File(["x"],"manifest"));
		const res=await worker.fetch(new Request("https://cloud.audit.invalid/api/sync/vaults/vault-1/versions",{method:"POST",headers:{authorization:"Bearer audit-owner"},body:form}),runtime.env,ctx);
		expect(res.status).toBe(400);expect(await res.json()).toMatchObject({code:"INVALID"});expect(db.sqlite.prepare("SELECT count(*) AS n FROM sync_transfer").get()?.n).toBe(0);
	});
	it("refunds cancelled/expired preparation once and hides other accounts",async()=>{
		await storedVault(1);const response=await prepare([{id:H,size:16}]);expect(response.status).toBe(200);const {ticket}=await response.json() as {ticket:string};
		expect((await request("/sync/vaults/vault-1/commit","POST",{ticket},"other")).status).toBe(404);
		for(let i=0;i<2;i++)expect((await request(`/sync/preparations/${ticket}`,"DELETE")).status).toBe(200);
		expect((await request("/sync/vaults/vault-1/commit","POST",{ticket})).status).toBe(410);expect(db.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='owner'").get()?.storage).toBe(0);
		const second=await prepare([{id:H,size:16}]);expect(second.status).toBe(200);vi.setSystemTime(Date.now()+31*60_000);await expireTransfers(runtime.env);expect(db.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='owner'").get()?.storage).toBe(0);
	});
	it("retains cancelled ticket metadata until failed temporary R2 deletion can retry",async()=>{
		await storedVault(1);const {ticket}=await(await prepare([{id:H,size:16}])).json() as {ticket:string};
		vi.setSystemTime(Date.now()+2*DAY);const deletion=vi.spyOn(notes,"delete").mockRejectedValue(new Error("fixture R2 unavailable"));
		await expireTransfers(runtime.env);expect(db.sqlite.prepare("SELECT completed_version FROM sync_transfer WHERE id=?").get(ticket)?.completed_version).toBe(0);
		expect([...notes.objects.keys()].some(k=>k.includes(`/t/${ticket}/`))).toBe(true);
		deletion.mockRestore();vi.setSystemTime(Date.now()+300_001);await expireTransfers(runtime.env);
		expect(db.sqlite.prepare("SELECT id FROM sync_transfer WHERE id=?").get(ticket)).toBeUndefined();expect([...notes.objects.keys()].some(k=>k.includes(`/t/${ticket}/`))).toBe(false);
	});
	it("keeps a no-new-block version even when retained history adds manifest bytes",async()=>{
		await storedVault(1);db.sqlite.prepare("UPDATE site SET bytes=? WHERE id='s-1'").run(2*1024*MB);await downgrade("pro");vi.setSystemTime(Date.now()+15*DAY);
		const response=await prepare([],1,"owner","a longer sealed manifest");expect(response.status).toBe(200);const {ticket}=await response.json() as {ticket:string};
		expect((await request("/sync/vaults/vault-1/commit","POST",{ticket})).status).toBe(200);expect(db.sqlite.prepare("SELECT count(*) AS n FROM vault_version WHERE vault_id='vault-1'").get()?.n).toBe(2);
	});
});
describe("manual history purge",()=>{
	it("keeps pending bytes charged on R2 failure and refunds once on retry",async()=>{
		await storedVault(1);db.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-1',0,'old-history',7,?)").run(Date.now()-DAY);await notes.put("old-history/r","");await notes.put("old-history/m","history");db.sqlite.exec("UPDATE sync_account SET storage_bytes=storage_bytes+7 WHERE user_id='owner'");
		const before=db.sqlite.prepare("SELECT storage_bytes AS n FROM sync_account WHERE user_id='owner'").get()?.n;
		const deletion=vi.spyOn(notes,"delete").mockRejectedValueOnce(new Error("fixture R2 unavailable"));
		expect(await(await request("/sync/vaults/vault-1/history","DELETE")).json()).toMatchObject({freedBytes:0,storageBytes:before});expect(db.sqlite.prepare("SELECT count(*) AS n FROM sync_garbage").get()?.n).toBe(1);
		deletion.mockRestore();expect(await drainSyncGarbage(runtime.env,"owner")).toBe(7);expect(await drainSyncGarbage(runtime.env,"owner")).toBe(0);
		expect(await(await request("/sync/vaults/vault-1/history","DELETE")).json()).toMatchObject({freedBytes:0});
	});
	it("reports unknown history while old head references cannot be read safely",async()=>{
		await storedVault(1);await notes.delete("u/owner/v/current/r");
		expect(await(await request("/me")).json()).toMatchObject({usage:{historyBytes:null}});
		expect((await request("/sync/vaults/vault-1/history","DELETE")).status).toBe(503);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM vault_version WHERE vault_id='vault-1'").get()?.n).toBe(1);
	});
	it("preserves the head and rejects a purge during active sync",async()=>{
		await storedVault(1);const ready=await prepare([{id:H,size:4}]);const {ticket}=await ready.json() as {ticket:string};
		expect((await request("/sync/vaults/vault-1/history","DELETE")).status).toBe(409);
		await request(`/sync/preparations/${ticket}`,"DELETE");const first=await request("/sync/vaults/vault-1/history","DELETE");expect(first.status).toBe(200);expect(await first.json()).toMatchObject({freedBytes:0});expect((await request("/sync/vaults/vault-other/history","DELETE")).status).toBe(404);
	});
	it("does not refund twice if R2 deletion is retried",async()=>{
		await storedVault(1);db.sqlite.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES ('vault-1',0,'old-history',7,?)").run(Date.now()-DAY);await notes.put("old-history/r","");await notes.put("old-history/m","history");db.sqlite.exec("UPDATE sync_account SET storage_bytes=storage_bytes+7 WHERE user_id='owner'");
		const first=await request("/sync/vaults/vault-1/history","DELETE");expect(first.status).toBe(200);expect(await first.json()).toMatchObject({freedBytes:7});
		await Promise.all([drainSyncGarbage(runtime.env,"owner"),drainSyncGarbage(runtime.env,"owner")]);expect(await(await request("/sync/vaults/vault-1/history","DELETE")).json()).toMatchObject({freedBytes:0});expect(db.sqlite.prepare("SELECT n FROM vault_version WHERE vault_id='vault-1'").all()).toEqual([{n:1}]);
	});
});
