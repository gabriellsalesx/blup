import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {discordWebhook,sealTarget} from "./channel-crypto";
import {pulseWrite,ownedProject,type PulseApp} from "./access";
import {projectId} from "./limits";
const channelParam=z.object({channel:opaqueId});
const channelType=z.enum(["discord","telegram"]);
// A new destination is made for one project: it becomes that project's
// destination of its type, in the same batch that creates it, taking the
// place of the one it had.
export const projectChoice=z.object({projectId}).strict();
export async function usableProject(env:Cloudflare.Env,owner:string,id:string){const project=await ownedProject(env,owner,id);return project&&project.active&&!project.planDisabled?project:null;}
export const bindToProject=(db:D1Database,owner:string,project:string,channel:string,type:z.infer<typeof channelType>)=>db.prepare("INSERT INTO pulse_project_channel(project_id,channel_id,type) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM pulse_channel WHERE id=? AND user_id=? AND type=?) ON CONFLICT(project_id,type) DO UPDATE SET channel_id=excluded.channel_id").bind(project,channel,type,channel,owner,type);
export const channelsApi=new Hono<PulseApp>()
 .get("/",async c=>{
  const channels=await c.env.DB.prepare("SELECT id,type,label,active,last_error AS lastError FROM pulse_channel WHERE user_id=? ORDER BY created_at LIMIT 20").bind(c.get("session").user.id).all<{id:string;type:string;label:string;active:number;lastError:string|null}>();
  const bindings=await c.env.DB.prepare("SELECT p.project_id AS projectId,p.channel_id AS channelId FROM pulse_project_channel p JOIN pulse_channel c ON c.id=p.channel_id WHERE c.user_id=?").bind(c.get("session").user.id).all<{projectId:string;channelId:string}>();
  return c.json({channels:c.get("access").readable?channels.results.map(row=>({...row,type:channelType.parse(row.type),active:!!row.active})):[],bindings:c.get("access").readable?bindings.results:[]});
 })
 .post("/",pulseWrite,validate("json",projectChoice.extend({type:z.literal("discord"),webhook:discordWebhook}).strict()),async c=>{
  const input=c.req.valid("json"),owner=c.get("session").user.id,id=crypto.randomUUID();if(!await usableProject(c.env,owner,input.projectId))return c.json({code:"NOT_FOUND"},404);let target:string;
  try{target=await sealTarget(c.env,owner,id,"discord",input.webhook.url);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  const [made]=await c.env.DB.batch([c.env.DB.prepare("INSERT INTO pulse_channel(id,user_id,type,target,label,created_at) SELECT ?,?,'discord',?,?,? WHERE (SELECT count(*) FROM pulse_channel WHERE user_id=? AND type='discord')<? RETURNING id").bind(id,owner,target,input.webhook.label,Date.now(),owner,c.get("access").limits?.projects??0),bindToProject(c.env.DB,owner,input.projectId,id,"discord")]);return made.results.length?c.json({id,label:input.webhook.label},201):c.json({code:"QUOTA"},409);
 })
 .delete("/:channel",validate("param",channelParam),async c=>{
  const id=c.req.valid("param").channel,owner=c.get("session").user.id;
  if(!await c.env.DB.prepare("SELECT id FROM pulse_channel WHERE id=? AND user_id=?").bind(id,owner).first())return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_outbox WHERE channel_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_project_channel WHERE channel_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_cooldown WHERE key LIKE ?").bind(`${id}:%`),c.env.DB.prepare("DELETE FROM pulse_channel WHERE id=? AND user_id=?").bind(id,owner)]);return c.json({ok:true});
 })
 .post("/:channel/test",pulseWrite,validate("param",channelParam),validate("json",z.object({}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").channel,now=Date.now();
  const current=await c.env.DB.prepare("SELECT id FROM pulse_channel WHERE id=? AND user_id=? AND active=1").bind(id,owner).first();if(!current)return c.json({code:"NOT_FOUND"},404);
  const claim=await c.env.DB.prepare("UPDATE pulse_channel SET test_at=? WHERE id=? AND user_id=? AND test_at<=? RETURNING id").bind(now,id,owner,now-60_000).first();if(!claim){c.header("Retry-After","60");return c.json({code:"TOO_MANY_REQUESTS"},429);}
  const event=crypto.randomUUID();await c.env.DB.prepare("INSERT INTO pulse_outbox(id,user_id,channel_id,ref_id,kind,created_at,expires_at,next_at) VALUES(?,?,?,?,'test',?,?,?)").bind(event,owner,id,event,now,now+86_400_000,now).run();return c.json({queued:true},202);
 });
export const projectChannelsApi=new Hono<PulseApp>()
 .put("/:project/channels",pulseWrite,validate("param",z.object({project:projectId})),validate("json",z.object({channelIds:z.array(opaqueId).max(2).refine(v=>new Set(v).size===v.length)}).strict()),async c=>{
  const owner=c.get("session").user.id,project=c.req.valid("param").project;if(!await ownedProject(c.env,owner,project))return c.json({code:"NOT_FOUND"},404);
  const selected:{id:string;type:string}[]=[];
  for(const id of c.req.valid("json").channelIds){const row=await c.env.DB.prepare("SELECT id,type FROM pulse_channel WHERE id=? AND user_id=? AND active=1").bind(id,owner).first<{id:string;type:string}>();if(!row)return c.json({code:"NOT_FOUND"},404);selected.push(row);}
  if(new Set(selected.map(r=>r.type)).size!==selected.length)return c.json({code:"INVALID"},400);
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_project_channel WHERE project_id=?").bind(project),...selected.map(row=>c.env.DB.prepare("INSERT INTO pulse_project_channel VALUES(?,?,?)").bind(project,row.id,row.type))]);return c.json({ok:true});
 });
