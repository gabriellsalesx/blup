import {z} from "zod";
import {languageTag,type Language} from "../../shared/language";
import {success,failure,wait,json,type Delivery} from "../pulse/transports";
import {discordWebhook} from "../pulse/channel-crypto";

// One post: the offer as the channel sees it. link is the short link;
// watch, for a video, is the video itself (Telegram shows it above).
export type Post={title:string;image:string|null;price:number|null;originalPrice:number|null;currency:string|null;discount:number|null;link:string;affiliate:boolean;kind:string;credit:string|null;watch?:string|null};
const words={
 pt:{free:"Grátis",open:"Ver oferta",video:"Vídeo novo",watch:"Ver o vídeo",videoAd:"#publi",ad:"#publi · link de afiliado",off:"de desconto",test:"Teste do Blup Deals: este canal está pronto para receber ofertas."},
 en:{free:"Free",open:"See the deal",video:"New video",watch:"Watch the video",videoAd:"#ad",ad:"#ad · affiliate link",off:"off",test:"Blup Deals test: this channel is ready to get deals."},
 es:{free:"Gratis",open:"Ver oferta",video:"Video nuevo",watch:"Ver el video",videoAd:"#publi",ad:"#publi · enlace de afiliado",off:"de descuento",test:"Prueba de Blup Deals: este canal está listo para recibir ofertas."},
} as const;
export const testText=(language:Language)=>words[language].test;
// The source a post must credit, as an active link (GamerPower's terms ask
// for a clickable attribution; CheapShark gets the same).
export const credits:Record<string,{label:string;url:string}>={gamerpower:{label:"Via GamerPower",url:"https://www.gamerpower.com/"},cheapshark:{label:"Via CheapShark",url:"https://www.cheapshark.com/"}};
const creditOf=(p:Post)=>p.credit&&Object.hasOwn(credits,p.credit)?credits[p.credit]:null;
export function money(cents:number,currency:string,language:Language){
 try{return new Intl.NumberFormat(languageTag[language],{style:"currency",currency}).format(cents/100);}catch{return `${(cents/100).toFixed(2)} ${currency}`;}
}
// The price line: "Grátis" for a giveaway, else the old price and the new.
function priceLine(p:Post,language:Language,strike:(s:string)=>string){
 const w=words[language];
 if(p.kind==="game_free")return p.originalPrice&&p.currency?`${strike(money(p.originalPrice,p.currency,language))} ${w.free}`:w.free;
 if(p.price===null||!p.currency)return null;
 const now=money(p.price,p.currency,language),before=p.originalPrice&&p.originalPrice>p.price?`${strike(money(p.originalPrice,p.currency,language))} `:"";
 return `${before}${now}${p.discount&&p.discount>0?` (${p.discount}% ${w.off})`:""}`;
}
const escapeHTML=(s:string)=>s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;");
// What a post says, in either platform's markup: a video's news ("New
// video: the product"), the link's words and the notice.
function parts(p:Post,language:Language){
 const w=words[language],video=p.kind==="video";
 return {title:video?`${w.video}: ${p.title}`:p.title,open:video?w.watch:w.open,ad:p.affiliate?video?w.videoAd:w.ad:null};
}
// Telegram (HTML): at most 1024 characters with a photo.
export function telegramText(p:Post,language:Language){
 const {title,open,ad}=parts(p,language),price=priceLine(p,language,s=>`<s>${escapeHTML(s)}</s>`);
 const lines=[`<b>${escapeHTML([...title].slice(0,300).join(""))}</b>`,...(price?[price]:[]),`<a href="${escapeHTML(p.link)}">${open}</a>`,...((c=>c?[`<a href="${c.url}">${c.label}</a>`]:[])(creditOf(p))),...(ad?[ad]:[])];
 return lines.join("\n");
}
export function discordPayload(p:Post,language:Language){
 const {title,open,ad}=parts(p,language),price=priceLine(p,language,s=>`~~${s}~~`),credit=creditOf(p);
 return {username:"Blup Deals",allowed_mentions:{parse:[]},embeds:[{title:[...title].slice(0,250).join(""),url:p.link,color:0xc2410c,description:[...(price?[price]:[]),`[${open}](${p.link})`,...(credit?[`[${credit.label}](${credit.url})`]:[]),...(ad?[ad]:[])].join("\n"),...(p.image?{image:{url:p.image}}:{})}]};
}

