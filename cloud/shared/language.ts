import { z } from "zod";
export const languageSchema=z.enum(["pt","en","es"]);
export type Language=z.infer<typeof languageSchema>;
export const languageTag={pt:"pt-BR",en:"en-US",es:"es"} as const;
export const stripeLocale={pt:"pt-BR",en:"en",es:"es"} as const;
export const LANGUAGE_KEY="blup-lang";
export function languageOf(value:unknown):Language|null {
 const parsed=languageSchema.safeParse(value);return parsed.success?parsed.data:null;
}
export function browserLanguage(values:readonly string[]):Language {
 for(const value of values){const language=languageOf(value.trim().toLowerCase().split(/[-_]/)[0]);if(language)return language;}
 return "en";
}
export function acceptedLanguage(header:string|null):Language {
 const choices=(header??"").slice(0,1024).split(",").slice(0,32).map((item,index)=>{
  const [tag,...params]=item.trim().split(";");const q=params.find(p=>/^q=/i.test(p.trim()));
  const quality=q===undefined?1:Number(q.trim().slice(2));return {tag:tag??"",quality,index};
 }).filter(v=>Number.isFinite(v.quality)&&v.quality>0&&v.quality<=1).sort((a,b)=>b.quality-a.quality||a.index-b.index);
 return browserLanguage(choices.map(v=>v.tag));
}
export function requestLanguage(headers:Headers|undefined):Language {
 return languageOf(headers?.get("x-blup-language"))??acceptedLanguage(headers?.get("accept-language")??null);
}
export function interpolate(text:string,values:Readonly<Record<string,string|number>>={}){
 return text.replace(/\{(\w+)\}/g,(_,name:string)=>String(values[name]??`{${name}}`));
}
export type Placeholders<S extends string>=S extends `${string}{${infer Name}}${infer Rest}`?Name|Placeholders<Rest>:never;
export type TranslationArgs<S extends string>=[Placeholders<S>] extends [never]?[]:[values:Readonly<Record<Placeholders<S>,string|number>>];
