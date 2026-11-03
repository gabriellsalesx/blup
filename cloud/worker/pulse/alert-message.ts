import {pulseAppURL} from "../../shared/panel-paths";
// One alert, the same for every destination: what kind it is (`label`),
// what it is about (`title`, linking to the panel), an optional line of
// text and a few fields (project, environment, release, address). Discord
// shows it as an embed with the tone on its side; Telegram as text with
// bold and a link, sent as entities (no parse mode, so nothing in an error
// title is read as markup).
export type AlertTone="red"|"amber"|"green"|"brand";
export type AlertField={name:string;value:string};
export type AlertMessage={label:string;title:string;url:string;tone:AlertTone;description?:string;fields:AlertField[]};

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
