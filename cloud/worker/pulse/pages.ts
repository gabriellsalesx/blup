import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {languageSchema} from "../../shared/language";
import {pulseWrite,type PulseApp} from "./access";
import {planHolds,planBinds} from "./limits";
import {pageSlug,slugShape,reservedSlugs,serviceStatus} from "./page-rules";
import {navLinks,linksOf} from "./page-look";
import {dropPageAssets} from "./page-asset-store";
// A service of the page: its monitor, the name visitors see and, if any,
// what it is (an API, the CDN...), shown by a help icon.
const component=z.strictObject({monitorId:opaqueId,name:z.string().trim().min(1).max(80),description:z.string().trim().max(200).regex(/^[^\r\n]*$/).default("")});
const components=z.array(component).max(50).refine(v=>new Set(v.map(c=>c.monitorId)).size===v.length);
// The header of the public page: the title shown or not and its links.
const look={showTitle:z.boolean().optional(),links:navLinks.optional()};
const pageBody=z.strictObject({title:z.string().trim().min(1).max(100),slug:pageSlug,components,language:languageSchema,...look});
export const incidentState=z.enum(["investigating","identified","monitoring","resolved"]);
export const impact=z.enum(["operational","degraded","partial_outage","major_outage"]);
// "none" (not affected) is not stored; the page's impact follows the
// worst of the services.
const services=z.array(z.strictObject({monitorId:opaqueId,status:serviceStatus})).max(50).refine(v=>new Set(v.map(s=>s.monitorId)).size===v.length);
type Services=z.infer<typeof services>;
function impactOf(list:Services,total:number):z.infer<typeof impact>{const down=list.filter(s=>s.status==="downtime").length;return down?(down===total?"major_outage":"partial_outage"):list.some(s=>s.status==="degraded")?"degraded":"operational";}
// Only services shown on this page.
export async function pageServices(db:D1Database,page:string){return new Set((await db.prepare("SELECT monitor_id AS id FROM pulse_component WHERE page_id=?").bind(page).all<{id:string}>()).results.map(r=>r.id));}
function setServices(db:D1Database,incident:string,list:Services){return [db.prepare("DELETE FROM pulse_incident_component WHERE incident_id=?").bind(incident),...list.filter(s=>s.status!=="none").map(s=>db.prepare("INSERT INTO pulse_incident_component(incident_id,monitor_id,status) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM pulse_incident WHERE id=?)").bind(incident,s.monitorId,s.status,incident))];}
const DAY=86_400_000;
const incidentParam=z.object({page:opaqueId,incident:opaqueId});
const incidentTitle=z.string().trim().min(1).max(120);
export const pageParam=z.object({page:opaqueId});
export async function ownedPage(env:Cloudflare.Env,owner:string,id:string){return env.DB.prepare("SELECT id,title,slug,published,language,plan_disabled AS planDisabled FROM pulse_page WHERE id=? AND user_id=? AND (retire_at IS NULL OR retire_at>?)").bind(id,owner,Date.now()).first<{id:string;title:string;slug:string;published:number;language:string;planDisabled:number}>();}
// Every monitor must be the owner's (one query, the ids are already unique).
async function validComponents(db:D1Database,owner:string,list:z.infer<typeof components>){
 if(!list.length)return true;
 const found=await db.prepare("SELECT count(*) AS n FROM pulse_monitor m JOIN pulse_project p ON p.id=m.project_id WHERE m.id IN(SELECT value FROM json_each(?)) AND m.user_id=? AND p.deleted_at IS NULL").bind(JSON.stringify(list.map(c=>c.monitorId)),owner).first<{n:number}>();
 return found?.n===list.length;
}
function setComponents(db:D1Database,page:string,list:z.infer<typeof components>){return [db.prepare("DELETE FROM pulse_component WHERE page_id=?").bind(page),...list.map((item,position)=>db.prepare("INSERT INTO pulse_component(page_id,monitor_id,name,position,description) VALUES(?,?,?,?,?)").bind(page,item.monitorId,item.name,position,item.description))];}
export const pagesApi=new Hono<PulseApp>()
 .get("/",async c=>{
  const pages=await c.env.DB.prepare("SELECT id,title,slug,published,language,plan_disabled AS planDisabled,retire_at AS retireAt,show_title AS showTitle,nav_links AS navLinks,logo_light AS logoLight,logo_dark AS logoDark FROM pulse_page WHERE user_id=? AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 3").bind(c.get("session").user.id,Date.now()).all<{id:string;title:string;slug:string;published:number;language:string;planDisabled:number;retireAt:number|null;showTitle:number;navLinks:string;logoLight:string|null;logoDark:string|null}>();
  const rows=c.get("access").readable?await Promise.all(pages.results.map(async ({showTitle,navLinks:stored,logoLight,logoDark,...p})=>({...p,showTitle:!!showTitle,links:linksOf(stored),logos:{light:logoLight,dark:logoDark},published:!!p.published&&!p.planDisabled,writable:!p.planDisabled&&c.get("access").writable,readOnlyUntil:p.retireAt===null?null:new Date(p.retireAt).toISOString(),components:(await c.env.DB.prepare("SELECT monitor_id AS monitorId,name,description FROM pulse_component WHERE page_id=? ORDER BY position").bind(p.id).all<{monitorId:string;name:string;description:string}>()).results}))):[];return c.json({pages:rows});
 })
 // Whether an address can be used, while the new page is being typed.
 // Addresses are public, so this tells nothing that visiting would not.
 .get("/slug",validate("query",z.object({slug:z.string().max(80)})),async c=>{
  const slug=c.req.valid("query").slug;
  if(!slugShape.safeParse(slug).success)return c.json({state:"invalid" as const});
  if(reservedSlugs.includes(slug))return c.json({state:"reserved" as const});
  const used=await c.env.DB.prepare("SELECT 1 FROM pulse_page WHERE slug=?").bind(slug).first();
  return c.json({state:used?"taken" as const:"available" as const});
 })
 .post("/",pulseWrite,validate("json",pageBody),async c=>{
  const input=c.req.valid("json"),owner=c.get("session").user.id;if(!await validComponents(c.env.DB,owner,input.components))return c.json({code:"NOT_FOUND"},404);
  const id=crypto.randomUUID();
  try{
   const result=await c.env.DB.batch([c.env.DB.prepare(`INSERT INTO pulse_page(id,user_id,slug,title,language,show_title,nav_links,created_at) SELECT ?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_page WHERE user_id=? AND plan_disabled=0)<? AND (SELECT count(*) FROM pulse_page WHERE user_id=?)<3 AND ${planHolds} RETURNING id`).bind(id,owner,input.slug,input.title,input.language,Number(input.showTitle??true),JSON.stringify(input.links??[]),Date.now(),owner,c.get("access").limits?.pages??0,owner,...planBinds(owner,c.get("access"))),...input.components.map((item,position)=>c.env.DB.prepare("INSERT INTO pulse_component(page_id,monitor_id,name,position,description) SELECT ?,?,?,?,? WHERE EXISTS(SELECT 1 FROM pulse_page WHERE id=?)").bind(id,item.monitorId,item.name,position,item.description,id))]);
   return result[0]?.results.length?c.json({id,slug:input.slug},201):c.json({code:"QUOTA"},409);
  }catch(e){if(e instanceof Error&&e.message.includes("UNIQUE"))return c.json({code:"SLUG_TAKEN"},409);throw e;}
 })
 .patch("/:page",pulseWrite,validate("param",pageParam),validate("json",z.strictObject({title:pageBody.shape.title.optional(),components:components.optional(),published:z.boolean().optional(),language:z.optional(languageSchema),...look}).refine(v=>Object.keys(v).length>0)),async c=>{
  const id=c.req.valid("param").page,owner=c.get("session").user.id,current=await ownedPage(c.env,owner,id);if(!current)return c.json({code:"NOT_FOUND"},404);const body=c.req.valid("json");if(current.planDisabled)return c.json({code:"PLAN"},403);
  if(body.components&&!await validComponents(c.env.DB,owner,body.components))return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.batch([c.env.DB.prepare("UPDATE pulse_page SET title=?,published=?,language=?,show_title=coalesce(?,show_title),nav_links=coalesce(?,nav_links) WHERE id=? AND user_id=?").bind(body.title??current.title,body.published===undefined?current.published:Number(body.published),body.language??current.language,body.showTitle===undefined?null:Number(body.showTitle),body.links===undefined?null:JSON.stringify(body.links),id,owner),...(body.components?setComponents(c.env.DB,id,body.components):[])]);return c.json({ok:true});
 })
 .delete("/:page",validate("param",pageParam),async c=>{
  const id=c.req.valid("param").page,current=await ownedPage(c.env,c.get("session").user.id,id);if(!current)return c.json({code:"NOT_FOUND"},404);
  // Provider removal is coordinated by the domain route/cleanup before
  // the row is lost. Keep the tombstone unpublished until it succeeds.
  await c.env.DB.prepare("UPDATE pulse_page SET published=0 WHERE id=?").bind(id).run();
  if(await c.env.DB.prepare("SELECT page_id FROM pulse_page_domain WHERE page_id=?").bind(id).first())return c.json({code:"DOMAIN_EXISTS"},409);
  await dropPageAssets(c.env,id);
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_incident_update WHERE incident_id IN(SELECT id FROM pulse_incident WHERE page_id=?)").bind(id),c.env.DB.prepare("DELETE FROM pulse_incident_component WHERE incident_id IN(SELECT id FROM pulse_incident WHERE page_id=?)").bind(id),c.env.DB.prepare("DELETE FROM pulse_incident WHERE page_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_maintenance_component WHERE maintenance_id IN(SELECT id FROM pulse_maintenance WHERE page_id=?)").bind(id),c.env.DB.prepare("DELETE FROM pulse_maintenance WHERE page_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_component WHERE page_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_page WHERE id=?").bind(id)]);return c.json({ok:true});
 })
 .get("/:page/incidents",validate("param",pageParam),async c=>{
  const page=c.req.valid("param").page;
  if(!await ownedPage(c.env,c.get("session").user.id,page)||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const cutoff=Date.now()-(c.get("access").limits?.retentionDays??30)*DAY;
  const [incidents,affected]=await c.env.DB.batch([
   c.env.DB.prepare("SELECT i.id,i.title,i.state,i.impact,i.created_at AS createdAt,i.updated_at AS updatedAt,(SELECT count(*) FROM pulse_incident_update u WHERE u.incident_id=i.id) AS updates FROM pulse_incident i WHERE i.page_id=? AND i.updated_at>? ORDER BY i.created_at DESC LIMIT 50").bind(page,cutoff),
   c.env.DB.prepare("SELECT ic.incident_id AS incidentId,ic.monitor_id AS monitorId,ic.status FROM pulse_incident_component ic JOIN pulse_incident i ON i.id=ic.incident_id WHERE i.page_id=? AND i.updated_at>?").bind(page,cutoff),
  ]) as [D1Result<{id:string;title:string;state:string;impact:string;createdAt:number;updatedAt:number;updates:number}>,D1Result<{incidentId:string;monitorId:string;status:string}>];
  return c.json({incidents:incidents.results.map(i=>({...i,state:incidentState.parse(i.state),impact:impact.parse(i.impact),services:affected.results.filter(a=>a.incidentId===i.id).map(a=>({monitorId:a.monitorId,status:serviceStatus.parse(a.status)}))}))});
 })
 // An incident's title, and removing it with all its updates.
 .patch("/:page/incidents/:incident",pulseWrite,validate("param",incidentParam),validate("json",z.strictObject({title:incidentTitle})),async c=>{
  const {page,incident}=c.req.valid("param"),owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);
  const done=await c.env.DB.prepare("UPDATE pulse_incident SET title=? WHERE id=? AND page_id=? RETURNING id").bind(c.req.valid("json").title,incident,page).first();return done?c.json({ok:true}):c.json({code:"NOT_FOUND"},404);
 })
 .delete("/:page/incidents/:incident",validate("param",incidentParam),async c=>{
  const {page,incident}=c.req.valid("param");if(!await ownedPage(c.env,c.get("session").user.id,page))return c.json({code:"NOT_FOUND"},404);
  if(!await c.env.DB.prepare("SELECT id FROM pulse_incident WHERE id=? AND page_id=?").bind(incident,page).first())return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_incident_update WHERE incident_id=?").bind(incident),c.env.DB.prepare("DELETE FROM pulse_incident_component WHERE incident_id=?").bind(incident),c.env.DB.prepare("DELETE FROM pulse_incident WHERE id=? AND page_id=?").bind(incident,page)]);return c.json({ok:true});
 })
 // The updates of one incident, newest first (what the public page shows).
 .get("/:page/incidents/:incident/updates",validate("param",z.object({page:opaqueId,incident:opaqueId})),async c=>{
  const {page,incident}=c.req.valid("param");
  if(!await ownedPage(c.env,c.get("session").user.id,page)||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  if(!await c.env.DB.prepare("SELECT id FROM pulse_incident WHERE id=? AND page_id=?").bind(incident,page).first())return c.json({code:"NOT_FOUND"},404);
  const updates=await c.env.DB.prepare("SELECT id,body,state,created_at AS createdAt FROM pulse_incident_update WHERE incident_id=? AND created_at>? ORDER BY created_at DESC,id DESC LIMIT 100").bind(incident,Date.now()-(c.get("access").limits?.retentionDays??30)*DAY).all<{id:string;body:string;state:string;createdAt:number}>();
  return c.json({updates:updates.results.map(u=>({...u,state:incidentState.parse(u.state)}))});
 })
 // A new incident: what happened, since when (publishedAt: up to the kept
 // history ago, not later than now) and how each service is left.
 .post("/:page/incidents",pulseWrite,validate("param",pageParam),validate("json",z.strictObject({title:incidentTitle,body:z.string().trim().min(1).max(2000),state:incidentState,publishedAt:z.iso.datetime({offset:true}),services})),async c=>{
  const page=c.req.valid("param").page;const owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);
  const input=c.req.valid("json"),id=crypto.randomUUID(),now=Date.now(),keep=(c.get("access").limits?.retentionDays??30)*DAY,published=Date.parse(input.publishedAt);
  if(published>now+60_000||published<now-keep)return c.json({code:"INVALID"},400);
  const shown=await pageServices(c.env.DB,page);if(input.services.some(s=>!shown.has(s.monitorId)))return c.json({code:"INVALID"},400);
  const made=await c.env.DB.batch([
   c.env.DB.prepare("INSERT INTO pulse_incident(id,page_id,title,state,impact,created_at,updated_at,expires_at) SELECT ?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_incident WHERE page_id=?)<200 RETURNING id").bind(id,page,input.title,input.state,impactOf(input.services,shown.size),Math.min(published,now),now,now+keep,page),
   c.env.DB.prepare("INSERT INTO pulse_incident_update(id,incident_id,body,state,created_at,expires_at) SELECT ?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM pulse_incident WHERE id=?)").bind(crypto.randomUUID(),id,input.body,input.state,Math.min(published,now),now+keep,id),
   ...setServices(c.env.DB,id,input.services),
  ]);return made[0]?.results.length?c.json({id},201):c.json({code:"QUOTA"},409);
 })
 .post("/:page/incidents/:incident/updates",pulseWrite,validate("param",z.object({page:opaqueId,incident:opaqueId})),validate("json",z.strictObject({body:z.string().trim().min(1).max(2000),state:incidentState,services})),async c=>{
  const {page,incident}=c.req.valid("param");const owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);const current=await c.env.DB.prepare("SELECT id FROM pulse_incident WHERE id=? AND page_id=?").bind(incident,page).first();if(!current)return c.json({code:"NOT_FOUND"},404);const input=c.req.valid("json"),now=Date.now(),updateId=crypto.randomUUID();
  const shown=await pageServices(c.env.DB,page);if(input.services.some(s=>!shown.has(s.monitorId)))return c.json({code:"INVALID"},400);
  const made=await c.env.DB.batch([c.env.DB.prepare("INSERT INTO pulse_incident_update(id,incident_id,body,state,created_at,expires_at) SELECT ?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_incident_update WHERE incident_id=?)<100 RETURNING id").bind(updateId,incident,input.body,input.state,now,now+(c.get("access").limits?.retentionDays??30)*86_400_000,incident),c.env.DB.prepare("UPDATE pulse_incident SET state=?,impact=?,updated_at=?,expires_at=? WHERE id=? AND EXISTS(SELECT 1 FROM pulse_incident_update WHERE id=?)").bind(input.state,impactOf(input.services,shown.size),now,now+(c.get("access").limits?.retentionDays??30)*86_400_000,incident,updateId),...setServices(c.env.DB,incident,input.services)]);return made[0]?.results.length?c.json({ok:true},201):c.json({code:"QUOTA"},409);
 });
