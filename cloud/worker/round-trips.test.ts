import { beforeEach,afterEach,describe,it,expect,vi } from "vitest";
import { drizzle } from "drizzle-orm/d1";
import { TestD1,TestBucket,seedUser } from "./security-test-support";
import * as schema from "./db/schema";

// How many times the hot upload routes wait on D1 and R2. Each D1 call
// (or batch) and each R2 operation is one network round trip in
// production; these bounds keep them from creeping back.
const runtime=vi.hoisted(()=>({env:{} as Cloudflare.Env,db:undefined as unknown as ReturnType<typeof drizzle>,sessions:0}));
vi.mock("cloudflare:workers",()=>({env:runtime.env,DurableObject:class{},WorkerEntrypoint:class{},waitUntil:()=>{}}));
vi.mock("./auth",()=>({db:new Proxy({},{get:(_,key)=>{const v=Reflect.get(runtime.db,key);return typeof v==="function"?v.bind(runtime.db):v;}}),auth:{api:{getSession:async({headers}:{headers:Headers})=>{
	runtime.sessions++;
	const id=headers.get("authorization")?.replace("Bearer audit-","");return id==="owner"?{user:{id,email:`${id}@audit.invalid`,emailVerified:true,createdAt:new Date(),name:id},session:{id:`session-${id}`,createdAt:new Date()}}:null;
}},handler:async()=>Response.json({ok:true})}}));
import worker from "./index";
// @ts-expect-error test-only installed Node builtin
import {createHash} from "node:crypto";

// Node lacks workerd's DigestStream and FixedLengthStream.
class TestDigest extends WritableStream<Uint8Array>{
	readonly digest:Promise<ArrayBuffer>;
	constructor(){
		let resolve!:(v:ArrayBuffer)=>void,reject!:(e:unknown)=>void;
		const result=new Promise<ArrayBuffer>((yes,no)=>{resolve=yes;reject=no;});
		const hash=createHash("sha256");
		super({write(chunk){hash.update(chunk);},close(){resolve(new Uint8Array(hash.digest()).buffer);},abort(e){reject(e);}});
		this.digest=result;
	}
}
class TestFixedLength extends TransformStream<Uint8Array,Uint8Array>{
	constructor(size:number){let n=0;super({transform(chunk,c){n+=chunk.byteLength;if(n>size)throw new Error("length");c.enqueue(chunk);},flush(){if(n!==size)throw new Error("length");}});}
}

let db:TestD1,notes:TestBucket,sites:TestBucket,pending:Promise<unknown>[];
const ctx={waitUntil:(p:Promise<unknown>)=>pending.push(p)} as unknown as ExecutionContext;
beforeEach(()=>{
	vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-09T12:00:00Z"));
	db=new TestD1();notes=new TestBucket();sites=new TestBucket();pending=[];runtime.sessions=0;
	runtime.db=drizzle(db.asD1(),{schema});
	Object.assign(crypto,{DigestStream:TestDigest});vi.stubGlobal("FixedLengthStream",TestFixedLength);
	Object.assign(runtime.env,{DB:db.asD1(),NOTES:notes.asR2(),SITES:sites.asR2(),APP_URL:"https://cloud.audit.invalid",SITES_DOMAIN:"getblup.site",STAFF_HOST:"",STAFF_OWNER_EMAIL:"",STAFF_DEV_EMAIL:"",ABUSE_LIMIT:{limit:async()=>({success:true})},BULK_LIMIT:{limit:async()=>({success:true})},OTP_LIMIT:{limit:async()=>({success:true})},SYNC_LIVE:{getByName:()=>({notify:async()=>{},revoke:async()=>{}})}});
	seedUser(db,"owner","max");
	db.sqlite.exec("INSERT INTO site(id,user_id,slug,title) VALUES ('s-1','owner','first-site','First');INSERT INTO vault(id,user_id,name) VALUES ('vault-1','owner','YQ==');INSERT INTO sync_account(user_id,wrapped_key) VALUES ('owner','{}')");
});
afterEach(async()=>{await Promise.allSettled(pending);db.close();vi.useRealTimers();vi.unstubAllGlobals();});

const hex=async(bytes:Uint8Array)=>[...new Uint8Array(await crypto.subtle.digest("SHA-256",new Uint8Array(bytes)))].map(b=>b.toString(16).padStart(2,"0")).join("");
async function measure(run:()=>Promise<Response>){
	// The first request of an account writes its plan state once.
	await send("/sync/vaults",{method:"GET"});
	db.roundTrips=0;notes.operations=0;sites.operations=0;runtime.sessions=0;
	const res=await run();
	return {status:res.status,d1:db.roundTrips,r2:notes.operations+sites.operations,sessions:runtime.sessions};
}
const send=(path:string,init:RequestInit)=>worker.fetch(new Request(`https://cloud.audit.invalid/api${path}`,{...init,headers:{authorization:"Bearer audit-owner",...init.headers}}),runtime.env,ctx);

