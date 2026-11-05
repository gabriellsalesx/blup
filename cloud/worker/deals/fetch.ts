import {monitorURL,publicDNS} from "../pulse/monitor-http";

// Reading a page the person pasted: HTTPS on a public address only (the
// monitors' guard: no private networks, no Blup hosts), checked again on
// every redirect, which is followed by hand. A timeout for the whole read
// and a cap on the bytes kept. global_fetch_strictly_public is the real
// boundary in production; these checks are defense in depth.
export class FetchError extends Error{constructor(readonly code:"FETCH_BLOCKED"|"FETCH_FAILED"){super(code);}}
export function publicHTTPS(raw:string):URL{
 let url:URL;try{url=monitorURL(raw.trim());}catch{throw new FetchError("FETCH_BLOCKED");}
 if(url.protocol!=="https:")throw new FetchError("FETCH_BLOCKED");
 return url;
}
export type Fetched={url:URL;status:number;contentType:string;text:string};
// until stops at the first address it accepts, without opening it (a
// short link followed only as far as where it points). read takes the
// body itself (an image's bytes) instead of the text.
type Options={maxBytes?:number;timeoutMs?:number;hops?:number;body?:boolean;until?:(url:URL)=>boolean;send?:typeof fetch;accept?:string};
export async function safeFetch(raw:string,options:Options={}):Promise<Fetched>{
 const got=await safeResponse(raw,options);
 if(!got.res)return {...got,text:""};
 return {url:got.url,status:got.status,contentType:got.contentType,text:new TextDecoder().decode((await cappedBytes(got.res,options.maxBytes??512*1024)).bytes)};
}
// An image a post shows (WhatsApp wants the file itself): the same guard,
// at most maxBytes; null when it is bigger, not an image or not there.
export async function safeImage(raw:string,{maxBytes=5*1024*1024,send=fetch}:{maxBytes?:number;send?:typeof fetch}={}):Promise<{bytes:Uint8Array;contentType:string}|null>{
 try{
  const got=await safeResponse(raw,{send,accept:"image/jpeg,image/png;q=0.9,image/*;q=0.5",timeoutMs:10_000});
  if(!got.res)return null;
  if(got.status!==200||!got.contentType.startsWith("image/")){await got.res.body?.cancel().catch(()=>{});return null;}
  const {bytes,full}=await cappedBytes(got.res,maxBytes);
  return full&&bytes.byteLength?{bytes,contentType:got.contentType}:null;
 }catch{return null;}
}
async function safeResponse(raw:string,{timeoutMs=8_000,hops=5,body=true,until,send=fetch,accept="text/html,application/xhtml+xml;q=0.9,*/*;q=0.5"}:Options):Promise<{url:URL;status:number;contentType:string;res:Response|null}>{
 let url=publicHTTPS(raw);const signal=AbortSignal.timeout(timeoutMs);
 for(let hop=0;hop<=hops;hop++){
  if(hop>0&&until?.(url))return {url,status:0,contentType:"",res:null};
  try{await publicDNS(url.hostname,signal,send);}catch{throw new FetchError("FETCH_BLOCKED");}
  let res:Response;
  try{res=await send(url,{redirect:"manual",signal,headers:{"user-agent":"Mozilla/5.0 (compatible; BlupDeals/1.0; +https://getblup.io)",accept,"accept-language":"pt-BR,pt;q=0.9,en;q=0.8"}});}
  catch{throw new FetchError("FETCH_FAILED");}
  if(res.status>=300&&res.status<400){
   const location=res.headers.get("location");await res.body?.cancel().catch(()=>{});
   if(!location)throw new FetchError("FETCH_FAILED");
   let next:URL;try{next=new URL(location,url);}catch{throw new FetchError("FETCH_FAILED");}
   url=publicHTTPS(next.href);continue;
  }
  const contentType=res.headers.get("content-type")??"";
  if(!body){await res.body?.cancel().catch(()=>{});return {url,status:res.status,contentType,res:null};}
  return {url,status:res.status,contentType,res};
 }
 throw new FetchError("FETCH_FAILED");
}
// The first maxBytes of the body as text; the rest is let go.
export async function capped(res:Response,maxBytes:number):Promise<string>{
 return new TextDecoder().decode((await cappedBytes(res,maxBytes)).bytes);
}
// The first maxBytes of the body; full says nothing was left out.
async function cappedBytes(res:Response,maxBytes:number):Promise<{bytes:Uint8Array;full:boolean}>{
 const reader=res.body?.getReader();if(!reader)return {bytes:new Uint8Array(),full:true};
 let full=false;
 const chunks:Uint8Array[]=[];let size=0;
 try{while(size<=maxBytes){const {done,value}=await reader.read();if(done){full=true;break;}chunks.push(value);size+=value.byteLength;}}
 catch{throw new FetchError("FETCH_FAILED");}
 finally{await reader.cancel().catch(()=>{});}
 const bytes=new Uint8Array(Math.min(size,maxBytes));let at=0;
 for(const c of chunks){const part=c.subarray(0,bytes.length-at);bytes.set(part,at);at+=part.byteLength;if(at>=bytes.length)break;}
 return {bytes,full:full&&size<=maxBytes};
}
// A fixed public API (GamerPower, CheapShark, the stores' affiliate APIs):
// no redirects, a timeout and a cap, JSON out.
export async function apiJSON(url:string,init:RequestInit={},send:typeof fetch=fetch,maxBytes=2*1024*1024):Promise<unknown>{
 let res:Response;
 try{res=await send(url,{...init,redirect:"manual",signal:AbortSignal.timeout(10_000)});}catch{throw new FetchError("FETCH_FAILED");}
 if(!res.ok){await res.body?.cancel().catch(()=>{});throw new FetchError("FETCH_FAILED");}
 try{return JSON.parse(await capped(res,maxBytes)) as unknown;}catch(e){if(e instanceof FetchError)throw e;throw new FetchError("FETCH_FAILED");}
}
