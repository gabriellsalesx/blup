import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {discordWebhook,slackWebhook,customWebhook,sealTarget,openTarget,webhookTarget,channelType,channelTypes,newSecret} from "./channel-crypto";
import {pulseWrite,ownedProject,type PulseApp} from "./access";
import {projectId,planHolds,planBinds} from "./limits";
// One destination of each type per project, and at most ten projects
// (the projects route): the whole list fits.
export const MAX_CHANNELS=10*channelTypes.length;
const channelParam=z.object({channel:opaqueId});
// Every destination belongs to one project. A new one is made for a project
// and becomes its destination of that type, in the same batch that creates
// it; the one it replaces is deleted with its queue, so no destination is
// ever left without a project.
export const projectChoice=z.strictObject({projectId});
// The alerts a destination can be tested with: a plain test, or an example
// of each kind it sends for a project (marked as a test).
export const testKinds=["test","issue","regression","monitor_down","monitor_up"] as const;
const testRequest=z.union([z.strictObject({}),z.strictObject({projectId,kinds:z.array(z.enum(testKinds)).min(1).max(testKinds.length).refine(v=>new Set(v).size===v.length)})]);
// One test per destination a minute; the dispatcher sends at each minute.
export const TEST_SPACING_MS=60_000;
const nextRound=(now:number)=>Math.floor(now/60_000)*60_000+60_000;
// A destination's new address, read by its own kind's rules.
const addressOf={discord:discordWebhook,slack:slackWebhook,webhook:customWebhook} as const;
type Stored={type:string;target:string};
const ownChannel=(db:D1Database,id:string,owner:string)=>db.prepare("SELECT type,target FROM pulse_channel WHERE id=? AND user_id=?").bind(id,owner).first<Stored>();
// Saving a destination again: sealed anew, active, without the last error,
// and free to send at once.
const resave=(db:D1Database,id:string,owner:string,target:string,label:string)=>db.prepare("UPDATE pulse_channel SET target=?,label=?,active=1,last_error=NULL,next_at=0 WHERE id=? AND user_id=?").bind(target,label,id,owner).run();
export async function usableProject(env:Cloudflare.Env,owner:string,id:string){const project=await ownedProject(env,owner,id);return project&&project.active&&!project.planDisabled?project:null;}
// A project's quota counts the destinations of a type, without the one
// being replaced: SQL for the check, with the binds that follow it.
export const typeRoom="(SELECT count(*) FROM pulse_channel WHERE user_id=? AND type=? AND id NOT IN(SELECT channel_id FROM pulse_project_channel WHERE project_id=? AND type=?))<?";
// Deletes the owner's destinations no project sends to (the ones just
// replaced), with their queue and cooldowns.
export const dropUnbound=(db:D1Database,owner:string)=>{const unbound="SELECT id FROM pulse_channel WHERE user_id=? AND id NOT IN(SELECT channel_id FROM pulse_project_channel)";return [
 db.prepare(`DELETE FROM pulse_outbox WHERE channel_id IN(${unbound})`).bind(owner),
 db.prepare(`DELETE FROM pulse_cooldown WHERE substr(key,1,instr(key,':')-1) IN(${unbound})`).bind(owner),
 db.prepare(`DELETE FROM pulse_channel WHERE id IN(${unbound})`).bind(owner),
];};
export const bindToProject=(db:D1Database,owner:string,project:string,channel:string,type:z.infer<typeof channelType>)=>db.prepare("INSERT INTO pulse_project_channel(project_id,channel_id,type) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM pulse_channel WHERE id=? AND user_id=? AND type=?) ON CONFLICT(project_id,type) DO UPDATE SET channel_id=excluded.channel_id").bind(project,channel,type,channel,owner,type);
export const channelsApi=new Hono<PulseApp>()
 .get("/",async c=>{
  const channels=await c.env.DB.prepare("SELECT id,type,label,active,last_error AS lastError,test_at AS testAt FROM pulse_channel WHERE user_id=? ORDER BY created_at LIMIT ?").bind(c.get("session").user.id,MAX_CHANNELS).all<{id:string;type:string;label:string;active:number;lastError:string|null;testAt:number}>();
  const bindings=await c.env.DB.prepare("SELECT p.project_id AS projectId,p.channel_id AS channelId FROM pulse_project_channel p JOIN pulse_channel c ON c.id=p.channel_id WHERE c.user_id=?").bind(c.get("session").user.id).all<{projectId:string;channelId:string}>();
  return c.json({channels:c.get("access").readable?channels.results.map(({testAt,...row})=>({...row,type:channelType.parse(row.type),active:!!row.active,testAgainAt:testAt>0?testAt+TEST_SPACING_MS:0})):[],bindings:c.get("access").readable?bindings.results:[]});
 })
 // A destination by an address the person pastes: a Discord or Slack
 // webhook, or a webhook of their own system (whose signing secret is
 // made here and shown in this answer only).
 .post("/",pulseWrite,validate("json",z.discriminatedUnion("type",[projectChoice.extend({type:z.literal("discord"),webhook:discordWebhook}).strict(),projectChoice.extend({type:z.literal("slack"),webhook:slackWebhook}).strict(),projectChoice.extend({type:z.literal("webhook"),url:customWebhook}).strict()])),async c=>{
  const input=c.req.valid("json"),owner=c.get("session").user.id,id=crypto.randomUUID();if(!await usableProject(c.env,owner,input.projectId))return c.json({code:"NOT_FOUND"},404);
  const secret=input.type==="webhook"?newSecret():null;
  const destination=input.type==="webhook"?{target:JSON.stringify({url:input.url.url,secret}),label:input.url.label}:{target:input.webhook.url,label:input.webhook.label};
  let target:string;try{target=await sealTarget(c.env,owner,id,input.type,destination.target);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  const [made]=await c.env.DB.batch([c.env.DB.prepare(`INSERT INTO pulse_channel(id,user_id,type,target,label,created_at) SELECT ?,?,?,?,?,? WHERE ${typeRoom} AND ${planHolds} RETURNING id`).bind(id,owner,input.type,target,destination.label,Date.now(),owner,input.type,input.projectId,input.type,c.get("access").limits?.projects??0,...planBinds(owner,c.get("access"))),bindToProject(c.env.DB,owner,input.projectId,id,input.type),...dropUnbound(c.env.DB,owner)]);
  return made?.results.length?c.json({id,label:destination.label,secret},201):c.json({code:"QUOTA"},409);
 })
 // A new address for a Discord, Slack or webhook destination (a webhook
 // keeps its signing secret). Telegram is linked again instead.
 .patch("/:channel",pulseWrite,validate("param",channelParam),validate("json",z.strictObject({address:z.string().max(2048)})),async c=>{
  const id=c.req.valid("param").channel,owner=c.get("session").user.id,stored=await ownChannel(c.env.DB,id,owner);if(!stored)return c.json({code:"NOT_FOUND"},404);
  const type=channelType.parse(stored.type);if(type==="telegram")return c.json({code:"INVALID"},400);
  const parsed=addressOf[type].safeParse(c.req.valid("json").address);if(!parsed.success)return c.json({code:"INVALID"},400);
  let target:string;
  try{
   const value=type==="webhook"?JSON.stringify({url:parsed.data.url,secret:webhookTarget.parse(JSON.parse(await openTarget(c.env,owner,id,type,stored.target))).secret}):parsed.data.url;
   target=await sealTarget(c.env,owner,id,type,value);
  }catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  await resave(c.env.DB,id,owner,target,parsed.data.label);return c.json({label:parsed.data.label});
 })
 // A webhook's new signing secret, shown once; the old one stops at once.
 .post("/:channel/secret",pulseWrite,validate("param",channelParam),validate("json",z.strictObject({})),async c=>{
  const id=c.req.valid("param").channel,owner=c.get("session").user.id,stored=await ownChannel(c.env.DB,id,owner);if(!stored||stored.type!=="webhook")return c.json({code:"NOT_FOUND"},404);
  const secret=newSecret();let target:string,label:string;
  try{const current=webhookTarget.parse(JSON.parse(await openTarget(c.env,owner,id,"webhook",stored.target)));label=`Webhook · ${new URL(current.url).hostname}`.slice(0,80);target=await sealTarget(c.env,owner,id,"webhook",JSON.stringify({url:current.url,secret}));}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  await resave(c.env.DB,id,owner,target,label);return c.json({secret});
 })
 .delete("/:channel",validate("param",channelParam),async c=>{
  const id=c.req.valid("param").channel,owner=c.get("session").user.id;
  if(!await c.env.DB.prepare("SELECT id FROM pulse_channel WHERE id=? AND user_id=?").bind(id,owner).first())return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_outbox WHERE channel_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_project_channel WHERE channel_id=?").bind(id),c.env.DB.prepare("DELETE FROM pulse_cooldown WHERE key LIKE ?").bind(`${id}:%`),c.env.DB.prepare("DELETE FROM pulse_channel WHERE id=? AND user_id=?").bind(id,owner)]);return c.json({ok:true});
 })
 .post("/:channel/test",pulseWrite,validate("param",channelParam),validate("json",testRequest),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").channel,body=c.req.valid("json"),now=Date.now();
  const current=await c.env.DB.prepare("SELECT id FROM pulse_channel WHERE id=? AND user_id=? AND active=1").bind(id,owner).first();if(!current)return c.json({code:"NOT_FOUND"},404);
  // Examples are for a project this destination is a destination of.
  const project="projectId" in body?body.projectId:null,kinds="kinds" in body?body.kinds:["test" as const];
  if(project&&!await c.env.DB.prepare("SELECT 1 FROM pulse_project_channel pc JOIN pulse_project p ON p.id=pc.project_id WHERE pc.project_id=? AND pc.channel_id=? AND p.user_id=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL").bind(project,id,owner).first())return c.json({code:"NOT_FOUND"},404);
  const claim=await c.env.DB.prepare("UPDATE pulse_channel SET test_at=? WHERE id=? AND user_id=? AND test_at<=? RETURNING id").bind(now,id,owner,now-TEST_SPACING_MS).first();if(!claim){c.header("Retry-After","60");return c.json({code:"TOO_MANY_REQUESTS"},429);}
  await c.env.DB.batch(kinds.map(kind=>{const event=crypto.randomUUID();return c.env.DB.prepare("INSERT INTO pulse_outbox(id,user_id,project_id,channel_id,ref_id,kind,created_at,expires_at,next_at) VALUES(?,?,?,?,?,?,?,?,?)").bind(event,owner,kind==="test"?null:project,id,event,kind==="test"?"test":`sample_${kind}`,now,now+86_400_000,now);}));
  return c.json({queued:kinds.length,sendsAt:nextRound(now),testAgainAt:now+TEST_SPACING_MS},202);
 });
