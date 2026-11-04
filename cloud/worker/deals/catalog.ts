import {z} from "zod";
import {sha256} from "../pulse/limits";
import {apiJSON} from "./fetch";
import {cleanLine,priceCents} from "./page";
import {dealsAccess} from "./access";
import {enqueue,DAY} from "./store";

// The global catalog: an offer is stored once, whoever finds it, by a hash
// of where it comes from and what it is. Each price change goes to the
// history (an honest "before" price later).
export type OfferInput={key:string;store:string;externalId:string|null;sourceId:string|null;title:string;url:string;imageUrl:string|null;price:number|null;originalPrice:number|null;currency:string|null;kind:"product"|"game_free"|"game_deal";platforms:string[];discount:number|null;credit:string|null;expiresAt:number|null};
export type StoredOffer={id:string;isNew:boolean};
export async function upsertOffer(db:D1Database,o:OfferInput,now=Date.now()):Promise<StoredOffer>{
 const hash=await sha256(o.key);
 const row=await db.prepare(`INSERT INTO deals_offer(id,hash,store,external_id,source_id,title,url,image_url,price,original_price,currency,first_seen_at,last_seen_at,expires_at,kind,platforms,discount,credit) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
  ON CONFLICT(hash) DO UPDATE SET title=excluded.title,url=excluded.url,image_url=coalesce(excluded.image_url,image_url),price=excluded.price,original_price=excluded.original_price,currency=excluded.currency,last_seen_at=excluded.last_seen_at,expires_at=excluded.expires_at,platforms=excluded.platforms,discount=excluded.discount
  RETURNING id,first_seen_at AS firstSeen`).bind(crypto.randomUUID(),hash,o.store,o.externalId,o.sourceId,o.title,o.url,o.imageUrl,o.price,o.originalPrice,o.currency,now,now,o.expiresAt,o.kind,JSON.stringify(o.platforms),o.discount,o.credit).first<{id:string;firstSeen:number}>();
 if(!row)throw new Error("OFFER");
 if(o.price!==null&&o.currency)await db.prepare("INSERT OR IGNORE INTO deals_price(offer_id,at,price,currency) SELECT ?,?,?,? WHERE (SELECT price FROM deals_price WHERE offer_id=? ORDER BY at DESC LIMIT 1) IS NOT ?").bind(row.id,now,o.price,o.currency,row.id,o.price).run();
 return {id:row.id,isNew:row.firstSeen===now};
}

// The platform ids the panel offers. games_deals (CheapShark, PC stores)
// takes only the PC ones; pc stands for every PC store.
export const platforms=["pc","steam","epic","gog","ubisoft","itchio","playstation","xbox","switch","android","ios"] as const;
export const pcPlatforms=["pc","steam","epic","gog","ubisoft"] as const;
const pcFamily=new Set<string>(["pc","steam","epic","gog","ubisoft","itchio"]);
const gamerPowerPlatforms:[RegExp,string][]=[[/^pc$/,"pc"],[/steam/,"steam"],[/epic/,"epic"],[/gog/,"gog"],[/ubisoft/,"ubisoft"],[/itch/,"itchio"],[/playstation|ps4|ps5/,"playstation"],[/xbox/,"xbox"],[/switch/,"switch"],[/android/,"android"],[/ios/,"ios"]];
export function gamerPowerPlatformIds(raw:string){
 const out=new Set<string>();for(const part of raw.toLowerCase().split(",").map(s=>s.trim()))for(const [re,id] of gamerPowerPlatforms)if(re.test(part))out.add(id);
 if([...out].some(p=>pcFamily.has(p)))out.add("pc");return [...out];
}
const cheapSharkStores:Record<string,string>={"1":"steam","7":"gog","13":"ubisoft","25":"epic"};

