import {languageOf,languageSchema,acceptedLanguage} from "../../shared/language";
import {pulseAccess,sha256} from "./limits";
import {z} from "zod";
import {pageSlug,serviceStatus,maintenanceStateAt} from "./page-rules";
import {renderStatusPage,renderIncidentPage,renderIncidentHistory,renderBadge,THEME_SCRIPT} from "./public-html";
import {publishedPage,incidentById,incidentsBetween,monthWindow,monthParam} from "./public-incidents";
import {navLinks,linksOf,logoVariant,logoKey,logoVersion,logoType} from "./page-look";
const affectedStatus=serviceStatus.exclude(["none"]);

const publicSchema=z.object({unavailable:z.literal(false),domain:z.string().max(253).nullable(),title:z.string().max(100),showTitle:z.boolean(),links:navLinks,logos:z.object({light:logoVersion.nullable(),dark:logoVersion.nullable()}),slug:pageSlug,language:z.enum(["pt","en","es"]),historyDays:z.number().int().min(1).max(90),components:z.array(z.object({name:z.string().max(80),description:z.string().max(200),state:z.enum(["up","down","unknown"]),lastCheckedAt:z.iso.datetime().nullable(),incident:z.enum(["downtime","degraded"]).nullable(),maintenance:z.boolean(),days:z.array(z.object({date:z.string(),up:z.number().int().nonnegative(),down:z.number().int().nonnegative(),unknown:z.number().int().nonnegative(),maintenance:z.number().int().nonnegative()})).max(90)})).max(50),maintenance:z.array(z.object({title:z.string().max(120),body:z.string().max(2000),state:z.enum(["scheduled","active"]),startsAt:z.iso.datetime(),endsAt:z.iso.datetime(),services:z.array(z.string().max(80)).max(50)})).max(10),incidents:z.array(z.object({id:z.uuid(),title:z.string().max(120),state:z.string(),impact:z.string(),createdAt:z.iso.datetime(),updatedAt:z.iso.datetime(),services:z.array(z.object({name:z.string().max(80),status:affectedStatus})).max(50),updates:z.array(z.object({body:z.string().max(2000),state:z.string(),createdAt:z.iso.datetime()})).max(20)})).max(20)});
export async function publicPage(env:Cloudflare.Env,slug:string){
 const parsed=pageSlug.safeParse(slug);if(!parsed.success)return null;
 const page=await env.DB.prepare("SELECT id,user_id AS userId,title,slug,language,show_title AS showTitle,nav_links AS navLinks,logo_light AS logoLight,logo_dark AS logoDark FROM pulse_page WHERE slug=? AND published=1 AND plan_disabled=0").bind(parsed.data).first<{id:string;userId:string;title:string;slug:string;language:string;showTitle:number;navLinks:string;logoLight:string|null;logoDark:string|null}>();if(!page)return null;
 const access=await pulseAccess(env,page.userId);if(!access.writable)return {unavailable:true as const};
 const cache=typeof caches==="undefined"?null:caches.default,key=new Request(`https://pulse.getblup.site/.pulse-cache/${page.id}`);
 // The gateway checks publication, account and plan before every cache
 // hit. Only this closed, public DTO is cached, for at most 15 seconds.
 try{const hit=await cache?.match(key);if(hit){const parsed=publicSchema.safeParse(await hit.json());if(parsed.success)return parsed.data;}}catch{}
 const cutoff=Date.now()-(access.limits?.retentionDays??30)*86_400_000;
 const components=await env.DB.prepare("SELECT c.monitor_id AS id,c.name,c.description,m.state,m.last_at AS lastAt,m.interval_minutes AS intervalMinutes,m.project_id AS projectId,p.object_name AS objectName FROM pulse_component c JOIN pulse_monitor m ON m.id=c.monitor_id JOIN pulse_project p ON p.id=m.project_id WHERE c.page_id=? AND m.user_id=? AND m.active=1 AND m.plan_disabled=0 AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL ORDER BY c.position LIMIT 50").bind(page.id,page.userId).all<{id:string;name:string;description:string;state:string;lastAt:number|null;intervalMinutes:number;projectId:string;objectName:string}>();
 const incidents=await env.DB.prepare("SELECT id,title,state,impact,created_at AS createdAt,updated_at AS updatedAt FROM pulse_incident WHERE page_id=? AND updated_at>? ORDER BY created_at DESC LIMIT 20").bind(page.id,cutoff).all<{id:string;title:string;state:string;impact:string;createdAt:number;updatedAt:number}>();
 // How each incident leaves the services of this page, by their public
 // names; a service down or slow in an open incident shows it.
 const affected=(await env.DB.prepare("SELECT ic.incident_id AS incidentId,ic.monitor_id AS monitorId,c.name,ic.status FROM pulse_incident_component ic JOIN pulse_incident i ON i.id=ic.incident_id JOIN pulse_component c ON c.page_id=i.page_id AND c.monitor_id=ic.monitor_id WHERE i.page_id=? AND i.updated_at>? ORDER BY c.position").bind(page.id,cutoff).all<{incidentId:string;monitorId:string;name:string;status:string}>()).results.flatMap(r=>{const status=affectedStatus.safeParse(r.status);return status.success?[{...r,status:status.data}]:[];});
 const open=new Set(incidents.results.filter(i=>i.state!=="resolved").map(i=>i.id));
 const incidentOf=(monitor:string)=>{const now=affected.filter(a=>open.has(a.incidentId)&&a.monitorId===monitor);return now.some(a=>a.status==="downtime")?"downtime" as const:now.some(a=>a.status==="degraded")?"degraded" as const:null;};
 // Maintenance not over yet, soonest first, with the public names of its
 // services; a service in one going on now shows it.
 const now=Date.now();
 const windows=(await env.DB.prepare("SELECT id,title,body,starts_at AS startsAt,ends_at AS endsAt FROM pulse_maintenance WHERE page_id=? AND ends_at>? ORDER BY starts_at LIMIT 10").bind(page.id,now).all<{id:string;title:string;body:string;startsAt:number;endsAt:number}>()).results;
 const inWindow=(await env.DB.prepare("SELECT mc.maintenance_id AS maintenanceId,mc.monitor_id AS monitorId,c.name FROM pulse_maintenance_component mc JOIN pulse_maintenance m ON m.id=mc.maintenance_id JOIN pulse_component c ON c.page_id=m.page_id AND c.monitor_id=mc.monitor_id WHERE m.page_id=? AND m.ends_at>? ORDER BY c.position").bind(page.id,now).all<{maintenanceId:string;monitorId:string;name:string}>()).results;
 const activeNow=new Set(windows.filter(w=>w.startsAt<=now).map(w=>w.id)),inMaintenance=(monitor:string)=>inWindow.some(m=>m.monitorId===monitor&&activeNow.has(m.maintenanceId));
 const publicMaintenance=windows.map(w=>({title:w.title,body:w.body,state:maintenanceStateAt(w.startsAt,w.endsAt,now)==="active"?"active" as const:"scheduled" as const,startsAt:new Date(w.startsAt).toISOString(),endsAt:new Date(w.endsAt).toISOString(),services:inWindow.filter(m=>m.maintenanceId===w.id).map(m=>m.name)}));
 const publicComponents=[];
 for(const c of components.results){
  const history=await env.PULSE_PROJECT.getByName(c.objectName).monitorHistory(c.id,access.limits?.retentionDays??30);
  const state=c.lastAt!==null&&Date.now()-c.lastAt<=Math.max(c.intervalMinutes,access.limits?.intervalMinutes??5)*120_000&&["up","down"].includes(c.state)?c.state:"unknown";
  // Closed public projection. No private names, URLs, DSNs or issues.
  publicComponents.push({name:c.name,description:c.description,state,incident:incidentOf(c.id),maintenance:inMaintenance(c.id),lastCheckedAt:c.lastAt===null?null:new Date(c.lastAt).toISOString(),days:history.days.map(d=>({date:new Date(d.day*86_400_000).toISOString().slice(0,10),up:d.up,down:d.down,unknown:d.unknown,maintenance:d.maintenance??0}))});
 }
 const publicIncidents=[];
 for(const item of incidents.results){const updates=await env.DB.prepare("SELECT body,state,created_at AS createdAt FROM pulse_incident_update WHERE incident_id=? AND created_at>? ORDER BY created_at DESC LIMIT 20").bind(item.id,cutoff).all<{body:string;state:string;createdAt:number}>();publicIncidents.push({id:item.id,title:item.title,state:item.state,impact:item.impact,createdAt:new Date(item.createdAt).toISOString(),updatedAt:new Date(item.updatedAt).toISOString(),services:affected.filter(a=>a.incidentId===item.id).map(a=>({name:a.name,status:a.status})),updates:updates.results.map(u=>({...u,createdAt:new Date(u.createdAt).toISOString()}))});}
 // The page's own domain, when it is live and the plan allows it.
 const domain=access.limits?.customDomain?(await env.DB.prepare("SELECT hostname FROM pulse_page_domain WHERE page_id=? AND verified_at IS NOT NULL AND status='active' AND ssl='active'").bind(page.id).first<{hostname:string}>())?.hostname??null:null;
 const result=publicSchema.parse({unavailable:false,domain,title:page.title,showTitle:!!page.showTitle,links:linksOf(page.navLinks),logos:{light:page.logoLight,dark:page.logoDark},slug:page.slug,language:languageOf(page.language)??"en",historyDays:access.limits?.retentionDays??30,components:publicComponents,incidents:publicIncidents,maintenance:publicMaintenance});
 try{await cache?.put(key,Response.json(result,{headers:{"cache-control":"public, max-age=15"}}));}catch{}
 return result;
}
// What a published page answers besides its HTML: its JSON, its badge
// and its logos.
const publicFiles=["/","/status.json","/badge","/logo-light","/logo-dark"];
const badgeTheme=z.enum(["light","dark","auto"]).catch("auto");
export async function servePulsePage(req:Request,env:Cloudflare.Env,slugOverride?:string):Promise<Response|null>{
 const url=new URL(req.url);let slug=slugOverride;
 if(!slug){
  if(url.hostname===env.PULSE_PUBLIC_HOST){if(/^\/[^/]+$/.test(url.pathname))return new Response(null,{status:301,headers:{location:`${url.origin}${url.pathname}/`,"cache-control":"no-store"}});const match=/^\/([^/]+)\/(status\.json|badge|logo-(?:light|dark)|incidents|incidents\/[0-9a-f-]{36})?$/.exec(url.pathname);if(!match)return new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});try{slug=decodeURIComponent(match[1]);}catch{return new Response("Not found",{status:404});}}
  else{const row=await env.DB.prepare("SELECT p.slug FROM pulse_page_domain d JOIN pulse_page p ON p.id=d.page_id WHERE d.hostname=? AND d.verified_at IS NOT NULL AND d.status='active' AND d.ssl='active' AND p.plan_disabled=0").bind(url.hostname).first<{slug:string}>();if(!row)return null;slug=row.slug;if(!publicFiles.includes(url.pathname)&&!/^\/incidents(?:\/[0-9a-f-]{36})?$/.test(url.pathname))return new Response("Not found",{status:404});}
 }
 if(!["GET","HEAD"].includes(req.method))return new Response(null,{status:405,headers:{allow:"GET, HEAD","cache-control":"no-store"}});
 if(!(await env.PULSE_PUBLIC_LIMIT.limit({key:`public:${req.headers.get("cf-connecting-ip")??"local"}`})).success)return Response.json({code:"TOO_MANY_REQUESTS"},{status:429,headers:{"Retry-After":"60","Cache-Control":"no-store"}});
 if(url.hostname!==env.PULSE_PUBLIC_HOST){const owned=await env.DB.prepare("SELECT p.user_id AS owner FROM pulse_page_domain d JOIN pulse_page p ON p.id=d.page_id WHERE p.slug=? AND d.hostname=? AND d.status='active' AND d.ssl='active'").bind(slug,url.hostname).first<{owner:string}>();if(!owned)return null;const access=await pulseAccess(env,owned.owner);if(!access.writable||!access.limits?.customDomain)return new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});}
 const page=await publicPage(env,slug);if(!page)return Response.json({code:"NOT_FOUND"},{status:404,headers:{"cache-control":"no-store"}});
 if(page.unavailable)return Response.json({code:"UNAVAILABLE"},{status:503,headers:{"cache-control":"no-store","Retry-After":"60"}});
 const headers={"cache-control":"public, max-age=0, must-revalidate","Content-Security-Policy":"default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'","X-Content-Type-Options":"nosniff","Referrer-Policy":"no-referrer"};
 const respond=async(body:string,type:string,extra:Record<string,string>={})=>{const etag=`"${await sha256(body)}"`,responseHeaders={...headers,...extra,etag,"content-type":type};return new Response(req.method==="HEAD"||req.headers.get("if-none-match")===etag?null:body,{status:req.headers.get("if-none-match")===etag?304:200,headers:responseHeaders});};
 if(url.pathname.endsWith("status.json")||slugOverride)return respond(JSON.stringify(page),"application/json; charset=utf-8");
 // The badge is meant for other sites' iframes. An iframe cannot read the
 // site's language, so it speaks the visitor's (their browser's), unless
 // the code fixes one (?lang=); without either, the page's.
 if(url.pathname.endsWith("/badge")){const asked=req.headers.get("accept-language"),language=languageSchema.safeParse(url.searchParams.get("lang")).data??(asked?acceptedLanguage(asked):page.language);const body=renderBadge(page,badgeTheme.parse(url.searchParams.get("theme")),language);const etag=`"${await sha256(body)}"`;return new Response(req.method==="HEAD"||req.headers.get("if-none-match")===etag?null:body,{status:req.headers.get("if-none-match")===etag?304:200,headers:{...headers,etag,"content-type":"text/html; charset=utf-8",vary:"Accept-Language","Content-Security-Policy":"default-src 'none'; style-src 'unsafe-inline'; frame-ancestors *; base-uri 'none'; form-action 'none'"}});}
 const logo=/\/logo-(light|dark)$/.exec(url.pathname);
 if(logo){
  const variant=logoVariant.parse(logo[1]),version=page.logos[variant];if(!version)return new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});
  const id=(await env.DB.prepare("SELECT id FROM pulse_page WHERE slug=? AND published=1").bind(slug).first<{id:string}>())?.id,object=id?await env.PULSE_ASSETS.get(logoKey(id,variant)):null;if(!object)return new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});
  // The address carries the version: that one is kept a year.
  const lasting=url.searchParams.get("v")===version;
  return new Response(req.method==="HEAD"?null:object.body,{headers:{"content-type":logoType.catch("image/webp").parse(object.httpMetadata?.contentType),"cache-control":lasting?"public, max-age=31536000, immutable":"public, max-age=300","x-content-type-options":"nosniff","content-security-policy":"default-src 'none'; sandbox","cross-origin-resource-policy":"cross-origin",etag:`"${version}"`}});
 }
 const lang=languageSchema.safeParse(url.searchParams.get("lang")).data;
 // Where the page lives: its own domain when it has one (from the shared
 // address too), else its address here (relative, set by the header).
 const shown={...page,home:page.domain&&url.hostname!==page.domain?`https://${page.domain}/`:undefined};
 const missing=()=>new Response("Not found",{status:404,headers:{"cache-control":"no-store"}});
 let body:string;
 // The previous incidents (three months at a time) and one incident's own
 // page read the incidents themselves: the main page only carries the
 // latest.
 const asked=/\/incidents\/([0-9a-f-]{36})$/.exec(url.pathname)?.[1];
 if(asked||/\/incidents$/.test(url.pathname)){
  const source=await publishedPage(env,slug);if(!source)return missing();
  if(asked){const incident=await incidentById(env,source,asked);if(!incident)return missing();body=renderIncidentPage(shown,incident,Date.now(),lang);}
  else{const window=monthWindow(monthParam.safeParse(url.searchParams.get("to")).data??null,source.cutoff);body=renderIncidentHistory(shown,window,await incidentsBetween(env,source,window.from,window.to),Date.now(),lang);}
 }else body=renderStatusPage(shown,Date.now(),lang);
 return respond(body,"text/html; charset=utf-8",{"Content-Security-Policy":`default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; script-src '${await themeScriptHash()}'; frame-ancestors 'none'; base-uri 'none'`});
}
// The CSP source of the page's one inline script (computed once).
let themeHash:Promise<string>|undefined;
const themeScriptHash=()=>themeHash??=crypto.subtle.digest("SHA-256",new TextEncoder().encode(THEME_SCRIPT)).then(d=>`sha256-${btoa(String.fromCharCode(...new Uint8Array(d)))}`);
