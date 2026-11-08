import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {pulseWrite,type PulseApp} from "./access";
import {ownedPage,pageParam,pageServices} from "./pages";
import {maintenanceStateAt,MAINTENANCE_MAX_MS,MAINTENANCE_AHEAD_MS} from "./page-rules";
// Scheduled maintenance of a status page: a window (start and end) and the
// services of the page in it. Where it stands (scheduled, active, done)
// always comes from the window and the time now.
const DAY=86_400_000;
const maintenanceParam=z.object({page:opaqueId,maintenance:opaqueId});
const maintenanceBody=z.strictObject({title:z.string().trim().min(1).max(120),body:z.string().trim().max(2000),startsAt:z.iso.datetime({offset:true}),endsAt:z.iso.datetime({offset:true}),services:z.array(opaqueId).min(1).max(50).refine(v=>new Set(v).size===v.length)});
type MaintenanceBody=z.infer<typeof maintenanceBody>;
type Row={id:string;title:string;body:string;startsAt:number;endsAt:number;createdAt:number;updatedAt:number};
// The window must end after it starts, last at most 30 days, end after the
// kept history begins and start within a year; the services must be shown
// on the page.
async function checkWindow(env:Cloudflare.Env,page:string,input:MaintenanceBody,keep:number,now:number){
 const startsAt=Date.parse(input.startsAt),endsAt=Date.parse(input.endsAt);
 if(endsAt<=startsAt||endsAt-startsAt>MAINTENANCE_MAX_MS||endsAt<=now-keep||startsAt>now+MAINTENANCE_AHEAD_MS)return null;
 const shown=await pageServices(env.DB,page);if(input.services.some(id=>!shown.has(id)))return null;
 return {startsAt,endsAt};
}
function setServices(db:D1Database,maintenance:string,services:string[]){return [db.prepare("DELETE FROM pulse_maintenance_component WHERE maintenance_id=?").bind(maintenance),...services.map(id=>db.prepare("INSERT INTO pulse_maintenance_component(maintenance_id,monitor_id) SELECT ?,? WHERE EXISTS(SELECT 1 FROM pulse_maintenance WHERE id=?)").bind(maintenance,id,maintenance))];}
export const maintenanceApi=new Hono<PulseApp>()
 .get("/:page/maintenance",validate("param",pageParam),async c=>{
  const page=c.req.valid("param").page,now=Date.now();
  if(!await ownedPage(c.env,c.get("session").user.id,page)||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const keep=(c.get("access").limits?.retentionDays??30)*DAY;
  const [rows,services]=await c.env.DB.batch([
   c.env.DB.prepare("SELECT id,title,body,starts_at AS startsAt,ends_at AS endsAt,created_at AS createdAt,updated_at AS updatedAt FROM pulse_maintenance WHERE page_id=? AND ends_at>? ORDER BY starts_at DESC LIMIT 50").bind(page,now-keep),
   c.env.DB.prepare("SELECT mc.maintenance_id AS maintenanceId,mc.monitor_id AS monitorId FROM pulse_maintenance_component mc JOIN pulse_maintenance m ON m.id=mc.maintenance_id WHERE m.page_id=? AND m.ends_at>?").bind(page,now-keep),
  ]) as [D1Result<Row>,D1Result<{maintenanceId:string;monitorId:string}>];
  return c.json({maintenance:rows.results.map(m=>({...m,state:maintenanceStateAt(m.startsAt,m.endsAt,now),services:services.results.filter(s=>s.maintenanceId===m.id).map(s=>s.monitorId)}))});
 })
 .post("/:page/maintenance",pulseWrite,validate("param",pageParam),validate("json",maintenanceBody),async c=>{
  const page=c.req.valid("param").page,owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);
  const input=c.req.valid("json"),now=Date.now(),keep=(c.get("access").limits?.retentionDays??30)*DAY,window=await checkWindow(c.env,page,input,keep,now);if(!window)return c.json({code:"INVALID"},400);
  const id=crypto.randomUUID();
  const made=await c.env.DB.batch([c.env.DB.prepare("INSERT INTO pulse_maintenance(id,page_id,title,body,starts_at,ends_at,created_at,updated_at,expires_at) SELECT ?,?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_maintenance WHERE page_id=?)<100 RETURNING id").bind(id,page,input.title,input.body,window.startsAt,window.endsAt,now,now,window.endsAt+keep,page),...setServices(c.env.DB,id,input.services)]);
  return made[0]?.results.length?c.json({id},201):c.json({code:"QUOTA"},409);
 })
 .put("/:page/maintenance/:maintenance",pulseWrite,validate("param",maintenanceParam),validate("json",maintenanceBody),async c=>{
  const {page,maintenance}=c.req.valid("param"),owned=await ownedPage(c.env,c.get("session").user.id,page);if(!owned)return c.json({code:"NOT_FOUND"},404);if(owned.planDisabled)return c.json({code:"PLAN"},403);
  if(!await c.env.DB.prepare("SELECT id FROM pulse_maintenance WHERE id=? AND page_id=?").bind(maintenance,page).first())return c.json({code:"NOT_FOUND"},404);
  const input=c.req.valid("json"),now=Date.now(),keep=(c.get("access").limits?.retentionDays??30)*DAY,window=await checkWindow(c.env,page,input,keep,now);if(!window)return c.json({code:"INVALID"},400);
  await c.env.DB.batch([c.env.DB.prepare("UPDATE pulse_maintenance SET title=?,body=?,starts_at=?,ends_at=?,updated_at=?,expires_at=? WHERE id=? AND page_id=?").bind(input.title,input.body,window.startsAt,window.endsAt,now,window.endsAt+keep,maintenance,page),...setServices(c.env.DB,maintenance,input.services)]);
  return c.json({ok:true});
 })
 .delete("/:page/maintenance/:maintenance",validate("param",maintenanceParam),async c=>{
  const {page,maintenance}=c.req.valid("param");if(!await ownedPage(c.env,c.get("session").user.id,page))return c.json({code:"NOT_FOUND"},404);
  const gone=await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_maintenance_component WHERE maintenance_id IN(SELECT id FROM pulse_maintenance WHERE id=? AND page_id=?)").bind(maintenance,page),c.env.DB.prepare("DELETE FROM pulse_maintenance WHERE id=? AND page_id=? RETURNING id").bind(maintenance,page)]);
  return gone[1]?.results.length?c.json({ok:true}):c.json({code:"NOT_FOUND"},404);
 });
