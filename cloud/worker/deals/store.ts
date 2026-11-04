import {z} from "zod";
import {sealTarget,openTarget} from "../pulse/channel-crypto";
import {sha256} from "../pulse/limits";
import {dealsLimits} from "../../shared/pulse-plans";
import {affiliateBody,type Affiliate,type Store} from "./links";

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
// Queues one offer for each channel, at most dealsLimits.postsPerDay per
// account and day (the day it goes out) and never the same offer twice in
// a channel. One D1 batch runs in order, so each insert counts the ones
// before it. Returns how many were queued.
// priceShown false: the post gives the discount but not the price (a
// price from another country's store).
export async function enqueue(db:D1Database,input:{owner:string;offerId:string;channels:readonly string[];link:string;affiliate:boolean;huntId:string|null;sendAt:number;now?:number;priceShown?:boolean}){
 const now=input.now??Date.now(),day=dayOf(input.sendAt);
 if(!input.channels.length)return 0;
 const results=await db.batch(input.channels.map(channel=>db.prepare("INSERT OR IGNORE INTO deals_post(id,user_id,hunt_id,channel_id,offer_id,code,link,affiliate,price_shown,state,next_at,day,created_at) SELECT ?,?,?,?,?,?,?,?,?,'queued',?,?,? WHERE (SELECT count(*) FROM deals_post WHERE user_id=? AND day=?)<? AND EXISTS(SELECT 1 FROM deals_channel WHERE id=? AND user_id=? AND active=1 AND plan_disabled=0)")
  .bind(crypto.randomUUID(),input.owner,input.huntId,channel,input.offerId,shortCode(),input.link,input.affiliate?1:0,input.priceShown===false?0:1,input.sendAt,day,now,input.owner,day,dealsLimits.postsPerDay,channel,input.owner)));
 return results.reduce((n,r)=>n+(r.meta.changes??0),0);
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
