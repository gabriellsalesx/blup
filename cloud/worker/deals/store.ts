import {z} from "zod";
import {sealTarget,openTarget} from "../pulse/channel-crypto";
import {sha256} from "../pulse/limits";
import {dealsLimits} from "../../shared/pulse-plans";
import {affiliateBody,type Affiliate,type Store} from "./links";
import {dealsPaces,headlineKinds,socialIds,socialNetworks,socialURL,hasAddress,HEADLINE_MAX,EXTRA_MAX,type HeadlineKind,type SocialNetwork} from "../../shared/deals-post";

export const DAY=86_400_000;
export const dayOf=(now=Date.now())=>new Date(now).toISOString().slice(0,10);
export const channelTypes=["telegram","discord"] as const;
export type ChannelType=(typeof channelTypes)[number];
// The shortest wait between two posts in the same channel.
export const CHANNEL_INTERVAL=2*60_000;
export class DealsError extends Error{constructor(readonly code:string,readonly status:400|403|404|409|429|502|503){super(code);}}

// Channel targets and store secrets are sealed with the Pulse destination
// key, under the Deals context (they never open as each other).
export const seal=(env:Cloudflare.Env,owner:string,id:string,type:string,value:string)=>sealTarget(env,owner,id,type,value,"blup-deals");
export const unseal=(env:Cloudflare.Env,owner:string,id:string,type:string,value:string)=>openTarget(env,owner,id,type,value,"blup-deals");
export const targetHash=(type:ChannelType,target:string)=>sha256(`deals:${type}:${target}`);

export async function affiliateOf(env:Cloudflare.Env,owner:string,store:Store):Promise<Affiliate|null>{
 const row=await env.DB.prepare("SELECT credentials FROM deals_affiliate WHERE user_id=? AND store=?").bind(owner,store).first<{credentials:string|null}>();
 if(!row?.credentials)return null;
 try{const parsed=affiliateBody.safeParse(JSON.parse(await unseal(env,owner,store,"affiliate",row.credentials)));return parsed.success&&parsed.data.store===store?parsed.data:null;}catch{return null;}
}
// The channels among ids that belong to the owner and are on.
export async function usableChannels(db:D1Database,owner:string,ids:readonly string[]){
 if(!ids.length)return [];
 const rows=await db.prepare("SELECT id FROM deals_channel WHERE user_id=? AND active=1 AND plan_disabled=0 AND id IN(SELECT value FROM json_each(?))").bind(owner,JSON.stringify([...new Set(ids)])).all<{id:string}>();
 return rows.results.map(r=>r.id);
}
const alphabet="ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789";
export function shortCode(){return Array.from(crypto.getRandomValues(new Uint8Array(8)),b=>alphabet[b%alphabet.length]).join("");}
export const shortCodeSchema=z.string().regex(/^[A-Za-z0-9]{8}$/);
// What the person sets for all their posts (PUT /api/deals/settings): the
// pace, the opening line per kind (missing: the default), the extra text
// and the social networks, one of each, on their own sites.
const written=z.string().trim().refine(t=>!hasAddress(t),{message:"ADDRESS"});
export const postStyles=["classic","emoji"] as const;
export const settingsBody=z.object({
 pace:z.literal(dealsPaces),style:z.enum(postStyles).default("classic"),
 headlines:z.partialRecord(z.enum(headlineKinds),written.pipe(z.string().min(1).max(HEADLINE_MAX))),
 extra:z.object({text:written.pipe(z.string().max(EXTRA_MAX)),on:z.boolean()}).strict(),
 socials:z.object({links:z.array(z.object({network:z.enum(socialIds),url:z.string().max(300)}).strict()).max(socialNetworks.length)
  .refine(l=>new Set(l.map(x=>x.network)).size===l.length,{message:"DUPLICATE"})
  .transform((l,ctx)=>l.map(x=>{const url=socialURL(x.network,x.url);if(!url)ctx.addIssue({code:"custom",message:"SOCIAL_URL",path:[x.network]});return {network:x.network,url:url??""};})),on:z.boolean()}).strict(),
}).strict();
export type DealsSettings=z.infer<typeof settingsBody>&{paceLast:number;droppedAt:number|null};
const savedHeadlines=z.partialRecord(z.enum(headlineKinds),z.string()).catch({});
const savedSocials=z.array(z.object({network:z.enum(socialIds),url:z.string()})).catch([]);
type SettingRow={pace:number;style:string;paceLast:number;droppedAt:number|null;headlines:string;extra:string|null;extraOn:number;socials:string;socialsOn:number};
const json=(v:string)=>{try{return JSON.parse(v) as unknown;}catch{return null;}};
export async function settingsOf(db:D1Database,owner:string):Promise<DealsSettings>{
 const r=await db.prepare("SELECT pace,style,pace_last AS paceLast,dropped_at AS droppedAt,headlines,extra,extra_on AS extraOn,socials,socials_on AS socialsOn FROM deals_setting WHERE user_id=?").bind(owner).first<SettingRow>();
 const pace=dealsPaces.find(p=>p===r?.pace)??0;
 return {pace,style:postStyles.find(v=>v===r?.style)??"classic",paceLast:r?.paceLast??0,droppedAt:r?.droppedAt??null,headlines:r?savedHeadlines.parse(json(r.headlines)):{},extra:{text:r?.extra??"",on:!!r?.extraOn},socials:{links:r?savedSocials.parse(json(r.socials)):[],on:!!r?.socialsOn}};
}
export const headlineOf=(s:Pick<DealsSettings,"headlines">,kind:string)=>(headlineKinds as readonly string[]).includes(kind)?s.headlines[kind as HeadlineKind]??null:null;
export const socialLinks=(s:Pick<DealsSettings,"socials">)=>s.socials.on?s.socials.links.flatMap(l=>{const net=socialNetworks.find(n=>n.id===l.network);return net&&socialURL(l.network as SocialNetwork,l.url)?[{name:net.name,url:l.url}]:[];}):[];

