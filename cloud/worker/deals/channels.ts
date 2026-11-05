import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId} from "../validation";
import {discordWebhook,customWebhook,webhookTarget,newSecret,sameSecret} from "../pulse/channel-crypto";
import {publicKey,sha256} from "../pulse/limits";
import {languageOf,type Language} from "../../shared/language";
import {postLanguages} from "../../shared/deals-post";
import {dealsWrite,type DealsApp} from "./access";
import {cleanLine} from "./page";
import {dealsChatId,testText,sendTelegram,sendDiscord,sendWebhook} from "./message";
import {seal,unseal,addChannel,channelTypes,targetHash} from "./store";
import {sendWhatsapp,WHATSAPP_INTERVAL} from "./bridge";
import {deliverPost} from "./dispatch";
import {samplePost,sampleKinds} from "./samples";
import type {Delivery} from "../pulse/transports";

const channelParam=z.object({channel:opaqueId});
const linkParam=z.object({link:opaqueId});
const LINK_TTL=15*60_000;
const botReady=(env:Cloudflare.Env)=>!!env.DEALS_TELEGRAM_TOKEN&&!!env.DEALS_TELEGRAM_WEBHOOK_SECRET&&!!env.PULSE_CHANNEL_KEY;
const botName=(env:Cloudflare.Env)=>/^[A-Za-z0-9_]{5,32}$/.test(env.DEALS_TELEGRAM_BOT??"")?env.DEALS_TELEGRAM_BOT:"BlupDealsBot";
// One round of examples a minute per channel.
const TEST_GAP=60_000;
type Row={id:string;platform:string;name:string;active:number;lastError:string|null;createdAt:number;language:string|null;sessionId:string|null;testAt:number};
const view=(r:Row,now=Date.now())=>({id:r.id,type:z.enum(channelTypes).parse(r.platform),label:r.name,active:!!r.active,lastError:r.lastError,language:languageOf(r.language),createdAt:new Date(r.createdAt).toISOString(),testAgainAt:r.testAt>now?r.testAt:0});

