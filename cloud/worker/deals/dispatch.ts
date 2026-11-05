import {siteLock,unlockSite} from "../quota";
import {languageOf} from "../../shared/language";
import {dealsAccess} from "./access";
import {telegramText,discordPayload,whatsappText,sendTelegram,sendDiscord,sendWebhook,type Post} from "./message";
import {sendWhatsapp,WHATSAPP_INTERVAL} from "./bridge";
import {unseal,settingsOf,headlineOf,socialLinks,channelTypes,CHANNEL_INTERVAL,DAY,dayOf,type ChannelType,type DealsSettings} from "./store";
import {headline,defaultHeadlines,headlineKinds,storeTitles,type HeadlineKind} from "../../shared/deals-post";
import {translate,gameTitle} from "./ai";
import type {Delivery} from "../pulse/transports";

type Due={id:string;userId:string;channelId:string;code:string;link:string;attempts:number;day:string;platform:string;target:string;channelNext:number;active:number;title:string;image:string|null;price:number|null;originalPrice:number|null;currency:string|null;discount:number|null;kind:string;credit:string|null;language:string;affiliate:number;priceShown:number;url:string;store:string;description:string|null;coupon:string|null;offerId:string;shortTitle:string|null;channelLanguage:string|null;sessionId:string|null;whatsappState:string|null};
// The post's opening line: the person's (or the default, in the account's
// language) with the offer's name, discount and store; a video keeps its
// own. A line that ends up empty falls back to the name.
export function openingLine(item:{title:string;kind:string;discount:number|null;store:string;language:string|null},own:Pick<DealsSettings,"headlines">){
 if(!(headlineKinds as readonly string[]).includes(item.kind))return item.title;
 const template=headlineOf(own,item.kind)??defaultHeadlines[languageOf(item.language)??"pt"][item.kind as HeadlineKind];
 return headline(template,{name:item.title,discount:item.discount,store:storeTitles[item.store]??""})||item.title;
}
export const shortLink=(env:Cloudflare.Env,code:string)=>`https://${env.DEALS_PUBLIC_HOST||"deals.getblup.site"}/r/${code}`;
// key is the post's own: the WhatsApp Bridge publishes it only once.
export async function deliverPost(env:Cloudflare.Env,type:ChannelType,target:string,post:Post,language:string|null,send:typeof fetch=fetch,whatsapp?:{session:string;key:string}):Promise<Delivery>{
 const lang=languageOf(language)??"pt";
 if(type==="webhook")return sendWebhook(target,post,lang,send);
 if(type==="whatsapp")return whatsapp?sendWhatsapp(env,whatsapp.session,target,whatsapp.key,whatsappText(post,lang),post.image,send):{sent:false,permanent:false,code:"WHATSAPP_DISCONNECTED",waitMs:300_000,global:false};
 return type==="telegram"?sendTelegram(env,target,telegramText(post,lang),post.image,send,post.watch??null):sendDiscord(target,discordPayload(post,lang),send);
}
// The shortest wait between two posts in a channel: on a WhatsApp Canal,
// the safety mode's 30 minutes (the Bridge checks it again).
const intervalOf=(type:ChannelType)=>type==="whatsapp"?WHATSAPP_INTERVAL:CHANNEL_INTERVAL;
// The Deals bot's own budget per platform, kept in D1 (the table Pulse
// uses, under other keys) so every run and every account shares it:
// Telegram at most one message every 40 ms (25 per second, under the
// about 30 per second Telegram allows a bot), Discord every 100 ms. A
// global 429 pauses the whole platform for its Retry-After. Webhooks go
// to many systems, each its own: only a light spacing.
export const DEALS_BUDGET={telegram:{key:"deals-telegram",spacing:40},discord:{key:"deals-discord",spacing:100},whatsapp:{key:"deals-whatsapp",spacing:200},webhook:{key:"deals-webhook",spacing:20}} as const;
async function slot(db:D1Database,type:ChannelType,at:number){
 const {key,spacing}=DEALS_BUDGET[type];
 await db.prepare("INSERT OR IGNORE INTO pulse_delivery_budget(key,next_at) VALUES(?,0)").bind(key).run();
 if(await db.prepare("UPDATE pulse_delivery_budget SET next_at=? WHERE key=? AND next_at<=? RETURNING key").bind(at+spacing,key,at).first())return {ready:true,nextAt:at};
 const row=await db.prepare("SELECT next_at AS nextAt FROM pulse_delivery_budget WHERE key=?").bind(key).first<{nextAt:number}>();
 return {ready:false,nextAt:row?.nextAt??at+1000};
}
// The send queue, every minute: one post per channel at a time, at least
// CHANNEL_INTERVAL apart, within the bot's budget, and only while the
// account has the plan. The daily cap was taken when the post was queued
// (store.enqueue).
export async function dispatchDeals(env:Cloudflare.Env,send:typeof fetch=fetch,now=Date.now()){
 const lock=await siteLock(env.DB,"deals-dispatch",55_000);if(!lock)return;
 // now is the run's clock (fixed in tests); time spent in the run adds up.
 const started=Date.now(),clock=()=>now+(Date.now()-started),paused=new Set<ChannelType>();
 try{
  const due=await env.DB.prepare(`SELECT p.id,p.user_id AS userId,p.channel_id AS channelId,p.code,p.link,p.attempts,p.day,c.platform,c.target,c.next_at AS channelNext,c.active,
   o.title,o.store,o.image_url AS image,o.price,o.original_price AS originalPrice,o.currency,o.discount,o.kind,o.credit,o.url,u.language,
   p.affiliate,p.price_shown AS priceShown,p.description,p.coupon,p.offer_id AS offerId,o.short_title AS shortTitle,c.language AS channelLanguage,c.session_id AS sessionId,w.state AS whatsappState
   FROM deals_post p JOIN deals_channel c ON c.id=p.channel_id JOIN deals_offer o ON o.id=p.offer_id JOIN user u ON u.id=p.user_id LEFT JOIN deals_whatsapp w ON w.id=c.session_id
   WHERE p.state='queued' AND p.next_at<=? AND p.claim_until<=? ORDER BY p.next_at,p.created_at LIMIT 50`).bind(now,now).all<Due>();
  const access=new Map<string,boolean>(),settings=new Map<string,DealsSettings>(),busy=new Set<string>();
  // One translation per post and language in a run (several channels).
  const translations=new Map<string,Promise<string[]|null>>();
  for(const item of due.results){
   // At most 45 s of work per run, measured on the run itself.
   if(clock()-now>45_000)break;
   const fail=(code:string)=>env.DB.prepare("UPDATE deals_post SET state='failed',error=?,claim=NULL,claim_until=0 WHERE id=? AND state='queued'").bind(code,item.id).run();
   if(!access.has(item.userId))access.set(item.userId,!(await dealsAccess(env,item.userId,clock())).readOnly);
   if(!access.get(item.userId)){await fail("PLAN");continue;}
   if(!item.active){await fail("DESTINATION_DISABLED");continue;}
   // Valid for two days after the day it was meant to go out (sendAt),
   // however far ahead it was scheduled; retries keep their own cap.
   if(item.day<dayOf(now-2*DAY)){await fail("EXPIRED");continue;}
   if(busy.has(item.channelId))continue;
   const type:ChannelType=channelTypes.find(t=>t===item.platform)??"discord";
   // A Canal whose WhatsApp is not connected waits (until it reconnects,
   // or the two days run out).
   if(type==="whatsapp"&&item.whatsappState!=="connected")continue;
   // The platform's budget: a short wait is fine; a long one (a global
   // pause) leaves the rest of that platform for the next run.
   if(paused.has(type))continue;
   // A timer may wake a little before the clock reaches its time, so a
   // short wait is retried (a few times, with a millisecond to spare).
   let budget=await slot(env.DB,type,clock());
   for(let tries=0;!budget.ready&&tries<3&&budget.nextAt-clock()<=1200;tries++){const wait=Math.max(1,budget.nextAt-clock()+1);await new Promise(resolve=>setTimeout(resolve,wait));budget=await slot(env.DB,type,clock());}
   if(!budget.ready){paused.add(type);continue;}
   busy.add(item.channelId);
   // The channel's turn: taken atomically, or the post waits for it.
   const turn=await env.DB.prepare("UPDATE deals_channel SET next_at=? WHERE id=? AND next_at<=? RETURNING id").bind(now+intervalOf(type),item.channelId,now).first();
   if(!turn){await env.DB.prepare("UPDATE deals_post SET next_at=max(next_at,?) WHERE id=? AND state='queued'").bind(item.channelNext,item.id).run();continue;}
   const claim=crypto.randomUUID();
   if(!await env.DB.prepare("UPDATE deals_post SET claim=?,claim_until=?,attempts=attempts+1 WHERE id=? AND state='queued' AND claim_until<=? RETURNING id").bind(claim,now+60_000,item.id,now).first())continue;
   let result:Delivery;
   try{
    const target=await unseal(env,item.userId,item.channelId,type,item.target);
    if(!settings.has(item.userId))settings.set(item.userId,await settingsOf(env.DB,item.userId));
    const own=settings.get(item.userId)!;
    // The name (shortened when the account asked the AI for it), the
    // opening line and, for a channel in another language, the post's own
    // texts translated (as they are when the AI can't).
    const account=languageOf(item.language)??"pt",lang=languageOf(item.channelLanguage)??account;
    const name=own.ai.titles?item.shortTitle??(item.kind==="game_free"?gameTitle(item.title):item.title):item.title;
    const custom=headlineOf(own,item.kind)!==null;
    let title=openingLine({...item,title:name,language:custom?account:lang},own),description=item.description,extra=own.extra.on&&own.extra.text?own.extra.text:null;
    if(lang!==account){
     const texts=[...(custom?[title]:[]),...(description?[description]:[]),...(extra?[extra]:[])];
     const key=`${item.userId}:${item.offerId}:${lang}`;
     if(texts.length&&!translations.has(key))translations.set(key,translate(env,item.userId,texts,lang).catch(()=>null));
     const done=texts.length?await translations.get(key)!:null;
     if(done){let k=0;if(custom)title=done[k++];if(description)description=done[k++];if(extra)extra=done[k++];}
    }
    result=await deliverPost(env,type,target,{store:item.store,title,extra,socials:socialLinks(own),style:own.style,description,coupon:item.coupon,image:item.image,price:item.priceShown?item.price:null,originalPrice:item.priceShown?item.originalPrice:null,currency:item.currency,discount:item.discount,link:shortLink(env,item.code),affiliate:!!item.affiliate,kind:item.kind,credit:item.credit,watch:item.kind==="video"?item.url:null},lang,send,item.sessionId?{session:item.sessionId,key:item.id}:undefined);
   }catch{result={sent:false,permanent:false,code:"CHANNEL_UNAVAILABLE",waitMs:300_000,global:false};}
   if(result.sent){
    await env.DB.batch([
     env.DB.prepare("UPDATE deals_post SET state='sent',sent_at=?,error=NULL,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(clock(),item.id,claim),
     env.DB.prepare("UPDATE deals_channel SET last_error=NULL,next_at=max(next_at,?) WHERE id=?").bind(now+Math.max(intervalOf(type),result.waitMs),item.channelId),
    ]);continue;
   }
   // The Bridge's "too soon" (another Canal of the same number posted a
   // moment ago) is a turn, not a failure: the post waits, nothing counts.
   const soon=result.code==="TOO_SOON",last=!soon&&(result.permanent||item.attempts+1>=5);
   if(result.global)paused.add(type);
   await env.DB.batch([
    ...(result.global?[env.DB.prepare("UPDATE pulse_delivery_budget SET next_at=max(next_at,?) WHERE key=?").bind(clock()+result.waitMs,DEALS_BUDGET[type].key)]:[]),
    env.DB.prepare(`UPDATE deals_post SET state=?,error=?,next_at=?,claim=NULL,claim_until=0${soon?",attempts=max(0,attempts-1)":""} WHERE id=? AND claim=?`).bind(last?"failed":"queued",soon?null:result.code,now+Math.max(60_000,result.waitMs),item.id,claim),
    soon?env.DB.prepare("UPDATE deals_channel SET next_at=max(next_at,?) WHERE id=?").bind(now+result.waitMs,item.channelId)
     :env.DB.prepare(`UPDATE deals_channel SET last_error=?${result.code==="DESTINATION_DISABLED"?",active=0":""},next_at=max(next_at,?) WHERE id=?`).bind(result.code,now+Math.max(60_000,result.waitMs),item.channelId),
   ]);
  }
 }finally{await unlockSite(env.DB,"deals-dispatch",lock);}
}