export const dealsChatId=z.string().regex(/^-?[1-9][0-9]{0,15}$/).refine(v=>Number.isSafeInteger(Number(v)));
const botToken=(env:Cloudflare.Env)=>/^[0-9]+:[A-Za-z0-9_-]{20,200}$/.test(env.DEALS_TELEGRAM_TOKEN??"")?env.DEALS_TELEGRAM_TOKEN:null;
const telegramResult=z.discriminatedUnion("ok",[z.object({ok:z.literal(true),result:z.object({message_id:z.number().int().nonnegative()})}),z.object({ok:z.literal(false),error_code:z.number().int(),parameters:z.object({retry_after:z.number().int().nonnegative().optional()}).optional()})]);
async function telegramCall(token:string,method:string,body:object,send:typeof fetch){
 const res=await send(`https://api.telegram.org/bot${token}/${method}`,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(10_000),headers:{"content-type":"application/json"},body:JSON.stringify(body)});
 return {res,result:telegramResult.safeParse(await json(res))};
}
// A photo with the caption; when Telegram refuses the photo (a store
// image it cannot fetch), the text alone. A video goes as text with the
// video's own preview above it (Telegram reads TikTok's page itself).
export async function sendTelegram(env:Cloudflare.Env,target:string,text:string,image:string|null,send:typeof fetch=fetch,watch:string|null=null):Promise<Delivery>{
 const chat=dealsChatId.safeParse(target);if(!chat.success)return failure("INVALID_TARGET",0,true);
 const token=botToken(env);if(!token)return failure("CHANNEL_UNAVAILABLE",300_000);
 try{
  const preview=watch?{url:watch,prefer_large_media:true,show_above_text:true}:{is_disabled:true};
  let {res,result}=image?await telegramCall(token,"sendPhoto",{chat_id:chat.data,photo:image,caption:text,parse_mode:"HTML"},send):await telegramCall(token,"sendMessage",{chat_id:chat.data,text,parse_mode:"HTML",link_preview_options:preview},send);
  if(image&&result.success&&!result.data.ok&&result.data.error_code===400)({res,result}=await telegramCall(token,"sendMessage",{chat_id:chat.data,text,parse_mode:"HTML",link_preview_options:{is_disabled:true}},send));
  if(!result.success)return failure("PROVIDER");if(result.data.ok)return res.ok?success(1000):failure("PROVIDER");
  if(result.data.error_code===429)return failure("RATE_LIMIT",wait(result.data.parameters?.retry_after??res.headers.get("retry-after")),false,true);
  if([400,403,404].includes(result.data.error_code))return failure("DESTINATION_DISABLED",0,true);
  return failure("PROVIDER");
 }catch{return failure("PROVIDER");}
}
export async function sendDiscord(target:string,payload:object,send:typeof fetch=fetch):Promise<Delivery>{
 const parsed=discordWebhook.safeParse(target);if(!parsed.success)return failure("INVALID_TARGET",0,true);
 const url=new URL(parsed.data.url);url.searchParams.set("wait","true");
 try{
  const res=await send(url,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(10_000),headers:{"content-type":"application/json"},body:JSON.stringify(payload)}),body=await json(res);
  if(res.status===429){const p=z.object({retry_after:z.number().finite().min(0).optional(),global:z.boolean().optional()}).safeParse(body);return failure("RATE_LIMIT",wait(p.success?p.data.retry_after??res.headers.get("retry-after"):res.headers.get("retry-after")),false,p.success&&p.data.global===true);}
  if([401,403,404].includes(res.status))return failure("DESTINATION_DISABLED",0,true);
  if(!res.ok||!z.object({id:z.string().regex(/^[0-9]{1,22}$/)}).safeParse(body).success)return failure("PROVIDER");
  return success(res.headers.get("x-ratelimit-remaining")==="0"?wait(res.headers.get("x-ratelimit-reset-after")):0);
 }catch{return failure("PROVIDER");}
}
