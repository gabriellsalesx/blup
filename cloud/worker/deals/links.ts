import {z} from "zod";
import {apiJSON,FetchError} from "./fetch";

// The stores Blup knows how to put the person's affiliate ID into. Any
// other store is "other": the link goes as it is, without commission.
export const stores=["amazon","mercadolivre","shopee","aliexpress","nuuvem","magalu","terabyte"] as const;
export const storeSchema=z.enum(stores);
export type Store=z.infer<typeof storeSchema>;

// What the person saves per store (PUT /affiliates/:store). Sealed at
// rest, never sent back.
const id=(max:number)=>z.string().trim().regex(new RegExp(`^[A-Za-z0-9_-]{2,${max}}$`));
export const affiliateBody=z.discriminatedUnion("store",[
 z.object({store:z.literal("amazon"),tag:id(64)}).strict(),
 z.object({store:z.literal("mercadolivre"),word:id(64),tool:z.string().trim().regex(/^[0-9]{1,20}$/).optional()}).strict(),
 z.object({store:z.literal("shopee"),appId:z.string().trim().regex(/^[0-9]{3,20}$/),secret:z.string().trim().regex(/^[A-Za-z0-9]{16,128}$/)}).strict(),
 z.object({store:z.literal("aliexpress"),appKey:z.string().trim().regex(/^[0-9]{3,20}$/),appSecret:z.string().trim().regex(/^[A-Za-z0-9]{16,128}$/),trackingId:id(64)}).strict(),
 // Nuuvem pays through Rakuten Advertising: the person's Rakuten ID (the
 // id= of any of their links, like *AbCdEfGh1).
 z.object({store:z.literal("nuuvem"),siteId:z.string().trim().regex(/^[A-Za-z0-9*/._-]{6,32}$/)}).strict(),
 // Parceiro Magalu: the promoter_id of the person's links and, for their
 // own showcase on Magazine Você, its name (magazine<loja>).
 z.object({store:z.literal("magalu"),promoterId:id(64),showcase:z.string().trim().regex(/^magazine[a-z0-9_-]{1,60}$/i).optional()}).strict(),
 // Terabyte's partner program: the p= of the person's links and, if they
 // use them, the program's UTM medium and campaign.
 z.object({store:z.literal("terabyte"),partner:id(64),utmMedium:id(64).optional(),utmCampaign:id(64).optional()}).strict(),
]);
export type Affiliate=z.infer<typeof affiliateBody>;
// A few characters so the person recognizes what is saved.
export function affiliateHint(a:Affiliate):string{
 const tail=(v:string)=>v.length<=4?"…":`…${v.slice(-4)}`;
 switch(a.store){case "amazon":return tail(a.tag);case "mercadolivre":return tail(a.word);case "shopee":return `App ID ${tail(a.appId)}`;case "aliexpress":return `App Key ${tail(a.appKey)}`;case "nuuvem":return tail(a.siteId);case "magalu":return a.showcase??tail(a.promoterId);case "terabyte":return tail(a.partner);}
}

const amazonHost=/^(?:www\.|smile\.|m\.)?amazon\.(?:com|com\.br|ca|com\.mx|co\.uk|de|es|fr|it|nl|se|pl|co\.jp|in|com\.au|sg|ae|sa|com\.tr|eg|com\.be)$/;
const mlHost=/(?:^|\.)(?:mercadolivre\.com\.br|mercadolibre\.com\.(?:ar|mx|co|uy|pe|ve|ec|bo|py|do|cr|gt|hn|ni|pa|sv)|mercadolibre\.cl)$/;
const shopeeHost=/(?:^|\.)shopee\.(?:com\.br|com\.my|com\.mx|co\.id|co\.th|com\.co|cl|sg|ph|vn|tw)$/;
const aliHost=/(?:^|\.)aliexpress\.(?:com|us|ru)$/;
const nuuvemHost=/(?:^|\.)nuuvem\.com$/;
const magaluHost=/(?:^|\.)(?:magazineluiza\.com\.br|magazinevoce\.com\.br|magalu\.com)$/;
const terabyteHost=/(?:^|\.)terabyteshop\.com\.br$/;
// Short addresses the stores hand out; the preview follows them (safely)
// to the product before anything is converted.
const shortHosts=new Set(["amzn.to","a.co","amzn.eu","meli.la","shope.ee","s.shopee.com.br","s.click.aliexpress.com","a.aliexpress.com","divulgador.magalu.com","magalu.onelink.me"]);
export const isShortLink=(url:URL)=>shortHosts.has(url.hostname);
export function storeOf(url:URL):Store|"other"{
 const host=url.hostname.toLowerCase();
 if(amazonHost.test(host))return "amazon";if(mlHost.test(host))return "mercadolivre";if(shopeeHost.test(host))return "shopee";if(aliHost.test(host))return "aliexpress";if(nuuvemHost.test(host))return "nuuvem";if(magaluHost.test(host))return "magalu";if(terabyteHost.test(host))return "terabyte";return "other";
}

