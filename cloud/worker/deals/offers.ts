import {Hono} from "hono";
import {z} from "zod";
import {validate} from "../validation";
import {dealsLimits} from "../../shared/pulse-plans";
import {dealsWrite,type DealsApp} from "./access";
import {safeFetch,publicHTTPS,FetchError} from "./fetch";
import {pageInfo,cleanLine} from "./page";
import {convertLink,isShortLink,storeOf,type Converted,type Store} from "./links";
import {upsertOffer} from "./catalog";
import {affiliateOf,enqueue,postsOn,usableChannels,dayOf,DAY} from "./store";
import {tiktokVideo,VideoError,type Video} from "./video";

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

type Channels=Awaited<ReturnType<typeof usableChannels>>;
// What every post checks before it is queued: a time from now up to 30
// days ahead, channels of the person's own and room in that day's cap.
async function readyToQueue(db:D1Database,owner:string,input:{channelIds:string[];sendAt?:string},now:number):Promise<{channels:Channels;sendAt:number}|{code:"INVALID"|"NOT_FOUND"|"DAILY_LIMIT";status:400|404|429}>{
 const sendAt=input.sendAt?Date.parse(input.sendAt):now;
 if(sendAt<now-60_000||sendAt>now+30*DAY)return {code:"INVALID",status:400};
 const channels=await usableChannels(db,owner,input.channelIds);
 if(channels.length!==new Set(input.channelIds).size)return {code:"NOT_FOUND",status:404};
 const at=Math.max(sendAt,now);
 if(await postsOn(db,owner,dayOf(at))>=dealsLimits.postsPerDay)return {code:"DAILY_LIMIT",status:429};
 return {channels,sendAt:at};
}
async function checkVideo(url:string):Promise<Video|{code:"FETCH_BLOCKED"|"NOT_A_VIDEO"|"VIDEO_NOT_FOUND"|"FETCH_FAILED";status:400|404|502}>{
 try{return await tiktokVideo(url);}
 catch(e){
  if(e instanceof VideoError)return e.code==="NOT_A_VIDEO"?{code:e.code,status:400}:{code:e.code,status:404};
  const code=fetchFailure(e);if(!code)throw e;return code==="FETCH_BLOCKED"?{code,status:400}:{code,status:502};
 }
}

