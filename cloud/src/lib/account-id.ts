import {language,textIn} from "./i18n.svelte";
import type {Language} from "../../shared/language";
export const accountIdHelp=(lang:Language=language())=>textIn(lang,"accountIdHelp");
export const shortAccountId=(id:string)=>id.length>12?`${id.slice(0,8)}…${id.slice(-4)}`:id;