export const channelsApi=new Hono<DealsApp>()
 .get("/",async c=>{
  const rows=await c.env.DB.prepare("SELECT id,platform,name,active,last_error AS lastError,created_at AS createdAt,language,session_id AS sessionId,test_at AS testAt FROM deals_channel WHERE user_id=? ORDER BY created_at,id LIMIT 50").bind(c.get("session").user.id).all<Row>();
  return c.json({channels:rows.results.map(r=>view(r))});
 })
 // A Discord webhook, or a webhook of the person's own system: its
 // signing secret is shown once, here (the same address again gets a new
 // one).
 .post("/",dealsWrite,validate("json",z.discriminatedUnion("type",[z.object({type:z.literal("discord"),webhook:discordWebhook}).strict(),z.object({type:z.literal("webhook"),url:customWebhook}).strict()])),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json");
  const secret=input.type==="webhook"?newSecret():null;
  const label=input.type==="webhook"?input.url.label:input.webhook.label;
  let saved:Awaited<ReturnType<typeof addChannel>>;
  try{saved=input.type==="webhook"?await addChannel(c.env,owner,"webhook",JSON.stringify({url:input.url.url,secret}),label,Date.now(),null,input.url.url):await addChannel(c.env,owner,"discord",input.webhook.url,label);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  if(!saved)return c.json({code:"QUOTA"},409);
  return c.json({id:saved.id,label,secret},saved.created?201:200);
 })
 // A webhook's new signing secret, shown once; the old one stops at once.
 .post("/:channel/secret",dealsWrite,validate("param",channelParam),validate("json",z.object({}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").channel;
  const row=await c.env.DB.prepare("SELECT target FROM deals_channel WHERE id=? AND user_id=? AND platform='webhook'").bind(id,owner).first<{target:string}>();
  if(!row)return c.json({code:"NOT_FOUND"},404);
  const secret=newSecret();
  try{const current=webhookTarget.parse(JSON.parse(await unseal(c.env,owner,id,"webhook",row.target)));await c.env.DB.prepare("UPDATE deals_channel SET target=? WHERE id=? AND user_id=?").bind(await seal(c.env,owner,id,"webhook",JSON.stringify({url:current.url,secret})),id,owner).run();}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  return c.json({secret});
 })
 // The language the channel's posts go out in (null: the account's).
 .patch("/:channel",dealsWrite,validate("param",channelParam),validate("json",z.object({language:z.enum(postLanguages).nullable()}).strict()),async c=>{
  const row=await c.env.DB.prepare("UPDATE deals_channel SET language=? WHERE id=? AND user_id=? RETURNING id").bind(c.req.valid("json").language,c.req.valid("param").channel,c.get("session").user.id).first();
  return row?c.json({ok:true}):c.json({code:"NOT_FOUND"},404);
 })
 .delete("/:channel",validate("param",channelParam),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").channel;
  const [removed]=await c.env.DB.batch([
   c.env.DB.prepare("DELETE FROM deals_channel WHERE id=? AND user_id=? RETURNING id").bind(id,owner),
   c.env.DB.prepare("DELETE FROM deals_post WHERE channel_id=? AND user_id=? AND state='queued'").bind(id,owner),
   c.env.DB.prepare("UPDATE deals_hunt SET channels=(SELECT json_group_array(value) FROM json_each(deals_hunt.channels) WHERE value<>?) WHERE user_id=? AND EXISTS(SELECT 1 FROM json_each(deals_hunt.channels) WHERE value=?)").bind(id,owner,id),
  ]);
  return removed.results.length?c.json({ok:true}):c.json({code:"NOT_FOUND"},404);
 })
 // Examples, right away: the plain test and posts of each kind (as they
 // would really go out, marked as examples), one round a minute. On
 // WhatsApp one at a time: each takes the Canal's 30-minute turn. The
 // answer says whether all arrived (else why) and when it can go again.
 .post("/:channel/test",dealsWrite,validate("param",channelParam),validate("json",z.object({kinds:z.array(z.enum(sampleKinds)).min(1).max(sampleKinds.length).refine(k=>new Set(k).size===k.length)}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").channel,kinds=c.req.valid("json").kinds,now=Date.now();
  const row=await c.env.DB.prepare("SELECT c.platform,c.target,c.session_id AS sessionId,c.language AS channelLanguage,c.test_at AS testAt,u.language FROM deals_channel c JOIN user u ON u.id=c.user_id WHERE c.id=? AND c.user_id=?").bind(id,owner).first<{platform:string;target:string;sessionId:string|null;channelLanguage:string|null;testAt:number;language:string|null}>();
  if(!row)return c.json({code:"NOT_FOUND"},404);
  const type=z.enum(channelTypes).parse(row.platform);
  if(type==="whatsapp"&&kinds.length>1)return c.json({code:"BAD_REQUEST"},400);
  // The round is taken atomically: two clicks never send twice.
  if(!await c.env.DB.prepare("UPDATE deals_channel SET test_at=? WHERE id=? AND user_id=? AND test_at<=? RETURNING id").bind(now+TEST_GAP,id,owner,now).first())return c.json({code:"TOO_SOON",testAgainAt:row.testAt},429);
  const account:Language=languageOf(row.language)??"pt",lang:Language=languageOf(row.channelLanguage)??account;
  let target:string;try{target=await unseal(c.env,owner,id,type,row.target);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  let result:Delivery={sent:true,permanent:false,code:null,waitMs:0,global:false},sent=0;
  for(const kind of kinds){
   if(sent)await new Promise(r=>setTimeout(r,Math.min(1500,Math.max(300,result.waitMs))));
   const key=`test:${id}:${now}:${kind}`,session=row.sessionId;
   if(kind==="test")result=type==="telegram"?await sendTelegram(c.env,target,testText(lang),null):type==="discord"?await sendDiscord(target,{username:"Blup Deals",allowed_mentions:{parse:[]},content:testText(lang)}):type==="webhook"?await sendWebhook(target,null,lang):session?await sendWhatsapp(c.env,session,target,key,testText(lang),null):{sent:false,permanent:false,code:"WHATSAPP_DISCONNECTED",waitMs:0,global:false};
   else result=await deliverPost(c.env,type,target,await samplePost(c.env.DB,owner,kind,lang,account,new URL(c.req.url).origin),lang,fetch,session?{session,key}:undefined);
   if(!result.sent)break;
   sent++;
  }
  // WhatsApp's turn: after a post, 30 minutes; a "too soon", what is left.
  const again=type!=="whatsapp"?now+TEST_GAP:result.code==="TOO_SOON"?now+Math.max(TEST_GAP,result.waitMs):result.sent?now+WHATSAPP_INTERVAL:now+TEST_GAP;
  // A "too soon" or the Bridge down says nothing about the channel itself.
  const keep=result.code==="TOO_SOON"||result.code==="BRIDGE_UNAVAILABLE";
  await c.env.DB.prepare(`UPDATE deals_channel SET test_at=?${keep?"":`,last_error=?${result.sent?",active=1":result.code==="DESTINATION_DISABLED"?",active=0":""}`} WHERE id=? AND user_id=?`).bind(...(keep?[again]:[again,result.sent?null:result.code]),id,owner).run();
  return c.json({ok:result.sent,code:result.code,sent,testAgainAt:again},202);
 });

// Linking Telegram: the panel opens the bot in private with a one-time
// code; the bot answers with buttons to add it as admin to a channel or a
// group; the chat where that same Telegram user adds it is the candidate.
export const telegramApi=new Hono<DealsApp>()
 .post("/link",dealsWrite,validate("json",z.object({}).strict()),async c=>{
  if(!botReady(c.env))return c.json({code:"TELEGRAM_UNAVAILABLE"},503);
  const owner=c.get("session").user.id,id=crypto.randomUUID(),code=publicKey(),now=Date.now();
  await c.env.DB.batch([c.env.DB.prepare("DELETE FROM deals_telegram_link WHERE user_id=? AND state<>'confirmed'").bind(owner),c.env.DB.prepare("INSERT INTO deals_telegram_link(id,user_id,hash,expires_at) VALUES(?,?,?,?)").bind(id,owner,await sha256(code),now+LINK_TTL)]);
  return c.json({id,url:`https://t.me/${botName(c.env)}?start=${code}`,expiresAt:new Date(now+LINK_TTL).toISOString()},201);
 })
 .get("/link/:link",validate("param",linkParam),async c=>{
  const row=await c.env.DB.prepare("SELECT state,expires_at AS expiresAt,chat_label AS label,chat_id AS chatId,channel_id AS channelId FROM deals_telegram_link WHERE id=? AND user_id=?").bind(c.req.valid("param").link,c.get("session").user.id).first<{state:string;expiresAt:number;label:string|null;chatId:string|null;channelId:string|null}>();
  if(!row)return c.json({code:"NOT_FOUND"},404);
  const state=row.state==="confirmed"?"confirmed":row.expiresAt<=Date.now()?"expired":row.state==="candidate"?"candidate":"pending";
  return c.json({state:z.enum(["pending","candidate","expired","confirmed"]).parse(state),candidate:state==="candidate"&&row.chatId?{id:row.chatId,label:row.label??"Telegram"}:null,channelId:row.channelId});
 })
 .post("/link/:link/confirm",dealsWrite,validate("param",linkParam),validate("json",z.object({}).strict()),async c=>{
  const owner=c.get("session").user.id,id=c.req.valid("param").link,now=Date.now();
  const link=await c.env.DB.prepare("SELECT state,expires_at AS expiresAt,chat_id AS chatId,chat_label AS label,channel_id AS channelId FROM deals_telegram_link WHERE id=? AND user_id=?").bind(id,owner).first<{state:string;expiresAt:number;chatId:string|null;label:string|null;channelId:string|null}>();
  if(!link)return c.json({code:"NOT_FOUND"},404);
  if(link.state==="confirmed"&&link.channelId)return c.json({id:link.channelId},200);
  const chat=dealsChatId.safeParse(link.chatId);
  if(link.state!=="candidate"||link.expiresAt<=now||!chat.success)return c.json({code:"LINK_EXPIRED"},409);
  let saved:Awaited<ReturnType<typeof addChannel>>;try{saved=await addChannel(c.env,owner,"telegram",chat.data,link.label??"Telegram",now);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
  if(!saved)return c.json({code:"QUOTA"},409);
  await c.env.DB.prepare("UPDATE deals_telegram_link SET state='confirmed',channel_id=? WHERE id=? AND user_id=?").bind(saved.id,id,owner).run();
  return c.json({id:saved.id},201);
 });

const chatSchema=z.object({id:z.number().int().min(-Number.MAX_SAFE_INTEGER).max(Number.MAX_SAFE_INTEGER),type:z.string().max(32),title:z.string().max(256).optional(),username:z.string().max(64).optional(),first_name:z.string().max(128).optional()});
const fromSchema=z.object({id:z.number().int().min(1).max(Number.MAX_SAFE_INTEGER),is_bot:z.boolean()});
const telegramUpdate=z.object({
 update_id:z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),
 message:z.object({text:z.string().max(4096).optional(),chat:chatSchema,from:fromSchema.optional()}).optional(),
 my_chat_member:z.object({chat:chatSchema,from:fromSchema,new_chat_member:z.object({status:z.string().max(32),can_post_messages:z.boolean().optional()})}).optional(),
});
const replies={
 pt:{text:"Quase lá! Agora adicione o Blup Deals como administrador do canal ou grupo que vai receber as ofertas.",channel:"Adicionar a um canal",group:"Adicionar a um grupo"},
 en:{text:"Almost there! Now add Blup Deals as an admin of the channel or group that will get the deals.",channel:"Add to a channel",group:"Add to a group"},
 es:{text:"¡Casi listo! Ahora agrega Blup Deals como administrador del canal o grupo que recibirá las ofertas.",channel:"Agregar a un canal",group:"Agregar a un grupo"},
} as const;
// Telegram calls this (secret in the header). The answer may carry one
// Bot API call, which is how the private reply goes without a request.
export const telegramWebhookApi=new Hono<{Bindings:Cloudflare.Env}>()
 .post("/webhook",async(c,next)=>{const token=z.string().min(1).max(256).safeParse(c.req.header("x-telegram-bot-api-secret-token"));if(!token.success||!sameSecret(token.data,c.env.DEALS_TELEGRAM_WEBHOOK_SECRET??""))return c.json({code:"UNAUTHORIZED"},401);return next();},validate("json",telegramUpdate),async c=>{
  const update=c.req.valid("json"),now=Date.now();
  const fresh=await c.env.DB.prepare("INSERT OR IGNORE INTO deals_telegram_update(id,received_at,applied) VALUES(?,?,1) RETURNING id").bind(update.update_id,now).first();
  if(!fresh)return c.json({ok:true});
  const message=update.message;
  if(message?.from&&!message.from.is_bot&&message.chat.type==="private"&&message.from.id===message.chat.id){
   const match=new RegExp(`^/start(?:@${botName(c.env)})? ([a-f0-9]{48})$`).exec(message.text??"");if(!match)return c.json({ok:true});
   const started=await c.env.DB.prepare("UPDATE deals_telegram_link SET state='started',telegram_user=? WHERE hash=? AND state IN('pending','started') AND expires_at>? RETURNING user_id AS userId").bind(String(message.from.id),await sha256(match[1]),now).first<{userId:string}>();
   if(!started)return c.json({ok:true});
   const language=languageOf((await c.env.DB.prepare("SELECT language FROM user WHERE id=?").bind(started.userId).first<{language:string|null}>())?.language)??"pt",r=replies[language],bot=botName(c.env);
   return c.json({method:"sendMessage",chat_id:message.chat.id,text:r.text,reply_markup:{inline_keyboard:[[{text:r.channel,url:`https://t.me/${bot}?startchannel&admin=post_messages`}],[{text:r.group,url:`https://t.me/${bot}?startgroup&admin=post_messages`}]]}});
  }
  const member=update.my_chat_member;
  if(member&&!member.from.is_bot&&["channel","group","supergroup"].includes(member.chat.type)){
   const chatId=String(member.chat.id),status=member.new_chat_member.status;
   // Only an admin bot posts (the panel asks for that): in a channel it must
   // also have the right to post, stated, not just not denied.
   const canPost=status==="administrator"&&(member.chat.type!=="channel"||member.new_chat_member.can_post_messages===true);
   if(!canPost){
    // Removed, demoted to member or without the right to post: every
    // account's channel there stops, and it is not a candidate.
    await c.env.DB.prepare("UPDATE deals_channel SET active=0,last_error='DESTINATION_DISABLED' WHERE target_hash=?").bind(await targetHash("telegram",chatId)).run();return c.json({ok:true});
   }
   const label=cleanLine(member.chat.title??(member.chat.username?`@${member.chat.username}`:"Telegram"),80)||"Telegram";
   await c.env.DB.prepare("UPDATE deals_telegram_link SET state='candidate',chat_id=?,chat_label=? WHERE id=(SELECT id FROM deals_telegram_link WHERE telegram_user=? AND state IN('started','candidate') AND expires_at>? ORDER BY expires_at DESC LIMIT 1)").bind(chatId,label,String(member.from.id),now).run();
  }
  return c.json({ok:true});
 });
