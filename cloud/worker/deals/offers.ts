import {Hono} from "hono";
import {z} from "zod";
import {validate} from "../validation";
import {dealsLimits} from "../../shared/pulse-plans";
import {dealsWrite,type DealsApp} from "./access";
import {safeFetch,publicHTTPS,FetchError} from "./fetch";
import {pageInfo} from "./page";
import {convertLink,isShortLink,storeOf,type Converted,type Store} from "./links";
import {upsertOffer} from "./catalog";
import {affiliateOf,enqueue,postsOn,usableChannels,dayOf,DAY} from "./store";

export type Preview={store:Store|"other";title:string;image:string|null;price:number|null;currency:string|null;link:string;affiliate:boolean;reason?:Converted["reason"];url:string};
// Reads the pasted page (safely, following short links), takes what it
// says about the product and puts the person's ID into the link.
export async function previewOffer(env:Cloudflare.Env,owner:string,raw:string,send:typeof fetch=fetch):Promise<Preview>{
 const start=publicHTTPS(raw);let final=start,info={title:null as string|null,image:null as string|null,price:null as number|null,currency:null as string|null};
 try{
  const page=await safeFetch(start.href,{send});final=page.url;
  if(page.status>=200&&page.status<300&&/html/i.test(page.contentType))info=pageInfo(page.text,page.url);
 }catch(e){
  // A blocked hop is never followed; a short link that cannot be opened
  // has no product to convert. A store page that refuses robots is fine.
  if(e instanceof FetchError&&(e.code==="FETCH_BLOCKED"||isShortLink(start)))throw e;
 }
 const store=storeOf(final),converted=await convertLink(final,store==="other"?null:await affiliateOf(env,owner,store),send);
 return {store:converted.store,title:info.title??final.hostname.replace(/^www\./,""),image:info.image,price:info.price,currency:info.currency,link:converted.link,affiliate:converted.affiliate,...(converted.reason?{reason:converted.reason}:{}),url:final.href};
}
const fetchFailure=(e:unknown)=>e instanceof FetchError?e.code:null;
const ids=z.array(z.string().regex(/^[A-Za-z0-9_-]{1,128}$/)).min(1).max(dealsLimits.channels);
type PostRow={id:string;title:string;image:string|null;store:string;link:string;state:string;channelId:string;sentAt:number|null;clicks:number;lastError:string|null};

export const offersApi=new Hono<DealsApp>()
 .post("/preview",dealsWrite,validate("json",z.object({url:z.string().max(2048)}).strict()),async c=>{
  try{
   const p=await previewOffer(c.env,c.get("session").user.id,c.req.valid("json").url);
   return c.json({store:p.store,title:p.title,image:p.image,price:p.price,currency:p.currency,link:p.link,affiliate:p.affiliate,...(p.reason?{reason:p.reason}:{})});
  }catch(e){const code=fetchFailure(e);if(!code)throw e;return code==="FETCH_BLOCKED"?c.json({code},400):c.json({code},502);}
 })
 .post("/",dealsWrite,validate("json",z.object({url:z.string().max(2048),channelIds:ids,sendAt:z.iso.datetime({offset:true}).optional()}).strict()),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json"),now=Date.now();
  const sendAt=input.sendAt?Date.parse(input.sendAt):now;
  if(sendAt<now-60_000||sendAt>now+30*DAY)return c.json({code:"INVALID"},400);
  const channels=await usableChannels(c.env.DB,owner,input.channelIds);
  if(channels.length!==new Set(input.channelIds).size)return c.json({code:"NOT_FOUND"},404);
  if(await postsOn(c.env.DB,owner,dayOf(Math.max(sendAt,now)))>=dealsLimits.postsPerDay)return c.json({code:"DAILY_LIMIT"},429);
  let p:Preview;
  try{p=await previewOffer(c.env,owner,input.url);}catch(e){const code=fetchFailure(e);if(!code)throw e;return code==="FETCH_BLOCKED"?c.json({code},400):c.json({code},502);}
  // The catalog keeps the product (without anyone's ID); each post keeps
  // the person's link.
  const product=new URL(p.url);product.hash="";
  const offer=await upsertOffer(c.env.DB,{key:`link:${p.store}:${product.origin}${product.pathname}`,store:p.store,externalId:null,sourceId:null,title:p.title,url:p.url,imageUrl:p.image,price:p.price,originalPrice:null,currency:p.currency,kind:"product",platforms:[],discount:null,credit:null,expiresAt:null},now);
  const queued=await enqueue(c.env.DB,{owner,offerId:offer.id,channels,link:p.link,affiliate:p.affiliate,huntId:null,sendAt:Math.max(sendAt,now),now});
  if(!queued)return await postsOn(c.env.DB,owner,dayOf(Math.max(sendAt,now)))>=dealsLimits.postsPerDay?c.json({code:"DAILY_LIMIT"},429):c.json({code:"DUPLICATE"},409);
  return c.json({id:offer.id,queued},201);
 })
 .get("/",validate("query",z.object({state:z.enum(["queued","sent","failed"]).optional()})),async c=>{
  const owner=c.get("session").user.id,state=c.req.valid("query").state;
  const rows=await c.env.DB.prepare(`SELECT p.id,o.title,o.image_url AS image,o.store,p.link,p.state,p.channel_id AS channelId,p.sent_at AS sentAt,p.error AS lastError,coalesce((SELECT sum(clicks) FROM deals_click_day WHERE post_id=p.id),0) AS clicks
   FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.user_id=? AND (? IS NULL OR p.state=?) ORDER BY coalesce(p.sent_at,p.next_at) DESC,p.id LIMIT 100`).bind(owner,state??null,state??null).all<PostRow>();
  return c.json({posts:rows.results.map(r=>({id:r.id,title:r.title,image:r.image,store:r.store,link:r.link,state:z.enum(["queued","sent","failed"]).parse(r.state),channelId:r.channelId,sentAt:r.sentAt===null?null:new Date(r.sentAt).toISOString(),clicks:r.clicks,lastError:r.state==="failed"?r.lastError:null}))});
 });
