import {z} from "zod";
import {dealCategories,gameTitle} from "../../shared/deals-post";
import {languageOf,type Language} from "../../shared/language";
import {THEME_SCRIPT} from "../pulse/public-html";
import {sha256} from "../pulse/limits";
import {linksOf} from "../pulse/page-look";
import {dealsAccess} from "./entitlement";
import {siteLogoKey,siteQuery,siteSlug,socialsOf,SITE_PAGE,SITE_DAYS} from "./site-rules";
const since=(now:number)=>now-SITE_DAYS*86_400_000;
import {SitePage,SITE_CSS,SEARCH_SCRIPT,searchResult,type DealCategory,type SiteDeal,type SiteInfo,type SitePageProps} from "./site-page";
import {DetailPage,SHARE_SCRIPT,HISTORY_DAYS,type PricePoint,type SiteDetail} from "./site-detail";

// The public deals pages: deals.getblup.site/<slug>/ and the account's own
// domain (sites-entry.ts), each deal also on its own page (/o/<code>/).
// Built on the server, light, cached:
// - one D1 read finds the page and its newest deal (the version), so a
//   visit that has the page already gets 304 without anything else;
// - a new version renders once and stays in the edge cache;
// - GET and HEAD only, within DEALS_PUBLIC_LIMIT per IP.

type Found={id:string;userId:string;slug:string;title:string;description:string;language:string;logoLight:string|null;logoDark:string|null;showTitle:number;links:string;socials:string;updatedAt:number;latest:number|null};
const found=`SELECT s.id,s.user_id AS userId,s.slug,s.title,s.description,s.language,s.logo_light AS logoLight,s.logo_dark AS logoDark,s.show_title AS showTitle,s.links,s.socials,s.updated_at AS updatedAt,
 (SELECT max(p.sent_at) FROM deals_post p WHERE p.channel_id=s.channel_id AND p.state='sent') AS latest FROM deals_site s`;
const siteBySlug=(env:Cloudflare.Env,slug:string)=>env.DB.prepare(`${found} WHERE s.slug=?`).bind(slug).first<Found>();
const siteByHost=(env:Cloudflare.Env,host:string)=>env.DB.prepare(`${found} JOIN deals_site_domain d ON d.site_id=s.id WHERE d.hostname=? AND d.verified_at IS NOT NULL AND d.status='active' AND d.ssl='active'`).bind(host).first<Found>();

type DealRow={platforms:string;endsAt:number|null;code:string;title:string;image:string|null;price:number|null;originalPrice:number|null;currency:string|null;discount:number|null;store:string;coupon:string|null;description:string|null;category:string|null;sentAt:number;kind:string};
const https=z.url({protocol:/^https$/}).max(2048);
const category=z.enum(dealCategories);

// One page of the deals the site's channel sent in the last SITE_DAYS,
// newest first: the offer's
// short name when it has one, its price only when the post showed it, and
// nothing whose offer ran out.
async function dealsOf(env:Cloudflare.Env,siteId:string,query:z.infer<typeof siteQuery>,now:number){
 const where=["p.channel_id=(SELECT channel_id FROM deals_site WHERE id=?1)","p.state='sent'","(o.expires_at IS NULL OR o.expires_at>?2)","p.sent_at>?3"],binds:(string|number)[]=[siteId,now,since(now)];
 if(query.before){binds.push(query.before);where.push(`p.sent_at<?${binds.length}`);}
 if(query.category){binds.push(query.category);where.push(`o.category=?${binds.length}`);}
 if(query.q){binds.push(`%${query.q.replace(/[\\%_]/g,m=>`\\${m}`)}%`);where.push(`(o.title LIKE ?${binds.length} ESCAPE '\\' OR o.short_title LIKE ?${binds.length} ESCAPE '\\')`);}
 binds.push(SITE_PAGE+1);
 const rows=await env.DB.prepare(`SELECT p.code,coalesce(o.short_title,o.title) AS title,o.image_url AS image,CASE WHEN p.price_shown=1 THEN o.price END AS price,CASE WHEN p.price_shown=1 THEN o.original_price END AS originalPrice,o.currency,o.discount,o.store,p.coupon,p.description,o.category,p.sent_at AS sentAt,o.kind,o.platforms,o.expires_at AS endsAt
  FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE ${where.join(" AND ")} ORDER BY p.sent_at DESC LIMIT ?${binds.length}`).bind(...binds).all<DealRow>();
 const list=rows.results.slice(0,SITE_PAGE).map(toDeal);
 return {deals:list,next:rows.results.length>SITE_PAGE?list.at(-1)?.sentAt??null:null};
}
// The categories the page has deals in, in the panel's order.
async function categoriesOf(env:Cloudflare.Env,siteId:string,now:number):Promise<DealCategory[]>{
 const rows=await env.DB.prepare("SELECT DISTINCT o.category FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.channel_id=(SELECT channel_id FROM deals_site WHERE id=?) AND p.state='sent' AND p.sent_at>? AND (o.expires_at IS NULL OR o.expires_at>?) AND o.category IS NOT NULL LIMIT 32").bind(siteId,since(now),now).all<{category:string}>();
 const present=new Set(rows.results.map(r=>r.category));
 return dealCategories.filter(c=>present.has(c));
}

