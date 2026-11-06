import {beforeEach,afterEach,describe,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
import {ProjectStore,type ProjectSQL} from "./project-store";
import {safeEvent} from "./event";
import {authKey,checkDepth,ingestBytes,normalizeEventId,parseEnvelope,IngestError} from "./protocol";
import {reserveGrant} from "./quota";
import {pulseAccess,sha256} from "./limits";
import {pulseLimits} from "./limits";
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
beforeEach(()=>{
 vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));d1=new TestD1();local=new TestD1();seedUser(d1,"alice","max_pulse");seedUser(d1,"bob","ultra_pulse");seedUser(d1,"free");const e=engine(local);store=new ProjectStore(e.sql,e.transaction);Object.assign(state.env,{DB:d1.asD1(),STAFF_OWNER_EMAIL:"",PULSE_INGEST_LIMIT:{limit:async()=>({success:true})}});});
afterEach(()=>{d1.close();local.close();vi.unstubAllGlobals();vi.useRealTimers();});
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
 it("takes JSON null as a missing field (sentry-elixir)",async()=>{
  const event=await safeEvent({platform:"elixir",message:null,level:"error",exception:{values:[{type:"RuntimeError",value:"failure",module:null,stacktrace:{frames:[{filename:"lib/app.ex",function:"App.checkout/0",lineno:3,colno:null,in_app:true,context_line:null}]}}]},dist:null},"e".repeat(32));
  expect(event).toMatchObject({platform:"elixir",title:"failure",exceptions:[{frames:[{function:"App.checkout/0",line:3,column:null}]}]});
 });
 it("validates all authentication sources without letting cookies authorize ingestion",()=>{
  const key="a".repeat(48),url=`https://ingest.audit.invalid/api/123/store/?sentry_key=${key}&sentry_version=7`;
  expect(authKey(new Request(url),"123",`https://${key}@ingest.audit.invalid/123`)).toBe(key);
  expect(()=>authKey(new Request(url,{headers:{"x-sentry-auth":`Sentry sentry_version=7,sentry_key=${"b".repeat(48)}`}}),"123")).toThrow(IngestError);
  // Any whole protocol version, as Sentry: sentry-elixir still states 5.
  expect(authKey(new Request("https://ingest.audit.invalid/api/123/envelope/",{headers:{"x-sentry-auth":`Sentry sentry_version=5, sentry_client=sentry-elixir/14.0.1, sentry_key=${key}`}}),"123")).toBe(key);
  expect(authKey(new Request(`https://ingest.audit.invalid/api/123/store/?sentry_key=${key}&sentry_version=5`),"123")).toBe(key);
  for(const v of ["0","7a","100",""])expect(()=>authKey(new Request(`https://ingest.audit.invalid/api/123/store/?sentry_key=${key}&sentry_version=${v}`),"123")).toThrow(IngestError);
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
  const value=await safeEvent({...raw(),request:{data:"PRIVATE_NOTE",headers:{authorization:"PRIVATE_TOKEN"}},user:{email:"PRIVATE_EMAIL"},extra:{note:"PRIVATE_EXTRA"},contexts:{note:"PRIVATE_CONTEXT"},breadcrumbs:[{message:"clicked",data:{note:"PRIVATE_CRUMB"}}],message:"token=unsafe person@example.com /home/person/private.md",exception:undefined},"a".repeat(32));
  const encoded=JSON.stringify(value);for(const forbidden of ["PRIVATE_","unsafe","person@example.com","/home/person"])expect(encoded).not.toContain(forbidden);
  expect((await safeEvent({...raw(),fingerprint:["bucket","{{ default }}"]},"b".repeat(32))).group).toBe((await safeEvent({...raw(),fingerprint:["bucket","{{ default }}"]},"c".repeat(32))).group);
  expect((await safeEvent({...raw(),fingerprint:["ab","c"]},"b".repeat(32))).group).not.toBe((await safeEvent({...raw(),fingerprint:["a","bc"]},"c".repeat(32))).group);
  expect((await safeEvent(raw(undefined,"first failure"),"b".repeat(32))).group).not.toBe((await safeEvent(raw(undefined,"different failure"),"c".repeat(32))).group);
  const withLine=(line:number)=>({...raw(),exception:{values:[{type:"Error",value:"failure",stacktrace:{frames:[{filename:"app.js",function:"?",lineno:line,in_app:true}]}}]}});
  expect((await safeEvent(withLine(1),"b".repeat(32))).group).not.toBe((await safeEvent(withLine(2),"c".repeat(32))).group);
  // Named frames group by file and function: a release that moves the code
  // keeps the issue, and frames further out (a framework's own, which vary
  // between calls) do not split it; another caller does.
  const stack=(frames:{filename:string;function:string;lineno:number}[])=>({...raw(),exception:{values:[{type:"Error",value:"failure",stacktrace:{frames:frames.map(f=>({...f,in_app:true}))}}]}});
  const group=async(frames:{filename:string;function:string;lineno:number}[])=>(await safeEvent(stack(frames),"d".repeat(32))).group;
  const thrower={filename:"boom.mjs",function:"rota",lineno:18},caller={filename:"boom.mjs",function:"Object.handler",lineno:20};
  expect(await group([{filename:"nitro.mjs",function:"Object.handler",lineno:1935},caller,thrower])).toBe(await group([{filename:"nitro.mjs",function:"?",lineno:1617},caller,{...thrower,lineno:42}]));
  expect(await group([caller,thrower])).not.toBe(await group([{filename:"other.mjs",function:"checkout",lineno:3},thrower]));
 });
});
describe("Pulse atomic admission and retention",()=>{
 it("separates quota grants and accepted events at the UTC year/month boundary",async()=>{
  const before=Date.UTC(2026,11,31,23,59,59,999),after=before+1;
  const old=await reserveGrant(d1.asD1(),"alice","1",0,32,before),fresh=await reserveGrant(d1.asD1(),"alice","1",0,32,after);
  expect(old.id).not.toBe(fresh.id);store.importGrant(old,"2026-12");store.importGrant(fresh,"2027-01");const event=await safeEvent(raw(),"a".repeat(32));store.accept(event,"max_pulse",before);store.accept({...event,id:"b".repeat(32)},"max_pulse",after);expect(store.usage("2026-12").accepted).toBe(1);expect(store.usage("2027-01").accepted).toBe(1);expect(d1.sqlite.prepare("SELECT month,reserved FROM pulse_month ORDER BY month").all()).toEqual([{month:"2026-12",reserved:32},{month:"2027-01",reserved:32}]);
 });
 it("expires deduplication and issue state before physical cleanup has run",async()=>{
  const now=Date.now(),old=now-31*86400000,event=await safeEvent(raw(),"a".repeat(32));store.importGrant({id:"old-window",amount:32},new Date(old).toISOString().slice(0,7));store.accept(event,"max_pulse",old);store.setState(event.group,"ignored",0);store.importGrant({id:"new-window",amount:32},new Date(now).toISOString().slice(0,7));expect(store.accept(event,"max_pulse",now).duplicate).toBe(false);expect(store.issues(now-30*86400000,{limit:5})[0]).toMatchObject({state:"open",firstAt:now,lastAt:now,count:1});expect(store.samples(event.group,now-30*86400000)).toHaveLength(1);
 });
 it("reserves physical database headroom without consuming an event or grant",async()=>{
  const e=engine(local),sql={exec:e.sql.exec.bind(e.sql),databaseSize:pulseLimits.max_pulse.projectBytes-1024*1024} as ProjectSQL,limited=new ProjectStore(sql,e.transaction),now=Date.now(),month=new Date(now).toISOString().slice(0,7),event=await safeEvent(raw(),"a".repeat(32));limited.importGrant({id:"capacity",amount:32},month);expect(()=>limited.accept(event,"max_pulse",now)).toThrow(IngestError);expect(limited.usage(month).accepted).toBe(0);expect(limited.seen(event.id,0)).toBe(false);
 });
 it("paginates issues sharing the same timestamp without skipping IDs",async()=>{
  const now=Date.now();store.importGrant({id:"pagination",amount:32},new Date(now).toISOString().slice(0,7));for(let i=1;i<=3;i++)store.accept(await safeEvent(raw(i.toString().repeat(32),["alpha failure","beta failure","gamma failure"][i-1]),i.toString().repeat(32)),"max_pulse",now);const first=store.issues(0,{limit:2}),last=first[1];const next=store.issues(0,{limit:2,before:last.lastAt,beforeId:last.id});expect(new Set([...first,...next].map(i=>i.id)).size).toBe(3);
 });
 it("lists the environments and releases of recent issues, newest first",async()=>{
  const now=Date.now();store.importGrant({id:"facets",amount:32},new Date(now).toISOString().slice(0,7));
  store.accept(await safeEvent({...raw("1".repeat(32),"one"),environment:"staging",release:"1.0.0"},"1".repeat(32)),"max_pulse",now-1000);
  store.accept(await safeEvent({...raw("2".repeat(32),"two"),environment:"production",release:"1.1.0"},"2".repeat(32)),"max_pulse",now);
  store.accept(await safeEvent(raw("3".repeat(32),"three"),"3".repeat(32)),"max_pulse",now);
  expect(store.facets(0)).toEqual({environments:["production","staging"],releases:["1.1.0","1.0.0"]});
  expect(store.issues(0,{environment:"staging",limit:10}).map(i=>i.title)).toEqual(["one"]);expect(store.issues(0,{release:"1.1.0",q:"tw",limit:10}).map(i=>i.title)).toEqual(["two"]);
 });
 it("counts issues by state and their occurrences, for an environment and a release",async()=>{
  const now=Date.now();store.importGrant({id:"counts",amount:32},new Date(now).toISOString().slice(0,7));
  for(const [id,title,environment,release] of [["1","one","production","1.0.0"],["2","two","production","1.1.0"],["3","three","staging","1.0.0"]] as const)store.accept(await safeEvent({...raw(id.repeat(32),title),environment,release},id.repeat(32)),"max_pulse",now);
  // A second occurrence of "one" (same group, new event).
  const again=await safeEvent({...raw("4".repeat(32),"one"),environment:"production",release:"1.0.0"},"4".repeat(32));store.accept({...again,group:store.issues(0,{q:"one",limit:1})[0].id},"max_pulse",now);
  const [two,three]=[store.issues(0,{q:"two",limit:1})[0],store.issues(0,{q:"three",limit:1})[0]];store.setState(two.id,"resolved",0);store.setState(three.id,"ignored",0);
  expect(store.counts(0,{})).toEqual({open:1,resolved:1,ignored:1,total:3,occurrences:4});
  expect(store.counts(0,{environment:"production"})).toEqual({open:1,resolved:1,ignored:0,total:2,occurrences:3});
  expect(store.counts(0,{release:"1.0.0"})).toMatchObject({open:1,ignored:1,total:2});
  expect(store.counts(now+1,{})).toEqual({open:0,resolved:0,ignored:0,total:0,occurrences:0});
 });
 it("gives one issue with its occurrences per day, and nothing for an unknown or old one",async()=>{
  const now=Date.now();store.importGrant({id:"detail",amount:32},new Date(now).toISOString().slice(0,7));const event=await safeEvent(raw("d".repeat(32),"detail failure"),"d".repeat(32));store.accept(event,"max_pulse",now);store.accept(await safeEvent(raw("e".repeat(32),"detail failure"),"e".repeat(32)),"max_pulse",now);
  const found=store.issue(event.group,0);expect(found?.issue).toMatchObject({id:event.group,title:"detail failure",count:2,state:"open"});expect(found?.days).toEqual([{day:Math.floor(now/86_400_000),count:2}]);
  expect(store.issue("f".repeat(64),0)).toBeNull();expect(store.issue(event.group,now+1)).toBeNull();
 });
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
 it("skips stored samples that no longer match the event projection",async()=>{
  const now=Date.now(),event=await safeEvent(raw(),"c".repeat(32));store.importGrant({id:"g",amount:10},new Date(now).toISOString().slice(0,7));
  store.accept(event,"max_pulse",now);store.accept({...event,id:"d".repeat(32)},"max_pulse",now+1);
  local.sqlite.prepare("UPDATE sample SET event=? WHERE id=?").run(JSON.stringify({...event,id:"d".repeat(32),injected:true}),"d".repeat(32));
  expect(store.samples(event.group,0).map(s=>s.id)).toEqual([event.id]);
  local.sqlite.prepare("UPDATE sample SET event='not json' WHERE id=?").run(event.id);
  expect(store.samples(event.group,0)).toEqual([]);
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
 it("keeps monitor writes inside the same project physical capacity boundary",async()=>{
  const created=await(await request("/projects","POST",{name:"Capacity",platform:"go"})).json() as {id:string};d1.sqlite.prepare("INSERT INTO pulse_monitor(id,user_id,project_id,name,url,interval_minutes,next_at,created_at) VALUES('capacity-monitor','alice',?,'monitor','https://example.com',5,0,0)").run(created.id);
  const e=engine(local),sql={exec:e.sql.exec.bind(e.sql),databaseSize:pulseLimits.max_pulse.projectBytes-1024*1024} as ProjectSQL,ctx={waitUntil:(p:Promise<unknown>)=>p.catch(()=>{}),storage:{sql,transactionSync:e.transaction,getAlarm:async()=>null,setAlarm:async()=>{}}} as unknown as DurableObjectState,object=new PulseProject(ctx,state.env),now=Date.now();
  const result=await object.recordChecks([{id:"capacity-check",monitor:"capacity-monitor",revision:1,scheduledAt:now,checkedAt:now,intervalMinutes:5,result:{outcome:"up",status:200,durationMs:1,reason:"http"}}]);expect(result[0].state).toBe("unknown");expect(local.sqlite.prepare("SELECT count(*) AS n FROM monitor_check").get()?.n).toBe(0);
 });
 it("rotates concurrent DSNs atomically and keeps only one active key",async()=>{
  const created=await(await request("/projects","POST",{name:"Rotation",platform:"go"})).json() as {id:string};const rotations=await Promise.all([request(`/projects/${created.id}/keys`,"POST"),request(`/projects/${created.id}/keys`,"POST")]);expect(rotations.map(r=>r.status)).toEqual([201,201]);expect(d1.sqlite.prepare("SELECT count(*) AS n FROM pulse_key WHERE project_id=? AND revoked_at IS NULL").get(created.id)?.n).toBe(1);
 });
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
describe("what an occurrence keeps beyond the error",()=>{
 const frames=[{filename:"/home/dev/app/node_modules/lib.js",function:"lib",lineno:1,in_app:false,pre_context:["library code"],context_line:"throw e",post_context:[]},{filename:"/home/dev/app/src/checkout.js",function:"pay",lineno:42,colno:7,in_app:true,pre_context:["a","b","c","d","e","f","const key='sk_live_abcdefghijklmnop';"],context_line:"return order.customer.id; // mail me at dev@example.com",post_context:["}","x".repeat(400),"3","4","5","6"]}];
 const full={event_id:"c".repeat(32),platform:"javascript",exception:{values:[{type:"TypeError",value:"x is undefined",stacktrace:{frames}}]},
  request:{url:"https://shop.example.com/orders/12345/items/550e8400-e29b-41d4-a716-446655440000?token=secret&email=a@b.co",method:"post",headers:{cookie:"session=1"},data:{card:"4242"}},
  contexts:{browser:{name:"Chrome",version:"131.0.6778.86"},os:{name:"Windows",version:"10.0.19045"},runtime:{name:"node",version:"v22.11.0"},device:{model:"secret"}},
  user:{email:"a@b.co",ip_address:"1.2.3.4"},
  breadcrumbs:{values:[{timestamp:1793710000.5,category:"ui.click",message:"button#pay"},{type:"http",category:"fetch",data:{method:"post",url:"https://api.example.com/charge/99?card=4242",status_code:402}},{category:"navigation",data:{from:"/cart?ref=x",to:"/checkout/7"}},{category:"console",message:"token=abc123 failed"},"not a crumb"]}};
 it("keeps code only of app frames, cleaned and bounded",async()=>{
  const event=await safeEvent(full,"c".repeat(32)),[lib,own]=event.exceptions[0].frames;
  expect(lib.context).toBeUndefined();
  expect(own.context?.pre).toEqual(["c","d","e","f","const key='[secret]';"]);
  expect(own.context?.line).toBe("return order.customer.id; // mail me at [email]");
  expect(own.context?.post).toHaveLength(5);expect(own.context?.post[1]).toHaveLength(200);
 });
 it("keeps the route without query or ids and only names of the browser, system and runtime",async()=>{
  const event=await safeEvent(full,"c".repeat(32));
  expect(event.context).toEqual({method:"POST",route:"/orders/:id/items/:id",browser:"Chrome 131",os:"Windows 10",runtime:"node 22"});
  const stored=JSON.stringify(event);for(const leaked of ["token=secret","sk_live","a@b.co","1.2.3.4","4242","session=1","shop.example.com","6778"])expect(stored).not.toContain(leaked);
 });
 it("keeps the trail of actions, with HTTP calls and page changes as method, route and status",async()=>{
  const event=await safeEvent(full,"c".repeat(32));
  expect(event.breadcrumbs?.map(b=>[b.category,b.message])).toEqual([["ui.click","button#pay"],["fetch","POST /charge/:id 402"],["navigation","/cart → /checkout/:id"],["console","token=[secret] failed"]]);
  expect(event.breadcrumbs?.[0].at).toBe(1793710000500);
 });
 it("drops odd shapes of these parts without losing the error",async()=>{
  const event=await safeEvent({...full,request:"x",contexts:[1],breadcrumbs:{values:"nope"},exception:{values:[{type:"E",stacktrace:{frames:[{function:"f",in_app:true,pre_context:"bad",context_line:5}]}}]}},"c".repeat(32));
  expect(event.context).toBeUndefined();expect(event.breadcrumbs).toBeUndefined();expect(event.exceptions[0].frames[0].context).toBeUndefined();expect(event.title).toBe("E");
 });
});
