import {z} from "zod";
import {monitorURL} from "./monitor-http";
const envelope=z.object({v:z.literal(1),iv:z.string().max(32),data:z.string().max(8192)}).strict();
const encode=(v:Uint8Array)=>btoa(String.fromCharCode(...v));
function decode(v:string){return Uint8Array.from(atob(v),c=>c.charCodeAt(0));}
async function key(env:Cloudflare.Env){const bytes=decode(env.PULSE_CHANNEL_KEY??"");if(bytes.length!==32)throw new Error("CHANNEL_UNAVAILABLE");return crypto.subtle.importKey("raw",bytes,{name:"AES-GCM"},false,["encrypt","decrypt"]);}
// The context keeps each product's sealed values apart: a Deals secret
// never opens as a Pulse destination, nor the other way round.
export type SealContext="blup-pulse"|"blup-deals";
const aad=(owner:string,id:string,type:string,context:SealContext)=>new TextEncoder().encode(JSON.stringify([context,1,owner,id,type]));
export async function sealTarget(env:Cloudflare.Env,owner:string,id:string,type:string,value:string,context:SealContext="blup-pulse"){
 const iv=crypto.getRandomValues(new Uint8Array(12)),data=await crypto.subtle.encrypt({name:"AES-GCM",iv,additionalData:aad(owner,id,type,context)},await key(env),new TextEncoder().encode(value));return JSON.stringify({v:1,iv:encode(iv),data:encode(new Uint8Array(data))});
}
export async function openTarget(env:Cloudflare.Env,owner:string,id:string,type:string,value:string,context:SealContext="blup-pulse"){
 const saved=envelope.parse(JSON.parse(value)),iv=decode(saved.iv);if(iv.length!==12)throw new Error("CHANNEL_UNAVAILABLE");
 const bytes=await crypto.subtle.decrypt({name:"AES-GCM",iv,additionalData:aad(owner,id,type,context)},await key(env),decode(saved.data));return new TextDecoder().decode(bytes);
}
export const discordWebhook=z.string().max(2048).transform((raw,ctx)=>{
 try{const u=new URL(raw),match=/^\/api\/(?:v[0-9]+\/)?webhooks\/([0-9]{15,22})\/([A-Za-z0-9._-]{40,200})\/?$/.exec(u.pathname);if(u.protocol!=="https:"||!["discord.com","discordapp.com"].includes(u.hostname)||u.port||u.username||u.password||u.search||u.hash||!match)throw new Error();return {url:`https://discord.com/api/webhooks/${match[1]}/${match[2]}`,label:`Discord · ${match[1].slice(-6)}`};}catch{ctx.addIssue({code:"custom",message:"webhook"});return z.NEVER;}
});
// The kinds of destination; Telegram is linked through the bot, the
// others by an address the person pastes.
export const channelTypes=["discord","telegram","slack","webhook"] as const;
export type ChannelType=(typeof channelTypes)[number];
export const channelType=z.enum(channelTypes);
// A Slack incoming webhook (Slack app "Incoming Webhooks"), only its own
// address shape.
export const slackWebhook=z.string().max(2048).transform((raw,ctx)=>{
 try{const u=new URL(raw),match=/^\/services\/(T[A-Z0-9]{6,20})\/(B[A-Z0-9]{6,20})\/([A-Za-z0-9]{16,64})\/?$/.exec(u.pathname);if(u.protocol!=="https:"||u.hostname!=="hooks.slack.com"||u.port||u.username||u.password||u.search||u.hash||!match)throw new Error();return {url:`https://hooks.slack.com/services/${match[1]}/${match[2]}/${match[3]}`,label:`Slack · ${match[2].slice(-6)}`};}catch{ctx.addIssue({code:"custom",message:"Invalid Slack webhook"});return z.NEVER;}
});
// A webhook of the person's own system: HTTPS on a public address only
// (the same guard as monitors: no private networks, no Blup hosts).
export const customWebhook=z.string().max(2048).transform((raw,ctx)=>{
 try{const u=monitorURL(raw);if(u.protocol!=="https:"||u.username||u.password||u.hash)throw new Error();return {url:u.href,label:`Webhook · ${u.hostname}`.slice(0,80)};}catch{ctx.addIssue({code:"custom",message:"Invalid webhook"});return z.NEVER;}
});
// What a generic webhook keeps sealed: its address and the secret its
// requests are signed with.
export const webhookTarget=z.object({url:z.url().max(2048),secret:z.string().regex(/^[a-f0-9]{64}$/)}).strict();
export const chatIdSchema=z.string().regex(/^[1-9][0-9]{0,15}$/).refine(v=>Number.isSafeInteger(Number(v)));
export function sameSecret(a:string,b:string){const x=new TextEncoder().encode(a),y=new TextEncoder().encode(b);if(!x.length||x.length!==y.length)return false;let difference=0;for(let i=0;i<x.length;i++)difference|=x[i]^y[i];return difference===0;}
