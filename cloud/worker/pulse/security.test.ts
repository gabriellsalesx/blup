import {beforeEach,afterEach,describe,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
import {ProjectStore,type ProjectSQL} from "./project-store";
import {safeEvent} from "./event";
import {authKey,checkDepth,ingestBytes,normalizeEventId,parseEnvelope,IngestError} from "./protocol";
import {reserveGrant} from "./quota";
import {pulseAccess,sha256} from "./limits";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{constructor(readonly ctx:DurableObjectState,readonly env:Cloudflare.Env){}}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
import {pulseIngest} from "./ingest";
import {PulseProject} from "./project";

// This adapter preserves real SQLite queries and transaction rollback;
// only the workerd cursor transport is represented structurally.
function engine(db:TestD1){
 const sql={get databaseSize(){return Number(db.sqlite.prepare("PRAGMA page_count").get()?.page_count)*4096;},exec(query:string,...args:unknown[]){
  if(query.includes(";")){db.sqlite.exec(query);return {toArray:()=>[]};}
  const rows=db.sqlite.prepare(query).all(...args);return {toArray:()=>rows};
 }} as unknown as ProjectSQL;
 const transaction=<T>(fn:()=>T):T=>{db.sqlite.exec("BEGIN");try{const result=fn();db.sqlite.exec("COMMIT");return result;}catch(e){db.sqlite.exec("ROLLBACK");throw e;}};
 return {sql,transaction};
}
let d1:TestD1,local:TestD1,store:ProjectStore;
beforeEach(()=>{d1=new TestD1();local=new TestD1();seedUser(d1,"alice","max_pulse");seedUser(d1,"bob","ultra_pulse");seedUser(d1,"free");const e=engine(local);store=new ProjectStore(e.sql,e.transaction);Object.assign(state.env,{DB:d1.asD1(),STAFF_OWNER_EMAIL:"",PULSE_INGEST_LIMIT:{limit:async()=>({success:true})}});});
afterEach(()=>{d1.close();local.close();vi.unstubAllGlobals();});
const raw=(id="a".repeat(32),message="failure")=>({event_id:id,message,platform:"go",exception:{values:[{type:"Error",value:message}]}});
const envelope=(event:unknown,header:unknown={})=>{const payload=JSON.stringify(event);return new TextEncoder().encode(`${JSON.stringify(header)}\n${JSON.stringify({type:"event",length:new TextEncoder().encode(payload).length})}\n${payload}`);};
const request=(path:string,method="GET",body?:unknown,user="alice")=>pulseApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method,headers:{...(user?{authorization:`Bearer ${user}`} : {}),...(body===undefined?{}:{"content-type":"application/json"})},body:body===undefined?undefined:JSON.stringify(body)}),state.env);
describe("Pulse protocol and privacy",()=>{
 it("respects UTF8 byte lengths and ignores binary attachments safely",()=>{
  const event=raw(undefined,"falha açú"),p=envelope(event,{event_id:event.event_id}),a=new TextEncoder().encode('{"type":"attachment","length":3}\n');const bytes=new Uint8Array(p.length+1+a.length+3);bytes.set(p);bytes[p.length]=10;bytes.set(a,p.length+1);bytes.set([0,10,255],p.length+1+a.length);expect(parseEnvelope(bytes).event).toEqual(event);
 });
 it("rejects framing errors, duplicate events, deep JSON and mismatched IDs",()=>{
  expect(()=>parseEnvelope(new TextEncoder().encode('{}\n{"type":"event","length":100}\n{}'))).toThrow(IngestError);
  const one=envelope(raw());const two=new Uint8Array([...one,10,...one.subarray(3)]);expect(()=>parseEnvelope(two)).toThrow(IngestError);
  let deep:unknown=1;for(let i=0;i<34;i++)deep={deep};expect(()=>checkDepth(deep)).toThrow(IngestError);
  expect(()=>normalizeEventId(raw(),"b".repeat(32))).toThrow(IngestError);
  expect(normalizeEventId({event_id:"12345678-1234-1234-1234-123456789abc"})).toBe("12345678123412341234123456789abc");
 });
 it("accepts the official Go 0.50 exception array without retaining absolute paths",async()=>{const event=await safeEvent({platform:"go",exception:[{type:"Error",value:"failure",stacktrace:{frames:[{abs_path:"/application/main.go",function:"main",lineno:4,in_app:true}]}}]},"a".repeat(32));expect(event.exceptions[0].frames[0].filename).toBe("main.go");expect(JSON.stringify(event)).not.toContain("/application/");});
 it("validates all authentication sources without letting cookies authorize ingestion",()=>{
  const key="a".repeat(48),url=`https://ingest.audit.invalid/api/123/store/?sentry_key=${key}&sentry_version=7`;
  expect(authKey(new Request(url),"123",`https://${key}@ingest.audit.invalid/123`)).toBe(key);
  expect(()=>authKey(new Request(url,{headers:{"x-sentry-auth":`Sentry sentry_version=7,sentry_key=${"b".repeat(48)}`}}),"123")).toThrow(IngestError);
  expect(()=>authKey(new Request("https://ingest.audit.invalid/api/123/store/",{headers:{cookie:"session=anything"}}),"123")).toThrow(IngestError);
  expect(()=>authKey(new Request(url),"123",`https://${key}@evil.invalid/123`)).toThrow(IngestError);
 });
 it("bounds decompression and rejects corrupt or unsupported streams",async()=>{
  for(const encoding of ["gzip","deflate"] as const){const compressed=new Blob([envelope(raw())]).stream().pipeThrough(new CompressionStream(encoding));const init={method:"POST",body:compressed,duplex:"half",headers:{"content-encoding":encoding}};expect(parseEnvelope(await ingestBytes(new Request("https://ingest.audit.invalid",init))).event?.event_id).toBe("a".repeat(32));}
  await expect(ingestBytes(new Request("https://ingest.audit.invalid",{method:"POST",body:"bad",headers:{"content-encoding":"gzip"}}))).rejects.toMatchObject({status:400});
  const bomb=new Blob(["a".repeat(1_100_000)]).stream().pipeThrough(new CompressionStream("gzip"));const init={method:"POST",body:bomb,duplex:"half",headers:{"content-encoding":"gzip"}};await expect(ingestBytes(new Request("https://ingest.audit.invalid",init))).rejects.toMatchObject({status:413});
  await expect(ingestBytes(new Request("https://ingest.audit.invalid",{method:"POST",body:"x",headers:{"content-encoding":"zstd"}}))).rejects.toMatchObject({status:415});
 });
 it("drops private structures and groups only sanitized bounded data",async()=>{
  const value=await safeEvent({...raw(),request:{data:"PRIVATE_NOTE",headers:{authorization:"PRIVATE_TOKEN"}},user:{email:"PRIVATE_EMAIL"},extra:{note:"PRIVATE_EXTRA"},contexts:{note:"PRIVATE_CONTEXT"},breadcrumbs:[{message:"PRIVATE_CRUMB"}],message:"token=unsafe person@example.com /home/person/private.md",exception:undefined},"a".repeat(32));
  const encoded=JSON.stringify(value);for(const forbidden of ["PRIVATE_","unsafe","person@example.com","/home/person"])expect(encoded).not.toContain(forbidden);
  expect((await safeEvent({...raw(),fingerprint:["bucket","{{ default }}"]},"b".repeat(32))).group).toBe((await safeEvent({...raw(),fingerprint:["bucket","{{ default }}"]},"c".repeat(32))).group);
  expect((await safeEvent({...raw(),fingerprint:["ab","c"]},"b".repeat(32))).group).not.toBe((await safeEvent({...raw(),fingerprint:["a","bc"]},"c".repeat(32))).group);
  expect((await safeEvent(raw(undefined,"first failure"),"b".repeat(32))).group).not.toBe((await safeEvent(raw(undefined,"different failure"),"c".repeat(32))).group);
  const withLine=(line:number)=>({...raw(),exception:{values:[{type:"Error",value:"failure",stacktrace:{frames:[{filename:"app.js",function:"?",lineno:line,in_app:true}]}}]}});
  expect((await safeEvent(withLine(1),"b".repeat(32))).group).not.toBe((await safeEvent(withLine(2),"c".repeat(32))).group);
 });
});
describe("Pulse atomic admission and retention",()=>{
 it("serializes grants across projects, imports once and never refunds uncertain grants",async()=>{
  const attempts=await Promise.allSettled([reserveGrant(d1.asD1(),"alice","1",0,32),reserveGrant(d1.asD1(),"alice","2",0,32)]);expect(attempts.filter(a=>a.status==="fulfilled")).toHaveLength(1);
  const grant=await reserveGrant(d1.asD1(),"alice","1",0,32);store.importGrant(grant,new Date().toISOString().slice(0,7));store.importGrant(grant,new Date().toISOString().slice(0,7));expect(store.slots(new Date().toISOString().slice(0,7))).toBe(32);
  expect(d1.sqlite.prepare("SELECT reserved FROM pulse_month").get()?.reserved).toBe(32);
 });
 it("deduplicates before consuming quota or reopening resolved issues; keeps only 100 samples",async()=>{
  const now=Date.now(),month=new Date(now).toISOString().slice(0,7),event=await safeEvent(raw(),"a".repeat(32));store.importGrant({id:"g",amount:300},month);
  expect(store.accept(event,"ultra_pulse",now).duplicate).toBe(false);store.setState(event.group,"resolved",0);expect(store.accept(event,"ultra_pulse",now).duplicate).toBe(true);expect(store.issues(0,{limit:10})[0].state).toBe("resolved");
  for(let i=1;i<103;i++)store.accept({...event,id:i.toString(16).padStart(32,"0")},"ultra_pulse",now+i);
  expect(store.samples(event.group,0,100)).toHaveLength(100);expect(store.issues(0,{limit:10})[0]).toMatchObject({count:103,state:"open"});expect(store.usage(month).accepted).toBe(103);
  store.setState(event.group,"ignored",0);store.accept({...event,id:"b".repeat(32)},"ultra_pulse",now+104);expect(store.issues(0,{limit:10})[0].state).toBe("ignored");
 });
 it("enforces burst atomically, hides expired data before cleanup and rolls back failed admissions",async()=>{
  const now=Date.now(),month=new Date(now).toISOString().slice(0,7),event=await safeEvent(raw(),"a".repeat(32));store.importGrant({id:"g",amount:100},month);
  for(let i=0;i<60;i++)store.accept({...event,id:i.toString(16).padStart(32,"0")},"max_pulse",now);
  expect(()=>store.accept(event,"max_pulse",now)).toThrow(IngestError);expect(store.usage(month).accepted).toBe(60);expect(store.issues(now+1,{limit:10})).toEqual([]);expect(store.samples(event.group,now+1)).toEqual([]);store.cleanup(now+1,now+32*86_400_000);store.cleanup(now+1,now+32*86_400_000);expect(store.issues(0,{limit:10})).toEqual([]);
 });
 it("grants owner courtesy, stops writes on expiry and never extends the 14 day window",async()=>{
  const env=state.env,now=Date.now();expect((await pulseAccess(env,"alice",now)).writable).toBe(true);d1.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");const first=await pulseAccess(env,"alice",now);expect(first.readable).toBe(true);expect(first.writable).toBe(false);expect((await pulseAccess(env,"alice",now+15*86_400_000)).readable).toBe(false);expect((await pulseAccess(env,"alice",now+1)).readOnlyUntil).toBe(first.readOnlyUntil);
  Object.assign(env,{STAFF_OWNER_EMAIL:"free@audit.invalid"});expect((await pulseAccess(env,"free")).plan).toBe("ultra_pulse");
 });
});
describe("Pulse actual routes and DO",()=>{
 it("rejects IDOR, arbitrary owner/limit, unpaid creation and excess projects",async()=>{
  const made=await request("/projects","POST",{name:"App",platform:"go"});expect(made.status).toBe(201);const body=await made.json() as {id:string;key:{dsn:string}};
  expect(body.key.dsn).toMatch(/^https:\/\/[a-f0-9]{48}@ingest.getblup.io\/\d+$/);
  expect((await request(`/projects/${body.id}`,"DELETE",undefined,"bob")).status).toBe(404);
  expect((await request("/projects","POST",{name:"App",platform:"go",owner:"bob"})).status).toBe(400);
  expect((await request("/projects","POST",{name:"App",platform:"go"},"free")).status).toBe(403);
  const rest=await Promise.all([1,2,3].map(()=>request("/projects","POST",{name:"App",platform:"go"})));expect(rest.map(r=>r.status).sort()).toEqual([201,201,409]);
  expect((await request("/projects","GET",undefined,"")).status).toBe(401);
 });
 it("keeps the earlier cleanup alarm under continuous ingestion",async()=>{
  const created=await (await request("/projects","POST",{name:"Alarm",platform:"go"})).json() as {id:string;key:{dsn:string}},key=new URL(created.key.dsn).username;
  const e=engine(local);let alarm:number|null=null;const set=vi.fn(async(n:number)=>{alarm=n;});
  const ctx={waitUntil:(p:Promise<unknown>)=>p.catch(()=>{}),storage:{sql:e.sql,transactionSync:e.transaction,getAlarm:async()=>alarm,setAlarm:set}} as unknown as DurableObjectState;
  const object=new PulseProject(ctx,state.env),now=Date.now(),clock=vi.spyOn(Date,"now").mockReturnValue(now);
  try{await object.ingest(created.id,await sha256(key),await safeEvent(raw(),"a".repeat(32)));clock.mockReturnValue(now+30*60_000);await object.ingest(created.id,await sha256(key),await safeEvent(raw(),"b".repeat(32)));expect(alarm).toBe(now+60*60_000);expect(set).toHaveBeenCalledTimes(1);}finally{clock.mockRestore();}
 });
 it("persists before success, returns compatible 429/CORS, and rejects a revoked DSN",async()=>{
  const made=await (await request("/projects","POST",{name:"App",platform:"go"})).json() as {id:string;key:{dsn:string}};const key=new URL(made.key.dsn).username;
  const e=engine(local),ctx={waitUntil:(p:Promise<unknown>)=>p.catch(()=>{}),storage:{sql:e.sql,transactionSync:e.transaction,getAlarm:async()=>null,setAlarm:async()=>{}}} as unknown as DurableObjectState;const object=new PulseProject(ctx,state.env);Object.assign(state.env,{PULSE_PROJECT:{getByName:()=>object}});
  const send=()=>pulseIngest(new Request(`https://ingest.audit.invalid/api/${made.id}/envelope/?sentry_key=${key}&sentry_version=7`,{method:"POST",body:envelope(raw()),headers:{"content-type":"text/plain"}}),state.env);
  expect((await send()).status).toBe(200);expect((await send()).status).toBe(200);expect(store.usage(new Date().toISOString().slice(0,7)).accepted).toBe(1);
  Object.assign(state.env,{PULSE_INGEST_LIMIT:{limit:async()=>({success:false})}});const limited=await send();expect(limited.status).toBe(429);expect(limited.headers.get("retry-after")).toBe("10");expect(limited.headers.get("x-sentry-rate-limits")).toContain("error;default");expect(limited.headers.get("access-control-allow-origin")).toBe("*");
  Object.assign(state.env,{PULSE_INGEST_LIMIT:{limit:async()=>({success:true})}});d1.sqlite.prepare("UPDATE pulse_key SET revoked_at=1 WHERE hash=?").run(await sha256(key));expect((await send()).status).toBe(403);
 });
});
