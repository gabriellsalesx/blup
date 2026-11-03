import {z} from "zod";
import {discordWebhook,slackWebhook,chatIdSchema} from "./channel-crypto";
import {boundedBody} from "../validation";
import {discordPayload,telegramPayload,slackPayload,webhookPayload,webhookSignature,type AlertMessage} from "./alert-message";
import {webhookTarget} from "./channel-crypto";
import {monitorURL,publicDNS} from "./monitor-http";
export type Delivery={sent:boolean;permanent:boolean;code:string|null;waitMs:number;global:boolean};
const success=(waitMs=0):Delivery=>({sent:true,permanent:false,code:null,waitMs,global:false});
const failure=(code:string,waitMs=15_000,permanent=false,global=false):Delivery=>({sent:false,permanent,code,waitMs,global});
const wait=(value:unknown)=>{if(value===null||value===undefined)return 15_000;const parsed=z.coerce.number().finite().min(0).max(86400).safeParse(value);return parsed.success?Math.max(500,Math.ceil(parsed.data*1000)):15_000;};
async function json(res:Response):Promise<unknown>{const init={method:"POST",body:res.body,duplex:"half"};const bytes=await boundedBody(new Request("https://provider-response.invalid",init),64*1024);try{return JSON.parse(new TextDecoder().decode(bytes)) as unknown;}catch{return null;}}
export async function deliverDiscord(target:string,message:AlertMessage):Promise<Delivery>{
 const parsed=discordWebhook.safeParse(target);if(!parsed.success)return failure("INVALID_TARGET",0,true);
 const url=new URL(parsed.data.url);url.searchParams.set("wait","true");
 try{
  const res=await fetch(url,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(10_000),headers:{"content-type":"application/json"},body:JSON.stringify(discordPayload(message))}),body=await json(res);
  if(res.status===429){const p=z.object({retry_after:z.number().finite().min(0).optional(),global:z.boolean().optional()}).safeParse(body);return failure("RATE_LIMIT",wait(p.success?p.data.retry_after??res.headers.get("retry-after"):res.headers.get("retry-after")),false,p.success&&p.data.global===true);}
  if([401,403,404].includes(res.status))return failure("DESTINATION_DISABLED",0,true);
  if(!res.ok||!z.object({id:z.string().regex(/^[0-9]{1,22}$/)}).safeParse(body).success)return failure("PROVIDER");
  const cooldown=res.headers.get("x-ratelimit-remaining")==="0"?wait(res.headers.get("x-ratelimit-reset-after")):0;return success(cooldown);
 }catch{return failure("PROVIDER");}
}
export async function deliverTelegram(env:Cloudflare.Env,target:string,message:AlertMessage):Promise<Delivery>{
 const parsed=chatIdSchema.safeParse(target);if(!parsed.success)return failure("INVALID_TARGET",0,true);
 if(!/^[0-9]+:[A-Za-z0-9_-]{20,200}$/.test(env.PULSE_TELEGRAM_TOKEN??""))return failure("CHANNEL_UNAVAILABLE",300_000);
 try{
  const res=await fetch(`https://api.telegram.org/bot${env.PULSE_TELEGRAM_TOKEN}/sendMessage`,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(10_000),headers:{"content-type":"application/json"},body:JSON.stringify({chat_id:parsed.data,...telegramPayload(message),link_preview_options:{is_disabled:true}})}),body=await json(res);
  const result=z.discriminatedUnion("ok",[z.object({ok:z.literal(true),result:z.object({message_id:z.number().int().nonnegative()})}),z.object({ok:z.literal(false),error_code:z.number().int(),parameters:z.object({retry_after:z.number().int().nonnegative().optional()}).optional()})]).safeParse(body);
  if(!result.success)return failure("PROVIDER");if(result.data.ok)return res.ok?success(1000):failure("PROVIDER");
  if(result.data.error_code===429)return failure("RATE_LIMIT",wait(result.data.parameters?.retry_after??res.headers.get("retry-after")),false,true);
  if([400,403,404].includes(result.data.error_code))return failure("DESTINATION_DISABLED",0,true);
  return failure("PROVIDER");
 }catch{return failure("PROVIDER");}
}
// Slack answers "ok" (200); a removed hook, a revoked app or an archived
// channel stop the destination; too many requests wait.
export async function deliverSlack(target:string,message:AlertMessage):Promise<Delivery>{
 const parsed=slackWebhook.safeParse(target);if(!parsed.success)return failure("INVALID_TARGET",0,true);
 try{
  const res=await fetch(parsed.data.url,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(10_000),headers:{"content-type":"application/json"},body:JSON.stringify(slackPayload(message))});await res.body?.cancel();
  if(res.status===429)return failure("RATE_LIMIT",wait(res.headers.get("retry-after")));
  if([403,404,410].includes(res.status))return failure("DESTINATION_DISABLED",0,true);
  return res.ok?success():failure("PROVIDER");
 }catch{return failure("PROVIDER");}
}
// The person's own system: signed JSON to a public HTTPS address, checked
// again before each request (no private networks, no redirects followed).
// 2xx is delivered; 410 stops the destination; 429 waits; the rest retry.
export async function deliverWebhook(target:string,message:AlertMessage,now=Date.now()):Promise<Delivery>{
 let parsed:ReturnType<typeof webhookTarget.parse>;try{parsed=webhookTarget.parse(JSON.parse(target));}catch{return failure("INVALID_TARGET",0,true);}
 const signal=AbortSignal.timeout(10_000);
 try{
  const url=monitorURL(parsed.url);await publicDNS(url.hostname,signal);
  const body=JSON.stringify(webhookPayload(message,now));
  const res=await fetch(url,{method:"POST",redirect:"manual",signal,headers:{"content-type":"application/json","user-agent":"BlupPulse/1.0 (+https://getblup.io)","x-blup-event":message.event,"x-blup-signature":await webhookSignature(parsed.secret,body,now)},body});await res.body?.cancel();
  if(res.status===429)return failure("RATE_LIMIT",wait(res.headers.get("retry-after")));
  if(res.status===410)return failure("DESTINATION_DISABLED",0,true);
  return res.ok?success():failure("PROVIDER");
 }catch(e){return e instanceof Error&&e.message==="INVALID_TARGET"?failure("INVALID_TARGET",0,true):failure("PROVIDER");}
}

