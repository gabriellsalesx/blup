import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {TestD1,TestBucket,seedUser} from "../security-test-support";
import {MonitorStore} from "./monitor-store";
import type {ProjectSQL} from "./project-store";
import {checkHTTP,monitorURL,publicIPv4,publicIPv6} from "./monitor-http";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{}}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {pulseApi} from "./api";
import {checkDue} from "./scheduler";
import {publicPage,servePulsePage} from "./public";
let db:TestD1,storage:TestD1,store:MonitorStore;
beforeEach(()=>{
 vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
 db=new TestD1();storage=new TestD1();seedUser(db,"alice","max_pulse");seedUser(db,"bob","ultra_pulse");
 db.sqlite.exec("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) VALUES('1','alice','Private project','go','object1',0),('2','bob','Other project','go','object2',0)");
 const sql={get databaseSize(){return 0;},exec(query:string,...args:unknown[]){if(query.includes(";")){storage.sqlite.exec(query);return {toArray:()=>[]};}const rows=storage.sqlite.prepare(query).all(...args);return {toArray:()=>rows};}} as unknown as ProjectSQL;
 storage.sqlite.exec("CREATE TABLE pending_alert(id TEXT PRIMARY KEY,issue_id TEXT,kind TEXT,created_at INTEGER)");
 store=new MonitorStore(sql,fn=>{storage.sqlite.exec("BEGIN");try{const result=fn();storage.sqlite.exec("COMMIT");return result;}catch(e){storage.sqlite.exec("ROLLBACK");throw e;}});
 Object.assign(state.env,{DB:db.asD1(),PULSE_ASSETS:new TestBucket().asR2(),STAFF_OWNER_EMAIL:"",PULSE_PUBLIC_HOST:"pulse.audit.invalid",PULSE_PUBLIC_LIMIT:{limit:async()=>({success:true})},PULSE_PROJECT:{getByName:()=>({recordChecks:async(observations:Parameters<MonitorStore["record"]>[0][])=>observations.map(o=>store.record(o)),monitorHistory:async(id:string,days:number)=>store.history(id,days)})}});
});
afterEach(()=>{db.close();storage.close();vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});
const request=(path:string,method="GET",body?:unknown,user="alice")=>pulseApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method,headers:{authorization:`Bearer ${user}`,...(body===undefined?{}:{"content-type":"application/json"})},body:body===undefined?undefined:JSON.stringify(body)}),state.env);
const fakePublic=vi.fn(async(input:RequestInfo|URL)=>String(input).startsWith("https://cloudflare-dns.com/")?Response.json({Status:0,Answer:[{type:1,data:"93.184.216.34"}]}):new Response("",{status:503}));
it("blocks URL tricks, literals, reserved ranges and our control hosts",()=>{
 for(const url of ["http://127.1","http://2130706433","http://0x7f000001","http://[::1]","https://[::ffff:127.0.0.1]","https://169.254.169.254/latest/meta-data","https://10.0.0.1","https://192.168.1.1","https://172.31.1.1","http://localhost","https://internal.local","http://user:pass@example.com","https://example.com:8080","file:///etc/passwd","https://getblup.io","https://cloud.getblup.io/api/me","https://something.workers.dev"]){expect(()=>monitorURL(url)).toThrow();}
 expect(publicIPv4("100.64.0.1")).toBe(false);expect(publicIPv4("198.18.0.1")).toBe(false);expect(publicIPv6("2001:db8::1")).toBe(false);expect(publicIPv6("2606:4700:4700::1111")).toBe(true);expect(monitorURL("https://example.com/").href).toBe("https://example.com/");expect(monitorURL("https://getblup.io/",true).hostname).toBe("getblup.io");
});
it("checks DNS and every redirect, sends no ambient secrets and cancels bodies",async()=>{
 let canceled=0;const send=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{if(String(input).startsWith("https://cloudflare-dns.com/"))return Response.json({Status:0,Answer:[{type:1,data:"93.184.216.34"}]});expect(init?.redirect).toBe("manual");const headers=new Headers(init?.headers);for(const h of ["authorization","cookie","x-blup-upload-ticket"])expect(headers.has(h)).toBe(false);return new Response(new ReadableStream({cancel(){canceled++;}}),{status:302,headers:{location:"http://169.254.169.254/"}});});
 expect(await checkHTTP("https://example.com/",false,send)).toMatchObject({outcome:"unknown",reason:"blocked"});expect(send).toHaveBeenCalledTimes(3);expect(canceled).toBe(1);
 const privateDNS=vi.fn(async()=>Response.json({Status:0,Answer:[{type:1,data:"127.0.0.1"}]}));expect(await checkHTTP("https://example.com/",false,privateDNS)).toMatchObject({outcome:"unknown",reason:"blocked"});expect(privateDNS).toHaveBeenCalledTimes(1);
});
it("limits redirects to three hops and turns network timeout into a bounded failure",async()=>{
 const send=vi.fn(async(input:RequestInfo|URL)=>String(input).startsWith("https://cloudflare-dns.com/")?Response.json({Status:0,Answer:[{type:1,data:"93.184.216.34"}]}):new Response(null,{status:302,headers:{location:"/next"}}));expect(await checkHTTP("https://example.com/",false,send)).toMatchObject({outcome:"down",status:302});expect(send).toHaveBeenCalledTimes(12);
 expect(await checkHTTP("https://example.com/",false,async()=>{throw new TypeError("network");})).toMatchObject({outcome:"down",reason:"network"});
});
it("requires two consecutive failures/successes, ignores duplicate checks and records gaps",()=>{
 const now=Date.UTC(2026,10,15,12),base={id:"one",monitor:"monitor1",revision:1,scheduledAt:now,checkedAt:now,intervalMinutes:5,result:{outcome:"down" as const,status:500,durationMs:1,reason:"http" as const}};
 expect(store.record(base).state).toBe("unknown");expect(store.record(base).duplicate).toBe(true);
 expect(store.record({...base,id:"two",scheduledAt:now+300_000,checkedAt:now+300_000}).state).toBe("down");
 for(let i=3;i<=4;i++)expect(store.record({...base,id:String(i),scheduledAt:now+(i-1)*300_000,checkedAt:now+(i-1)*300_000,result:{...base.result,outcome:"up",status:200}}).state).toBe(i===4?"up":"down");
 expect(store.record({...base,id:"gap",scheduledAt:now+1_200_000,checkedAt:now+3_600_000}).state).toBe("unknown");expect(store.history("monitor1",30,now+3_600_000).days[0]).toMatchObject({up:2,down:2,unknown:1});
 expect(store.history("monitor1",30,now+8*86_400_000).checks).toEqual([]);store.cleanup(30,now+31*86_400_000);expect(store.history("monitor1",30,now+31*86_400_000).days).toEqual([]);
});
it.each([Date.UTC(2026,10,15,23,5),Date.UTC(2026,10,30,23,5),Date.UTC(2026,11,31,23,5)])("keeps transitions and UTC daily buckets correct across midnight at %s",now=>{
 const base={id:"one",monitor:"monitor1",revision:1,scheduledAt:now,checkedAt:now,intervalMinutes:5,result:{outcome:"down" as const,status:503,durationMs:1,reason:"http" as const}};
 expect(store.record(base).state).toBe("unknown");expect(store.record({...base,id:"two",scheduledAt:now+300000,checkedAt:now+300000}).state).toBe("down");
 for(let i=3;i<=4;i++)expect(store.record({...base,id:String(i),scheduledAt:now+(i-1)*300000,checkedAt:now+(i-1)*300000,result:{...base.result,outcome:"up",status:200}}).state).toBe(i===4?"up":"down");
 expect(store.record({...base,id:"gap",scheduledAt:now+1200000,checkedAt:now+3600000}).state).toBe("unknown");
 const days=store.history("monitor1",30,now+3600000).days;
 expect(days).toEqual([expect.objectContaining({day:Math.floor(now/86400000)+1,up:0,down:0,unknown:1}),expect.objectContaining({day:Math.floor(now/86400000),up:2,down:2,unknown:0})]);
});
it("keeps cleanup scheduled while expired monitor checks exceed one batch",()=>{
 const now=Date.now(),old=now-8*86400000;
 for(let i=0;i<501;i++)store.record({id:`expired-${i}`,monitor:"monitor1",revision:1,scheduledAt:old,checkedAt:old,intervalMinutes:5,result:{outcome:"up",status:200,durationMs:1,reason:"http"}});
 expect(store.cleanup(30,now)).toBe(true);expect(store.sql.exec<{n:number}>("SELECT count(*) AS n FROM monitor_check").toArray()[0].n).toBe(1);expect(store.cleanup(30,now)).toBe(false);expect(store.sql.exec<{n:number}>("SELECT count(*) AS n FROM monitor_check").toArray()[0].n).toBe(0);
});
it("enforces owner, interval and count; cron overlap cannot check twice",async()=>{
 expect((await request("/monitors","POST",{projectId:"2",name:"x",url:"https://example.com",intervalMinutes:5})).status).toBe(404);
 expect((await request("/monitors","POST",{projectId:"1",name:"x",url:"https://example.com",intervalMinutes:2})).status).toBe(403);
 const made=await request("/monitors","POST",{projectId:"1",name:"Private monitor",url:"https://example.com/private?secret=hidden",intervalMinutes:5});expect(made.status).toBe(201);const {id}=await made.json() as {id:string};
 expect((await request(`/monitors/${id}`,"DELETE",undefined,"bob")).status).toBe(404);
 vi.stubGlobal("fetch",fakePublic);const now=Date.now(),clock=vi.spyOn(Date,"now").mockReturnValue(now);await Promise.all([checkDue(state.env,now),checkDue(state.env,now)]);expect(fakePublic).toHaveBeenCalledTimes(3);clock.mockReturnValue(now+300_000);await checkDue(state.env,now+300_000);expect(fakePublic).toHaveBeenCalledTimes(6);expect(db.sqlite.prepare("SELECT state FROM pulse_monitor WHERE id=?").get(id)?.state).toBe("down");
});
it("keeps concurrent monitor creation within the account cap",async()=>{
 const responses=await Promise.all(Array.from({length:11},(_,i)=>request("/monitors","POST",{projectId:"1",name:`Monitor ${i}`,url:"https://example.com/",intervalMinutes:5})));
 expect(responses.filter(r=>r.status===201)).toHaveLength(10);expect(responses.filter(r=>r.status===409)).toHaveLength(1);
});
it("publishes only explicit component names and incident text, escapes attacks and revokes on expiry",async()=>{
 const monitor=await(await request("/monitors","POST",{projectId:"1",name:"PRIVATE_MONITOR_NAME",url:"https://example.com/private-secret",intervalMinutes:5})).json() as {id:string};
 expect((await request("/pages","POST",{title:"Public",slug:"public-page",language:"en",components:[{monitorId:monitor.id,name:"Public service"}]},"bob")).status).toBe(404);
 const created=await request("/pages","POST",{title:"<script>attack()</script>",slug:"public-page",language:"en",components:[{monitorId:monitor.id,name:"Public service"}]});expect(created.status).toBe(201);const {id}=await created.json() as {id:string};
 await request(`/pages/${id}`,"PATCH",{published:true});expect((await request(`/pages/${id}/incidents`,"POST",{title:"Incident",body:'<img src=x onerror="attack()">',state:"investigating",publishedAt:new Date().toISOString(),services:[{monitorId:monitor.id,status:"degraded"}]})).status).toBe(201);
 const cache=new Map<string,Response>();vi.stubGlobal("caches",{default:{match:async(key:Request)=>cache.get(key.url)?.clone(),put:async(key:Request,response:Response)=>{cache.set(key.url,response);}}});
 const page=await publicPage(state.env,"public-page"),data=JSON.stringify(page);for(const privateValue of ["PRIVATE_MONITOR_NAME","private-secret","object1","alice","Private project"])expect(data).not.toContain(privateValue);
 const html=await(await servePulsePage(new Request("https://pulse.audit.invalid/public-page/"),state.env))?.text();expect(html).toContain("&lt;script&gt;");expect(html).not.toContain("<script>attack");expect(html?.match(/<script>/g)).toHaveLength(1);expect(html).not.toContain("<img");
 // The list links each incident to its own page, with every update.
 const incidentId=(page as {incidents:{id:string}[]}).incidents[0].id;expect(html).toContain(`href="incidents/${incidentId}?lang=en"`);
 const own=await servePulsePage(new Request(`https://pulse.audit.invalid/public-page/incidents/${incidentId}`),state.env),ownHtml=await own?.text()??"";
 expect(own?.status).toBe(200);expect(ownHtml).toContain("&lt;img src=x");expect(ownHtml).not.toContain("<img src=x");expect(ownHtml).toContain('href="../incidents?lang=en"');expect(ownHtml).toContain('class="brand" href="../"');
 // The previous incidents, three months at a time.
 const history=await(await servePulsePage(new Request("https://pulse.audit.invalid/public-page/incidents"),state.env))?.text()??"";
 expect(history).toContain("Previous incidents");expect(history).toContain(`href="incidents/${incidentId}?lang=en"`);expect(history).toContain("November 2026");expect(history).toContain("September 2026");expect(history).toContain('class="arrow off"');
 const older=await(await servePulsePage(new Request("https://pulse.audit.invalid/public-page/incidents?to=2026-08"),state.env))?.text()??"";expect(older).toContain("August 2026");expect(older).not.toContain(`incidents/${incidentId}`);
expect(ownHtml).toContain("Public service");
 expect((await servePulsePage(new Request(`https://pulse.audit.invalid/public-page/incidents/${crypto.randomUUID()}`),state.env))?.status).toBe(404);
 db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");expect(await publicPage(state.env,"public-page")).toEqual({unavailable:true});expect((await servePulsePage(new Request("https://pulse.audit.invalid/public-page/"),state.env))?.status).toBe(503);
});
it("keeps how an incident leaves each service and when it was published",async()=>{
 const monitor=await(await request("/monitors","POST",{projectId:"1",name:"API",url:"https://example.com/",intervalMinutes:5})).json() as {id:string};
 const other=await(await request("/monitors","POST",{projectId:"1",name:"Off page",url:"https://example.com/other",intervalMinutes:5})).json() as {id:string};
 const {id}=await(await request("/pages","POST",{title:"Status",slug:"status-services",language:"en",components:[{monitorId:monitor.id,name:"Public API"}]})).json() as {id:string};
 await request(`/pages/${id}`,"PATCH",{published:true});
 const base={title:"Slow API",body:"Looking into it",state:"investigating"};const now=Date.now();
 expect((await request(`/pages/${id}/incidents`,"POST",{...base,publishedAt:new Date(now+3_600_000).toISOString(),services:[]})).status).toBe(400);
 expect((await request(`/pages/${id}/incidents`,"POST",{...base,publishedAt:new Date(now-400*86_400_000).toISOString(),services:[]})).status).toBe(400);
 expect((await request(`/pages/${id}/incidents`,"POST",{...base,publishedAt:new Date(now).toISOString(),services:[{monitorId:other.id,status:"downtime"}]})).status).toBe(400);
 expect((await request(`/pages/${id}/incidents`,"POST",{...base,publishedAt:new Date(now).toISOString(),impact:"degraded",services:[]})).status).toBe(400);
 const earlier=now-2*3_600_000,made=await request(`/pages/${id}/incidents`,"POST",{...base,publishedAt:new Date(earlier).toISOString(),services:[{monitorId:monitor.id,status:"downtime"}]});expect(made.status).toBe(201);const incident=(await made.json() as {id:string}).id;
 const listed=(await(await request(`/pages/${id}/incidents`)).json() as {incidents:{createdAt:number;impact:string;services:unknown[]}[]}).incidents[0];
 expect(listed).toMatchObject({createdAt:earlier,impact:"major_outage",updates:1,services:[{monitorId:monitor.id,status:"downtime"}]});
 let page=await publicPage(state.env,"status-services");if(!page||page.unavailable)throw new Error("page");
 expect(page.components[0]).toMatchObject({name:"Public API",incident:"downtime"});expect(page.incidents[0].services).toEqual([{name:"Public API",status:"downtime"}]);
 expect((await request(`/pages/${id}/incidents/${incident}/updates`,"POST",{body:"Fixed",state:"resolved",services:[{monitorId:monitor.id,status:"resolved"}]})).status).toBe(201);
 page=await publicPage(state.env,"status-services");if(!page||page.unavailable)throw new Error("page");
 expect(page.components[0].incident).toBeNull();expect(page.incidents[0].services).toEqual([{name:"Public API",status:"resolved"}]);
 expect(db.sqlite.prepare("SELECT impact FROM pulse_incident WHERE id=?").get(incident)?.impact).toBe("operational");
 const updates=(await(await request(`/pages/${id}/incidents/${incident}/updates`)).json() as {updates:{body:string;state:string}[]}).updates;
 expect(updates.map(u=>[u.body,u.state])).toEqual([["Fixed","resolved"],["Looking into it","investigating"]]);
 expect((await request(`/pages/${id}/incidents/${incident}/updates`,"GET",undefined,"bob")).status).toBe(404);
 expect((await request(`/pages/${id}/incidents/${incident}`,"PATCH",{title:"Renamed"},"bob")).status).toBe(404);expect((await request(`/pages/${id}/incidents/${incident}`,"PATCH",{title:"",state:"x"})).status).toBe(400);
 expect((await request(`/pages/${id}/incidents/${incident}`,"PATCH",{title:"Renamed"})).status).toBe(200);expect(db.sqlite.prepare("SELECT title FROM pulse_incident WHERE id=?").get(incident)?.title).toBe("Renamed");
 expect((await request(`/pages/${id}/incidents/${incident}`,"DELETE",undefined,"bob")).status).toBe(404);expect((await request(`/pages/${id}/incidents/${incident}`,"DELETE")).status).toBe(200);
 for(const table of ["pulse_incident","pulse_incident_update","pulse_incident_component"])expect(db.sqlite.prepare(`SELECT count(*) AS n FROM ${table}`).get()?.n).toBe(0);
});
it("keeps checks during a maintenance apart: no state change, no alert, not in the uptime",()=>{
 const now=Date.UTC(2026,10,15,12),check=(id:string,at:number,outcome:"up"|"down")=>({id,monitor:"m",revision:1,scheduledAt:at,checkedAt:at,intervalMinutes:5,result:{outcome,status:outcome==="up"?200:500,durationMs:1,reason:"http" as const}});
 store.record(check("a",now,"up"));expect(store.record(check("b",now+300_000,"up")).state).toBe("up");
 expect(store.record(check("c",now+600_000,"down"),true).state).toBe("up");expect(store.record(check("d",now+900_000,"down"),true).state).toBe("up");
 expect(storage.sqlite.prepare("SELECT count(*) AS n FROM pending_alert WHERE kind='monitor_down'").get()?.n).toBe(0);
 expect(store.history("m",30,now+900_000).days[0]).toMatchObject({up:2,down:0,maintenance:2});
 store.record(check("e",now+1_200_000,"down"));expect(store.record(check("f",now+1_500_000,"down")).state).toBe("down");
});
it("schedules maintenance on a page's services and shows it while it lasts",async()=>{
 const monitor=await(await request("/monitors","POST",{projectId:"1",name:"API",url:"https://example.com/",intervalMinutes:5})).json() as {id:string};
 const other=await(await request("/monitors","POST",{projectId:"1",name:"Off page",url:"https://example.com/other",intervalMinutes:5})).json() as {id:string};
 const {id}=await(await request("/pages","POST",{title:"Status",slug:"status-maintenance",language:"en",components:[{monitorId:monitor.id,name:"Public API"}]})).json() as {id:string};
 await request(`/pages/${id}`,"PATCH",{published:true});
 const now=Date.now(),at=(ms:number)=>new Date(ms).toISOString(),body={title:"Database upgrade",body:"Short downtime expected.",services:[monitor.id]};
 for(const bad of [{...body,startsAt:at(now+7_200_000),endsAt:at(now+3_600_000)},{...body,startsAt:at(now),endsAt:at(now+31*86_400_000)},{...body,startsAt:at(now),endsAt:at(now+3_600_000),services:[other.id]},{...body,startsAt:at(now),endsAt:at(now+3_600_000),services:[]},{...body,startsAt:at(now),endsAt:at(now+3_600_000),extra:true}])expect((await request(`/pages/${id}/maintenance`,"POST",bad)).status).toBe(400);
 expect((await request(`/pages/${id}/maintenance`,"POST",{...body,startsAt:at(now),endsAt:at(now+3_600_000)},"bob")).status).toBe(404);
 const later=await request(`/pages/${id}/maintenance`,"POST",{...body,title:"Later",startsAt:at(now+86_400_000),endsAt:at(now+90_000_000)});expect(later.status).toBe(201);
 const made=await request(`/pages/${id}/maintenance`,"POST",{...body,startsAt:at(now-60_000),endsAt:at(now+3_600_000)});expect(made.status).toBe(201);const maintenance=(await made.json() as {id:string}).id;
 const listed=(await(await request(`/pages/${id}/maintenance`)).json() as {maintenance:{id:string;state:string;services:string[]}[]}).maintenance;
 expect(listed.map(m=>[m.state,m.services])).toEqual([["scheduled",[monitor.id]],["active",[monitor.id]]]);
 let page=await publicPage(state.env,"status-maintenance");if(!page||page.unavailable)throw new Error("page");
 expect(page.components[0].maintenance).toBe(true);expect(page.maintenance.map(m=>[m.title,m.state,m.services])).toEqual([["Database upgrade","active",["Public API"]],["Later","scheduled",["Public API"]]]);
 const html=await(await servePulsePage(new Request("https://pulse.audit.invalid/status-maintenance/"),state.env))?.text();expect(html).toContain("Database upgrade");
 expect((await request(`/pages/${id}/maintenance/${maintenance}`,"PUT",{...body,startsAt:at(now-3_600_000),endsAt:at(now-60_000)})).status).toBe(200);
 page=await publicPage(state.env,"status-maintenance");if(!page||page.unavailable)throw new Error("page");expect(page.components[0].maintenance).toBe(false);expect(page.maintenance.map(m=>m.title)).toEqual(["Later"]);
 expect((await request(`/pages/${id}/maintenance/${maintenance}`,"DELETE",undefined,"bob")).status).toBe(404);expect((await request(`/pages/${id}/maintenance/${maintenance}`,"DELETE")).status).toBe(200);
 expect(db.sqlite.prepare("SELECT count(*) AS n FROM pulse_maintenance_component WHERE maintenance_id=?").get(maintenance)?.n).toBe(0);
});
it("alerts a confirmed fall and its return, not a new monitor that is up",()=>{
 const now=Date.UTC(2026,10,15,12),check=(id:string,step:number,outcome:"up"|"down")=>({id,monitor:"alerted",revision:1,scheduledAt:now+step*300_000,checkedAt:now+step*300_000,intervalMinutes:5,result:{outcome,status:outcome==="up"?200:500,durationMs:1,reason:"http" as const}});
 const kinds=()=>storage.sqlite.prepare("SELECT kind FROM pending_alert WHERE issue_id='alerted' ORDER BY created_at").all().map(r=>r.kind);
 store.record(check("a",0,"up"));store.record(check("b",1,"up"));expect(kinds()).toEqual([]);
 store.record(check("c",2,"down"));store.record(check("d",3,"down"));store.record(check("e",4,"down"));expect(kinds()).toEqual(["monitor_down"]);
 store.record(check("f",5,"up"));store.record(check("g",6,"up"));store.record(check("h",7,"up"));expect(kinds()).toEqual(["monitor_down","monitor_up"]);
 // Down again, then a check without an answer, then back: still one green.
 store.record(check("i",8,"down"));store.record(check("j",9,"down"));store.record({...check("k",10,"up"),result:{outcome:"unknown",status:null,durationMs:1,reason:"network"}});
 store.record(check("l",11,"up"));store.record(check("m",12,"up"));expect(kinds()).toEqual(["monitor_down","monitor_up","monitor_down","monitor_up"]);
});
it("checks with the monitor's method, statuses, keyword, timeout and headers",async()=>{
 const dns=(input:RequestInfo|URL)=>String(input).startsWith("https://cloudflare-dns.com/")?Response.json({Status:0,Answer:[{type:1,data:"93.184.216.34"}]}):null;
 const seen:{url:string;method:string;auth:string|null}[]=[];
 const site=(status:number,body="",location?:string)=>vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const d=dns(input);if(d)return d;seen.push({url:String(input),method:init?.method??"GET",auth:new Headers(init?.headers).get("authorization")});return new Response(init?.method==="HEAD"?null:body,{status,headers:location?{location}:{}});});
 const base={method:"GET" as const,expectedStatus:[] as number[],keyword:null,timeoutSeconds:10};
 expect((await checkHTTP("https://example.com/",false,site(404),base)).outcome).toBe("down");
 expect((await checkHTTP("https://example.com/",false,site(404),{...base,expectedStatus:[404]})).outcome).toBe("up");
 expect(await checkHTTP("https://example.com/",false,site(200,"<h1>Erro interno</h1>"),{...base,keyword:{text:"Bem-vindo",mode:"contains"}})).toMatchObject({outcome:"down",reason:"keyword"});
 expect((await checkHTTP("https://example.com/",false,site(200,"<h1>BEM-VINDO</h1>"),{...base,keyword:{text:"bem-vindo",mode:"contains"}})).outcome).toBe("up");
 expect(await checkHTTP("https://example.com/",false,site(200,"Fatal error"),{...base,keyword:{text:"fatal error",mode:"absent"}})).toMatchObject({outcome:"down",reason:"keyword"});
 seen.length=0;await checkHTTP("https://example.com/",false,site(200),{...base,method:"HEAD"});expect(seen[0].method).toBe("HEAD");
 seen.length=0;const hop=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const d=dns(input);if(d)return d;seen.push({url:String(input),method:"GET",auth:new Headers(init?.headers).get("authorization")});return String(input).startsWith("https://example.com/")?new Response(null,{status:302,headers:{location:"https://other.example.org/login"}}):new Response("ok");});
 await checkHTTP("https://example.com/api",false,hop,base,[{name:"Authorization",value:"Bearer secret-value"}]);
 expect(seen.map(s=>[new URL(s.url).hostname,s.auth])).toEqual([["example.com","Bearer secret-value"],["other.example.org",null]]);
 const slow=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{const d=dns(input);if(d)return d;return await new Promise<Response>((_,reject)=>init?.signal?.addEventListener("abort",()=>reject(new Error("aborted"))));});
 vi.useRealTimers();expect(await checkHTTP("https://example.com/",false,slow,{...base,timeoutSeconds:1})).toMatchObject({outcome:"down",reason:"timeout"});
});
it("keeps request headers sealed and never sends their values back",async()=>{
 Object.assign(state.env,{PULSE_CHANNEL_KEY:btoa("k".repeat(32))});
 const options={method:"GET",expectedStatus:[200],keyword:{text:"ok",mode:"contains"},timeoutSeconds:5};
 expect((await request("/monitors","POST",{projectId:"1",name:"API",url:"https://example.com/",intervalMinutes:5,options:{...options,method:"HEAD"}})).status).toBe(400);
 expect((await request("/monitors","POST",{projectId:"1",name:"API",url:"https://example.com/",intervalMinutes:5,headers:[{name:"Host",value:"x"}]})).status).toBe(400);
 const made=await request("/monitors","POST",{projectId:"1",name:"API",url:"https://example.com/",intervalMinutes:5,options,headers:[{name:"Authorization",value:"Bearer TOP_SECRET_VALUE"},{name:"X-Env",value:"prod"}]});expect(made.status).toBe(201);const {id}=await made.json() as {id:string};
 expect(JSON.stringify(db.sqlite.prepare("SELECT * FROM pulse_monitor WHERE id=?").get(id))).not.toContain("TOP_SECRET_VALUE");
 const listed=await(await request("/monitors")).json() as {monitors:{id:string;options:unknown;headerNames:string[]}[]},mine=listed.monitors.find(m=>m.id===id);
 expect(mine).toMatchObject({options,headerNames:["Authorization","X-Env"]});expect(JSON.stringify(listed)).not.toContain("TOP_SECRET_VALUE");
 expect((await request(`/monitors/${id}`,"PATCH",{headers:[{name:"Unknown",value:null}]})).status).toBe(400);
 expect((await request(`/monitors/${id}`,"PATCH",{headers:[{name:"Authorization",value:null}]})).status).toBe(200);
 const {openHeaders}=await import("./monitor-options");const row=db.sqlite.prepare("SELECT headers FROM pulse_monitor WHERE id=?").get(id) as {headers:string};
 expect(await openHeaders(state.env,"alice",id,row.headers)).toEqual([{name:"Authorization",value:"Bearer TOP_SECRET_VALUE"}]);
});
// A WebP header (lossless) of a given size, as the browser makes them.
function webp(width:number,height:number,extra=16){
 const bytes=new Uint8Array(25+extra),view=new DataView(bytes.buffer),text=(at:number,v:string)=>[...v].forEach((c,i)=>bytes[at+i]=c.charCodeAt(0));
 text(0,"RIFF");view.setUint32(4,bytes.length-8,true);text(8,"WEBP");text(12,"VP8L");view.setUint32(16,bytes.length-20,true);bytes[20]=0x2f;view.setUint32(21,((width-1)&0x3fff)|(((height-1)&0x3fff)<<14),true);
 return bytes;
}
it("shows the page's logo, links and title choice, and a badge and favicon with its state",async()=>{
 const monitor=await(await request("/monitors","POST",{projectId:"1",name:"Api",url:"https://example.com/",intervalMinutes:5})).json() as {id:string};
 const {id}=await(await request("/pages","POST",{title:"Acme",slug:"acme-status",language:"en",components:[{monitorId:monitor.id,name:"Api"}]})).json() as {id:string};
 expect((await request(`/pages/${id}`,"PATCH",{links:[{label:"Site",url:"http://acme.example/"}]})).status).toBe(400);
 expect((await request(`/pages/${id}`,"PATCH",{links:Array.from({length:6},(_,i)=>({label:`L${i}`,url:"https://acme.example/"}))})).status).toBe(400);
 expect((await request(`/pages/${id}`,"PATCH",{published:true,showTitle:false,links:[{label:"Site <b>",url:"https://acme.example/"}]})).status).toBe(200);
 const upload=(body:BodyInit,user="alice",type="image/webp")=>pulseApi.fetch(new Request(`https://cloud.audit.invalid/pages/${id}/logo/dark`,{method:"PUT",headers:{authorization:`Bearer ${user}`,"content-type":type},body}),state.env);
 expect((await upload(new Uint8Array([137,80,78,71,13,10,26,10]))).status).toBe(400);
 expect((await upload(webp(120,40),"alice","image/png")).status).toBe(400);expect((await upload(webp(120,40),"alice","image/gif")).status).toBe(400);
 expect((await upload(webp(4000,40))).status).toBe(400);
 expect((await upload(webp(120,40),"bob")).status).toBe(404);
 const sent=await upload(webp(120,40));expect(sent.status).toBe(201);const {version}=await sent.json() as {version:string};expect(version).toMatch(/^[a-f0-9]{16}$/);
 const listed=await(await request("/pages")).json() as {pages:{showTitle:boolean;links:unknown[];logos:{light:string|null;dark:string|null}}[]};
 expect(listed.pages[0]).toMatchObject({showTitle:false,links:[{label:"Site <b>",url:"https://acme.example/"}],logos:{light:null,dark:version}});
 const served=await servePulsePage(new Request("https://pulse.audit.invalid/acme-status/"),state.env),html=await served?.text()??"";
 expect(html).toContain(`src="logo-dark?v=${version}"`);expect(html).toContain('alt="Acme"');expect(html).not.toContain("<span>Acme</span>");expect(html).toContain(`class="site-header"`);expect(html).toContain(`popovertarget="mobile-menu"`);
 expect(html).toContain('<a href="https://acme.example/" rel="noopener nofollow">Site &lt;b&gt;</a>');expect(html).toContain('rel="icon" type="image/svg+xml" href="data:image/svg+xml,');
 const csp=served?.headers.get("content-security-policy")??"";expect(csp).toContain("img-src 'self' data:");
 // Only the page's own theme script runs, by its hash.
 const script=/<script>([^<]*)<\/script>/.exec(html)?.[1]??"",hash=btoa(String.fromCharCode(...new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(script)))));expect(csp).toContain(`script-src 'sha256-${hash}'`);
 expect(html).toContain('data-set-theme="dark"');expect(html).toContain('<html lang="en">');
 const spanish=await(await servePulsePage(new Request("https://pulse.audit.invalid/acme-status/?lang=es"),state.env))?.text()??"";expect(spanish).toContain('<html lang="es">');expect(spanish).toContain('aria-checked="true">Español');
 expect(await(await servePulsePage(new Request("https://pulse.audit.invalid/acme-status/?lang=xx"),state.env))?.text()).toContain('<html lang="en">');
 const logo=await servePulsePage(new Request(`https://pulse.audit.invalid/acme-status/logo-dark?v=${version}`),state.env);expect(logo?.headers.get("content-type")).toBe("image/webp");expect(logo?.headers.get("cache-control")).toContain("immutable");
 expect((await servePulsePage(new Request("https://pulse.audit.invalid/acme-status/logo-light"),state.env))?.status).toBe(404);
 const badge=await servePulsePage(new Request("https://pulse.audit.invalid/acme-status/badge?theme=dark"),state.env);expect(badge?.headers.get("content-security-policy")).toContain("frame-ancestors *");
 const badgeHtml=await badge?.text()??"";expect(badgeHtml).toContain('href="./" target="_blank"');expect(badgeHtml).toContain("color:#f5f5f5");
 // A PNG, where the browser could not make WebP.
 const png=new Uint8Array(40);png.set([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82]);new DataView(png.buffer).setUint32(16,200);new DataView(png.buffer).setUint32(20,60);
 const light=pulseApi.fetch(new Request(`https://cloud.audit.invalid/pages/${id}/logo/light`,{method:"PUT",headers:{authorization:"Bearer alice","content-type":"image/png"},body:png}),state.env);expect((await light).status).toBe(201);
 expect((await servePulsePage(new Request("https://pulse.audit.invalid/acme-status/logo-light"),state.env))?.headers.get("content-type")).toBe("image/png");
 expect((await request(`/pages/${id}/logo/dark`,"DELETE")).status).toBe(200);expect((await servePulsePage(new Request(`https://pulse.audit.invalid/acme-status/logo-dark?v=${version}`),state.env))?.status).toBe(404);
});