export const offersApi=new Hono<DealsApp>()
 .post("/preview",dealsWrite,validate("json",z.object({url:z.string().max(2048)}).strict()),async c=>{
  try{
   const p=await previewOffer(c.env,c.get("session").user.id,c.req.valid("json").url);
   return c.json({store:p.store,title:p.title,image:p.image,price:p.price,currency:p.currency,link:p.link,affiliate:p.affiliate,...(p.reason?{reason:p.reason}:{})});
  }catch(e){const code=fetchFailure(e);if(!code)throw e;return code==="FETCH_BLOCKED"?c.json({code},400):c.json({code},502);}
 })
 .post("/",dealsWrite,validate("json",z.object({url:z.string().max(2048),channelIds:ids,sendAt:z.iso.datetime({offset:true}).optional()}).strict()),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json"),now=Date.now();
  const ready=await readyToQueue(c.env.DB,owner,input,now);if("code" in ready)return c.json({code:ready.code},ready.status);
  let p:Preview;
  try{p=await previewOffer(c.env,owner,input.url);}catch(e){const code=fetchFailure(e);if(!code)throw e;return code==="FETCH_BLOCKED"?c.json({code},400):c.json({code},502);}
  // The catalog keeps the product (without anyone's ID); each post keeps
  // the person's link.
  const product=new URL(p.url);product.hash="";
  const offer=await upsertOffer(c.env.DB,{key:`link:${p.store}:${product.origin}${product.pathname}`,store:p.store,externalId:null,sourceId:null,title:p.title,url:p.url,imageUrl:p.image,price:p.price,originalPrice:null,currency:p.currency,kind:"product",platforms:[],discount:null,credit:null,expiresAt:null},now);
  const queued=await enqueue(c.env.DB,{owner,offerId:offer.id,channels:ready.channels,link:p.link,affiliate:p.affiliate,huntId:null,sendAt:ready.sendAt,now});
  if(!queued)return await postsOn(c.env.DB,owner,dayOf(ready.sendAt))>=dealsLimits.postsPerDay?c.json({code:"DAILY_LIMIT"},429):c.json({code:"DUPLICATE"},409);
  return c.json({id:offer.id,queued},201);
 })
 // A new TikTok video about a product: checked first, then announced. The
 // person names the product (TikTok tells nothing about it without its
 // API) and, if they want, its price; the click goes to the video.
 .post("/video/preview",dealsWrite,validate("json",z.object({url:z.string().max(2048)}).strict()),async c=>{
  const video=await checkVideo(c.req.valid("json").url);
  return "code" in video?c.json({code:video.code},video.status):c.json({id:video.id,author:video.author,url:video.url});
 })
 .post("/video",dealsWrite,validate("json",z.object({url:z.string().max(2048),title:z.string().trim().min(1).max(200),price:z.number().int().min(0).max(100_000_00).optional(),channelIds:ids,sendAt:z.iso.datetime({offset:true}).optional()}).strict()),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json"),now=Date.now();
  const title=cleanLine(input.title,200);if(!title)return c.json({code:"INVALID"},400);
  const ready=await readyToQueue(c.env.DB,owner,input,now);if("code" in ready)return c.json({code:ready.code},ready.status);
  const video=await checkVideo(input.url);if("code" in video)return c.json({code:video.code},video.status);
  // One announcement per video and account (a second one changes nothing);
  // the catalog keeps the video with the product it shows.
  if(await c.env.DB.prepare("SELECT 1 FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.user_id=? AND o.store='tiktok' AND o.external_id=? LIMIT 1").bind(owner,video.id).first())return c.json({code:"VIDEO_ANNOUNCED"},409);
  const offer=await upsertOffer(c.env.DB,{key:`tiktok:${owner}:${video.id}`,store:"tiktok",externalId:video.id,sourceId:null,title,url:video.url,imageUrl:null,price:input.price??null,originalPrice:null,currency:input.price===undefined?null:"BRL",kind:"video",platforms:[],discount:null,credit:null,expiresAt:null},now);
  const queued=await enqueue(c.env.DB,{owner,offerId:offer.id,channels:ready.channels,link:video.url,affiliate:true,huntId:null,sendAt:ready.sendAt,now});
  if(!queued)return await postsOn(c.env.DB,owner,dayOf(ready.sendAt))>=dealsLimits.postsPerDay?c.json({code:"DAILY_LIMIT"},429):c.json({code:"DUPLICATE"},409);
  return c.json({id:offer.id,queued},201);
 })
 .get("/",validate("query",z.object({state:z.enum(["queued","sent","failed"]).optional()})),async c=>{
  const owner=c.get("session").user.id,state=c.req.valid("query").state;
  const rows=await c.env.DB.prepare(`SELECT p.id,o.title,o.image_url AS image,o.store,p.link,p.state,p.channel_id AS channelId,p.sent_at AS sentAt,p.error AS lastError,coalesce((SELECT sum(clicks) FROM deals_click_day WHERE post_id=p.id),0) AS clicks
   FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.user_id=? AND (? IS NULL OR p.state=?) ORDER BY coalesce(p.sent_at,p.next_at) DESC,p.id LIMIT 100`).bind(owner,state??null,state??null).all<PostRow>();
  return c.json({posts:rows.results.map(r=>({id:r.id,title:r.title,image:r.image,store:r.store,link:r.link,state:z.enum(["queued","sent","failed"]).parse(r.state),channelId:r.channelId,sentAt:r.sentAt===null?null:new Date(r.sentAt).toISOString(),clicks:r.clicks,lastError:r.state==="failed"?r.lastError:null}))});
 });
