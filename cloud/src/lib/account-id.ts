import {translate} from "./i18n/catalog";
import type {Language} from "../../shared/language";
export const accountIdHelp=(locale:Language="pt")=>translate(locale,"accountIdHelp");
export const shortAccountId=(id:string)=>id.length>12?`${id.slice(0,8)}…${id.slice(-4)}`:id;
export async function copyAccountId(id:string,clipboard:Pick<Clipboard,"writeText">|undefined=globalThis.navigator?.clipboard):Promise<boolean>{
	if(!clipboard||!id)return false;
	try{await clipboard.writeText(id);return true;}catch{return false;}
}
