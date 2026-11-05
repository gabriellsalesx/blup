import {z} from "zod";
import {languageTag,type Language} from "../../shared/language";
import {currencyDigits} from "../../shared/deals-markets";
import {success,failure,wait,json,postSigned,type Delivery} from "../pulse/transports";
import {discordWebhook} from "../pulse/channel-crypto";

// One post: the offer as the channel sees it. link is the short link;
// watch, for a video, is the video itself (Telegram shows it above).
// title is the opening line already written; extra, the person's own text,
// and socials, their networks, go under the link.
// description and coupon are the person's own for this post; style, the
// account's layout (classic, or emoji: a mark for each part). store goes to
// webhooks; sample marks an example sent from the panel, and photo is its
// picture's own bytes (a JPEG sent as a file, not fetched by address).
export type Post={store?:string|null;sample?:boolean;photo?:Uint8Array|null;title:string;image:string|null;price:number|null;originalPrice:number|null;currency:string|null;discount:number|null;link:string;affiliate:boolean;kind:string;credit:string|null;watch?:string|null;extra?:string|null;socials?:readonly {name:string;url:string}[];description?:string|null;coupon?:string|null;style?:"classic"|"emoji"};
const words={
 pt:{free:"Grátis",open:"Ver oferta",video:"Vídeo novo",watch:"Ver o vídeo",videoAd:"#publi",ad:"#publi · link de afiliado",off:"de desconto",value:"Valor",coupon:"Cupom",link:"Link",test:"Teste do Blup Deals: este canal está pronto para receber ofertas.",sample:"Exemplo do Blup Deals: é assim que as ofertas chegam aqui."},
 en:{free:"Free",open:"See the deal",video:"New video",watch:"Watch the video",videoAd:"#ad",ad:"#ad · affiliate link",off:"off",value:"Price",coupon:"Coupon",link:"Link",test:"Blup Deals test: this channel is ready to get deals.",sample:"Blup Deals example: this is how deals arrive here."},
 es:{free:"Gratis",open:"Ver oferta",video:"Video nuevo",watch:"Ver el video",videoAd:"#publi",ad:"#publi · enlace de afiliado",off:"de descuento",value:"Precio",coupon:"Cupón",link:"Enlace",test:"Prueba de Blup Deals: este canal está listo para recibir ofertas.",sample:"Ejemplo de Blup Deals: así llegan las ofertas aquí."},
} as const;
export const testText=(language:Language)=>words[language].test;
// The source a post must credit, as an active link (GamerPower's terms ask
// for a clickable attribution): on Discord the small line above the title,
// on Telegram the last line.
export const credits:Record<string,{name:string;label:string;url:string}>={gamerpower:{name:"GamerPower",label:"Via GamerPower",url:"https://www.gamerpower.com/"}};
const creditOf=(p:Post)=>p.credit&&Object.hasOwn(credits,p.credit)?credits[p.credit]:null;
// Money in minor units, as many decimals as the currency has (none for
// the yen), the same as the panel.
export function money(minor:number,currency:string,language:Language){
 const digits=currencyDigits(currency);
 try{return new Intl.NumberFormat(languageTag[language],{style:"currency",currency}).format(minor/10**digits);}catch{return `${(minor/10**digits).toFixed(digits)} ${currency}`;}
}
// The price line: "Grátis" for a giveaway (its worth is in dollars, so it
// is left out), else the old price and the new; without a price (another
// country's), the discount alone.
function priceLine(p:Post,language:Language,strike:(s:string)=>string){
 const w=words[language];
 if(p.kind==="game_free")return w.free;
 if(p.price===null||!p.currency)return p.discount&&p.discount>0?`${p.discount}% ${w.off}`:null;
 const now=money(p.price,p.currency,language),before=p.originalPrice&&p.originalPrice>p.price?`${strike(money(p.originalPrice,p.currency,language))} `:"";
 return `${before}${now}${p.discount&&p.discount>0?` (${p.discount}% ${w.off})`:""}`;
}
const escapeHTML=(s:string)=>s.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;");
// Discord reads markdown: the person's text goes as typed, not as markup.
const escapeMarkdown=(s:string)=>s.replace(/[\\*_~`|>\[\]()<]/g,c=>`\\${c}`).replace(/^([#-])/,"\\$1");
const markdownURL=(u:string)=>u.replace(/[()\s]/g,c=>encodeURIComponent(c));
// What a post says, in either platform's markup: a video's news ("New
// video: the product"), the link's words and the notice.
function parts(p:Post,language:Language){
 const w=words[language],video=p.kind==="video";
 return {title:video?`${w.video}: ${p.title}`:p.title,open:video?w.watch:w.open,ad:p.affiliate?video?w.videoAd:w.ad:null};
}
// The lines under the title, in either platform's markup (esc: the
// person's text as typed; link; strike). Classic: description, price,
// coupon, the link's words; emoji: each part on its own line with its mark
// and the address itself, as the deal groups write. Then the person's
// extra text and networks, the notice and, on an example, that it is one.
type Markup={esc:(s:string)=>string;link:(text:string,url:string)=>string;strike:(s:string)=>string};
function bodyLines(p:Post,language:Language,m:Markup){
 const w=words[language],{open,ad}=parts(p,language),price=priceLine(p,language,m.strike);
 const desc=p.description?m.esc(p.description):null,coupon=p.coupon?m.esc(p.coupon):null;
 const tail=[...(p.extra?[m.esc(p.extra)]:[]),...(p.socials?.length?[p.socials.map(n=>m.link(m.esc(n.name),n.url)).join(" · ")]:[]),...(ad?[ad]:[]),...(p.sample?[w.sample]:[])];
 if(p.style!=="emoji")return [...(desc?[desc]:[]),...(price?[price]:[]),...(coupon?[`${w.coupon}: ${coupon}`]:[]),m.link(open,p.link),...tail];
 const lines=[...(desc?["",desc]:[]),"",...(price?[`💵 ${w.value}: ${price}`]:[]),...(coupon?[`🧾 ${w.coupon}: ${coupon}`]:[]),...(price||coupon?[""]:[]),`📌 ${p.kind==="video"?w.watch:w.link}: ${m.link(m.esc(p.link.replace(/^https:\/\//,"")),p.link)}`,...(tail.length?["",...tail]:[])];
 // No blank line twice or at the end (the first one parts it from the title).
 return lines.filter((l,i,a)=>l!==""||(a[i-1]!==""&&i<a.length-1));
}
// The title: in the emoji layout, a flame before it and, on a discount, a
// falling chart after it.
const headTitle=(p:Post,title:string)=>p.style==="emoji"?`🔥 ${title}${p.discount&&p.discount>0&&p.kind!=="game_free"?" 📉":""}`:title;
// Telegram (HTML): at most 1024 characters with a photo.
export function telegramText(p:Post,language:Language){
 const {title}=parts(p,language);
 const lines=[`<b>${escapeHTML([...headTitle(p,title)].slice(0,300).join(""))}</b>`,...bodyLines(p,language,{esc:escapeHTML,link:(t,u)=>`<a href="${escapeHTML(u)}">${t}</a>`,strike:s=>`<s>${escapeHTML(s)}</s>`}),...((c=>c?[`<a href="${c.url}">${c.label}</a>`]:[])(creditOf(p)))];
 return lines.join("\n");
}
export function discordPayload(p:Post,language:Language){
 const {title}=parts(p,language),credit=creditOf(p);
 const description=bodyLines(p,language,{esc:escapeMarkdown,link:(t,u)=>`[${t}](${markdownURL(u)})`,strike:s=>`~~${s}~~`}).join("\n").replace(/^\n+/,"");
 return {username:"Blup Deals",allowed_mentions:{parse:[]},embeds:[{...(credit?{author:{name:credit.name,url:credit.url}}:{}),title:[...headTitle(p,title)].slice(0,250).join(""),url:p.link,color:0xc2410c,description,...(p.image?{image:{url:p.image}}:{})}]};
}

// WhatsApp (its own marks: *bold*, ~strike~; addresses go as they are,
// WhatsApp makes them clickable). The person's text as typed.
export function whatsappText(p:Post,language:Language){
 const {title}=parts(p,language);
 const link=(t:string,u:string)=>u.endsWith(t)?u:`${t}: ${u}`;
 const lines=[`*${[...headTitle(p,title)].slice(0,300).join("").replace(/\*/g,"")}*`,...bodyLines(p,language,{esc:s=>s,link,strike:s=>`~${s}~`}),...((c=>c?[`${c.label}: ${c.url}`]:[])(creditOf(p)))];
 return lines.join("\n").slice(0,4000);
}

// Plain text, for webhooks: no marks, each address after its words.
function plainText(p:Post,language:Language){
 const {title}=parts(p,language);
 return [[...headTitle(p,title)].slice(0,300).join(""),...bodyLines(p,language,{esc:s=>s,link:(t,u)=>u.endsWith(t)?u:`${t}: ${u}`,strike:s=>s}),...((c=>c?[`${c.label}: ${c.url}`]:[])(creditOf(p)))].join("\n").slice(0,4000);
}
// The person's own system: the post as data and as plain text (null: the
// plain test message), signed like Pulse's webhooks (X-Blup-Signature).
// event "deal" is an offer (test: true on an example); "test", the test.
export function dealsWebhookPayload(p:Post|null,language:Language,sentAt:number){
 return {version:1 as const,source:"blup-deals" as const,event:p?"deal" as const:"test" as const,test:!p||p.sample===true,language,text:p?plainText(p,language):testText(language),
  deal:p?{title:parts(p,language).title,kind:p.kind,store:p.store??null,price:p.price,originalPrice:p.originalPrice,currency:p.currency,discount:p.discount,link:p.link,image:p.image,video:p.watch??null,description:p.description??null,coupon:p.coupon??null,affiliate:p.affiliate,credit:creditOf(p)?.name??null}:null,
  sentAt:new Date(sentAt).toISOString()};
}
export function sendWebhook(target:string,p:Post|null,language:Language,send:typeof fetch=fetch,now=Date.now()){
 const payload=dealsWebhookPayload(p,language,now);
 return postSigned(target,payload.event,payload,"BlupDeals/1.0 (+https://getblup.io)",now,send);
}

export const dealsChatId=z.string().regex(/^-?[1-9][0-9]{0,15}$/).refine(v=>Number.isSafeInteger(Number(v)));
const botToken=(env:Cloudflare.Env)=>/^[0-9]+:[A-Za-z0-9_-]{20,200}$/.test(env.DEALS_TELEGRAM_TOKEN??"")?env.DEALS_TELEGRAM_TOKEN:null;
const telegramResult=z.discriminatedUnion("ok",[z.object({ok:z.literal(true),result:z.object({message_id:z.number().int().nonnegative()})}),z.object({ok:z.literal(false),error_code:z.number().int(),parameters:z.object({retry_after:z.number().int().nonnegative().optional()}).optional()})]);
async function telegramCall(token:string,method:string,body:object|FormData,send:typeof fetch){
 const res=await send(`https://api.telegram.org/bot${token}/${method}`,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(15_000),...(body instanceof FormData?{body}:{headers:{"content-type":"application/json"},body:JSON.stringify(body)})});
 return {res,result:telegramResult.safeParse(await json(res))};
}
// A picture's bytes as a form's file.
function photoForm(fields:Record<string,string>,name:string,bytes:Uint8Array){const form=new FormData();for(const [k,v] of Object.entries(fields))form.append(k,v);form.append(name,new Blob([bytes],{type:"image/jpeg"}),"exemplo.jpg");return form;}
// A photo with the caption; when Telegram refuses the photo (a store
// image it cannot fetch), the text alone. A video goes as text with the
// video's own preview above it (Telegram reads TikTok's page itself). A
// photo's bytes go as a file.
export async function sendTelegram(env:Cloudflare.Env,target:string,text:string,image:string|null,send:typeof fetch=fetch,watch:string|null=null,photo:Uint8Array|null=null):Promise<Delivery>{
 const chat=dealsChatId.safeParse(target);if(!chat.success)return failure("INVALID_TARGET",0,true);
 const token=botToken(env);if(!token)return failure("CHANNEL_UNAVAILABLE",300_000);
 try{
  const preview=watch?{url:watch,prefer_large_media:true,show_above_text:true}:{is_disabled:true};
  let {res,result}=photo?await telegramCall(token,"sendPhoto",photoForm({chat_id:chat.data,caption:text,parse_mode:"HTML"},"photo",photo),send):image?await telegramCall(token,"sendPhoto",{chat_id:chat.data,photo:image,caption:text,parse_mode:"HTML"},send):await telegramCall(token,"sendMessage",{chat_id:chat.data,text,parse_mode:"HTML",link_preview_options:preview},send);
  if((photo||image)&&result.success&&!result.data.ok&&result.data.error_code===400)({res,result}=await telegramCall(token,"sendMessage",{chat_id:chat.data,text,parse_mode:"HTML",link_preview_options:{is_disabled:true}},send));
  if(!result.success)return failure("PROVIDER");if(result.data.ok)return res.ok?success(1000):failure("PROVIDER");
  if(result.data.error_code===429)return failure("RATE_LIMIT",wait(result.data.parameters?.retry_after??res.headers.get("retry-after")),false,true);
  if([400,403,404].includes(result.data.error_code))return failure("DESTINATION_DISABLED",0,true);
  return failure("PROVIDER");
 }catch{return failure("PROVIDER");}
}
// A photo's bytes go as the message's file, shown in the embed
// (attachment://exemplo.jpg).
export async function sendDiscord(target:string,payload:object,send:typeof fetch=fetch,photo:Uint8Array|null=null):Promise<Delivery>{
 const parsed=discordWebhook.safeParse(target);if(!parsed.success)return failure("INVALID_TARGET",0,true);
 const url=new URL(parsed.data.url);url.searchParams.set("wait","true");
 try{
  const res=await send(url,{method:"POST",redirect:"manual",signal:AbortSignal.timeout(15_000),...(photo?{body:photoForm({payload_json:JSON.stringify({...payload,attachments:[{id:0,filename:"exemplo.jpg"}]})},"files[0]",photo)}:{headers:{"content-type":"application/json"},body:JSON.stringify(payload)})}),body=await json(res);
  if(res.status===429){const p=z.object({retry_after:z.number().finite().min(0).optional(),global:z.boolean().optional()}).safeParse(body);return failure("RATE_LIMIT",wait(p.success?p.data.retry_after??res.headers.get("retry-after"):res.headers.get("retry-after")),false,p.success&&p.data.global===true);}
  if([401,403,404].includes(res.status))return failure("DESTINATION_DISABLED",0,true);
  if(!res.ok||!z.object({id:z.string().regex(/^[0-9]{1,22}$/)}).safeParse(body).success)return failure("PROVIDER");
  return success(res.headers.get("x-ratelimit-remaining")==="0"?wait(res.headers.get("x-ratelimit-reset-after")):0);
 }catch{return failure("PROVIDER");}
}