// One deal of the page, by its short code: what the list shows, whether its
// offer ran out, and the version of its prices (for the page's ETag).
type DetailRow=DealRow&{offerId:string;expiresAt:number|null;priced:number|null};
const detailOf=(env:Cloudflare.Env,siteId:string,code:string,now:number)=>env.DB.prepare(`SELECT p.code,coalesce(o.short_title,o.title) AS title,o.image_url AS image,CASE WHEN p.price_shown=1 THEN o.price END AS price,CASE WHEN p.price_shown=1 THEN o.original_price END AS originalPrice,o.currency,o.discount,o.store,p.coupon,p.description,o.category,p.sent_at AS sentAt,o.kind,o.platforms,o.expires_at AS endsAt,o.id AS offerId,o.expires_at AS expiresAt,(SELECT max(at) FROM deals_price WHERE offer_id=o.id) AS priced
 FROM deals_post p JOIN deals_offer o ON o.id=p.offer_id WHERE p.code=? AND p.channel_id=(SELECT channel_id FROM deals_site WHERE id=?) AND p.state='sent' AND p.sent_at>?`).bind(code,siteId,since(now)).first<DetailRow>();
// The prices of the last days (and the one before them, where the line
// starts), in the offer's currency.
async function pricesOf(env:Cloudflare.Env,offerId:string,currency:string,now:number):Promise<PricePoint[]>{
 const start=now-HISTORY_DAYS*86_400_000;
 const rows=await env.DB.prepare("SELECT at,price FROM deals_price WHERE offer_id=?1 AND currency=?2 AND at>=coalesce((SELECT max(at) FROM deals_price WHERE offer_id=?1 AND currency=?2 AND at<=?3),?3) ORDER BY at LIMIT 400").bind(offerId,currency,start).all<PricePoint>();
 return rows.results;
}
// A list of platform ids, as stored (JSON); anything else is none.
const platformList=z.array(z.string().max(32)).max(16).catch([]);
// What a deal shows: a free game without the giveaway's words in its
// name ("(Epic Games) Giveaway"), as the posts do.
const toDeal=(r:DealRow):SiteDeal=>{
 let platforms:string[]=[];try{platforms=platformList.parse(JSON.parse(r.platforms));}catch{/* none */}
 return {code:r.code,title:r.kind==="game_free"?gameTitle(r.title):r.title,image:https.safeParse(r.image).success?r.image:null,price:r.price,originalPrice:r.originalPrice,currency:r.currency,discount:r.discount,store:r.store,coupon:r.coupon,description:r.description,category:category.safeParse(r.category).data??null,sentAt:r.sentAt,kind:r.kind,platforms,endsAt:r.endsAt};
};

