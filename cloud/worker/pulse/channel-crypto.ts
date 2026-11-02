import {z} from "zod";
const envelope=z.object({v:z.literal(1),iv:z.string().max(32),data:z.string().max(8192)}).strict();
const encode=(v:Uint8Array)=>btoa(String.fromCharCode(...v));
function decode(v:string){return Uint8Array.from(atob(v),c=>c.charCodeAt(0));}
async function key(env:Cloudflare.Env){const bytes=decode(env.PULSE_CHANNEL_KEY??"");if(bytes.length!==32)throw new Error("CHANNEL_UNAVAILABLE");return crypto.subtle.importKey("raw",bytes,{name:"AES-GCM"},false,["encrypt","decrypt"]);}
const aad=(owner:string,id:string,type:string)=>new TextEncoder().encode(JSON.stringify(["blup-pulse",1,owner,id,type]));
export async function sealTarget(env:Cloudflare.Env,owner:string,id:string,type:string,value:string){
 const iv=crypto.getRandomValues(new Uint8Array(12)),data=await crypto.subtle.encrypt({name:"AES-GCM",iv,additionalData:aad(owner,id,type)},await key(env),new TextEncoder().encode(value));return JSON.stringify({v:1,iv:encode(iv),data:encode(new Uint8Array(data))});
}
export async function openTarget(env:Cloudflare.Env,owner:string,id:string,type:string,value:string){
 const saved=envelope.parse(JSON.parse(value)),iv=decode(saved.iv);if(iv.length!==12)throw new Error("CHANNEL_UNAVAILABLE");
 const bytes=await crypto.subtle.decrypt({name:"AES-GCM",iv,additionalData:aad(owner,id,type)},await key(env),decode(saved.data));return new TextDecoder().decode(bytes);
}
export const discordWebhook=z.string().max(2048).transform((raw,ctx)=>{
 try{const u=new URL(raw),match=/^\/api\/(?:v[0-9]+\/)?webhooks\/([0-9]{15,22})\/([A-Za-z0-9._-]{40,200})\/?$/.exec(u.pathname);if(u.protocol!=="https:"||!["discord.com","discordapp.com"].includes(u.hostname)||u.port||u.username||u.password||u.search||u.hash||!match)throw new Error();return {url:`https://discord.com/api/webhooks/${match[1]}/${match[2]}`,label:`Discord · ${match[1].slice(-6)}`};}catch{ctx.addIssue({code:"custom",message:"webhook"});return z.NEVER;}
});
export const chatIdSchema=z.string().regex(/^[1-9][0-9]{0,15}$/).refine(v=>Number.isSafeInteger(Number(v)));
export function sameSecret(a:string,b:string){const x=new TextEncoder().encode(a),y=new TextEncoder().encode(b);if(!x.length||x.length!==y.length)return false;let difference=0;for(let i=0;i<x.length;i++)difference|=x[i]^y[i];return difference===0;}