const gamerPowerItem=z.object({id:z.number().int().positive(),title:z.string().max(512),worth:z.string().max(32),image:z.string().max(1024).optional(),thumbnail:z.string().max(1024).optional(),open_giveaway_url:z.url().max(1024),platforms:z.string().max(512),end_date:z.string().max(32),status:z.string().max(32),type:z.string().max(32)}).loose();
const cheapSharkItem=z.object({dealID:z.string().regex(/^[A-Za-z0-9%+/=]{8,256}$/),title:z.string().max(512),storeID:z.string().regex(/^[0-9]{1,4}$/),salePrice:z.string().max(16),normalPrice:z.string().max(16),savings:z.string().max(32),thumb:z.string().max(1024).optional(),gameID:z.string().max(32).optional()}).loose();
const httpsOnly=(v:string|undefined)=>{if(!v)return null;try{const u=new URL(v);return u.protocol==="https:"?u.href:null;}catch{return null;}};
export function gamerPowerOffers(value:unknown,now=Date.now()):OfferInput[]{
 const list=z.array(z.unknown()).max(1000).safeParse(value);if(!list.success)return [];
 return list.data.flatMap(raw=>{
  const g=gamerPowerItem.safeParse(raw);if(!g.success||g.data.status.toLowerCase()!=="active"||g.data.type.toLowerCase()!=="game")return [];
  const url=httpsOnly(g.data.open_giveaway_url);if(!url)return [];
  const end=Date.parse(g.data.end_date.replace(" ","T")+"Z"),title=cleanLine(g.data.title,200);if(!title||Number.isFinite(end)&&end<=now)return [];
  const worth=priceCents(g.data.worth);
  return [{key:`gamerpower:${g.data.id}`,store:"gamerpower",externalId:String(g.data.id),sourceId:"gamerpower",title,url,imageUrl:httpsOnly(g.data.image??g.data.thumbnail),price:0,originalPrice:worth&&worth>0?worth:null,currency:"USD",kind:"game_free" as const,platforms:gamerPowerPlatformIds(g.data.platforms),discount:100,credit:"gamerpower",expiresAt:Number.isFinite(end)?end:null}];
 });
}
export function cheapSharkOffers(value:unknown):OfferInput[]{
 const list=z.array(z.unknown()).max(1000).safeParse(value);if(!list.success)return [];
 return list.data.flatMap(raw=>{
  const d=cheapSharkItem.safeParse(raw);if(!d.success)return [];
  const store=cheapSharkStores[d.data.storeID];if(!store)return [];
  const price=priceCents(d.data.salePrice),original=priceCents(d.data.normalPrice),title=cleanLine(d.data.title,200),discount=Math.round(Number(d.data.savings));
  if(price===null||!title||!Number.isFinite(discount))return [];
  // The CheapShark link itself, never an affiliate swap (their terms).
  return [{key:`cheapshark:${d.data.dealID}`,store:"cheapshark",externalId:d.data.dealID,sourceId:"cheapshark",title,url:`https://www.cheapshark.com/redirect?dealID=${encodeURIComponent(decodeURIComponent(d.data.dealID))}`,imageUrl:httpsOnly(d.data.thumb),price,originalPrice:original,currency:"USD",kind:"game_deal" as const,platforms:["pc",store],discount:Math.max(0,Math.min(100,discount)),credit:"cheapshark",expiresAt:null}];
 });
}

// The shared sources, each read once per round for every account.
const sources={
 gamerpower:{url:"https://www.gamerpower.com/api/giveaways?type=game",parse:gamerPowerOffers},
 cheapshark:{url:"https://www.cheapshark.com/api/1.0/deals?storeID=1,7,13,25&onSale=1&sortBy=Recent&pageSize=60",parse:cheapSharkOffers},
} as const;
type SourceId=keyof typeof sources;
const ROUND=30*60_000;

export const huntFilters=z.object({platforms:z.array(z.enum(platforms)).max(platforms.length).optional(),minDiscount:z.number().int().min(0).max(100).optional(),maxPrice:z.number().int().min(0).max(100_000_00).optional()}).strict();
export type HuntFilters=z.infer<typeof huntFilters>;
export const huntKinds=["games_free","games_deals"] as const;
export type HuntKind=(typeof huntKinds)[number];
const offerKind={games_free:"game_free",games_deals:"game_deal"} as const;
export const huntSource={games_free:"gamerpower",games_deals:"cheapshark"} as const;
type CatalogRow={id:string;url:string;price:number|null;discount:number|null;platforms:string};
export function matches(filters:HuntFilters,offer:CatalogRow){
 const have:string[]=z.array(z.string()).catch([]).parse(JSON.parse(offer.platforms));
 if(filters.platforms?.length&&!filters.platforms.some(p=>have.includes(p)))return false;
 if(filters.minDiscount!==undefined&&(offer.discount??0)<filters.minDiscount)return false;
 if(filters.maxPrice!==undefined&&(offer.price===null||offer.price>filters.maxPrice))return false;
 return true;
}
type Hunt={id:string;userId:string;kind:HuntKind;filters:HuntFilters;channels:string[]};
const PER_ROUND=3;
// Sends a hunt the newest matching offers it has not sent yet (at most
// PER_ROUND each time), to every channel it names.
export async function feedHunt(env:Cloudflare.Env,hunt:Hunt,since:number,now=Date.now()){
 const rows=await env.DB.prepare("SELECT id,url,price,discount,platforms FROM deals_offer WHERE kind=? AND first_seen_at>=? AND (expires_at IS NULL OR expires_at>?) ORDER BY first_seen_at DESC,id LIMIT 200").bind(offerKind[hunt.kind],since,now).all<CatalogRow>();
 let sent=0;
 for(const offer of rows.results){
  if(sent>=PER_ROUND)break;if(!matches(hunt.filters,offer))continue;
  const fresh=await env.DB.prepare("SELECT 1 FROM deals_offer WHERE id=? AND NOT EXISTS(SELECT 1 FROM deals_post WHERE offer_id=? AND user_id=?)").bind(offer.id,offer.id,hunt.userId).first();if(!fresh)continue;
  const queued=await enqueue(env.DB,{owner:hunt.userId,offerId:offer.id,channels:hunt.channels,link:offer.url,affiliate:false,huntId:hunt.id,sendAt:now,now});
  if(queued)sent++;else if(!await env.DB.prepare("SELECT 1 FROM deals_post WHERE offer_id=? AND user_id=?").bind(offer.id,hunt.userId).first())break;
 }
 if(sent)await env.DB.prepare("UPDATE deals_hunt SET last_post_at=? WHERE id=?").bind(now,hunt.id).run();
 return sent;
}
export const huntRow=z.object({id:z.string(),userId:z.string(),kind:z.enum(huntKinds),filters:z.string().transform(v=>huntFilters.catch({}).parse(JSON.parse(v))),channels:z.string().transform(v=>z.array(z.string()).catch([]).parse(JSON.parse(v)))});

