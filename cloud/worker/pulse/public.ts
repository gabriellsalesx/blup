import {languageOf} from "../../shared/language";
import {translate} from "../../src/lib/i18n/catalog";
import {pulseAccess,sha256} from "./limits";
import {z} from "zod";
import {pageSlug} from "./page-rules";

const escape=(v:string)=>v.replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]??""));
const publicSchema=z.object({unavailable:z.literal(false),title:z.string().max(100),slug:pageSlug,language:z.enum(["pt","en","es"]),components:z.array(z.object({name:z.string().max(80),state:z.enum(["up","down","unknown"]),lastCheckedAt:z.iso.datetime().nullable(),days:z.array(z.object({date:z.string(),up:z.number().int().nonnegative(),down:z.number().int().nonnegative(),unknown:z.number().int().nonnegative()})).max(90)})).max(50),incidents:z.array(z.object({title:z.string().max(120),state:z.string(),impact:z.string(),createdAt:z.iso.datetime(),updatedAt:z.iso.datetime(),updates:z.array(z.object({body:z.string().max(2000),state:z.string(),createdAt:z.iso.datetime()})).max(20)})).max(20)});
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
 const publicComponents=[];
 for(const c of components.results){
  const history=await env.PULSE_PROJECT.getByName(c.objectName).monitorHistory(c.id,access.limits?.retentionDays??30);
  const state=c.lastAt!==null&&Date.now()-c.lastAt<=Math.max(c.intervalMinutes,access.limits?.intervalMinutes??5)*120_000&&["up","down"].includes(c.state)?c.state:"unknown";
  // Closed public projection. No private names, URLs, DSNs or issues.
  publicComponents.push({name:c.name,state,lastCheckedAt:c.lastAt===null?null:new Date(c.lastAt).toISOString(),days:history.days.map(d=>({date:new Date(d.day*86_400_000).toISOString().slice(0,10),up:d.up,down:d.down,unknown:d.unknown}))});
 }
 const incidents=await env.DB.prepare("SELECT id,title,state,impact,created_at AS createdAt,updated_at AS updatedAt FROM pulse_incident WHERE page_id=? AND updated_at>? ORDER BY created_at DESC LIMIT 20").bind(page.id,cutoff).all<{id:string;title:string;state:string;impact:string;createdAt:number;updatedAt:number}>();
 const publicIncidents=[];
 for(const item of incidents.results){const updates=await env.DB.prepare("SELECT body,state,created_at AS createdAt FROM pulse_incident_update WHERE incident_id=? AND created_at>? ORDER BY created_at DESC LIMIT 20").bind(item.id,cutoff).all<{body:string;state:string;createdAt:number}>();publicIncidents.push({title:item.title,state:item.state,impact:item.impact,createdAt:new Date(item.createdAt).toISOString(),updatedAt:new Date(item.updatedAt).toISOString(),updates:updates.results.map(u=>({...u,createdAt:new Date(u.createdAt).toISOString()}))});}
 const result=publicSchema.parse({unavailable:false,title:page.title,slug:page.slug,language:languageOf(page.language)??"en",components:publicComponents,incidents:publicIncidents});
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
 const t=(key:"pulseOperational"|"pulseDown"|"pulseUnknown"|"pulseIncidents"|"pulseObservations"|"pulseNoIncidents")=>translate(page.language,key);
 const body=`<!doctype html><html lang="${page.language}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="refresh" content="60"><title>${escape(page.title)}</title><style>body{font:16px system-ui;max-width:780px;margin:4rem auto;padding:0 1.5rem;color:#16352d}article{border:1px solid #d2e7df;padding:1.2rem;margin:1rem 0;border-radius:1rem}small,footer{color:#507267}h1{font-size:2rem}p{white-space:pre-wrap;overflow-wrap:anywhere}</style></head><body><h1>${escape(page.title)}</h1>${page.components.map(c=>`<article><h2>${escape(c.name)}</h2><p>${escape(t(c.state==="up"?"pulseOperational":c.state==="down"?"pulseDown":"pulseUnknown"))}</p><small>${escape(t("pulseObservations"))}: ${c.days.reduce((n,d)=>n+d.up+d.down,0).toLocaleString(page.language)}</small></article>`).join("")}<h2>${escape(t("pulseIncidents"))}</h2>${page.incidents.map(i=>`<article><h3>${escape(i.title)}</h3>${i.updates.map(u=>`<p>${escape(u.body)}</p><small>${escape(new Intl.DateTimeFormat(page.language,{dateStyle:"medium",timeStyle:"short",timeZone:"UTC"}).format(new Date(u.createdAt)))}</small>`).join("")}</article>`).join("")||`<p>${escape(t("pulseNoIncidents"))}</p>`}<footer>Blup Pulse</footer></body></html>`;
 return respond(body,"text/html; charset=utf-8");
}
