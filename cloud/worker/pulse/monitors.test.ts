import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
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
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",PULSE_PUBLIC_HOST:"pulse.audit.invalid",PULSE_PUBLIC_LIMIT:{limit:async()=>({success:true})},PULSE_PROJECT:{getByName:()=>({recordChecks:async(observations:Parameters<MonitorStore["record"]>[0][])=>observations.map(o=>store.record(o)),monitorHistory:async(id:string,days:number)=>store.history(id,days)})}});
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
 const html=await(await servePulsePage(new Request("https://pulse.audit.invalid/public-page/"),state.env))?.text();expect(html).toContain("&lt;script&gt;");expect(html).not.toContain("<script>");expect(html).not.toContain("<img");
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
});
