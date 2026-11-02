import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {languageSchema} from "../../shared/language";
import {pulseWrite,type PulseApp} from "./access";
import {pageSlug} from "./page-rules";
const component=z.object({monitorId:opaqueId,name:z.string().trim().min(1).max(80)}).strict();
const components=z.array(component).max(50).refine(v=>new Set(v.map(c=>c.monitorId)).size===v.length);
const pageBody=z.object({title:z.string().trim().min(1).max(100),slug:pageSlug,components,language:languageSchema}).strict();
export const incidentState=z.enum(["investigating","identified","monitoring","resolved"]);
export const impact=z.enum(["operational","degraded","partial_outage","major_outage"]);
export const pageParam=z.object({page:opaqueId});
export async function ownedPage(env:Cloudflare.Env,owner:string,id:string){return env.DB.prepare("SELECT id,title,slug,published,language,plan_disabled AS planDisabled FROM pulse_page WHERE id=? AND user_id=? AND (retire_at IS NULL OR retire_at>?)").bind(id,owner,Date.now()).first<{id:string;title:string;slug:string;published:number;language:string;planDisabled:number}>();}
async function validComponents(db:D1Database,owner:string,list:z.infer<typeof components>){
 for(const item of list){if(!await db.prepare("SELECT m.id FROM pulse_monitor m JOIN pulse_project p ON p.id=m.project_id WHERE m.id=? AND m.user_id=? AND p.deleted_at IS NULL").bind(item.monitorId,owner).first())return false;}return true;
}
function setComponents(db:D1Database,page:string,list:z.infer<typeof components>){return [db.prepare("DELETE FROM pulse_component WHERE page_id=?").bind(page),...list.map((item,position)=>db.prepare("INSERT INTO pulse_component VALUES(?,?,?,?)").bind(page,item.monitorId,item.name,position))];}
export const pagesApi=new Hono<PulseApp>()
 .get("/",async c=>{
  const pages=await c.env.DB.prepare("SELECT id,title,slug,published,language,plan_disabled AS planDisabled,retire_at AS retireAt FROM pulse_page WHERE user_id=? AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 3").bind(c.get("session").user.id,Date.now()).all<{id:string;title:string;slug:string;published:number;language:string;planDisabled:number;retireAt:number|null}>();
  const rows=c.get("access").readable?await Promise.all(pages.results.map(async p=>({...p,published:!!p.published&&!p.planDisabled,writable:!p.planDisabled&&c.get("access").writable,readOnlyUntil:p.retireAt===null?null:new Date(p.retireAt).toISOString(),components:(await c.env.DB.prepare("SELECT monitor_id AS monitorId,name FROM pulse_component WHERE page_id=? ORDER BY position").bind(p.id).all<{monitorId:string;name:string}>()).results}))):[];return c.json({pages:rows});
 })
 .post("/",pulseWrite,validate("json",pageBody),async c=>{
  const input=c.req.valid("json"),owner=c.get("session").user.id;if(!await validComponents(c.env.DB,owner,input.components))return c.json({code:"NOT_FOUND"},404);
  const id=crypto.randomUUID();
  try{
   const result=await c.env.DB.batch([c.env.DB.prepare("INSERT INTO pulse_page(id,user_id,slug,title,language,created_at) SELECT ?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_page WHERE user_id=? AND plan_disabled=0)<? AND (SELECT count(*) FROM pulse_page WHERE user_id=?)<3 RETURNING id").bind(id,owner,input.slug,input.title,input.language,Date.now(),owner,c.get("access").limits?.pages??0,owner),...input.components.map((item,position)=>c.env.DB.prepare("INSERT INTO pulse_component SELECT ?,?,?,? WHERE EXISTS(SELECT 1 FROM pulse_page WHERE id=?)").bind(id,item.monitorId,item.name,position,id))]);
   return result[0].results.length?c.json({id,slug:input.slug},201):c.json({code:"QUOTA"},409);
  }catch(e){if(e instanceof Error&&e.message.includes("UNIQUE"))return c.json({code:"SLUG_TAKEN"},409);throw e;}
 })
 .patch("/:page",pulseWrite,validate("param",pageParam),validate("json",z.object({title:pageBody.shape.title.optional(),components:components.optional(),published:z.boolean().optional(),language:languageSchema.optional()}).strict().refine(v=>Object.keys(v).length>0)),async c=>{
  const id=c.req.valid("param").page,owner=c.get("session").user.id,current=await ownedPage(c.env,owner,id);if(!current)return c.json({code:"NOT_FOUND"},404);const body=c.req.valid("json");if(current.planDisabled)return c.json({code:"PLAN"},403);
  if(body.components&&!await validComponents(c.env.DB,owner,body.components))return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.batch([c.env.DB.prepare("UPDATE pulse_page SET title=?,published=?,language=? WHERE id=? AND user_id=?").bind(body.title??current.title,body.published===undefined?current.published:Number(body.published),body.language??current.language,id,owner),...(body.components?setComponents(c.env.DB,id,body.components):[])]);return c.json({ok:true});
 })
 .delete("/:page",validate("param",pageParam),async c=>{
  const id=c.req.valid("param").page,current=await ownedPage(c.env,c.get("session").user.id,id);if(!current)return c.json({code:"NOT_FOUND"},404);
  // Provider removal is coordinated by the domain route/cleanup before
  // the row is lost. Keep the tombstone unpublished until it succeeds.
  await c.env.DB.prepare("UPDATE pulse_page SET published=0 WHERE id=?").bind(id).run();
  if(await c.env.DB.prepare("SELECT page_id FROM pulse_page_domain WHERE page_id=?").bind(id).first())return c.json({code:"DOMAIN_EXISTS"},409);
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_incident_update WHERE incident_id IN(SELECT id FROM pulse_incident WHERE page_id=?)").bind(id),c.env.DB.prepare("DELETE FROM pulse_incident WHERE page_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_component WHERE page_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_page WHERE id=?").bind(id)]);return c.json({ok:true});
 })
 .get("/:page/incidents",validate("param",pageParam),async c=>{
  if(!await ownedPage(c.env,c.get("session").user.id,c.req.valid("param").page)||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const incidents=await c.env.DB.prepare("SELECT id,title,state,impact,created_at AS createdAt,updated_at AS updatedAt FROM pulse_incident WHERE page_id=? AND updated_at>? ORDER BY created_at DESC LIMIT 50").bind(c.req.valid("param").page,Date.now()-(c.get("access").limits?.retentionDays??30)*86_400_000).all<{id:string;title:string;state:string;impact:string;createdAt:number;updatedAt:number}>();return c.json({incidents:incidents.results});
 })
 .post("/:page/incidents",pulseWrite,validate("param",pageParam),validate("json",z.object({title:z.string().trim().min(1).max(120),body:z.string().trim().min(1).max(2000),state:incidentState,impact}).strict()),async c=>{
  const page=c.req.valid("param").page;const owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);const input=c.req.valid("json"),id=crypto.randomUUID(),now=Date.now();
  const made=await c.env.DB.batch([c.env.DB.prepare("INSERT INTO pulse_incident(id,page_id,title,state,impact,created_at,updated_at,expires_at) SELECT ?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_incident WHERE page_id=?)<200 RETURNING id").bind(id,page,input.title,input.state,input.impact,now,now,now+(c.get("access").limits?.retentionDays??30)*86_400_000,page),c.env.DB.prepare("INSERT INTO pulse_incident_update(id,incident_id,body,state,created_at,expires_at) SELECT ?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM pulse_incident WHERE id=?)").bind(crypto.randomUUID(),id,input.body,input.state,now,now+(c.get("access").limits?.retentionDays??30)*86_400_000,id)]);return made[0].results.length?c.json({id},201):c.json({code:"QUOTA"},409);
 })
 .post("/:page/incidents/:incident/updates",pulseWrite,validate("param",z.object({page:opaqueId,incident:opaqueId})),validate("json",z.object({body:z.string().trim().min(1).max(2000),state:incidentState}).strict()),async c=>{
  const {page,incident}=c.req.valid("param");const owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);const current=await c.env.DB.prepare("SELECT id FROM pulse_incident WHERE id=? AND page_id=?").bind(incident,page).first();if(!current)return c.json({code:"NOT_FOUND"},404);const input=c.req.valid("json"),now=Date.now(),updateId=crypto.randomUUID();
  const made=await c.env.DB.batch([c.env.DB.prepare("INSERT INTO pulse_incident_update(id,incident_id,body,state,created_at,expires_at) SELECT ?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_incident_update WHERE incident_id=?)<100 RETURNING id").bind(updateId,incident,input.body,input.state,now,now+(c.get("access").limits?.retentionDays??30)*86_400_000,incident),c.env.DB.prepare("UPDATE pulse_incident SET state=?,updated_at=?,expires_at=? WHERE id=? AND EXISTS(SELECT 1 FROM pulse_incident_update WHERE id=?)").bind(input.state,now,now+(c.get("access").limits?.retentionDays??30)*86_400_000,incident,updateId)]);return made[0].results.length?c.json({ok:true},201):c.json({code:"QUOTA"},409);
 });
