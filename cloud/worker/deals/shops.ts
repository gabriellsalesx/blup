import {z} from "zod";
import {cleanLine,priceMinor} from "./page";
import {shopeeCall,aliexpressCall,aliexpressHost,shopeeLinkHost,type Affiliate} from "./links";
import {shopeeCurrency,aliexpressCurrency,aliexpressLanguage,currencyDigits,type ShopeeCountry,type AliexpressCountry} from "../../shared/deals-markets";
import {upsertOffer,huntFilters,type HuntFilters,type OfferInput} from "./catalog";
import {dealsAccess} from "./access";
import {affiliateOf,enqueue,postedLately,settingsOf,accountLanguage,DAY} from "./store";
import {curate,describe} from "./ai";
import {money} from "./message";
import {FetchError} from "./fetch";
import {storeTitles} from "../../shared/deals-post";

// A store hunt: Shopee's or AliExpress's own search, with the person's own
// API keys, for the words they chose, sorted by sales, in the money of
// where it sells: Shopee in its account's country, AliExpress in the
// country the hunt picks (market). What it finds on
// sale (the minimum discount, the price limit) goes out with the link the
// store's API made for that person (their commission), the biggest
// discounts first. Each hunt reads for itself (its own deals_source row,
// shop:<hunt id>), every 30 minutes.
const ROUND=30*60_000;
const PER_ROUND=3;
// How many finds the AI looks at each round, for the PER_ROUND that go.
const CANDIDATES=10;
const PAGE=50;
export const shopSourceId=(hunt:string)=>`shop:${hunt}`;
// The store refused the keys (or the account): shown on the hunt.
export class ShopRefused extends Error{constructor(){super("STORE_REFUSED");}}
export type Found={offer:OfferInput;link:string};

