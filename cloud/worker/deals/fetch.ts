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
type Options={maxBytes?:number;timeoutMs?:number;hops?:number;body?:boolean;send?:typeof fetch};
export async function safeFetch(raw:string,{maxBytes=512*1024,timeoutMs=8_000,hops=5,body=true,send=fetch}:Options={}):Promise<Fetched>{
 let url=publicHTTPS(raw);const signal=AbortSignal.timeout(timeoutMs);
 for(let hop=0;hop<=hops;hop++){
  try{await publicDNS(url.hostname,signal,send);}catch{throw new FetchError("FETCH_BLOCKED");}
  let res:Response;
  try{res=await send(url,{redirect:"manual",signal,headers:{"user-agent":"Mozilla/5.0 (compatible; BlupDeals/1.0; +https://getblup.io)",accept:"text/html,application/xhtml+xml;q=0.9,*/*;q=0.5","accept-language":"pt-BR,pt;q=0.9,en;q=0.8"}});}
  catch{throw new FetchError("FETCH_FAILED");}
  if(res.status>=300&&res.status<400){
   const location=res.headers.get("location");await res.body?.cancel().catch(()=>{});
   if(!location)throw new FetchError("FETCH_FAILED");
   let next:URL;try{next=new URL(location,url);}catch{throw new FetchError("FETCH_FAILED");}
   url=publicHTTPS(next.href);continue;
  }
  const contentType=res.headers.get("content-type")??"";
  if(!body){await res.body?.cancel().catch(()=>{});return {url,status:res.status,contentType,text:""};}
  return {url,status:res.status,contentType,text:await capped(res,maxBytes)};
 }
 throw new FetchError("FETCH_FAILED");
}
// The first maxBytes of the body as text; the rest is let go.
export async function capped(res:Response,maxBytes:number):Promise<string>{
 const reader=res.body?.getReader();if(!reader)return "";
 const chunks:Uint8Array[]=[];let size=0;
 try{while(size<maxBytes){const {done,value}=await reader.read();if(done)break;chunks.push(value);size+=value.byteLength;}}
 catch{throw new FetchError("FETCH_FAILED");}
 finally{await reader.cancel().catch(()=>{});}
 const bytes=new Uint8Array(Math.min(size,maxBytes));let at=0;
 for(const c of chunks){const part=c.subarray(0,bytes.length-at);bytes.set(part,at);at+=part.byteLength;if(at>=bytes.length)break;}
 return new TextDecoder().decode(bytes);
}
// A fixed public API (GamerPower, CheapShark, the stores' affiliate APIs):
// no redirects, a timeout and a cap, JSON out.
export async function apiJSON(url:string,init:RequestInit={},send:typeof fetch=fetch,maxBytes=2*1024*1024):Promise<unknown>{
 let res:Response;
 try{res=await send(url,{...init,redirect:"manual",signal:AbortSignal.timeout(10_000)});}catch{throw new FetchError("FETCH_FAILED");}
 if(!res.ok){await res.body?.cancel().catch(()=>{});throw new FetchError("FETCH_FAILED");}
 try{return JSON.parse(await capped(res,maxBytes)) as unknown;}catch(e){if(e instanceof FetchError)throw e;throw new FetchError("FETCH_FAILED");}
}
