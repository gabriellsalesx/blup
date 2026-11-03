import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
import {sealTarget,openTarget,discordWebhook} from "./channel-crypto";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
import {deliverDiscord,deliverTelegram} from "./transports";
import {enqueueSignals,dispatchAlerts} from "./outbox";
let db:TestD1;
const WEBHOOK=`https://discord.com/api/webhooks/123456789012345678/${"test_fixture_".repeat(6)}`;
beforeEach(()=>{
 vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
 db=new TestD1();seedUser(db,"alice","max_pulse");seedUser(db,"bob","ultra_pulse");seedUser(db,"free");
 db.sqlite.exec("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) VALUES('1','alice','App','go','one',0),('2','bob','Other','go','two',0)");
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",PULSE_CHANNEL_KEY:btoa("x".repeat(32)),PULSE_TELEGRAM_TOKEN:`123456:${"local_fixture_".repeat(3)}`,PULSE_TELEGRAM_WEBHOOK_SECRET:"local-webhook-test-fixture",PULSE_PROJECT:{getByName:()=>({notification:async()=>({state:"open",title:"Sanitized error"})})}});
 vi.stubGlobal("fetch",vi.fn(async()=>{throw new Error("Unmocked provider forbidden");}));
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});
const request=(path:string,method="GET",body?:unknown,user="alice",headers:Record<string,string>={})=>pulseApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method,headers:{...(user?{authorization:`Bearer ${user}`} : {}),...(body===undefined?{}:{"content-type":"application/json"}),...headers},body:body===undefined?undefined:JSON.stringify(body)}),state.env);
async function channel(){return await(await request("/channels","POST",{type:"discord",webhook:WEBHOOK})).json() as {id:string};}
it("encrypts destinations with owner/channel/type binding and rejects forged targets",async()=>{
 const sealed=await sealTarget(state.env,"alice","channel","discord",WEBHOOK);expect(sealed).not.toContain(WEBHOOK);expect(await openTarget(state.env,"alice","channel","discord",sealed)).toBe(WEBHOOK);
 for(const [owner,id,type] of [["bob","channel","discord"],["alice","other","discord"],["alice","channel","telegram"]])await expect(openTarget(state.env,owner,id,type,sealed)).rejects.toThrow();
 for(const url of ["http://discord.com/api/webhooks/123/token","https://discord.com.evil.invalid/api/webhooks/123/token","https://127.0.0.1/api/webhooks/123/token",WEBHOOK+"?thread_id=1","https://user:password@discord.com/api/webhooks/123/token"])expect(discordWebhook.safeParse(url).success).toBe(false);
});
it("returns masked channels, enforces ownership and does not send on create/read",async()=>{
 const created=await channel();expect(created.id).toBeTruthy();const response=await request("/channels"),encoded=await response.text();expect(encoded).not.toContain("test_fixture_");expect(encoded).not.toContain("target");expect((await request(`/channels/${created.id}`,"DELETE",undefined,"bob")).status).toBe(404);
 expect((await request("/projects/2/channels","PUT",{channelIds:[created.id]})).status).toBe(404);expect((await request("/channels","POST",{type:"discord",webhook:WEBHOOK},"free")).status).toBe(403);expect(fetch).not.toHaveBeenCalled();
 expect((await request(`/channels/${created.id}/test`,"POST",{})).status).toBe(202);expect((await request(`/channels/${created.id}/test`,"POST",{})).status).toBe(429);expect(fetch).not.toHaveBeenCalled();
});
it("links Telegram only through a signed update, expiring code and authenticated confirmation",async()=>{
 const result=await(await request("/telegram/link","POST",{})).json() as {id:string;url:string};const code=new URL(result.url).searchParams.get("start");expect(code).toHaveLength(48);expect(db.sqlite.prepare("SELECT hash FROM pulse_telegram_link").get()?.hash).not.toBe(code);
 const update={update_id:123,message:{text:`/start ${code}`,chat:{id:12345,type:"private",username:"localtest"},from:{id:12345,is_bot:false}}};
 expect((await request("/telegram/webhook","POST",update,"")).status).toBe(401);
 const headers={"x-telegram-bot-api-secret-token":state.env.PULSE_TELEGRAM_WEBHOOK_SECRET};expect((await request("/telegram/webhook","POST",update,"",headers)).status).toBe(200);
 await request("/telegram/webhook","POST",{...update,message:{...update.message,chat:{...update.message.chat,id:999},from:{id:999,is_bot:false}}},"",headers);
 expect((await(await request(`/telegram/link/${result.id}`)).json() as {candidate:{id:string}}).candidate.id).toBe("12345");
 expect((await request(`/telegram/link/${result.id}/confirm`,"POST",{},"bob")).status).toBe(404);
 const confirmed=await Promise.all([request(`/telegram/link/${result.id}/confirm`,"POST",{}),request(`/telegram/link/${result.id}/confirm`,"POST",{})]);expect(confirmed.map(r=>r.status).sort()).toEqual([200,201]);expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_channel WHERE type='telegram'").get()?.n).toBe(1);
 const expired=await(await request("/telegram/link","POST",{})).json() as {id:string};db.sqlite.exec("UPDATE pulse_telegram_link SET state='candidate',chat_id='12345',expires_at=0");expect((await request(`/telegram/link/${expired.id}/confirm`,"POST",{})).status).toBe(409);
});
it("suppresses duplicates/cooldown and rechecks bindings and entitlement before delivery",async()=>{
 const created=await channel();await request("/projects/1/channels","PUT",{channelIds:[created.id]});
 const signal={id:"event-1",refId:"group",kind:"issue" as const,createdAt:Date.now()};await enqueueSignals(state.env,"1",[signal]);await enqueueSignals(state.env,"1",[signal,{...signal,id:"event-2"}]);expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_outbox").get()?.n).toBe(1);
 db.sqlite.exec("DELETE FROM pulse_project_channel");await dispatchAlerts(state.env);expect(fetch).not.toHaveBeenCalled();expect(db.sqlite.prepare("SELECT state FROM pulse_outbox").get()?.state).toBe("discarded");
 await request(`/channels/${created.id}/test`,"POST",{});db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");await dispatchAlerts(state.env);expect(fetch).not.toHaveBeenCalled();
});
it("obeys fractional Discord retry, disables invalid destinations and strips mentions",async()=>{
 const send=vi.fn(async(_input:RequestInfo|URL,init?:RequestInit)=>{expect(JSON.parse(String(init?.body))).toMatchObject({allowed_mentions:{parse:[]}});return Response.json({retry_after:2.25,global:true},{status:429});});vi.stubGlobal("fetch",send);expect(await deliverDiscord(WEBHOOK,"@everyone notice")).toMatchObject({sent:false,waitMs:2250,global:true});
 vi.stubGlobal("fetch",vi.fn(async()=>new Response(null,{status:404})));expect(await deliverDiscord(WEBHOOK,"test")).toMatchObject({permanent:true});
 vi.stubGlobal("fetch",vi.fn(async(_input:RequestInfo|URL,init?:RequestInit)=>{const body=JSON.parse(String(init?.body));expect(body).not.toHaveProperty("parse_mode");expect(body).not.toHaveProperty("allow_paid_broadcast");return Response.json({ok:false,error_code:429,parameters:{retry_after:7}});}));expect(await deliverTelegram(state.env,"12345","test")).toMatchObject({waitMs:7000,global:true});
});
it("persists retries without exceeding five attempts or leaking provider content",async()=>{
 const created=await channel();await request(`/channels/${created.id}/test`,"POST",{});
 vi.stubGlobal("fetch",vi.fn(async()=>new Response("private provider error with token",{status:500})));const clock=vi.spyOn(Date,"now"),initial=Date.now();
 for(let i=0;i<6;i++){clock.mockReturnValue(initial+i*400_000);await dispatchAlerts(state.env,initial+i*400_000);}
 expect(fetch).toHaveBeenCalledTimes(5);expect(db.sqlite.prepare("SELECT state,attempts FROM pulse_outbox").get()).toMatchObject({state:"discarded",attempts:5});expect(JSON.stringify(db.sqlite.prepare("SELECT * FROM pulse_channel").all())).not.toContain("private provider error");
});
it("sends a monitor test alert to the project's destinations with the monitor name",async()=>{
 db.sqlite.exec("INSERT INTO pulse_monitor(id,user_id,project_id,name,url,interval_minutes,next_at,created_at) VALUES('m1','alice','1','Checkout API','https://example.com',5,0,0),('m2','bob','2','Other','https://example.com',5,0,0)");
 expect((await request("/monitors/m1/test","POST",{})).status).toBe(409);
 const created=await channel();await request("/projects/1/channels","PUT",{channelIds:[created.id]});
 expect((await request("/monitors/m2/test","POST",{})).status).toBe(404);expect((await request("/monitors/m1/test","POST",{extra:1})).status).toBe(400);
 expect((await request("/monitors/m1/test","POST",{})).status).toBe(202);expect((await request("/monitors/m1/test","POST",{})).status).toBe(429);
 const send=vi.fn(async(_input:RequestInfo|URL,_init?:RequestInit)=>new Response(null,{status:204}));vi.stubGlobal("fetch",send);await dispatchAlerts(state.env);
 expect(send).toHaveBeenCalledTimes(1);expect(String(JSON.parse(String(send.mock.calls[0][1]?.body)).content)).toContain("Checkout API");
});
