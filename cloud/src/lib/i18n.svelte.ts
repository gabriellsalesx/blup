import {z} from "zod/mini";
import {browserLanguage,interpolate,languageOf,languageTag,LANGUAGE_KEY,type Language,type TranslationArgs} from "../../shared/language";
import type {messages,MessageKey} from "./i18n/catalog";
export function chooseLanguage(stored:unknown,cookie:unknown,languages:readonly string[]):Language {
 return languageOf(stored)??languageOf(cookie)??browserLanguage(languages);
}
function initialLanguage():Language {
 let stored:unknown=null,cookie:unknown=null;
 try{stored=globalThis.localStorage?.getItem(LANGUAGE_KEY);}catch{}
 try{if(typeof document!=="undefined")cookie=document.cookie.split(";").map(v=>v.trim()).find(v=>v.startsWith(`${LANGUAGE_KEY}=`))?.slice(LANGUAGE_KEY.length+1);}catch{}
 return chooseLanguage(stored,cookie,typeof navigator==="undefined"?[]:navigator.languages??[navigator.language]);
}
// CSR only (routes/+layout.ts). Worker/email rendering imports pure
// shared catalogs instead, so no account language lives in server globals.
export const i18n=$state({locale:initialLanguage(),staff:false,revision:0});
export const language=():Language=>i18n.staff?"pt":i18n.locale;
export const intlLocale=()=>languageTag[language()];
// The texts come one language at a time (/i18n/<lang>.json, made
// from the catalog at build: shared/i18n-split.ts); the catalog only gives
// the keys and their placeholders here. static/theme.js starts the file of
// the language in use in the <head>; the panel waits for it (loadLanguage
// in the root layout) before its first page.
type Texts=Readonly<Record<string,string>>;
const textsSchema=z.record(z.string(),z.string());
const loaded:Partial<Record<Language,Texts>>={};
const loading:Partial<Record<Language,Promise<void>>>={};
type Early={lang:string;table:Promise<unknown>};
const early=()=>(globalThis as {__blupTexts?:Early}).__blupTexts;
async function fetchTexts(lang:Language):Promise<Texts|null>{
 const head=early();
 const fromHead=head?.lang===lang?await head.table.catch(()=>null):null;
 // Under Vitest there is no file to fetch: the catalog right away.
 const body=fromHead??(import.meta.env.VITEST?null:await fetch(`/i18n/${lang}.json`).then(r=>r.ok?r.json():null).catch(()=>null));
 const parsed=textsSchema.safeParse(body);
 if(parsed.success)return parsed.data;
 // No file (offline, a test, an old tab after a deploy): the catalog itself,
 // in a chunk of its own that only this path downloads.
 const {messages:all}=await import("./i18n/catalog");
 const index={pt:0,en:1,es:2}[lang];
 return Object.fromEntries(Object.entries(all).map(([key,texts])=>[key,texts[index]]));
}
/** Makes the texts of lang ready; t() shows them from then on. */
export function loadLanguage(lang:Language):Promise<void>{
 if(loaded[lang])return Promise.resolve();
 return loading[lang]??=fetchTexts(lang).then(texts=>{if(texts){loaded[lang]=texts;i18n.revision++;}}).finally(()=>{delete loading[lang];});
}
export function t<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>):string{
 return textIn(language(),key,...args);
}
/** A text in a given language (t() is the one in use). */
export function textIn<K extends MessageKey>(lang:Language,key:K,...args:TranslationArgs<(typeof messages)[K][0]>):string{
 void i18n.revision;
 const texts=loaded[lang];
 // Not ready yet (a language switch in flight): the texts on screen stay in
 // the language already loaded until this one arrives.
 if(!texts)void loadLanguage(lang);
 const text=texts?.[key]??Object.values(loaded).find(v=>v?.[key]!==undefined)?.[key]??key;
 return interpolate(text,args[0]);
}
export function setLanguage(value:Language){
 i18n.locale=value;i18n.revision++;
 void loadLanguage(value);
 try{globalThis.localStorage?.setItem(LANGUAGE_KEY,value);}catch{}
 if(typeof document!=="undefined"){
  try{document.cookie=`${LANGUAGE_KEY}=${value}; Path=/; Max-Age=31536000; SameSite=Lax${location.protocol==="https:"?"; Secure":""}`;}catch{}
  document.documentElement.lang=languageTag[value];
 }
}
export function formatDate(value:Date|string,options:Intl.DateTimeFormatOptions={dateStyle:"medium",timeStyle:"short"}){return new Intl.DateTimeFormat(intlLocale(),options).format(new Date(value));}
export function relativeTime(value:string){
 const minutes=Math.round((new Date(value).getTime()-Date.now())/60000),hours=Math.round(minutes/60),days=Math.round(hours/24);
 return new Intl.RelativeTimeFormat(intlLocale(),{numeric:"auto"}).format(Math.abs(minutes)<60?minutes:Math.abs(hours)<24?hours:days,Math.abs(minutes)<60?"minute":Math.abs(hours)<24?"hour":"day");
}
// The tests read t() right away, in any language.
if(import.meta.env.VITEST)await Promise.all((["pt","en","es"] as const).map(loadLanguage));