// A hunt's find waits at most this long in the queue; past it (a pace too
// slow for what the hunts find) it is not queued, and the panel warns.
export const QUEUE_HORIZON=2*DAY;
// The pace's next free time: one pace after the last one given, never
// before sendAt. Taken atomically (two runs never get the same time) and
// given back when nothing was queued.
async function paceSlot(db:D1Database,owner:string,sendAt:number,now:number,horizon:number):Promise<{at:number;undo:()=>Promise<unknown>}|"full"|null>{
 for(let tries=0;tries<3;tries++){
  const row=await db.prepare("SELECT pace,pace_last AS last FROM deals_setting WHERE user_id=?").bind(owner).first<{pace:number;last:number}>();
  if(!row||row.pace<=0)return null;
  const at=Math.max(sendAt,row.last+row.pace*60_000);
  if(at>now+horizon){await db.prepare("UPDATE deals_setting SET dropped_at=? WHERE user_id=?").bind(now,owner).run();return "full";}
  if(await db.prepare("UPDATE deals_setting SET pace_last=? WHERE user_id=? AND pace_last=? RETURNING user_id").bind(at,owner,row.last).first())
   return {at,undo:()=>db.prepare("UPDATE deals_setting SET pace_last=? WHERE user_id=? AND pace_last=?").bind(row.last,owner,at).run()};
 }
 return "full";
}
// Queues one offer for each channel, at most dealsLimits.postsPerDay per
// account and day (the day it goes out) and never the same offer twice in
// a channel. One D1 batch runs in order, so each insert counts the ones
// before it. Returns how many were queued.
// priceShown false: the post gives the discount but not the price (a
// price from another country's store). paced: it waits its turn in the
// account's pace (a hunt's find, a link put in the queue), at most horizon
// ahead.
export async function enqueue(db:D1Database,input:{owner:string;offerId:string;channels:readonly string[];link:string;affiliate:boolean;huntId:string|null;sendAt:number;now?:number;priceShown?:boolean;paced?:boolean;horizon?:number;description?:string|null;coupon?:string|null}){
 const now=input.now??Date.now();
 if(!input.channels.length)return 0;
 const slot=input.paced?await paceSlot(db,input.owner,input.sendAt,now,input.horizon??QUEUE_HORIZON):null;
 if(slot==="full")return 0;
 const sendAt=slot?.at??input.sendAt,day=dayOf(sendAt);
 const results=await db.batch(input.channels.map(channel=>db.prepare("INSERT OR IGNORE INTO deals_post(id,user_id,hunt_id,channel_id,offer_id,code,link,affiliate,price_shown,paced,description,coupon,state,next_at,day,created_at) SELECT ?,?,?,?,?,?,?,?,?,?,?,?,'queued',?,?,? WHERE (SELECT count(*) FROM deals_post WHERE user_id=? AND day=?)<? AND EXISTS(SELECT 1 FROM deals_channel WHERE id=? AND user_id=? AND active=1 AND plan_disabled=0)")
  .bind(crypto.randomUUID(),input.owner,input.huntId,channel,input.offerId,shortCode(),input.link,input.affiliate?1:0,input.priceShown===false?0:1,input.paced?1:0,input.description||null,input.coupon||null,sendAt,day,now,input.owner,day,dealsLimits.postsPerDay,channel,input.owner)));
 const queued=results.reduce((n,r)=>n+(r.meta.changes??0),0);
 if(!queued&&slot)await slot.undo();
 return queued;
}
// A new pace spaces again what waits in it, in its order: the first one a
// pace after the last paced post sent (or now), the next a pace later.
// Posts of one offer to several channels share their time.
export async function respace(db:D1Database,owner:string,pace:number,now=Date.now()){
 const rows=await db.prepare("SELECT id,next_at AS at FROM deals_post WHERE user_id=? AND state='queued' AND paced=1 AND claim_until<=? ORDER BY next_at,created_at").bind(owner,now).all<{id:string;at:number}>();
 const last=(await db.prepare("SELECT max(sent_at) AS at FROM deals_post WHERE user_id=? AND paced=1 AND state='sent'").bind(owner).first<{at:number|null}>())?.at??0;
 const groups=new Map<number,string[]>();for(const r of rows.results)groups.set(r.at,[...(groups.get(r.at)??[]),r.id]);
 const step=pace*60_000;let at=step?Math.max(now,last+step):now,given=last;
 const updates:D1PreparedStatement[]=[];
 for(const ids of groups.values()){updates.push(db.prepare("UPDATE deals_post SET next_at=?,day=? WHERE user_id=? AND state='queued' AND id IN(SELECT value FROM json_each(?))").bind(at,dayOf(at),owner,JSON.stringify(ids)));given=at;if(step)at+=step;}
 updates.push(db.prepare("UPDATE deals_setting SET pace_last=? WHERE user_id=?").bind(given,owner));
 for(let i=0;i<updates.length;i+=50)await db.batch(updates.slice(i,i+50));
}
export async function postsOn(db:D1Database,owner:string,day:string){
 return (await db.prepare("SELECT count(*) AS n FROM deals_post WHERE user_id=? AND day=?").bind(owner,day).first<{n:number}>())?.n??0;
}
// Whether the person already posted this product lately: a priced one in
// the last 30 days at a price not more than 1% above this one, an unpriced
// one in the last 7 days. The same product comes back only cheaper.
export async function postedLately(db:D1Database,owner:string,store:string,externalId:string,price:number|null,now=Date.now()){
 if(price!==null){
  const best=await db.prepare("SELECT min(o.price) AS price FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.user_id=? AND o.store=? AND o.external_id=? AND p.created_at>?").bind(owner,store,externalId,now-30*DAY).first<{price:number|null}>();
  return best?.price!=null&&price>=best.price*0.99;
 }
 return !!await db.prepare("SELECT 1 FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.user_id=? AND o.store=? AND o.external_id=? AND p.created_at>? LIMIT 1").bind(owner,store,externalId,now-7*DAY).first();
}