// The page's own code: a new theme or script makes every page new (with
// the version published, below), so no cache keeps the old one.
let themeVersion:Promise<string>|undefined;
const codeVersion=()=>themeVersion??=sha256(`${SITE_CSS}${THEME_SCRIPT}${SEARCH_SCRIPT}${SHARE_SCRIPT}`).then(h=>h.slice(0,12));
const scriptHash=(code:string)=>crypto.subtle.digest("SHA-256",new TextEncoder().encode(code)).then(d=>`'sha256-${btoa(String.fromCharCode(...new Uint8Array(d)))}'`);
let hashes:Promise<string>|undefined;
const scriptHashes=()=>hashes??=Promise.all([scriptHash(THEME_SCRIPT),scriptHash(SEARCH_SCRIPT),scriptHash(SHARE_SCRIPT)]).then(h=>h.join(" "));
const notFound=()=>new Response("Not found",{status:404,headers:{"content-type":"text/plain; charset=utf-8","cache-control":"public, max-age=60"}});

// Answers a request for a deals page, or null when it is not one (the
// caller goes on: another page of the host, or 404).
export async function serveDealsSite(req:Request,env:Cloudflare.Env):Promise<Response|null>{
 const url=new URL(req.url),own=url.hostname!==env.DEALS_PUBLIC_HOST;
 // The path after the page's own part: "/<slug>" on the Deals host.
 let rest=url.pathname,site:Found|null=null;
 if(!own){
  const match=/^\/([^/]+)(\/.*)?$/.exec(url.pathname);const slug=siteSlug.safeParse(match?.[1]);
  if(!slug.success)return null;
  if(match?.[2]===undefined)return new Response(null,{status:301,headers:{location:`${url.origin}/${slug.data}/${url.search}`,"cache-control":"public, max-age=3600"}});
  rest=match[2];
 }
 // A deal's own page: /o/<code>/ (the address without the last slash goes
 // to the one with it).
 const deal=/^\/o\/([A-Za-z0-9]{8})(\/)?$/.exec(rest);
 if(deal&&!deal[2])return new Response(null,{status:301,headers:{location:`${url.origin}${url.pathname}/${url.search}`,"cache-control":"public, max-age=3600"}});
 if(!deal&&rest!=="/"&&rest!=="/search"&&rest!=="/logo-light"&&rest!=="/logo-dark")return null;
 // On another host, only a domain a page has (else the host's other
 // owners answer: a Pulse page, a Blup site).
 site=own?await siteByHost(env,url.hostname):await siteBySlug(env,/^\/([^/]+)/.exec(url.pathname)?.[1]??"");
 if(!site)return own?null:notFound();
 if(!["GET","HEAD"].includes(req.method))return new Response(null,{status:405,headers:{allow:"GET, HEAD","cache-control":"no-store"}});
 if(!(await env.DEALS_PUBLIC_LIMIT.limit({key:`site:${req.headers.get("cf-connecting-ip")??"local"}`})).success)return new Response("Too many requests",{status:429,headers:{"retry-after":"60","cache-control":"no-store"}});

 // The logos: a version in the address keeps them for a year.
 if(rest==="/logo-light"||rest==="/logo-dark"){
  const variant=rest==="/logo-light"?"light":"dark",object=await env.DEALS_ASSETS.get(siteLogoKey(site.id,variant));
  if(!object)return notFound();
  const versioned=url.searchParams.get("v")===(variant==="light"?site.logoLight:site.logoDark);
  return new Response(req.method==="HEAD"?null:object.body,{headers:{"content-type":object.httpMetadata?.contentType??"image/webp","cache-control":versioned?"public, max-age=31536000, immutable":"public, max-age=300","x-content-type-options":"nosniff","content-security-policy":"default-src 'none'; sandbox","cross-origin-resource-policy":"cross-origin"}});
 }

 const query=siteQuery.parse(Object.fromEntries(url.searchParams));
 const lang:Language=query.lang??languageOf(site.language)??"pt";
 // The search dialog's results, as the person types (eight at most).
 if(rest==="/search"){
  const json=(body:unknown,status=200)=>new Response(req.method==="HEAD"?null:JSON.stringify(body),{status,headers:{"content-type":"application/json; charset=utf-8","cache-control":status===200?"public, max-age=60":"no-store","x-content-type-options":"nosniff"}});
  if((await dealsAccess(env,site.userId,Date.now())).readOnly)return json({code:"NOT_FOUND"},404);
  if(!query.q)return json({results:[]});
  const info:SiteInfo={slug:site.slug,title:site.title,description:site.description,language:languageOf(site.language)??"pt",logos:{light:site.logoLight,dark:site.logoDark},showTitle:site.showTitle===1,links:linksOf(site.links),socials:socialsOf(site.socials)};
  const {deals}=await dealsOf(env,site.id,{q:query.q},Date.now());
  return json({results:deals.slice(0,8).map(d=>searchResult({base:own?"":`/${site.slug}`,lang,site:info},d))});
 }
 // The page's version: what it shows changes only with the site or a new
 // deal (and the day, for offers that run out). A visit that has it gets
 // 304 at once.
 const now=Date.now(),day=Math.floor(now/86_400_000);
 // A deal's page changes with its own post and prices too.
 const detail=deal?await detailOf(env,site.id,deal[1]!,now):null;
 if(deal&&!detail)return notFound();
 const etag=`"${(await sha256(`${site.id}:${site.updatedAt}:${site.latest??0}:${day}:${url.pathname}:${url.search}:${env.CF_VERSION_METADATA?.id??""}:${await codeVersion()}:${detail?`${detail.priced??0}:${detail.expiresAt??0}`:""}`)).slice(0,32)}"`;
 const headers:Record<string,string>={
  "content-type":"text/html; charset=utf-8",etag,
  "cache-control":"public, max-age=0, must-revalidate",
  "content-security-policy":`default-src 'none'; style-src 'unsafe-inline'; img-src 'self' https: data:; script-src ${await scriptHashes()}; connect-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'`,
  "x-content-type-options":"nosniff","referrer-policy":"strict-origin-when-cross-origin","vary":"accept-encoding",
 };
 if(req.headers.get("if-none-match")===etag)return new Response(null,{status:304,headers});
 // The same version once rendered: from the edge cache.
 const cache=typeof caches==="undefined"?null:caches.default,key=new Request(`https://deals-site.cache/${site.id}/${etag.slice(1,-1)}`);
 const kept=await cache?.match(key);
 if(kept)return new Response(req.method==="HEAD"?null:kept.body,{headers});
 // Without the Deals plan (or a domain the account lost), no page.
 if((await dealsAccess(env,site.userId,now)).readOnly)return own?null:notFound();
 const info:SiteInfo={slug:site.slug,title:site.title,description:site.description,language:languageOf(site.language)??"pt",logos:{light:site.logoLight,dark:site.logoDark},showTitle:site.showTitle===1,links:linksOf(site.links),socials:socialsOf(site.socials)};
 const shared={now,site:info,lang,base:own?"":`/${site.slug}`,origin:url.origin,links:`https://${env.DEALS_PUBLIC_HOST||"deals.getblup.site"}`};
 let html:string;
 if(detail){
  const full:SiteDetail={...toDeal(detail),expired:detail.expiresAt!==null&&detail.expiresAt<=now,prices:detail.currency?await pricesOf(env,detail.offerId,detail.currency,now):[]};
  html=`<!doctype html>${DetailPage({...shared,deals:[],query:null,category:null,categories:[],next:null,detail:detail.code,deal:full,now})}`;
 }else{
  const [{deals,next},categories]=await Promise.all([dealsOf(env,site.id,query,now),categoriesOf(env,site.id,now)]);
  const props:SitePageProps={...shared,deals,query:query.q??null,category:query.category??null,categories,next};
  html=`<!doctype html>${SitePage(props)}`;
 }
 if(cache)await cache.put(key,new Response(html,{headers:{"content-type":"text/html; charset=utf-8","cache-control":"public, max-age=600"}}));
 return new Response(req.method==="HEAD"?null:html,{headers});
}
