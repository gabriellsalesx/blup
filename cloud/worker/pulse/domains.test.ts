import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {TestD1,TestBucket,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
import {disablePulseDomain} from "./domain-lifecycle";
import {servePulsePage} from "./public";
import {pulseMaintenance} from "./cleanup";
import {purgePulseAccount} from "./cleanup";
let db:TestD1,proved:boolean,providerFailure:boolean,calls:{method:string;path:string;body:unknown}[];
beforeEach(()=>{
 vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
 db=new TestD1();seedUser(db,"alice","ultra_pulse");seedUser(db,"bob","ultra_pulse");seedUser(db,"max","max_pulse");db.sqlite.exec("INSERT INTO pulse_page(id,user_id,slug,title,created_at,published) VALUES('page1','alice','public-page','Public',0,1)");
 Object.assign(state.env,{DB:db.asD1(),PULSE_ASSETS:new TestBucket().asR2(),STAFF_OWNER_EMAIL:"",CF_ZONE_ID:"a".repeat(32),CF_API_TOKEN:"fixture-only",SITES_CNAME_TARGET:"sites.audit.invalid",PULSE_PUBLIC_HOST:"pulse.audit.invalid",PULSE_PUBLIC_LIMIT:{limit:async()=>({success:true})},PULSE_PROJECT:{getByName:()=>({purge:async()=>{},cleanup:async()=>{}})}});proved=false;providerFailure=false;calls=[];
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
  const url=new URL(String(input));if(url.hostname==="cloudflare-dns.com"){
   const row=db.sqlite.prepare("SELECT proof FROM pulse_page_domain WHERE page_id='page1'").get();return Response.json({Status:0,Answer:proved?[{name:"_blup.status.audit.invalid.",type:16,data:`"${row?.proof}"`}]:[]});
  }
  expect(url.hostname).toBe("api.cloudflare.com");calls.push({method:init?.method??"GET",path:url.pathname,body:init?.body?JSON.parse(String(init.body)):null});if(providerFailure)return new Response("provider details",{status:503});if(init?.method==="DELETE")return Response.json({success:true});return Response.json({success:true,result:{id:"provider-fixture",hostname:"status.audit.invalid",status:"active",ssl:{status:"active"}}});
 }));
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});
const request=(method:string,body?:unknown,user="alice")=>pulseApi.fetch(new Request("https://cloud.audit.invalid/pages/page1/domain",{method,headers:{authorization:`Bearer ${user}`,"content-type":"application/json"},body:body===undefined?undefined:JSON.stringify(body)}),state.env);
it("proves TXT ownership before provisioning and enables Early Hints without exposing credentials",async()=>{
 const pending=await request("POST",{hostname:"status.audit.invalid"});expect(pending.status).toBe(200);expect(await pending.json()).toMatchObject({status:"pending_ownership",records:[{type:"TXT",name:"_blup.status.audit.invalid"},{type:"CNAME"}]});expect(calls).toHaveLength(0);
 proved=true;db.sqlite.exec("UPDATE pulse_page_domain SET checked_at=NULL");const active=await request("GET");expect(active.status).toBe(200);expect(await active.json()).toMatchObject({status:"active",ssl:"active"});expect(calls[0].body).toMatchObject({hostname:"status.audit.invalid",ssl:{settings:{early_hints:"on"}}});expect(JSON.stringify(await(await request("GET")).json())).not.toContain("fixture-only");
 expect((await servePulsePage(new Request("https://status.audit.invalid/"),state.env))?.status).toBe(200);
});
it("rejects another owner, forbidden hostname, extra body keys and a verified Cloud claim",async()=>{
 expect((await request("POST",{hostname:"status.audit.invalid"},"bob")).status).toBe(404);expect((await request("POST",{hostname:"pulse.getblup.site"})).status).toBe(400);expect((await request("POST",{hostname:"status.audit.invalid",provider_id:"foreign"})).status).toBe(400);
 await request("POST",{hostname:"status.audit.invalid"});db.sqlite.prepare("INSERT INTO hostname_claim VALUES(?,'site','other-site')").run("status.audit.invalid");proved=true;db.sqlite.exec("UPDATE pulse_page_domain SET checked_at=NULL");const taken=await request("GET");expect(taken.status).toBe(409);expect(await taken.json()).toMatchObject({code:"DOMAIN_TAKEN"});expect(calls).toHaveLength(0);
});
it("stops serving immediately after loss of domain eligibility and retains provider ID for failed deletion",async()=>{
 await request("POST",{hostname:"status.audit.invalid"});proved=true;db.sqlite.exec("UPDATE pulse_page_domain SET checked_at=NULL");await request("GET");db.sqlite.exec("UPDATE subscription SET plan='max_pulse' WHERE reference_id='alice'");expect((await servePulsePage(new Request("https://status.audit.invalid/"),state.env))?.status).toBe(404);
 providerFailure=true;await expect(disablePulseDomain(state.env,"page1",false,true)).rejects.toThrow();expect(db.sqlite.prepare("SELECT status,provider_id FROM pulse_page_domain").get()).toEqual({status:"disabled",provider_id:"provider-fixture"});providerFailure=false;await disablePulseDomain(state.env,"page1",false,true);expect(db.sqlite.prepare("SELECT provider_id FROM pulse_page_domain").get()?.provider_id).toBe(null);expect(db.sqlite.prepare("SELECT count(*) AS n FROM hostname_claim").get()?.n).toBe(0);
 expect((await request("DELETE")).status).toBe(200);expect(await(await request("GET")).json()).toBe(null);
});
it("does not remove a domain when an upgrade wins before the conditional disable claim",async()=>{
 db.sqlite.exec("INSERT INTO pulse_page_domain(page_id,hostname,proof,verified_at,provider_id,status,ssl) VALUES('page1','status.audit.invalid','proof',1,'provider-fixture','active','active')");await disablePulseDomain(state.env,"page1",false,true);expect(calls).toHaveLength(0);expect(db.sqlite.prepare("SELECT status FROM pulse_page_domain").get()?.status).toBe("active");
});
it("physically purges retired resources in bounded maintenance after their read-only window",async()=>{
 db.sqlite.prepare("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at,plan_disabled,retire_at) VALUES('1','alice','old','go','retired',0,1,?)").run(Date.now()-1000);db.sqlite.prepare("UPDATE pulse_page SET plan_disabled=1,retire_at=? WHERE id='page1'").run(Date.now()-1000);await pulseMaintenance(state.env);expect(db.sqlite.prepare("SELECT id FROM pulse_project WHERE id='1'").get()).toBeUndefined();expect(db.sqlite.prepare("SELECT id FROM pulse_page WHERE id='page1'").get()).toBeUndefined();
});
it("keeps monthly admission receipts after Pulse expiry so resubscribing cannot reset the quota",async()=>{
 const now=Date.now(),month=new Date(now).toISOString().slice(0,7);db.sqlite.prepare("INSERT INTO pulse_account(user_id,last_plan,read_only_until) VALUES('alice','ultra_pulse',?)").run(now-1);db.sqlite.prepare("INSERT INTO pulse_month VALUES('alice',?,32)").run(month);db.sqlite.prepare("INSERT INTO pulse_grant VALUES('receipt','alice','retired-project',?,32,?)").run(month,now);await pulseMaintenance(state.env,false,now);expect(db.sqlite.prepare("SELECT reserved FROM pulse_month WHERE user_id='alice'").get()?.reserved).toBe(32);expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_grant").get()?.n).toBe(1);await purgePulseAccount(state.env,"alice");expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_month").get()?.n).toBe(0);
});
it("tells whether a status page address is free, taken, reserved or malformed",async()=>{
 const check=async(slug:string,user:string|null="bob")=>pulseApi.fetch(new Request(`https://cloud.audit.invalid/pages/slug?slug=${encodeURIComponent(slug)}`,{headers:user?{authorization:`Bearer ${user}`}:{}}),state.env);
 expect(await (await check("nova-pagina")).json()).toEqual({state:"available"});
 expect(await (await check("public-page")).json()).toEqual({state:"taken"});
 expect(await (await check("status")).json()).toEqual({state:"reserved"});
 expect(await (await check("Com Espaco")).json()).toEqual({state:"invalid"});
 expect((await check("nova-pagina",null)).status).toBe(401);
});