export type Converted={link:string;affiliate:boolean;reason?:"not_configured"|"unsupported_store"|"conversion_failed"};
// Earlier affiliate marks are taken out, so the commission is the person's.
const amazonTracking=["tag","ascsubtag","linkCode","linkId","creative","creativeASIN","camp","ref_","psc","th","smid","pd_rd_r","pd_rd_w","pd_rd_wg","pf_rd_p","pf_rd_r","content-id","sr","qid","keywords","dib","dib_tag","crid","sprefix"];
export function amazonLink(url:URL,tag:string):string{
 const asin=/\/(?:dp|gp\/product|gp\/aw\/d|exec\/obidos\/ASIN)\/([A-Z0-9]{10})(?:[/?]|$)/i.exec(url.pathname)?.[1];
 const host=url.hostname.replace(/^(?:smile|m)\./,"www.");
 if(asin){const out=new URL(`https://${host}/dp/${asin.toUpperCase()}`);out.searchParams.set("tag",tag);return out.href;}
 const out=new URL(url.href);out.hostname=host;out.hash="";for(const k of amazonTracking)out.searchParams.delete(k);out.searchParams.set("tag",tag);return out.href;
}
export function mercadoLivreLink(url:URL,word:string,tool?:string):string{
 const out=new URL(url.href);out.hash="";
 for(const k of [...out.searchParams.keys()])if(/^matt_/.test(k)||k==="tracking_id"||k==="forceInApp")out.searchParams.delete(k);
 out.searchParams.set("matt_word",word);if(tool)out.searchParams.set("matt_tool",tool);return out.href;
}
// A Rakuten Advertising link (someone's, pasted again) carries the store
// page in murl; that page is what gets the person's own ID.
export function rakutenTarget(url:URL):URL|null{
 if(url.hostname.toLowerCase()!=="click.linksynergy.com")return null;
 try{const target=new URL(url.searchParams.get("murl")??"");return target.protocol==="https:"?target:null;}catch{return null;}
}
// Rakuten's deep link: the person's ID, Nuuvem's advertiser number and
// the page, without the tracking someone else added to it.
export const NUUVEM_MID="46796";
export function nuuvemLink(url:URL,siteId:string):string{
 const page=new URL(url.href);page.hash="";
 for(const k of [...page.searchParams.keys()])if(/^(?:utm_|ranmid$|raneaid$|ransiteid$|siteid$)/i.test(k))page.searchParams.delete(k);
 const out=new URL("https://click.linksynergy.com/deeplink");out.searchParams.set("id",siteId);out.searchParams.set("mid",NUUVEM_MID);out.searchParams.set("murl",page.href);return out.href;
}
// Magalu: the person's promoter_id (and showcase) on any Magalu page; a
// Magazine Você page moves to the person's own showcase, keeping the rest
// of the path (the product, the search).
export function magaluLink(url:URL,promoterId:string,showcase?:string):string{
 const out=new URL(url.href);out.hash="";
 for(const k of [...out.searchParams.keys()])if(/^(?:promoter_id|showcase|partner_id|utm_.+)$/i.test(k))out.searchParams.delete(k);
 if(showcase&&/(?:^|\.)magazinevoce\.com\.br$/.test(out.hostname)){const parts=out.pathname.split("/");if(parts[1])parts[1]=showcase;else parts.splice(1,0,showcase);out.pathname=parts.join("/");}
 out.searchParams.set("promoter_id",promoterId);if(showcase)out.searchParams.set("showcase",showcase);return out.href;
}
// Terabyte: the partner's p= (and the program's UTMs, when set), without
// someone else's.
export function terabyteLink(url:URL,partner:string,utmMedium?:string,utmCampaign?:string):string{
 const out=new URL(url.href);out.hash="";
 for(const k of [...out.searchParams.keys()])if(k==="p"||/^utm_/i.test(k))out.searchParams.delete(k);
 out.searchParams.set("p",partner);if(utmMedium)out.searchParams.set("utm_medium",utmMedium);if(utmCampaign)out.searchParams.set("utm_campaign",utmCampaign);return out.href;
}
const hex=(bytes:ArrayBuffer)=>Array.from(new Uint8Array(bytes),b=>b.toString(16).padStart(2,"0")).join("");
// Shopee Affiliate Open API (GraphQL, Brazil): the request is signed with
// SHA256(appId + timestamp + payload + secret).
export async function shopeeCall(appId:string,secret:string,query:string,variables:Record<string,unknown>,send:typeof fetch=fetch,now=Date.now()):Promise<unknown>{
 const timestamp=Math.floor(now/1000),payload=JSON.stringify({query,variables});
 const signature=hex(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(`${appId}${timestamp}${payload}${secret}`)));
 return apiJSON("https://open-api.affiliate.shopee.com.br/graphql",{method:"POST",headers:{"content-type":"application/json",authorization:`SHA256 Credential=${appId}, Timestamp=${timestamp}, Signature=${signature}`},body:payload},send);
}
export async function shopeeLink(url:URL,appId:string,secret:string,send:typeof fetch=fetch,now=Date.now()):Promise<string>{
 const body=await shopeeCall(appId,secret,"mutation($url:String!){generateShortLink(input:{originUrl:$url,subIds:[\"blupdeals\"]}){shortLink}}",{url:new URL(url.origin+url.pathname).href},send,now);
 const parsed=z.object({data:z.object({generateShortLink:z.object({shortLink:z.url().max(512)})})}).safeParse(body);
 if(!parsed.success||!/^https:\/\/(?:s\.)?shope(?:e\.com\.br|\.ee)\//.test(parsed.data.data.generateShortLink.shortLink))throw new FetchError("FETCH_FAILED");
 return parsed.data.data.generateShortLink.shortLink;
}
// AliExpress Open Platform (IOP, /sync): parameters sorted by name, each
// name and value joined, signed with HMAC-SHA256 under the app secret,
// in upper-case hex.
export async function aliexpressSign(params:Record<string,string>,secret:string){
 const text=Object.keys(params).sort().map(k=>`${k}${params[k]}`).join("");
 const key=await crypto.subtle.importKey("raw",new TextEncoder().encode(secret),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
 return hex(await crypto.subtle.sign("HMAC",key,new TextEncoder().encode(text))).toUpperCase();
}
// The answer of aliexpress.affiliate.link.generate. The documented example
// has resp_code as the string "200" and promotion_links as a list; older
// answers wrap the list in {promotion_link:[...]}. Both pass, nothing else.
const promotionLink=z.object({promotion_link:z.url().max(1024)}).loose();
const promotionLinks=z.array(promotionLink).min(1).max(50);
export const aliexpressResponse=z.object({aliexpress_affiliate_link_generate_response:z.object({resp_result:z.object({
 resp_code:z.union([z.literal(200),z.literal("200")]),
 result:z.object({promotion_links:z.union([promotionLinks,z.object({promotion_link:promotionLinks}).loose()])}).loose(),
}).loose()}).loose()}).loose();
// Only AliExpress itself (or a subdomain): a look-alike host is refused.
export function aliexpressHost(raw:string){
 try{const u=new URL(raw),host=u.hostname.toLowerCase();return u.protocol==="https:"&&!u.username&&!u.password&&!u.port&&(host==="aliexpress.com"||host.endsWith(".aliexpress.com"));}catch{return false;}
}
// One signed call to the IOP gateway: the method and its parameters.
export async function aliexpressCall(appKey:string,appSecret:string,method:string,args:Record<string,string>,send:typeof fetch=fetch,now=Date.now()):Promise<unknown>{
 const params:Record<string,string>={app_key:appKey,method,sign_method:"sha256",timestamp:String(now),...args};
 const query=new URLSearchParams({...params,sign:await aliexpressSign(params,appSecret)});
 return apiJSON(`https://api-sg.aliexpress.com/sync?${query}`,{method:"POST"},send);
}
export async function aliexpressLink(url:URL,appKey:string,appSecret:string,trackingId:string,send:typeof fetch=fetch,now=Date.now()):Promise<string>{
 const item=/\/item\/([0-9]{6,24})\.html/.exec(url.pathname)?.[1],source=item?`https://www.aliexpress.com/item/${item}.html`:new URL(url.origin+url.pathname).href;
 const body=await aliexpressCall(appKey,appSecret,"aliexpress.affiliate.link.generate",{promotion_link_type:"0",source_values:source,tracking_id:trackingId},send,now);
 const parsed=aliexpressResponse.safeParse(body);
 const links=parsed.success?parsed.data.aliexpress_affiliate_link_generate_response.resp_result.result.promotion_links:null;
 const link=links?(Array.isArray(links)?links:links.promotion_link)[0].promotion_link:null;
 if(!link||!aliexpressHost(link))throw new FetchError("FETCH_FAILED");
 return link;
}
// The product link with the person's ID, or the link as it is and why not.
export async function convertLink(url:URL,affiliate:Affiliate|null,send:typeof fetch=fetch):Promise<Converted&{store:Store|"other"}>{
 const store=storeOf(url),plain=(reason:Converted["reason"])=>({store,link:url.href,affiliate:false,reason});
 if(store==="other")return plain("unsupported_store");
 if(!affiliate||affiliate.store!==store)return plain("not_configured");
 try{
  switch(affiliate.store){
   case "amazon":return {store,link:amazonLink(url,affiliate.tag),affiliate:true};
   case "mercadolivre":return {store,link:mercadoLivreLink(url,affiliate.word,affiliate.tool),affiliate:true};
   case "shopee":return {store,link:await shopeeLink(url,affiliate.appId,affiliate.secret,send),affiliate:true};
   case "aliexpress":return {store,link:await aliexpressLink(url,affiliate.appKey,affiliate.appSecret,affiliate.trackingId,send),affiliate:true};
   case "nuuvem":return {store,link:nuuvemLink(url,affiliate.siteId),affiliate:true};
   case "magalu":return {store,link:magaluLink(url,affiliate.promoterId,affiliate.showcase),affiliate:true};
   case "terabyte":return {store,link:terabyteLink(url,affiliate.partner,affiliate.utmMedium,affiliate.utmCampaign),affiliate:true};
  }
 }catch{return plain("conversion_failed");}
}
