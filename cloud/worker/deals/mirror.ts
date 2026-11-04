import {z} from "zod";
import {safeFetch,FetchError} from "./fetch";
import {cleanLine,decodeEntities,pageInfo,priceCents} from "./page";
import {convertLink,isShortLink,storeOf,type Store} from "./links";
import {upsertOffer,huntFilters,type HuntFilters} from "./catalog";
import {dealsAccess} from "./access";
import {affiliateOf,enqueue,postedLately,DAY} from "./store";

// Mirroring a public Telegram channel: its public preview (t.me/s/<name>),
// with no login, session or userbot, read once for everyone who mirrors
// it. Each new post with a link of a store Blup knows becomes a post of
// Blup's own (title, price, the person's affiliate link), never a copy of
// the other person's text. A refused read (403, 429, gone) pauses the
// channel with its reason: no proxies, no hidden reading.
export const channelName=z.string().trim().transform(v=>v.replace(/^(?:https?:\/\/)?(?:www\.)?(?:t\.me|telegram\.me)\/(?:s\/)?/i,"").replace(/^@/,"").replace(/[/?#].*$/,"")).pipe(z.string().regex(/^[A-Za-z][A-Za-z0-9_]{3,31}$/));
export const mirrorSourceId=(name:string)=>`telegram:${name.toLowerCase()}`;
const ROUND=10*60_000;
// A post older than this when read is no deal any more (prices move).
const FRESH=3*60*60_000;
const MAX_POSTS=20;

export type ChannelPost={id:number;at:number;text:string;links:string[];photo:string|null};
export type ChannelPage={title:string|null;posts:ChannelPost[]};
const attr=(html:string,re:RegExp)=>{const m=re.exec(html);return m?decodeEntities(m[1]):null;};
const httpsURL=(raw:string|null,host?:RegExp)=>{if(!raw)return null;try{const u=new URL(raw);return u.protocol==="https:"&&(!host||host.test(u.hostname))?u.href:null;}catch{return null;}};
// The preview's own markup (tgme_widget_message): read as text and links
// only, never kept or shown as HTML.
export function readChannel(html:string,name:string):ChannelPage{
 const title=attr(html,/<meta property="og:title" content="([^"]{1,256})"/);
 const posts:ChannelPost[]=[];
 const parts=html.split(/<div class="tgme_widget_message_wrap/).slice(1);
 for(const part of parts){
  const id=Number(new RegExp(`data-post="${name}/([0-9]{1,12})"`,"i").exec(part)?.[1]);if(!Number.isSafeInteger(id))continue;
  const at=Date.parse(attr(part,/<time datetime="([^"]{10,40})"/)??"");
  const body=/<div class="tgme_widget_message_text[^"]*"[^>]*>([\s\S]{0,8192}?)<\/div>/.exec(part)?.[1]??"";
  const text=decodeEntities(body.replace(/<br\s*\/?>/gi,"\n").replace(/<[^>]+>/g,"")).replace(/[\u0000-\u0009\u000b-\u001f\u007f]/g,"").trim();
  const links=[...body.matchAll(/<a [^>]*href="([^"]{1,2048})"/g),...part.matchAll(/class="tgme_widget_message_link_preview" href="([^"]{1,2048})"/g)].map(m=>httpsURL(decodeEntities(m[1]))).filter((v):v is string=>!!v&&!/^https:\/\/t\.me\//.test(v));
  const photo=httpsURL(attr(part,/tgme_widget_message_photo_wrap[^"]*"[^>]*background-image:url\('([^']{1,2048})'\)/),/(?:^|\.)telesco\.pe$/);
  posts.push({id,at:Number.isFinite(at)?at:0,text,links:[...new Set(links)],photo});
 }
 return {title:title?cleanLine(title,120):null,posts};
}
// "de R$ 199,90 por R$ 89,90": the last amount is the price, an earlier
// higher one the price before.
export function textPrices(text:string){
 const amounts=[...text.matchAll(/R\$\s?([0-9]{1,3}(?:\.[0-9]{3})*(?:,[0-9]{1,2})?|[0-9]+(?:,[0-9]{1,2})?)/g)].map(m=>priceCents(m[1])).filter((v):v is number=>v!==null&&v>0);
 const price=amounts.at(-1)??null,before=amounts.length>1?Math.max(...amounts.slice(0,-1)):null;
 return {price,before:before!==null&&price!==null&&before>price?before:null};
}
// The post's first line that says something (a title), without the
// symbols and emoji promotion posts open with.
export function textTitle(text:string){
 for(const line of text.split("\n")){const clean=cleanLine(line.replace(/^[^\p{L}\p{N}]+/u,""),200);if(clean.length>=4&&!/^https?:/i.test(clean))return clean;}
 return null;
}


// Reads the preview (public HTTPS only, through the safe fetch).
export async function fetchChannel(name:string,send:typeof fetch=fetch){
 const page=await safeFetch(`https://t.me/s/${name}`,{send,maxBytes:1024*1024});
 if(page.status!==200&&page.status!==404)return {status:page.status,page:null};
 // A missing or private channel answers with the preview's empty page.
 if(page.status===404||!/tgme_channel_info|tgme_widget_message/.test(page.text))return {status:404 as const,page:null};
 return {status:200 as const,page:readChannel(page.text,name)};
}

type Product={store:Store;url:URL;title:string;image:string|null;price:number|null;before:number|null};
// The store product a post points to: its first link of a known store,
// short links followed safely; title, image and price from the store's
// page when it gives them, else from the post itself.
async function productOf(post:ChannelPost,send:typeof fetch):Promise<Product|null>{
 for(const raw of post.links.slice(0,5)){
  let url:URL;try{url=new URL(raw);}catch{continue;}
  let info={title:null as string|null,image:null as string|null,price:null as number|null,currency:null as string|null};
  if(storeOf(url)==="other"&&!isShortLink(url))continue;
  try{const page=await safeFetch(url.href,{send});url=page.url;if(page.status>=200&&page.status<300&&/html/i.test(page.contentType))info=pageInfo(page.text,page.url);}
  catch(e){if(e instanceof FetchError&&(e.code==="FETCH_BLOCKED"||isShortLink(url)))continue;}
  const store=storeOf(url);if(store==="other")continue;
  const fromText=textPrices(post.text);
  const title=info.title??textTitle(post.text);if(!title)continue;
  const price=info.price!==null&&info.currency==="BRL"?info.price:fromText.price;
  return {store,url,title,image:info.image??post.photo,price,before:fromText.before!==null&&price!==null&&fromText.before>price?fromText.before:null};
 }
 return null;
}
type Hunt={id:string;userId:string;filters:HuntFilters;channels:string[]};
// One product to one hunt: the person's own link (or nothing, if they have
// no ID for that store), once per product unless it got cheaper (more
// than 1%), within the daily cap.
async function deliver(env:Cloudflare.Env,hunt:Hunt,product:Product,offerId:string,send:typeof fetch,now:number){
 if(hunt.filters.stores?.length&&!hunt.filters.stores.includes(product.store))return 0;
 const affiliate=await affiliateOf(env,hunt.userId,product.store);if(!affiliate)return 0;
 const converted=await convertLink(product.url,affiliate,send);if(!converted.affiliate)return 0;
 if(await postedLately(env.DB,hunt.userId,product.store,productKey(product.url),product.price,now))return 0;
 const queued=await enqueue(env.DB,{owner:hunt.userId,offerId,channels:hunt.channels,link:converted.link,affiliate:true,huntId:hunt.id,sendAt:now,now});
 if(queued)await env.DB.prepare("UPDATE deals_hunt SET last_post_at=? WHERE id=?").bind(now,hunt.id).run();
 return queued;
}
// The product's identity across posts and channels: store, host and path
// (no one's tracking in it).
const productKey=(url:URL)=>`${url.hostname.replace(/^www\./,"")}${url.pathname.replace(/\/+$/,"")}`.slice(0,512);
const huntRow=z.object({id:z.string(),userId:z.string(),filters:z.string().transform(v=>huntFilters.catch({}).parse(JSON.parse(v))),channels:z.string().transform(v=>z.array(z.string()).catch([]).parse(JSON.parse(v)))});

// Each run reads the channels whose turn it is (a few a minute), each at
// most every 10 minutes, and only while some active mirror uses it.
export async function collectMirrors(env:Cloudflare.Env,send:typeof fetch=fetch,now=Date.now(),perRun=5){
 const due=await env.DB.prepare("SELECT s.id,s.key,s.cursor,s.last_at AS lastAt FROM deals_source s WHERE s.kind='telegram' AND s.next_at<=? AND s.claim_until<=? AND EXISTS(SELECT 1 FROM deals_hunt h WHERE h.source_id=s.id AND h.kind='mirror' AND h.active=1 AND h.plan_disabled=0 AND (h.retire_at IS NULL OR h.retire_at>?)) ORDER BY s.next_at LIMIT ?").bind(now,now,now,perRun).all<{id:string;key:string;cursor:string|null;lastAt:number|null}>();
 for(const source of due.results){
  const claim=crypto.randomUUID();
  if(!await env.DB.prepare("UPDATE deals_source SET claim=?,claim_until=? WHERE id=? AND claim_until<=? RETURNING id").bind(claim,now+120_000,source.id,now).first())continue;
  const name=source.key.replace(/^telegram:/,"");
  try{
   const read=await fetchChannel(name,send);
   if(!read.page){
    // Gone or refused: wait longer each time, with the reason shown.
    const code=read.status===404?"CHANNEL_GONE":read.status===429?"RATE_LIMIT":"CHANNEL_REFUSED";
    await env.DB.prepare("UPDATE deals_source SET next_at=?+min(failures+1,6)*1800000,failures=failures+1,error=?,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(now,code,source.id,claim).run();continue;
   }
   const last=Number(source.cursor??0),newest=Math.max(last,...read.page.posts.map(p=>p.id));
   // The first read only marks where the channel is (no flood of old posts).
   const fresh=source.lastAt===null?[]:read.page.posts.filter(p=>p.id>last&&now-p.at<=FRESH).sort((a,b)=>a.id-b.id).slice(-MAX_POSTS);
   if(fresh.length){
    const hunts=(await env.DB.prepare("SELECT id,user_id AS userId,filters,channels FROM deals_hunt WHERE source_id=? AND kind='mirror' AND active=1 AND plan_disabled=0 AND (retire_at IS NULL OR retire_at>?) ORDER BY id LIMIT 500").bind(source.id,now).all()).results.flatMap(r=>{const h=huntRow.safeParse(r);return h.success?[h.data]:[];});
    const allowed=new Map<string,boolean>();
    for(const hunt of hunts)if(!allowed.has(hunt.userId))allowed.set(hunt.userId,!(await dealsAccess(env,hunt.userId)).readOnly);
    for(const post of fresh){
     const product=await productOf(post,send);if(!product)continue;
     const offer=await upsertOffer(env.DB,{key:`mirror:${product.store}:${productKey(product.url)}:${product.price??"?"}`,store:product.store,externalId:productKey(product.url),sourceId:source.id,title:product.title,url:product.url.href,imageUrl:product.image,price:product.price,originalPrice:product.before,currency:product.price===null?null:"BRL",kind:"product",platforms:[],discount:product.before&&product.price?Math.round((1-product.price/product.before)*100):null,credit:null,expiresAt:now+DAY},now);
     for(const hunt of hunts)if(allowed.get(hunt.userId))await deliver(env,hunt,product,offer.id,send,now).catch(()=>0);
    }
   }
   await env.DB.prepare("UPDATE deals_source SET next_at=?,last_at=?,cursor=?,config=?,failures=0,error=NULL,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(now+ROUND,now,String(newest),JSON.stringify({title:read.page.title}),source.id,claim).run();
  }catch{
   await env.DB.prepare("UPDATE deals_source SET next_at=?+min(failures+1,6)*600000,failures=failures+1,error='FETCH_FAILED',claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(now,source.id,claim).run();
  }
 }
}
