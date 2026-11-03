import {Hono} from "hono";
import {z} from "zod";
import {ownerMatches} from "../plans";
import {validate,opaqueId} from "../validation";
import {pulseWrite,ownedProject,type PulseApp} from "./access";
import {projectId} from "./limits";
import {monitorURL,monitorURLSchema} from "./monitor-http";
const params=z.object({monitor:opaqueId});
const body=z.object({projectId,name:z.string().trim().min(1).max(80),url:monitorURLSchema,intervalMinutes:z.number().int().min(2).max(1440)}).strict();
export const monitorsApi=new Hono<PulseApp>()
 .get("/",async c=>{
  const monitors=await c.env.DB.prepare("SELECT id,project_id AS projectId,name,url,interval_minutes AS intervalMinutes,active,state,last_at AS lastAt,plan_disabled AS planDisabled,retire_at AS retireAt FROM pulse_monitor WHERE user_id=? AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 50").bind(c.get("session").user.id,Date.now()).all<{id:string;projectId:string;name:string;url:string;intervalMinutes:number;active:number;state:string;lastAt:number|null;planDisabled:number;retireAt:number|null}>();
  return c.json({monitors:c.get("access").readable?monitors.results.map(m=>({...m,active:!!m.active&&!m.planDisabled,writable:!m.planDisabled&&c.get("access").writable,readOnlyUntil:m.retireAt===null?null:new Date(m.retireAt).toISOString(),lastAt:m.lastAt===null?null:new Date(m.lastAt).toISOString()})):[]});
 })
 .post("/",pulseWrite,validate("json",body),async c=>{
  const input=c.req.valid("json"),user=c.get("session").user,access=c.get("access");
  if(input.intervalMinutes<(access.limits?.intervalMinutes??1440))return c.json({code:"PLAN"},403);
  const project=await ownedProject(c.env,user.id,input.projectId);if(!project||!project.active||project.planDisabled)return c.json({code:"NOT_FOUND"},404);
  let url:URL;try{url=monitorURL(input.url,ownerMatches(c.env.STAFF_OWNER_EMAIL,user.email,user.emailVerified));}catch{return c.json({code:"INVALID_TARGET"},400);}
  const id=crypto.randomUUID(),now=Date.now();
  const made=await c.env.DB.prepare("INSERT INTO pulse_monitor(id,user_id,project_id,name,url,interval_minutes,next_at,created_at) SELECT ?,?,?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_monitor WHERE user_id=? AND plan_disabled=0)<? AND (SELECT count(*) FROM pulse_monitor WHERE user_id=?)<50 AND EXISTS(SELECT 1 FROM pulse_project WHERE id=? AND user_id=? AND active=1 AND plan_disabled=0 AND deleted_at IS NULL) RETURNING id").bind(id,user.id,project.id,input.name,url.href,input.intervalMinutes,now,now,user.id,access.limits?.monitors??0,user.id,project.id,user.id).first();return made?c.json({id},201):c.json({code:"QUOTA"},409);
 })
 .patch("/:monitor",pulseWrite,validate("param",params),validate("json",z.object({name:body.shape.name.optional(),url:monitorURLSchema.optional(),intervalMinutes:body.shape.intervalMinutes.optional(),active:z.boolean().optional()}).strict().refine(v=>Object.keys(v).length>0)),async c=>{
  const user=c.get("session").user,input=c.req.valid("json"),id=c.req.valid("param").monitor;
  const current=await c.env.DB.prepare("SELECT name,url,interval_minutes AS intervalMinutes,active,plan_disabled AS planDisabled FROM pulse_monitor WHERE id=? AND user_id=?").bind(id,user.id).first<{name:string;url:string;intervalMinutes:number;active:number;planDisabled:number}>();if(!current)return c.json({code:"NOT_FOUND"},404);
  if(current.planDisabled)return c.json({code:"PLAN"},403);
  if((input.intervalMinutes??current.intervalMinutes)<(c.get("access").limits?.intervalMinutes??1440))return c.json({code:"PLAN"},403);
  let url:URL;try{url=monitorURL(input.url??current.url,ownerMatches(c.env.STAFF_OWNER_EMAIL,user.email,user.emailVerified));}catch{return c.json({code:"INVALID_TARGET"},400);}
  await c.env.DB.prepare("UPDATE pulse_monitor SET name=?,url=?,interval_minutes=?,active=?,claim=NULL,claim_until=0,next_at=?,state='unknown',last_at=NULL,revision=revision+1 WHERE id=? AND user_id=?").bind(input.name??current.name,url.href,input.intervalMinutes??current.intervalMinutes,input.active===undefined?current.active:Number(input.active),Date.now(),id,user.id).run();return c.json({ok:true});
 })
 .delete("/:monitor",validate("param",params),async c=>{const deleted=await c.env.DB.prepare("DELETE FROM pulse_monitor WHERE id=? AND user_id=? RETURNING id").bind(c.req.valid("param").monitor,c.get("session").user.id).first();return deleted?c.json({ok:true},200):c.json({code:"NOT_FOUND"},404);})
 // A test alert to every destination of the monitor's project, telling it
 // apart by its name. Each destination takes one test a minute, like the
 // destination's own test.
 .post("/:monitor/test",pulseWrite,validate("param",params),validate("json",z.object({}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").monitor,now=Date.now();
  const monitor=await c.env.DB.prepare("SELECT m.project_id AS projectId FROM pulse_monitor m JOIN pulse_project p ON p.id=m.project_id WHERE m.id=? AND m.user_id=? AND m.plan_disabled=0 AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL").bind(id,owner).first<{projectId:string}>();if(!monitor)return c.json({code:"NOT_FOUND"},404);
  const channels=await c.env.DB.prepare("SELECT c.id FROM pulse_project_channel pc JOIN pulse_channel c ON c.id=pc.channel_id WHERE pc.project_id=? AND c.user_id=? AND c.active=1 LIMIT 2").bind(monitor.projectId,owner).all<{id:string}>();if(!channels.results.length)return c.json({code:"NO_CHANNELS"},409);
  const claimed:string[]=[];
  for(const channel of channels.results)if(await c.env.DB.prepare("UPDATE pulse_channel SET test_at=? WHERE id=? AND user_id=? AND test_at<=? RETURNING id").bind(now,channel.id,owner,now-60_000).first())claimed.push(channel.id);
  if(!claimed.length){c.header("Retry-After","60");return c.json({code:"TOO_MANY_REQUESTS"},429);}
  await c.env.DB.batch(claimed.map(channel=>c.env.DB.prepare("INSERT INTO pulse_outbox(id,user_id,project_id,channel_id,ref_id,kind,created_at,expires_at,next_at) VALUES(?,?,?,?,?,'test',?,?,?)").bind(crypto.randomUUID(),owner,monitor.projectId,channel,id,now,now+86_400_000,now)));
  return c.json({queued:claimed.length},202);
 })
 .get("/:monitor/checks",validate("param",params),async c=>{
  const row=await c.env.DB.prepare("SELECT p.object_name AS objectName FROM pulse_monitor m JOIN pulse_project p ON p.id=m.project_id WHERE m.id=? AND m.user_id=? AND p.deleted_at IS NULL AND (m.retire_at IS NULL OR m.retire_at>?)").bind(c.req.valid("param").monitor,c.get("session").user.id,Date.now()).first<{objectName:string}>();if(!row||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  return c.json(await c.env.PULSE_PROJECT.getByName(row.objectName).monitorHistory(c.req.valid("param").monitor,c.get("access").limits?.retentionDays??30));
 });
