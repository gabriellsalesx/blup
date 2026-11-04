import {t} from "#lib/i18n.svelte.ts";
import {dealsPaces,type DealsPace} from "../../../shared/deals-post";
export {dealsPaces,type DealsPace};
// A pace in words: "Right away", "30 minutes", "3 hours".
export const paceName=(pace:number)=>pace===0?t("dealsPaceNow"):pace<60?t("dealsPaceMinutes",{count:pace}):pace===60?t("dealsPaceHour"):t("dealsPaceHours",{count:pace/60});
// In a sentence: "every 3 hours", or "Right away".
export const paceEvery=(pace:number)=>pace===0?paceName(0):t("dealsPaceEvery",{time:paceName(pace)});
export const paceOptions=()=>dealsPaces.map(value=>({value,label:paceName(value)}));