describe("round trips of the upload routes",()=>{
	it("PUT /sync/blocks/:id (without a ticket)",async()=>{
		const body=new Uint8Array([1,2,3]),id=await hex(body);
		const m=await measure(()=>send(`/sync/blocks/${id}`,{method:"PUT",body}));
		console.info("PUT /sync/blocks/:id",m);
		expect(m.status).toBe(201);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_lease").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM resource_reservation").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='owner'").get()?.storage).toBe(0);
		expect(m.sessions).toBe(1);expect(m.d1).toBeLessThanOrEqual(3);expect(m.r2).toBe(1);
		// A block that is already stored: no lock, no reservation, no R2.
		const again=await measure(()=>send(`/sync/blocks/${id}`,{method:"PUT",body}));
		expect(again.status).toBe(200);expect(again.d1).toBeLessThanOrEqual(2);expect(again.r2).toBe(0);
	});
	it("POST /sync/blocks/missing with 1000 ids",async()=>{
		const ids=Array.from({length:1000},(_,i)=>i.toString(16).padStart(64,"0"));
		db.sqlite.prepare("INSERT INTO sync_block(user_id,id,size) VALUES ('owner',?,1)").run(ids[500]);
		const m=await measure(()=>send("/sync/blocks/missing",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({ids})}));
		console.info("POST /sync/blocks/missing",m);
		expect(m.status).toBe(200);expect(m.sessions).toBe(1);expect(m.d1).toBeLessThanOrEqual(2);
	});
	it("PUT /sync/blocks/:id with a ticket",async()=>{
		const body=new Uint8Array([4,5,6]),id=await hex(body);
		const form=new FormData();form.set("parent","0");form.set("blocks",JSON.stringify([{id,size:body.byteLength}]));form.set("manifest",new File(["m"],"manifest"));
		const prepared=await send("/sync/vaults/vault-1/prepare",{method:"POST",body:form});
		const {ticket}=await prepared.json() as {ticket:string};
		const m=await measure(()=>send(`/sync/blocks/${id}`,{method:"PUT",body,headers:{"x-blup-upload-ticket":ticket}}));
		console.info("PUT /sync/blocks/:id (ticket)",m);
		expect(m.status).toBe(200);expect(m.sessions).toBe(1);expect(m.d1).toBeLessThanOrEqual(4);expect(m.r2).toBe(1);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_lease WHERE site_id LIKE 'sync-block:%'").get()?.n).toBe(0);
		expect((await send(`/sync/vaults/vault-1/commit`,{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({ticket})})).status).toBe(200);
	});
	it("PUT /sites/:id/files/:hash",async()=>{
		const body=new TextEncoder().encode("<h1>hi</h1>"),hash=await hex(body);
		const m=await measure(()=>send(`/sites/s-1/files/${hash}`,{method:"PUT",body,headers:{"content-length":String(body.byteLength)}}));
		console.info("PUT /sites/:id/files/:hash",m);
		expect(m.status).toBe(201);expect(m.sessions).toBe(1);expect(m.d1).toBeLessThanOrEqual(3);expect(m.r2).toBe(1);
		expect(sites.objects.has(`s/s-1/f/${hash}`)).toBe(true);expect([...sites.objects.keys()].some(k=>k.includes("/tmp/"))).toBe(false);
		expect(db.sqlite.prepare("SELECT bytes FROM site WHERE id='s-1'").get()?.bytes).toBe(body.byteLength);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_lease").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM resource_reservation").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='owner'").get()?.storage).toBe(0);
	});
	it("refuses a site file whose bytes are not its name, without keeping anything",async()=>{
		const body=new TextEncoder().encode("other bytes"),hash="c".repeat(64);
		const res=await send(`/sites/s-1/files/${hash}`,{method:"PUT",body,headers:{"content-length":String(body.byteLength)}});
		expect(res.status).toBe(400);expect(await res.json()).toEqual({code:"HASH_MISMATCH"});
		expect(sites.objects.has(`s/s-1/f/${hash}`)).toBe(false);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_blob").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_lease").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM resource_reservation").get()?.n).toBe(0);
		expect(db.sqlite.prepare("SELECT storage FROM resource_quota WHERE user_id='owner'").get()?.storage).toBe(0);
	});
	it("a site file already stored answers without lock, reservation or R2",async()=>{
		const hash="d".repeat(64);db.sqlite.prepare("INSERT INTO site_blob(site_id,hash,size) VALUES ('s-1',?,3)").run(hash);
		const m=await measure(()=>send(`/sites/s-1/files/${hash}`,{method:"PUT",body:new Uint8Array([1,2,3]),headers:{"content-length":"3"}}));
		expect(m.status).toBe(200);expect(m.r2).toBe(0);expect(m.d1).toBeLessThanOrEqual(2);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_lease").get()?.n).toBe(0);
	});
});
