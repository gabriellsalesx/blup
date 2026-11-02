import {browserLanguage,languageOf,languageTag,LANGUAGE_KEY,type Language,type TranslationArgs} from "../../shared/language";
import {translate,messages,type MessageKey} from "./i18n/catalog";
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
export function t<K extends MessageKey>(key:K,...args:TranslationArgs<(typeof messages)[K][0]>){return translate(language(),key,...args);}
export function setLanguage(value:Language){
 i18n.locale=value;i18n.revision++;
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
