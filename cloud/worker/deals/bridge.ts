import {z} from "zod";
import {failure,success,type Delivery} from "../pulse/transports";
import {safeImage} from "./fetch";

// The WhatsApp Bridge (cloud/bridge), seen from here: the signed requests
// to it, a post to a Canal and forgetting a connection. Apart from the
// routes (whatsapp.ts), so the queue and the purge can use it.
export const WHATSAPP_INTERVAL=30*60_000;
export const MIN_SECRET=32;
export const bridgeReady=(env:Cloudflare.Env)=>{
 const url=URL.parse(env.DEALS_BRIDGE_URL??"");
 return !!url&&(url.protocol==="https:"||url.hostname==="127.0.0.1")&&(env.DEALS_BRIDGE_SECRET??"").length>=MIN_SECRET;
};

const hex=(b:ArrayBuffer|Uint8Array)=>[...new Uint8Array(b)].map(x=>x.toString(16).padStart(2,"0")).join("");
async function hmac(secret:string,text:string){
 const key=await crypto.subtle.importKey("raw",new TextEncoder().encode(secret),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
 return hex(await crypto.subtle.sign("HMAC",key,new TextEncoder().encode(text)));
}
const bodyHash=async(body:string)=>hex(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(body)));
// The signed text, the Bridge's: time, nonce, method, path and the body's
// SHA-256.
export async function signature(secret:string,at:string,nonce:string,method:string,path:string,body:string){
 return hmac(secret,`${at}\n${nonce}\n${method}\n${path}\n${await bodyHash(body)}`);
}

export type Answer={status:number;data:unknown};
export async function bridge(env:Cloudflare.Env,method:"GET"|"POST"|"DELETE",path:string,body?:object,send:typeof fetch=fetch):Promise<Answer>{
 const base=new URL(env.DEALS_BRIDGE_URL),url=new URL(path,base),raw=body===undefined?"":JSON.stringify(body);
 const at=String(Math.floor(Date.now()/1000)),nonce=hex(crypto.getRandomValues(new Uint8Array(16)));
 const res=await send(url,{method,redirect:"manual",signal:AbortSignal.timeout(60_000),headers:{
  "content-type":"application/json","x-bridge-timestamp":at,"x-bridge-nonce":nonce,
  "x-bridge-signature":await signature(env.DEALS_BRIDGE_SECRET,at,nonce,method,url.pathname+url.search,raw),
 },...(body===undefined?{}:{body:raw})});
 return {status:res.status,data:res.status===204?null:await res.json().catch(()=>null)};
}

// A post to a Canal: the image (the store's, fetched here behind the same
// guard as a pasted page; JPEG or PNG only, else the text alone) and the
// caption go to the Bridge, which posts at most once per key (the post's
// id) and once every 30 minutes per Canal. A photo's own bytes (an
// example's picture) go as they are.
export async function sendWhatsapp(env:Cloudflare.Env,session:string,channel:string,key:string,text:string,image:string|null,send:typeof fetch=fetch,photo:Uint8Array|null=null):Promise<Delivery>{
// BRIDGE_UNAVAILABLE: the Bridge is not set up or did not answer (the
// connection is fine, nothing to reconnect: the WHATSAPP_ codes say that).
 if(!bridgeReady(env))return failure("BRIDGE_UNAVAILABLE",300_000);
 let picture:string|null=photo&&isJpegOrPng(photo)?base64(photo):null;
 if(image&&!picture){
  const got=await safeImage(image,{send});
  if(got&&isJpegOrPng(got.bytes))picture=base64(got.bytes);
 }
 let answer:Answer;
 try{answer=await bridge(env,"POST",`/v1/sessions/${session}/posts`,{channel,key:`deals:${key}`,text,...(picture?{image:picture}:{})},send);}catch{return failure("BRIDGE_UNAVAILABLE",60_000);}
 if(answer.status===200)return success(0);
 if(answer.status===429){const p=z.object({retryAfter:z.number().int().min(1).max(86_400)}).safeParse(answer.data);return failure("TOO_SOON",(p.success?p.data.retryAfter:60)*1000);}
 if(answer.status===403)return failure("DESTINATION_DISABLED",0,true);
 if(answer.status===404||answer.status===409)return failure("WHATSAPP_DISCONNECTED",300_000);
 return failure("PROVIDER",60_000);
}
const isJpegOrPng=(b:Uint8Array)=>(b[0]===0xff&&b[1]===0xd8&&b[2]===0xff)||(b[0]===0x89&&b[1]===0x50&&b[2]===0x4e&&b[3]===0x47);
function base64(bytes:Uint8Array){let s="";for(let i=0;i<bytes.length;i+=0x8000)s+=String.fromCharCode(...bytes.subarray(i,i+0x8000));return btoa(s);}

// Forgetting the account's WhatsApp at the Bridge (the account is purged).
export async function forgetWhatsapp(env:Cloudflare.Env,owner:string){
 const row=await env.DB.prepare("SELECT id FROM deals_whatsapp WHERE user_id=?").bind(owner).first<{id:string}>();
 if(row&&bridgeReady(env)){try{await bridge(env,"DELETE",`/v1/sessions/${row.id}`);}catch{}}
}