// One round: every due source read once, the catalog updated, and each
// active hunt of that source fed with what is new. The very first read
// only fills the catalog (no flood of old offers).
export async function collectDeals(env:Cloudflare.Env,send:typeof fetch=fetch,now=Date.now()){
 await env.DB.batch((Object.keys(sources) as SourceId[]).map(id=>env.DB.prepare("INSERT OR IGNORE INTO deals_source(id,kind,key,next_at,created_at) VALUES(?,?,?,0,?)").bind(id,id,id,now)));
 for(const id of Object.keys(sources) as SourceId[]){
  const claim=crypto.randomUUID();
  const source=await env.DB.prepare("UPDATE deals_source SET claim=?,claim_until=? WHERE id=? AND next_at<=? AND claim_until<=? RETURNING last_at AS lastAt").bind(claim,now+120_000,id,now,now).first<{lastAt:number|null}>();
  if(!source)continue;
  try{
   const offers=sources[id].parse(await apiJSON(sources[id].url,{headers:{accept:"application/json","user-agent":"BlupDeals/1.0 (+https://getblup.io)"}},send),now);
   for(const offer of offers)await upsertOffer(env.DB,offer,now);
   await env.DB.prepare("UPDATE deals_source SET next_at=?,last_at=?,failures=0,error=NULL,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(now+ROUND,now,id,claim).run();
   if(source.lastAt===null)continue;
   const hunts=await env.DB.prepare("SELECT id,user_id AS userId,kind,filters,channels FROM deals_hunt WHERE source_id=? AND active=1 AND plan_disabled=0 AND (retire_at IS NULL OR retire_at>?) ORDER BY id LIMIT 500").bind(id,now).all();
   const allowed=new Map<string,boolean>();
   for(const raw of hunts.results){
    const hunt=huntRow.safeParse(raw);if(!hunt.success)continue;
    if(!allowed.has(hunt.data.userId))allowed.set(hunt.data.userId,!(await dealsAccess(env,hunt.data.userId)).readOnly);
    if(allowed.get(hunt.data.userId))await feedHunt(env,hunt.data,now,now);
   }
  }catch{
   await env.DB.prepare("UPDATE deals_source SET next_at=?+min(failures+1,6)*600000,failures=failures+1,error='FETCH_FAILED',claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(now,id,claim).run();
  }
 }
}
// Old offers, prices, posts and clicks go after the history window.
export async function dealsCleanup(env:Cloudflare.Env,historyDays:number,now=Date.now()){
 const cutoff=now-historyDays*DAY,day=new Date(cutoff).toISOString().slice(0,10);
 await env.DB.batch([
  env.DB.prepare("DELETE FROM deals_click_day WHERE day<?").bind(day),
  env.DB.prepare("DELETE FROM deals_post WHERE created_at<? AND state<>'queued'").bind(cutoff),
  // A queued post past its window (the dispatcher's EXPIRED) is closed.
  env.DB.prepare("UPDATE deals_post SET state='failed',error='EXPIRED',claim=NULL,claim_until=0 WHERE state='queued' AND day<?").bind(new Date(now-2*DAY).toISOString().slice(0,10)),
  env.DB.prepare("DELETE FROM deals_price WHERE at<?").bind(cutoff),
  env.DB.prepare("DELETE FROM deals_offer WHERE last_seen_at<? AND NOT EXISTS(SELECT 1 FROM deals_post WHERE offer_id=deals_offer.id)").bind(cutoff),
  env.DB.prepare("DELETE FROM deals_telegram_link WHERE expires_at<?").bind(now-DAY),
  env.DB.prepare("DELETE FROM deals_telegram_update WHERE received_at<?").bind(now-DAY),
 ]);
}
