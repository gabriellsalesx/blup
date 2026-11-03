import {pulseAppURL} from "../../shared/panel-paths";
// One alert, the same for every destination: what kind it is (`label`),
// what it is about (`title`, linking to the panel), an optional line of
// text and a few fields (project, environment, release, address). Discord
// shows it as an embed with the tone on its side; Telegram as text with
// bold and a link, sent as entities (no parse mode, so nothing in an error
// title is read as markup); Slack as an attachment with its color; a
// generic webhook as signed JSON.
export type AlertTone="red"|"amber"|"green"|"brand";
export type AlertField={name:string;value:string};
// What happened, for systems that read the generic webhook.
export type AlertEvent="issue"|"regression"|"monitor_down"|"monitor_up"|"test";
// `test` marks an example of a real kind, sent by a destination's test.
export type AlertMessage={event:AlertEvent;test?:true;label:string;title:string;url:string;tone:AlertTone;description?:string;fields:AlertField[]};

const colors:Record<AlertTone,number>={red:0xef4444,amber:0xf59e0b,green:0x10b981,brand:0x16846b};
const cut=(value:string,max:number)=>{const text=value.trim()||"-";return text.length>max?`${text.slice(0,max-1)}…`:text;};

// The Pulse panel's page for what the alert is about.
export const panelLink=(path:`/${string}`)=>new URL(path.slice(1),pulseAppURL).href;

export function discordPayload(message:AlertMessage){
 return {
  username:"Blup Pulse",
  allowed_mentions:{parse:[]},
  embeds:[{
   author:{name:cut(message.label,256)},
   title:cut(message.title,256),
   url:message.url,
   color:colors[message.tone],
   ...(message.description?{description:cut(message.description,2000)}:{}),
   fields:message.fields.slice(0,10).map(f=>({name:cut(f.name,256),value:cut(f.value,1024),inline:true})),
  }],
 };
}

type Entity={type:"bold";offset:number;length:number}|{type:"text_link";offset:number;length:number;url:string};
// Offsets count UTF-16 code units, as JavaScript strings do.
export function telegramPayload(message:AlertMessage){
 let text="";const entities:Entity[]=[];
 const add=(part:string,...marks:({type:"bold"}|{type:"text_link";url:string})[])=>{const offset=text.length;text+=part;for(const mark of marks)entities.push({...mark,offset,length:part.length});};
 add(`Blup Pulse · ${cut(message.label,120)}`,{type:"bold"});text+="\n";
 add(cut(message.title,300),{type:"bold"},{type:"text_link",url:message.url});
 if(message.description){text+="\n";add(cut(message.description,1000));}
 if(message.fields.length)text+="\n";
 for(const field of message.fields.slice(0,10)){text+="\n";add(`${cut(field.name,60)}: `,{type:"bold"});add(cut(field.value,300));}
 return {text:text.slice(0,4096),entities:entities.filter(e=>e.offset+e.length<=4096)};
}

// Slack: the tone on the attachment's side, the title linking to the
// panel and the fields side by side. Text is escaped (&, <, >) so nothing
// in an error title becomes a Slack mention or link.
const slackText=(v:string)=>v.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
export function slackPayload(message:AlertMessage){
 const title=slackText(cut(message.title,300));
 return {
  text:`${slackText(cut(message.label,120))}: ${title}`,
  attachments:[{
   color:`#${colors[message.tone].toString(16).padStart(6,"0")}`,
   blocks:[
    {type:"context",elements:[{type:"mrkdwn",text:slackText(`Blup Pulse · ${cut(message.label,120)}`)}]},
    {type:"section",text:{type:"mrkdwn",text:`*<${message.url}|${title}>*${message.description?`\n${slackText(cut(message.description,1000))}`:""}`}},
    ...(message.fields.length?[{type:"section",fields:message.fields.slice(0,10).map(f=>({type:"mrkdwn",text:`*${slackText(cut(f.name,60))}*\n${slackText(cut(f.value,300))}`}))}]:[]),
   ],
  }],
 };
}

// A generic webhook: the alert as JSON, versioned, for the person's system;
// `test` is true for a test or an example of a kind.
export function webhookPayload(message:AlertMessage,sentAt:number){
 return {version:1 as const,source:"blup-pulse" as const,event:message.event,test:message.test===true||message.event==="test",label:message.label,title:message.title,url:message.url,tone:message.tone,description:message.description??null,fields:message.fields.slice(0,10),sentAt:new Date(sentAt).toISOString()};
}
// Its signature header: "t=<unix seconds>,v1=<hex HMAC-SHA256 of
// `${t}.${body}` with the webhook's secret>" (the receiver recomputes it
// and checks t is recent).
export async function webhookSignature(secret:string,body:string,sentAt:number){
 const t=Math.floor(sentAt/1000),key=await crypto.subtle.importKey("raw",new TextEncoder().encode(secret),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
 const mac=new Uint8Array(await crypto.subtle.sign("HMAC",key,new TextEncoder().encode(`${t}.${body}`)));
 return `t=${t},v1=${Array.from(mac,b=>b.toString(16).padStart(2,"0")).join("")}`;
}