const https=(raw:string|null|undefined,host:RegExp)=>{if(!raw)return null;try{const u=new URL(raw);return u.protocol==="https:"&&!u.username&&!u.password&&!u.port&&host.test(u.hostname)?u:null;}catch{return null;}};
const shopeeImageHost=/(?:^|\.)(?:susercontent\.com|shopee\.com\.br)$/;
const shopeeNode=z.object({itemId:z.union([z.number().int().positive(),z.string().regex(/^[0-9]{1,20}$/)]),productName:z.string().max(512),imageUrl:z.string().max(1024).nullish(),productLink:z.string().max(2048),offerLink:z.string().max(2048),priceMin:z.string().max(32),priceDiscountRate:z.number().int().min(0).max(100).nullish()}).loose();
const shopeeAnswer=z.object({data:z.object({productOfferV2:z.object({nodes:z.array(z.unknown()).max(100)})})});
// Shopee's errors carry a code in extensions: 10000 (its own failure) and
// 10030 (the rate limit) pass, so the hunt tries again in a few minutes;
// any other (10020 signature or credential, 10031 to 10035 the account
// without access) is the store refusing the keys.
const shopeeErrors=z.object({errors:z.array(z.unknown()).min(1).max(20)}).loose();
const shopeeCode=z.object({extensions:z.object({code:z.union([z.number(),z.string()])}).loose()}).loose();
const shopeePassing=new Set(["10000","10030"]);
function shopeeFailure(errors:unknown[]):Error{
 const codes=errors.map(e=>{const c=shopeeCode.safeParse(e);return c.success?String(c.data.extensions.code):null;});
 return codes.length&&codes.every(c=>c!==null&&shopeePassing.has(c))?new FetchError("FETCH_FAILED"):new ShopRefused();
}
// Shopee's answer to productOfferV2: the items on sale, in the account
// country's money. Shopee gives the discount but not the price before, so
// the post says the price and the discount.
export function shopeeOffers(value:unknown,now=Date.now(),country:ShopeeCountry="br"):Found[]{
 const currency=shopeeCurrency[country],digits=currencyDigits(currency);
 const parsed=shopeeAnswer.safeParse(value);
 if(!parsed.success){const errors=shopeeErrors.safeParse(value);if(errors.success)throw shopeeFailure(errors.data.errors);return [];}
 return parsed.data.data.productOfferV2.nodes.flatMap(raw=>{
  const n=shopeeNode.safeParse(raw);if(!n.success)return [];
  const title=cleanLine(n.data.productName,200),price=priceMinor(n.data.priceMin,digits),discount=n.data.priceDiscountRate??0;
  const page=https(n.data.productLink,shopeeLinkHost),link=https(n.data.offerLink,shopeeLinkHost);
  if(!title||price===null||price<=0||discount<=0||!page||!link)return [];
  const id=String(n.data.itemId);
  return [{link:link.href,offer:{key:`shopee:${country}:${id}:${price}`,store:"shopee",externalId:id,sourceId:"shopee",title,url:new URL(page.origin+page.pathname).href,imageUrl:https(n.data.imageUrl,shopeeImageHost)?.href??null,price,originalPrice:null,currency,kind:"store_deal",platforms:[],discount,credit:null,expiresAt:now+DAY}}];
 });
}
const aliProduct=z.object({product_id:z.union([z.number().int().positive(),z.string().regex(/^[0-9]{1,24}$/)]),product_title:z.string().max(512),product_main_image_url:z.string().max(1024).nullish(),product_detail_url:z.string().max(2048).nullish(),promotion_link:z.string().max(2048).nullish(),target_sale_price:z.string().max(32).nullish(),target_sale_price_currency:z.string().max(8).nullish(),target_original_price:z.string().max(32).nullish(),discount:z.string().max(8).nullish()}).loose();
const aliAnswer=z.object({aliexpress_affiliate_product_query_response:z.object({resp_result:z.object({resp_code:z.union([z.number(),z.string()]),result:z.object({products:z.union([z.array(z.unknown()),z.object({product:z.array(z.unknown())}).loose()]).optional()}).loose().optional()}).loose()}).loose()}).loose();
// The gateway's error ({type,code,message}, alone or in error_response):
// a call limit or the gateway's own failure (ISP, SYSTEM) passes and the
// hunt tries again; a developer error (ISV: bad keys, signature, app) is
// the store refusing them.
const aliErrorBody=z.object({type:z.string().max(32).optional(),code:z.union([z.string().max(128),z.number()]).optional()}).loose();
const aliError=z.union([z.object({error_response:aliErrorBody}).loose().transform(v=>v.error_response),z.object({type:z.string().max(32),code:z.string().max(128)}).loose()]);
function aliFailure(error:z.infer<typeof aliErrorBody>):Error{
 const type=(error.type??"").toUpperCase(),code=String(error.code??"");
 return /CallLimit|Throttl|Frequency|ServiceUnavailable/i.test(code)||type==="ISP"||type==="SYSTEM"?new FetchError("FETCH_FAILED"):new ShopRefused();
}
// AliExpress's answer to aliexpress.affiliate.product.query, in the
// currency asked (target_currency): the sale price, the price before and the link
// with the person's tracking ID. An error answer (bad keys, a refused
// app) comes as error_response or as the gateway's {type,code,message}.
export function aliexpressOffers(value:unknown,now=Date.now(),currency="BRL"):Found[]{
 const digits=currencyDigits(currency);
 const parsed=aliAnswer.safeParse(value);
 if(!parsed.success){const error=aliError.safeParse(value);if(error.success)throw aliFailure(error.data);return [];}
 const result=parsed.data.aliexpress_affiliate_product_query_response.resp_result;
 if(String(result.resp_code)!=="200")return [];
 const products=result.result?.products,list=Array.isArray(products)?products:products?.product??[];
 return list.slice(0,100).flatMap(raw=>{
  const p=aliProduct.safeParse(raw);if(!p.success||p.data.target_sale_price_currency!==currency)return [];
  const title=cleanLine(p.data.product_title,200),price=priceMinor(p.data.target_sale_price,digits),before=priceMinor(p.data.target_original_price,digits);
  const link=p.data.promotion_link&&aliexpressHost(p.data.promotion_link)?p.data.promotion_link:null;
  if(!title||price===null||price<=0||before===null||before<=price||!link)return [];
  const id=String(p.data.product_id),off=Number(/^([0-9]{1,3})%$/.exec(p.data.discount??"")?.[1]);
  return [{link,offer:{key:`aliexpress:${currency}:${id}:${price}`,store:"aliexpress",externalId:id,sourceId:"aliexpress",title,url:`https://www.aliexpress.com/item/${id}.html`,imageUrl:https(p.data.product_main_image_url,/(?:^|\.)(?:alicdn\.com|aliexpress-media\.com)$/)?.href??null,price,originalPrice:before,currency,kind:"store_deal",platforms:[],discount:Number.isInteger(off)&&off>0&&off<100?off:Math.round((1-price/before)*100),credit:null,expiresAt:now+DAY}}];
 });
}

