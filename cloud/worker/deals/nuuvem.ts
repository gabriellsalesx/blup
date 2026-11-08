import {z} from "zod";
import {capped,FetchError} from "./fetch";
import {cleanLine,decodeEntities,priceCents} from "./page";
import {NUUVEM_MID} from "./links";
import type {OfferInput} from "./catalog";

// Nuuvem's catalog through Rakuten Advertising's Product Search, with
// Blup's own Rakuten account (DEALS_RAKUTEN_*): one read for everyone,
// a few pages a minute until the whole catalog is read. Each person's post
// gets their own Rakuten ID (links.nuuvemLink); Blup's account only reads.
const API="https://api.linksynergy.com";
export const NUUVEM_PAGE=100;
export const NUUVEM_PAGES_PER_RUN=8;
type Credentials={clientId:string;clientSecret:string;sid:string};
export function rakutenCredentials(env:Cloudflare.Env):Credentials|null{
 const clientId=env.DEALS_RAKUTEN_CLIENT_ID??"",clientSecret=env.DEALS_RAKUTEN_CLIENT_SECRET??"",sid=env.DEALS_RAKUTEN_SID??"";
 return /^[A-Za-z0-9_-]{8,128}$/.test(clientId)&&/^[A-Za-z0-9_-]{8,128}$/.test(clientSecret)&&/^[0-9]{3,12}$/.test(sid)?{clientId,clientSecret,sid}:null;
}
async function call(url:string,init:RequestInit,send:typeof fetch,maxBytes:number){
 let res:Response;
 try{res=await send(url,{...init,redirect:"manual",signal:AbortSignal.timeout(15_000)});}catch{throw new FetchError("FETCH_FAILED");}
 if(!res.ok){await res.body?.cancel().catch(()=>{});throw new FetchError("FETCH_FAILED");}
 return capped(res,maxBytes);
}
const tokenAnswer=z.object({access_token:z.string().regex(/^[A-Za-z0-9._~+/=-]{16,512}$/)}).loose();
// A token for one run (an hour long; a run needs it for seconds).
export async function rakutenToken(credentials:Credentials,send:typeof fetch=fetch){
 const key=btoa(`${credentials.clientId}:${credentials.clientSecret}`);
 const text=await call(`${API}/token`,{method:"POST",headers:{authorization:`Bearer ${key}`,"content-type":"application/x-www-form-urlencoded"},body:new URLSearchParams({grant_type:"client_credentials",scope:credentials.sid}).toString()},send,16*1024);
 let parsed;try{parsed=tokenAnswer.safeParse(JSON.parse(text));}catch{throw new FetchError("FETCH_FAILED");}
 if(!parsed.success)throw new FetchError("FETCH_FAILED");
 return parsed.data.access_token;
}

// Credits and currency packs (FC Points, Dragon Crystals, gift cards)
// share the games' category; their names tell them apart.
const creditWords=/\b(?:points?|pontos|crystals?|cristais|gems?|gemas|diamonds?|diamantes|coins?|moedas|gold|silver|tokens?|cr[eé]ditos?|credits?|v-?bucks|robux|gift ?cards?|cart[aã]o|recarga|top-?up)\b/i;
export const isCredit=(title:string)=>creditWords.test(title)||/\s-\s[0-9][0-9.,]*\s/.test(title);

const tag=(item:string,name:string)=>{const m=new RegExp(`<${name}(?:\\s[^>]*)?>([\\s\\S]{0,4096}?)</${name}>`).exec(item)?.[1];return m===undefined?null:decodeEntities(m.trim());};
const total=(xml:string)=>{const n=Number(/<TotalPages>([0-9]{1,6})<\/TotalPages>/.exec(xml)?.[1]);return Number.isInteger(n)?n:0;};
const httpsOnly=(raw:string|null,host:RegExp)=>{if(!raw)return null;try{const u=new URL(raw);return u.protocol==="https:"&&host.test(u.hostname)?u:null;}catch{return null;}};
// The games of one page that are on sale: the sale price is the price,
// the list price is the "before". The key carries the sale price, so a
// new sale (or a lower one) is a new offer and an unchanged one is not.
export function nuuvemOffers(xml:string,now=Date.now()):OfferInput[]{
 const out:OfferInput[]=[];
 for(const [,item] of xml.matchAll(/<item>([\s\S]{0,65536}?)<\/item>/g)){
  if(item===undefined)continue;
  if(tag(item,"mid")!==NUUVEM_MID)continue;
  const sku=tag(item,"sku")??"",title=cleanLine(tag(item,"productname")??"",200);
  const list=priceCents(tag(item,"price")),sale=priceCents(tag(item,"saleprice"));
  if(!/^[0-9A-Za-z_-]{1,40}$/.test(sku)||!title||list===null||sale===null||sale<=0||sale>=list)continue;
  if(!/currency="BRL"/.test(/<saleprice[^>]*>/.exec(item)?.[0]??""))continue;
  // The store page is the link's murl (the link itself is Blup's).
  const link=httpsOnly(tag(item,"linkurl"),/^click\.linksynergy\.com$/),page=httpsOnly(link?.searchParams.get("murl")??null,/(?:^|\.)nuuvem\.com$/);
  if(!page)continue;
  const image=httpsOnly(tag(item,"imageurl"),/(?:^|\.)nuuvem\.com$/);
  out.push({key:`nuuvem:${sku}:${sale}`,store:"nuuvem",externalId:sku,sourceId:"nuuvem",title,url:page.href,imageUrl:image?.href??null,price:sale,originalPrice:list,currency:"BRL",kind:"nuuvem_deal",platforms:[],discount:Math.round((1-sale/list)*100),credit:null,
   // Seen again on each full read while the sale lasts; gone after.
   expiresAt:now+2*86_400_000});
 }
 return out;
}
// One run's share of the catalog, from page `from`: its sales and the next
// page to read (null once the last page is read).
export async function readNuuvem(credentials:Credentials,from:number,send:typeof fetch=fetch,now=Date.now()){
 const token=await rakutenToken(credentials,send),offers:OfferInput[]=[];let page=from,last=from;
 for(let read=0;read<NUUVEM_PAGES_PER_RUN;read++,page++){
  const xml=await call(`${API}/productsearch/1.0?${new URLSearchParams({mid:NUUVEM_MID,max:String(NUUVEM_PAGE),pagenumber:String(page)})}`,{headers:{authorization:`Bearer ${token}`,accept:"application/xml"}},send,2*1024*1024);
  offers.push(...nuuvemOffers(xml,now));last=total(xml);
  if(page>=last)return {offers,next:null};
 }
 return {offers,next:page};
}
