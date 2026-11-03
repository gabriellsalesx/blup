import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {pulseWrite,type PulseApp} from "./access";
import {publicKey,sha256} from "./limits";
import {sameSecret,sealTarget,chatIdSchema} from "./channel-crypto";
import {cleanText} from "./event";
import {projectChoice,usableProject,bindToProject} from "./channels";
const params=z.object({link:opaqueId});
const botReady=(env:Cloudflare.Env)=>!!env.PULSE_TELEGRAM_TOKEN&&!!env.PULSE_TELEGRAM_WEBHOOK_SECRET&&!!env.PULSE_CHANNEL_KEY;
export const telegramApi=new Hono<PulseApp>()
 .post("/link",pulseWrite,validate("json",z.object({}).strict()),async c=>{
  if(!botReady(c.env))return c.json({code:"CHANNEL_UNAVAILABLE"},503);
  const owner=c.get("session").user.id,id=crypto.randomUUID(),code=publicKey(),now=Date.now();
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM pulse_telegram_link WHERE user_id=?").bind(owner),c.env.DB.prepare("INSERT INTO pulse_telegram_link(id,user_id,hash,expires_at) VALUES(?,?,?,?)").bind(id,owner,await sha256(code),now+600_000)]);return c.json({id,url:`https://t.me/BlupPulseBot?start=${code}`,expiresAt:new Date(now+600_000).toISOString()},201);
 })
 .get("/link/:link",validate("param",params),async c=>{
  const row=await c.env.DB.prepare("SELECT id,state,expires_at AS expiresAt,chat_id AS chatId,chat_label AS chatLabel,channel_id AS channelId FROM pulse_telegram_link WHERE id=? AND user_id=?").bind(c.req.valid("param").link,c.get("session").user.id).first<{id:string;state:string;expiresAt:number;chatId:string|null;chatLabel:string|null;channelId:string|null}>();if(!row)return c.json({code:"NOT_FOUND"},404);
  return c.json({id:row.id,state:row.expiresAt<=Date.now()?"expired":row.state,expiresAt:new Date(row.expiresAt).toISOString(),candidate:row.chatId?{id:row.chatId,label:row.chatLabel??"Telegram"}:null,channelId:row.channelId});
 })
 .post("/link/:link/confirm",pulseWrite,validate("param",params),validate("json",projectChoice),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").link,project=c.req.valid("json").projectId,now=Date.now();if(!await usableProject(c.env,owner,project))return c.json({code:"NOT_FOUND"},404);
  const link=await c.env.DB.prepare("SELECT state,expires_at AS expiresAt,chat_id AS chatId,chat_label AS label,channel_id AS channelId FROM pulse_telegram_link WHERE id=? AND user_id=?").bind(id,owner).first<{state:string;expiresAt:number;chatId:string|null;label:string|null;channelId:string|null}>();if(!link)return c.json({code:"NOT_FOUND"},404);if(link.state==="confirmed"&&link.channelId){await bindToProject(c.env.DB,owner,project,link.channelId,"telegram").run();return c.json({id:link.channelId});}
  if(link.expiresAt<=now||link.state!=="candidate"||!chatIdSchema.safeParse(link.chatId).success||!link.chatId)return c.json({code:"LINK_EXPIRED"},409);
  const channel=id;let target:string;try{target=await sealTarget(c.env,owner,channel,"telegram",link.chatId);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  const result=await c.env.DB.batch([
   c.env.DB.prepare("INSERT OR IGNORE INTO pulse_channel(id,user_id,type,target,label,created_at) SELECT ?,?,'telegram',?,?,? WHERE EXISTS(SELECT 1 FROM pulse_telegram_link WHERE id=? AND user_id=? AND state='candidate' AND expires_at>?) AND (SELECT count(*) FROM pulse_channel WHERE user_id=? AND type='telegram')<? RETURNING id").bind(channel,owner,target,link.label??"Telegram",now,id,owner,now,owner,c.get("access").limits?.projects??0),
   c.env.DB.prepare("UPDATE pulse_telegram_link SET state='confirmed',channel_id=? WHERE id=? AND user_id=? AND EXISTS(SELECT 1 FROM pulse_channel WHERE id=? AND user_id=?)").bind(channel,id,owner,channel,owner),
   bindToProject(c.env.DB,owner,project,channel,"telegram"),
  ]);if(result[0].results.length)return c.json({id:channel},201);
  const existing=await c.env.DB.prepare("SELECT id FROM pulse_channel WHERE id=? AND user_id=?").bind(channel,owner).first();return existing?c.json({id:channel},200):c.json({code:"QUOTA"},409);
 });
const telegramUpdate=z.object({update_id:z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),message:z.object({text:z.string().max(4096).optional(),chat:z.object({id:z.number().int().min(-Number.MAX_SAFE_INTEGER).max(Number.MAX_SAFE_INTEGER),type:z.string().max(32),username:z.string().max(64).optional(),first_name:z.string().max(128).optional()}),from:z.object({id:z.number().int().min(1).max(Number.MAX_SAFE_INTEGER),is_bot:z.boolean()}).optional()}).optional()});
export const telegramWebhookApi=new Hono<PulseApp>()
 .post("/webhook",async(c,next)=>{const token=z.string().min(1).max(256).safeParse(c.req.header("x-telegram-bot-api-secret-token"));if(!token.success||!sameSecret(token.data,c.env.PULSE_TELEGRAM_WEBHOOK_SECRET??""))return c.json({code:"UNAUTHORIZED"},401);return next();},validate("json",telegramUpdate),async c=>{
  const update=c.req.valid("json"),message=update.message;
  if(!message||message.chat.type!=="private"||!message.from||message.from.is_bot||message.from.id!==message.chat.id)return c.json({ok:true});
  const match=/^\/start(?:@BlupPulseBot)? ([a-f0-9]{48})$/.exec(message.text??"");if(!match)return c.json({ok:true});
  const now=Date.now(),hash=await sha256(match[1]);
  if(!await c.env.DB.prepare("SELECT id FROM pulse_telegram_link WHERE hash=? AND state='pending' AND expires_at>?").bind(hash,now).first())return c.json({ok:true});
  await c.env.DB.batch([
   c.env.DB.prepare("INSERT OR IGNORE INTO pulse_telegram_update(id,received_at) VALUES(?,?)").bind(update.update_id,now),
   c.env.DB.prepare("UPDATE pulse_telegram_link SET state='candidate',chat_id=?,chat_label=? WHERE hash=? AND state='pending' AND expires_at>? AND EXISTS(SELECT 1 FROM pulse_telegram_update WHERE id=? AND applied=0)").bind(String(message.chat.id),cleanText(message.chat.username?`@${message.chat.username}`:message.chat.first_name??"Telegram",80),hash,now,update.update_id),
   c.env.DB.prepare("UPDATE pulse_telegram_update SET applied=1 WHERE id=?").bind(update.update_id),
  ]);return c.json({ok:true});
 });