// One search: the first page of the store's results for the words.
export async function searchShop(affiliate:Affiliate,keyword:string,market:AliexpressCountry,send:typeof fetch=fetch,now=Date.now()):Promise<Found[]>{
 if(affiliate.store==="shopee"){const country=affiliate.country??"br";return shopeeOffers(await shopeeCall(affiliate.appId,affiliate.secret,"query($keyword:String,$limit:Int){productOfferV2(keyword:$keyword,sortType:2,page:1,limit:$limit){nodes{itemId productName imageUrl productLink offerLink priceMin priceDiscountRate}}}",{keyword,limit:PAGE},send,now,country),now,country);}
 if(affiliate.store==="aliexpress"){const currency=aliexpressCurrency(market);return aliexpressOffers(await aliexpressCall(affiliate.appKey,affiliate.appSecret,"aliexpress.affiliate.product.query",{keywords:keyword,page_no:"1",page_size:String(PAGE),sort:"LAST_VOLUME_DESC",target_currency:currency,target_language:aliexpressLanguage(market),ship_to_country:market.toUpperCase(),tracking_id:affiliate.trackingId},send,now),now,currency);}
 return [];
}
export function fitsHunt(filters:HuntFilters,offer:OfferInput){
 if(filters.minDiscount!==undefined&&(offer.discount??0)<filters.minDiscount)return false;
 if(filters.maxPrice!==undefined&&(offer.price===null||offer.price>filters.maxPrice))return false;
 return true;
}
const huntRow=z.object({id:z.string(),userId:z.string(),filters:z.string().transform(v=>huntFilters.catch({}).parse(JSON.parse(v))),channels:z.string().transform(v=>z.array(z.string()).catch([]).parse(JSON.parse(v)))});

