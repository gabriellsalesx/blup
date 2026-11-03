import {languageOf} from "../../shared/language";
import {pulseAccess,sha256} from "./limits";
import {z} from "zod";
import {pageSlug,serviceStatus} from "./page-rules";
import {renderStatusPage} from "./public-html";
const affectedStatus=serviceStatus.exclude(["none"]);

const publicSchema=z.object({unavailable:z.literal(false),title:z.string().max(100),slug:pageSlug,language:z.enum(["pt","en","es"]),historyDays:z.number().int().min(1).max(90),components:z.array(z.object({name:z.string().max(80),state:z.enum(["up","down","unknown"]),lastCheckedAt:z.iso.datetime().nullable(),incident:z.enum(["downtime","degraded"]).nullable(),days:z.array(z.object({date:z.string(),up:z.number().int().nonnegative(),down:z.number().int().nonnegative(),unknown:z.number().int().nonnegative()})).max(90)})).max(50),incidents:z.array(z.object({title:z.string().max(120),state:z.string(),impact:z.string(),createdAt:z.iso.datetime(),updatedAt:z.iso.datetime(),services:z.array(z.object({name:z.string().max(80),status:affectedStatus})).max(50),updates:z.array(z.object({body:z.string().max(2000),state:z.string(),createdAt:z.iso.datetime()})).max(20)})).max(20)});
export async function publicPage(env:Cloudflare.Env,slug:string){
 const parsed=pageSlug.safeParse(slug);if(!parsed.success)return null;
 const page=await env.DB.prepare("SELECT id,user_id AS userId,title,slug,language FROM pulse_page WHERE slug=? AND published=1 AND plan_disabled=0").bind(parsed.data).first<{id:string;userId:string;title:string;slug:string;language:string}>();if(!page)return null;
 const access=await pulseAccess(env,page.userId);if(!access.writable)return {unavailable:true as const};
 const cache=typeof caches==="undefined"?null:caches.default,key=new Request(`https://pulse.getblup.site/.pulse-cache/${page.id}`);
 // The gateway checks publication, account and plan before every cache
 // hit. Only this closed, public DTO is cached, for at most 15 seconds.
 try{const hit=await cache?.match(key);if(hit){const parsed=publicSchema.safeParse(await hit.json());if(parsed.success)return parsed.data;}}catch{}
 const cutoff=Date.now()-(access.limits?.retentionDays??30)*86_400_000;
 const components=await env.DB.prepare("SELECT c.monitor_id AS id,c.name,m.state,m.last_at AS lastAt,m.interval_minutes AS intervalMinutes,m.project_id AS projectId,p.object_name AS objectName FROM pulse_component c JOIN pulse_monitor m ON m.id=c.monitor_id JOIN pulse_project p ON p.id=m.project_id WHERE c.page_id=? AND m.user_id=? AND m.active=1 AND m.plan_disabled=0 AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL ORDER BY c.position LIMIT 50").bind(page.id,page.userId).all<{id:string;name:string;state:string;lastAt:number|null;intervalMinutes:number;projectId:string;objectName:string}>();
 const incidents=await env.DB.prepare("SELECT id,title,state,impact,created_at AS createdAt,updated_at AS updatedAt FROM pulse_incident WHERE page_id=? AND updated_at>? ORDER BY created_at DESC LIMIT 20").bind(page.id,cutoff).all<{id:string;title:string;state:string;impact:string;createdAt:number;updatedAt:number}>();
 // How each incident leaves the services of this page, by their public
 // names; a service down or slow in an open incident shows it.
 const affected=(await env.DB.prepare("SELECT ic.incident_id AS incidentId,ic.monitor_id AS monitorId,c.name,ic.status FROM pulse_incident_component ic JOIN pulse_incident i ON i.id=ic.incident_id JOIN pulse_component c ON c.page_id=i.page_id AND c.monitor_id=ic.monitor_id WHERE i.page_id=? AND i.updated_at>? ORDER BY c.position").bind(page.id,cutoff).all<{incidentId:string;monitorId:string;name:string;status:string}>()).results.flatMap(r=>{const status=affectedStatus.safeParse(r.status);return status.success?[{...r,status:status.data}]:[];});
 const open=new Set(incidents.results.filter(i=>i.state!=="resolved").map(i=>i.id));
 const incidentOf=(monitor:string)=>{const now=affected.filter(a=>open.has(a.incidentId)&&a.monitorId===monitor);return now.some(a=>a.status==="downtime")?"downtime" as const:now.some(a=>a.status==="degraded")?"degraded" as const:null;};
 const publicComponents=[];
 for(const c of components.results){
  const history=await env.PULSE_PROJECT.getByName(c.objectName).monitorHistory(c.id,access.limits?.retentionDays??30);
  const state=c.lastAt!==null&&Date.now()-c.lastAt<=Math.max(c.intervalMinutes,access.limits?.intervalMinutes??5)*120_000&&["up","down"].includes(c.state)?c.state:"unknown";
  // Closed public projection. No private names, URLs, DSNs or issues.
  publicComponents.push({name:c.name,state,incident:incidentOf(c.id),lastCheckedAt:c.lastAt===null?null:new Date(c.lastAt).toISOString(),days:history.days.map(d=>({date:new Date(d.day*86_400_000).toISOString().slice(0,10),up:d.up,down:d.down,unknown:d.unknown}))});
 }
 const publicIncidents=[];
 for(const item of incidents.results){const updates=await env.DB.prepare("SELECT body,state,created_at AS createdAt FROM pulse_incident_update WHERE incident_id=? AND created_at>? ORDER BY created_at DESC LIMIT 20").bind(item.id,cutoff).all<{body:string;state:string;createdAt:number}>();publicIncidents.push({title:item.title,state:item.state,impact:item.impact,createdAt:new Date(item.createdAt).toISOString(),updatedAt:new Date(item.updatedAt).toISOString(),services:affected.filter(a=>a.incidentId===item.id).map(a=>({name:a.name,status:a.status})),updates:updates.results.map(u=>({...u,createdAt:new Date(u.createdAt).toISOString()}))});}
 const result=publicSchema.parse({unavailable:false,title:page.title,slug:page.slug,language:languageOf(page.language)??"en",historyDays:access.limits?.retentionDays??30,components:publicComponents,incidents:publicIncidents});
 try{await cache?.put(key,Response.json(result,{headers:{"cache-control":"public, max-age=15"}}));}catch{}
 return result;
}
export async function servePulsePage(req:Request,env:Cloudflare.Env,slugOverride?:string):Promise<Response|null>{
 const url=new URL(req.url);let slug=slugOverride;
 if(!slug){
  if(url.hostname===env.PULSE_PUBLIC_HOST){if(/^\/[^/]+$/.test(url.pathname))return new Response(null,{status:301,headers:{location:`${url.origin}${url.pathname}/`,"cache-control":"no-store"}});const match=/^\/([^/]+)\/(status\.json)?$/.exec(url.pathname);if(!match)return new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});try{slug=decodeURIComponent(match[1]);}catch{return new Response("Not found",{status:404});}}
  else{const row=await env.DB.prepare("SELECT p.slug FROM pulse_page_domain d JOIN pulse_page p ON p.id=d.page_id WHERE d.hostname=? AND d.verified_at IS NOT NULL AND d.status='active' AND d.ssl='active' AND p.plan_disabled=0").bind(url.hostname).first<{slug:string}>();if(!row)return null;slug=row.slug;if(!["/","/status.json"].includes(url.pathname))return new Response("Not found",{status:404});}
 }
 if(!["GET","HEAD"].includes(req.method))return new Response(null,{status:405,headers:{allow:"GET, HEAD","cache-control":"no-store"}});
 if(!(await env.PULSE_PUBLIC_LIMIT.limit({key:`public:${req.headers.get("cf-connecting-ip")??"local"}`})).success)return Response.json({code:"TOO_MANY_REQUESTS"},{status:429,headers:{"Retry-After":"60","Cache-Control":"no-store"}});
 if(url.hostname!==env.PULSE_PUBLIC_HOST){const owned=await env.DB.prepare("SELECT p.user_id AS owner FROM pulse_page_domain d JOIN pulse_page p ON p.id=d.page_id WHERE p.slug=? AND d.hostname=? AND d.status='active' AND d.ssl='active'").bind(slug,url.hostname).first<{owner:string}>();if(!owned)return null;const access=await pulseAccess(env,owned.owner);if(!access.writable||!access.limits?.customDomain)return new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});}
 const page=await publicPage(env,slug);if(!page)return Response.json({code:"NOT_FOUND"},{status:404,headers:{"cache-control":"no-store"}});
 if(page.unavailable)return Response.json({code:"UNAVAILABLE"},{status:503,headers:{"cache-control":"no-store","Retry-After":"60"}});
 const headers={"cache-control":"public, max-age=0, must-revalidate","Content-Security-Policy":"default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'","X-Content-Type-Options":"nosniff","Referrer-Policy":"no-referrer"};
 const respond=async(body:string,type:string)=>{const etag=`"${await sha256(body)}"`,responseHeaders={...headers,etag,"content-type":type};return new Response(req.method==="HEAD"||req.headers.get("if-none-match")===etag?null:body,{status:req.headers.get("if-none-match")===etag?304:200,headers:responseHeaders});};
 if(url.pathname.endsWith("status.json")||slugOverride)return respond(JSON.stringify(page),"application/json; charset=utf-8");
 const body=renderStatusPage(page);
 return respond(body,"text/html; charset=utf-8");
}
