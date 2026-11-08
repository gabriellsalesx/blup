import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
import {sealTarget,openTarget,discordWebhook} from "./channel-crypto";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
import {deliverDiscord,deliverTelegram} from "./transports";
import {enqueueSignals,dispatchAlerts} from "./outbox";
import {purgeProject} from "./cleanup";
import {discordPayload,telegramPayload,slackPayload,webhookSignature,type AlertMessage} from "./alert-message";
import {at as nth} from "./test-at";
const note=(title:string):AlertMessage=>({event:"issue",label:"New issue",title,url:"https://pulse.getblup.io/issues",tone:"red",fields:[]});
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
async function channel(){return await(await request("/channels","POST",{projectId:"1",type:"discord",webhook:WEBHOOK})).json() as {id:string};}
it("encrypts destinations with owner/channel/type binding and rejects forged targets",async()=>{
 const sealed=await sealTarget(state.env,"alice","channel","discord",WEBHOOK);expect(sealed).not.toContain(WEBHOOK);expect(await openTarget(state.env,"alice","channel","discord",sealed)).toBe(WEBHOOK);
 for(const [owner,id,type] of [["bob","channel","discord"],["alice","other","discord"],["alice","channel","telegram"]] as const)await expect(openTarget(state.env,owner,id,type,sealed)).rejects.toThrow();
 for(const url of ["http://discord.com/api/webhooks/123/token","https://discord.com.evil.invalid/api/webhooks/123/token","https://127.0.0.1/api/webhooks/123/token",WEBHOOK+"?thread_id=1","https://user:password@discord.com/api/webhooks/123/token"])expect(discordWebhook.safeParse(url).success).toBe(false);
});
it("returns masked channels, enforces ownership and does not send on create/read",async()=>{
 const created=await channel();expect(created.id).toBeTruthy();const response=await request("/channels"),encoded=await response.text();expect(encoded).not.toContain("test_fixture_");expect(encoded).not.toContain("target");expect((await request(`/channels/${created.id}`,"DELETE",undefined,"bob")).status).toBe(404);
 expect((await request("/projects/2/channels","PUT",{channelIds:[created.id]})).status).toBe(404);expect((await request("/channels","POST",{projectId:"1",type:"discord",webhook:WEBHOOK},"free")).status).toBe(403);expect(fetch).not.toHaveBeenCalled();
 expect((await request(`/channels/${created.id}/test`,"POST",{})).status).toBe(202);expect((await request(`/channels/${created.id}/test`,"POST",{})).status).toBe(429);expect(fetch).not.toHaveBeenCalled();
});
it("links Telegram only through a signed update, expiring code and authenticated confirmation",async()=>{
 const result=await(await request("/telegram/link","POST",{})).json() as {id:string;url:string};const code=new URL(result.url).searchParams.get("start");expect(code).toHaveLength(48);expect(db.sqlite.prepare("SELECT hash FROM pulse_telegram_link").get()?.hash).not.toBe(code);
 const update={update_id:123,message:{text:`/start ${code}`,chat:{id:12345,type:"private",username:"localtest"},from:{id:12345,is_bot:false}}};
 expect((await request("/telegram/webhook","POST",update,"")).status).toBe(401);
 const headers={"x-telegram-bot-api-secret-token":state.env.PULSE_TELEGRAM_WEBHOOK_SECRET};expect((await request("/telegram/webhook","POST",update,"",headers)).status).toBe(200);
 await request("/telegram/webhook","POST",{...update,message:{...update.message,chat:{...update.message.chat,id:999},from:{id:999,is_bot:false}}},"",headers);
 expect((await(await request(`/telegram/link/${result.id}`)).json() as {candidate:{id:string}}).candidate.id).toBe("12345");
 expect((await request(`/telegram/link/${result.id}/confirm`,"POST",{projectId:"1"},"bob")).status).toBe(404);
 const confirmed=await Promise.all([request(`/telegram/link/${result.id}/confirm`,"POST",{projectId:"1"}),request(`/telegram/link/${result.id}/confirm`,"POST",{projectId:"1"})]);expect(confirmed.map(r=>r.status).sort()).toEqual([200,201]);expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_channel WHERE type='telegram'").get()?.n).toBe(1);
 const expired=await(await request("/telegram/link","POST",{})).json() as {id:string};db.sqlite.exec("UPDATE pulse_telegram_link SET state='candidate',chat_id='12345',expires_at=0");expect((await request(`/telegram/link/${expired.id}/confirm`,"POST",{projectId:"1"})).status).toBe(409);
});
it("suppresses duplicates/cooldown and rechecks bindings and entitlement before delivery",async()=>{
 const created=await channel();
 const signal={id:"event-1",refId:"group",kind:"issue" as const,createdAt:Date.now()};await enqueueSignals(state.env,"1",[signal]);await enqueueSignals(state.env,"1",[signal,{...signal,id:"event-2"}]);expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_outbox").get()?.n).toBe(1);
 db.sqlite.exec("DELETE FROM pulse_project_channel");await dispatchAlerts(state.env);expect(fetch).not.toHaveBeenCalled();expect(db.sqlite.prepare("SELECT state FROM pulse_outbox").get()?.state).toBe("discarded");
 await request(`/channels/${created.id}/test`,"POST",{});db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");await dispatchAlerts(state.env);expect(fetch).not.toHaveBeenCalled();
});
it("obeys fractional Discord retry, disables invalid destinations and strips mentions",async()=>{
 const send=vi.fn(async(_input:RequestInfo|URL,init?:RequestInit)=>{expect(JSON.parse(String(init?.body))).toMatchObject({allowed_mentions:{parse:[]}});return Response.json({retry_after:2.25,global:true},{status:429});});vi.stubGlobal("fetch",send);expect(await deliverDiscord(WEBHOOK,note("@everyone notice"))).toMatchObject({sent:false,waitMs:2250,global:true});
 vi.stubGlobal("fetch",vi.fn(async()=>new Response(null,{status:404})));expect(await deliverDiscord(WEBHOOK,note("test"))).toMatchObject({permanent:true});
 vi.stubGlobal("fetch",vi.fn(async(_input:RequestInfo|URL,init?:RequestInit)=>{const body=JSON.parse(String(init?.body));expect(body).not.toHaveProperty("parse_mode");expect(body).not.toHaveProperty("allow_paid_broadcast");return Response.json({ok:false,error_code:429,parameters:{retry_after:7}});}));expect(await deliverTelegram(state.env,"12345",note("test"))).toMatchObject({waitMs:7000,global:true});
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
 await channel();
 expect((await request("/monitors/m2/test","POST",{})).status).toBe(404);expect((await request("/monitors/m1/test","POST",{extra:1})).status).toBe(400);
 expect((await request("/monitors/m1/test","POST",{})).status).toBe(202);expect((await request("/monitors/m1/test","POST",{})).status).toBe(429);
 const send=vi.fn(async(_input:RequestInfo|URL,_init?:RequestInit)=>new Response(null,{status:204}));vi.stubGlobal("fetch",send);await dispatchAlerts(state.env);
 expect(send).toHaveBeenCalledTimes(1);const sent=JSON.parse(String(nth(send.mock.calls)[1]?.body)) as ReturnType<typeof discordPayload>;expect(sent).toMatchObject({username:"Blup Pulse",allowed_mentions:{parse:[]},embeds:[{title:"Checkout API",url:"https://pulse.getblup.io/monitors/m1",fields:[{name:"Projeto",value:"App"},{name:"Endereço",value:"https://example.com/"}]}]});
});
it("binds a new destination to the chosen project, in place of the one it had",async()=>{
 expect((await request("/channels","POST",{type:"discord",webhook:WEBHOOK})).status).toBe(400);expect((await request("/channels","POST",{projectId:"2",type:"discord",webhook:WEBHOOK})).status).toBe(404);
 const first=await channel(),second=await channel();
 expect(db.sqlite.prepare("SELECT channel_id AS id FROM pulse_project_channel WHERE project_id='1' AND type='discord'").all()).toEqual([{id:second.id}]);expect(first.id).not.toBe(second.id);
 const link=await(await request("/telegram/link","POST",{})).json() as {id:string};db.sqlite.prepare("UPDATE pulse_telegram_link SET state='candidate',chat_id='12345' WHERE id=?").run(link.id);
 expect((await request(`/telegram/link/${link.id}/confirm`,"POST",{projectId:"2"})).status).toBe(404);expect((await request(`/telegram/link/${link.id}/confirm`,"POST",{projectId:"1"})).status).toBe(201);
 expect(db.sqlite.prepare("SELECT type FROM pulse_project_channel WHERE project_id='1' ORDER BY type").all()).toEqual([{type:"discord"},{type:"telegram"}]);
});
it("shapes one alert as a Discord embed and as Telegram text with entities",()=>{
 const message:AlertMessage={event:"issue",label:"New issue",title:"TypeError: <b>x</b> is undefined",url:"https://pulse.getblup.io/issues?project=1",tone:"red",fields:[{name:"Project",value:"Meu App"},{name:"Environment",value:"development"},{name:"Release",value:""}]};
 expect(discordPayload(message).embeds[0]).toEqual({author:{name:"New issue"},title:"TypeError: <b>x</b> is undefined",url:"https://pulse.getblup.io/issues?project=1",color:0xef4444,fields:[{name:"Project",value:"Meu App",inline:true},{name:"Environment",value:"development",inline:true},{name:"Release",value:"-",inline:true}]});
 const tg=telegramPayload(message);
 expect(tg.text).toBe("Blup Pulse · New issue\nTypeError: <b>x</b> is undefined\n\nProject: Meu App\nEnvironment: development\nRelease: -");
 const at=(e:{offset:number;length:number})=>tg.text.slice(e.offset,e.offset+e.length);
 expect(tg.entities.map(e=>[e.type,at(e)])).toEqual([["bold","Blup Pulse · New issue"],["bold","TypeError: <b>x</b> is undefined"],["text_link","TypeError: <b>x</b> is undefined"],["bold","Project: "],["bold","Environment: "],["bold","Release: "]]);
 expect(nth(discordPayload({...message,title:"x".repeat(400)}).embeds).title).toHaveLength(256);
});
it("connects Slack and signed generic webhooks, refusing other or private addresses",async()=>{
 const SLACK="https://hooks.slack.com/services/T0ABCDEF1/B0ABCDEF2/abcdefghijklmnopqrstuvwx";
 const create=(body:Record<string,unknown>)=>request("/channels","POST",{projectId:"1",...body});
 for(const webhook of ["https://hooks.slack.com/workflows/x","http://hooks.slack.com/services/T0ABCDEF1/B0ABCDEF2/abcdefghijklmnopqrstuvwx","https://evil.example/services/T0ABCDEF1/B0ABCDEF2/abcdefghijklmnopqrstuvwx"])expect((await create({type:"slack",webhook})).status).toBe(400);
 for(const url of ["http://hooks.example.com/x","https://localhost/x","https://10.0.0.1/x","https://cloud.getblup.io/api","https://user:pw@hooks.example.com/x"])expect((await create({type:"webhook",url})).status).toBe(400);
 const slack=await create({type:"slack",webhook:SLACK});expect(slack.status).toBe(201);expect((await slack.json() as {secret:unknown}).secret).toBeNull();
 const made=await create({type:"webhook",url:"https://hooks.example.com/blup?team=1"});expect(made.status).toBe(201);const {secret}=await made.json() as {secret:string};expect(secret).toMatch(/^[a-f0-9]{64}$/);
 const listed=await(await request("/channels")).json() as {channels:{type:string}[];bindings:unknown[]};
 expect(listed.channels.map(c=>c.type).sort()).toEqual(["slack","webhook"]);expect(listed.bindings).toHaveLength(2);expect(JSON.stringify(listed)).not.toContain(secret);expect(JSON.stringify(db.sqlite.prepare("SELECT target FROM pulse_channel").all())).not.toContain(secret);
 const calls:{url:string;init?:RequestInit}[]=[];
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const url=String(input);if(url.startsWith("https://cloudflare-dns.com/"))return Response.json({Status:0,Answer:[{type:1,data:"93.184.216.34"}]});calls.push({url,init});return new Response("ok");}));
 state.env.PULSE_PROJECT={getByName:()=>({notification:async()=>({state:"open",title:"<!channel> Boom & co",environment:"production",release:"1.0.0"})})} as unknown as Cloudflare.Env["PULSE_PROJECT"];
 await enqueueSignals(state.env,"1",[{id:"event-s",refId:"group-s",kind:"issue",createdAt:Date.now()}]);await dispatchAlerts(state.env);
 const toSlack=calls.find(c=>c.url.startsWith("https://hooks.slack.com/")),toHook=calls.find(c=>c.url.startsWith("https://hooks.example.com/"));
 const slackBody=JSON.parse(String(toSlack?.init?.body)) as ReturnType<typeof slackPayload>;expect(nth(slackBody.attachments).color).toBe("#ef4444");expect(JSON.stringify(slackBody)).not.toContain("<!channel>");expect(JSON.stringify(slackBody)).toContain("&lt;!channel&gt; Boom &amp; co");
 const body=String(toHook?.init?.body),headers=new Headers(toHook?.init?.headers);expect(JSON.parse(body)).toMatchObject({version:1,event:"issue",title:"<!channel> Boom & co",fields:[{name:"Projeto",value:"App"},{name:"Ambiente",value:"production"},{name:"Versão",value:"1.0.0"}]});
 const [t,v1]=(headers.get("x-blup-signature")??"").split(",").map(p=>p.split("=")[1]);expect(v1).toBe(await webhookSignature(secret,body,Number(t)*1000).then(s=>s.split("v1=")[1]));
 expect(toHook?.init?.redirect).toBe("manual");expect(headers.get("x-blup-event")).toBe("issue");
});
it("tests a destination with an example of each chosen kind, only for its own projects",async()=>{
 const made=await request("/channels","POST",{projectId:"1",type:"webhook",url:"https://hooks.example.com/blup"});const {id}=await made.json() as {id:string};
 expect((await request(`/channels/${id}/test`,"POST",{projectId:"1",kinds:["issue","issue"]})).status).toBe(400);
 expect((await request(`/channels/${id}/test`,"POST",{projectId:"1",kinds:["other"]})).status).toBe(400);
 expect((await request(`/channels/${id}/test`,"POST",{projectId:"2",kinds:["issue"]})).status).toBe(404);
 const asked=await request(`/channels/${id}/test`,"POST",{projectId:"1",kinds:["test","issue","regression","monitor_down","monitor_up"]});expect(asked.status).toBe(202);
 const result=await asked.json() as {queued:number;sendsAt:number;testAgainAt:number};expect(result.queued).toBe(5);expect(result.sendsAt%60_000).toBe(0);expect(result.sendsAt).toBeGreaterThan(Date.now());
 expect((await request(`/channels/${id}/test`,"POST",{projectId:"1",kinds:["issue"]})).status).toBe(429);
 const listed=await(await request("/channels")).json() as {channels:{id:string;testAgainAt:number}[]};expect(listed.channels.find(c=>c.id===id)?.testAgainAt).toBe(result.testAgainAt);
 const bodies:Record<string,unknown>[]=[];
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const url=String(input);if(url.startsWith("https://cloudflare-dns.com/"))return Response.json({Status:0,Answer:[{type:1,data:"93.184.216.34"}]});bodies.push(JSON.parse(String(init?.body)));return new Response("ok");}));
 // The clock is still here; each round moves it past the 100 ms spacing.
 for(let i=0;i<5;i++){vi.setSystemTime(Date.now()+200);await dispatchAlerts(state.env);}
 expect(bodies.map(b=>b.event).sort()).toEqual(["issue","monitor_down","monitor_up","regression","test"]);expect(bodies.every(b=>b.test===true)).toBe(true);
 expect(bodies.find(b=>b.event==="monitor_down")).toMatchObject({tone:"red",label:"Indisponibilidade confirmada (teste)",description:"Este é um aviso de teste. Nada aconteceu de verdade."});
 expect(bodies.find(b=>b.event==="regression")).toMatchObject({tone:"amber",fields:[{name:"Projeto",value:"App"},{name:"Ambiente",value:"production"},{name:"Versão",value:"1.0.0"}]});
});
it("changes a destination's address and a webhook's secret, never showing the old one",async()=>{
 const discord=await channel(),NEXT=`https://discord.com/api/webhooks/223456789012345678/${"next_fixture_".repeat(6)}`;
 db.sqlite.exec(`UPDATE pulse_channel SET active=0,last_error='DESTINATION_DISABLED' WHERE id='${discord.id}'`);
 expect((await request(`/channels/${discord.id}`,"PATCH",{address:"https://evil.example/api/webhooks/1/x"})).status).toBe(400);
 expect((await request(`/channels/${discord.id}`,"PATCH",{address:NEXT},"bob")).status).toBe(404);
 const changed=await request(`/channels/${discord.id}`,"PATCH",{address:NEXT});expect(changed.status).toBe(200);expect(await changed.text()).not.toContain("next_fixture_");
 const row=db.sqlite.prepare("SELECT target,active,last_error AS lastError FROM pulse_channel WHERE id=?").get(discord.id) as {target:string;active:number;lastError:string|null};
 expect(row).toMatchObject({active:1,lastError:null});expect(row.target).not.toContain("next_fixture_");expect(await openTarget(state.env,"alice",discord.id,"discord",row.target)).toBe(NEXT);
 const made=await request("/channels","POST",{projectId:"1",type:"webhook",url:"https://hooks.example.com/blup"});const {id,secret}=await made.json() as {id:string;secret:string};
 expect((await request(`/channels/${discord.id}/secret`,"POST",{})).status).toBe(404);
 const moved=await request(`/channels/${id}`,"PATCH",{address:"https://hooks.example.com/other"});expect(moved.status).toBe(200);expect((await moved.json() as {label:string}).label).toBe("Webhook · hooks.example.com");
 const kept=JSON.parse(await openTarget(state.env,"alice",id,"webhook",(db.sqlite.prepare("SELECT target FROM pulse_channel WHERE id=?").get(id) as {target:string}).target)) as {url:string;secret:string};expect(kept).toEqual({url:"https://hooks.example.com/other",secret});
 const rotated=await(await request(`/channels/${id}/secret`,"POST",{})).json() as {secret:string};expect(rotated.secret).toMatch(/^[a-f0-9]{64}$/);expect(rotated.secret).not.toBe(secret);
 expect(JSON.parse(await openTarget(state.env,"alice",id,"webhook",(db.sqlite.prepare("SELECT target FROM pulse_channel WHERE id=?").get(id) as {target:string}).target))).toEqual({url:"https://hooks.example.com/other",secret:rotated.secret});
});
it("keeps every destination in a project: a replaced one and a removed project's leave",async()=>{
 db.sqlite.exec("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) VALUES('3','alice','Site','go','three',0),('4','alice','Api','go','four',0)");
 const first=await channel();await request(`/channels/${first.id}/test`,"POST",{});
 for(const project of ["3","4"])expect((await request("/channels","POST",{projectId:project,type:"discord",webhook:WEBHOOK})).status).toBe(201);
 // At the plan's three Discord destinations, replacing one still fits.
 const replaced=await request("/channels","POST",{projectId:"1",type:"discord",webhook:WEBHOOK});expect(replaced.status).toBe(201);const second=await replaced.json() as {id:string};
 const ids=()=>(db.sqlite.prepare("SELECT id FROM pulse_channel ORDER BY id").all() as {id:string}[]).map(r=>r.id);
 expect(ids()).not.toContain(first.id);expect(ids()).toContain(second.id);expect(ids()).toHaveLength(3);
 expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_outbox WHERE channel_id=?").get(first.id)?.n).toBe(0);
 expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_channel c WHERE NOT EXISTS(SELECT 1 FROM pulse_project_channel pc WHERE pc.channel_id=c.id)").get()?.n).toBe(0);
 expect((await request("/projects/1/channels","PUT",{channelIds:[second.id]})).status).toBe(404);
 state.env.PULSE_PROJECT={getByName:()=>({purge:async()=>{}})} as unknown as Cloudflare.Env["PULSE_PROJECT"];Object.assign(state.env,{PULSE_SOURCEMAPS:{list:async()=>({objects:[],truncated:false}),delete:async()=>{}}});
 await purgeProject(state.env,"1","one");expect(ids()).not.toContain(second.id);expect(ids()).toHaveLength(2);
});
it("turns a Telegram destination off only when the chat itself is gone",async()=>{
 const {telegramGone}=await import("./transports");
 expect(telegramGone(403,"Forbidden: bot was blocked by the user")).toBe(true);
 expect(telegramGone(400,"Bad Request: chat not found")).toBe(true);
 expect(telegramGone(400,"Bad Request: can't parse entities: unsupported start tag")).toBe(false);
 expect(telegramGone(400)).toBe(false);expect(telegramGone(500,"chat not found")).toBe(false);
});
it("lists every destination an Ultra account can have (ten projects, four types)",async()=>{
 const insert=db.sqlite.prepare("INSERT INTO pulse_channel(id,user_id,type,target,label,created_at) VALUES(?,?,?,'x',?,?)");
 const types=["discord","telegram","slack","webhook"];
 for(let i=0;i<40;i++)insert.run(`ch${i}`,"bob",types[i%4],`D${i}`,i);
 const list=await(await request("/channels","GET",undefined,"bob")).json() as {channels:unknown[]};
 expect(list.channels).toHaveLength(40);
});