// Each run reads the store hunts whose turn it is (a few a minute). A hunt
// whose store keys are gone waits with NOT_CONFIGURED; refused keys with
// STORE_REFUSED, longer each time.
export async function collectShops(env:Cloudflare.Env,send:typeof fetch=fetch,now=Date.now(),perRun=5){
 const due=await env.DB.prepare("SELECT s.id,h.id AS huntId,h.user_id AS userId,h.filters,h.channels FROM deals_source s JOIN deals_hunt h ON h.source_id=s.id AND h.kind='store_deals' AND h.active=1 AND h.plan_disabled=0 AND (h.retire_at IS NULL OR h.retire_at>?) WHERE s.kind IN('shopee','aliexpress') AND s.next_at<=? AND s.claim_until<=? ORDER BY s.next_at LIMIT ?").bind(now,now,now,perRun).all<{id:string;huntId:string;userId:string;filters:string;channels:string}>();
 for(const source of due.results){
  const claim=crypto.randomUUID();
  if(!await env.DB.prepare("UPDATE deals_source SET claim=?,claim_until=? WHERE id=? AND claim_until<=? RETURNING id").bind(claim,now+120_000,source.id,now).first())continue;
  const done=(error:string|null,wait:number)=>env.DB.prepare(`UPDATE deals_source SET next_at=?,${error?"failures=failures+1":"last_at=?,failures=0"},error=?,claim=NULL,claim_until=0 WHERE id=? AND claim=?`).bind(now+wait,...(error?[]:[now]),error,source.id,claim).run();
  const fail=(error:string)=>env.DB.prepare("UPDATE deals_source SET next_at=?+min(failures+1,6)*?,failures=failures+1,error=?,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(now,error==="STORE_REFUSED"?1_800_000:600_000,error,source.id,claim).run();
  try{
   const parsed=huntRow.safeParse({id:source.huntId,userId:source.userId,filters:source.filters,channels:source.channels});
   const hunt=parsed.success?parsed.data:null,store=hunt?.filters.store;
   if(!hunt||!store||!hunt.filters.keywords?.length){await done("INVALID",ROUND);continue;}
   if((await dealsAccess(env,hunt.userId)).readOnly){await done(null,ROUND);continue;}
   const affiliate=await affiliateOf(env,hunt.userId,store);
   if(!affiliate){await done("NOT_CONFIGURED",ROUND);continue;}
   const found=new Map<string,Found>();
   for(const keyword of hunt.filters.keywords)for(const f of await searchShop(affiliate,keyword,hunt.filters.market??"br",send,now))if(fitsHunt(hunt.filters,f.offer)&&!found.has(f.offer.externalId??""))found.set(f.offer.externalId??"",f);
   // The biggest discounts not posted lately; the AI (when asked) gives
   // each a short name and a category and leaves out what is not what the
   // search asked for. Without it (no allowance left), they go as found.
   const fresh:Found[]=[];
   for(const f of [...found.values()].sort((a,b)=>(b.offer.discount??0)-(a.offer.discount??0))){
    if(fresh.length>=CANDIDATES)break;
    if(!await postedLately(env.DB,hunt.userId,f.offer.store,f.offer.externalId??"",f.offer.price,now))fresh.push(f);
   }
   const own=await settingsOf(env.DB,hunt.userId),language=await accountLanguage(env.DB,hunt.userId),cats=hunt.filters.categories;
   const sorted=fresh.length&&(hunt.filters.aiFilter||cats||own.ai.titles)?await curate(env,hunt.userId,hunt.filters.keywords,fresh.map(f=>f.offer.title),language):null;
   let sent=0;
   for(const [i,f] of fresh.entries()){
    if(sent>=PER_ROUND)break;
    const ai=sorted?.[i];
    if(ai&&((hunt.filters.aiFilter&&!ai.keep)||(cats&&!cats.includes(ai.category))))continue;
    const offer=await upsertOffer(env.DB,f.offer,now);
    if(ai)await env.DB.prepare("UPDATE deals_offer SET short_title=coalesce(?,short_title),category=? WHERE id=?").bind(ai.title,ai.category,offer.id).run();
    const words=own.ai.describe?await describe(env,hunt.userId,{title:ai?.title??f.offer.title,store:storeTitles[f.offer.store]??null,price:f.offer.price!==null&&f.offer.currency?money(f.offer.price,f.offer.currency,language):null},language):null;
    const queued=await enqueue(env.DB,{owner:hunt.userId,offerId:offer.id,channels:hunt.channels,link:f.link,affiliate:true,huntId:hunt.id,sendAt:now,now,paced:true,description:words&&"text" in words?words.text:null});
    if(!queued)break;sent++;
   }
   if(sent)await env.DB.prepare("UPDATE deals_hunt SET last_post_at=? WHERE id=?").bind(now,hunt.id).run();
   await done(null,ROUND);
  }catch(e){await fail(e instanceof ShopRefused?"STORE_REFUSED":"FETCH_FAILED");}
 }
}
