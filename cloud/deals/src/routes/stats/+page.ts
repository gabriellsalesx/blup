import {dealsApi,parseResponse,type Stats} from "../../lib/api";
import {attempt} from "../../lib/load";
// Clicks and posts per day for 7, 30 or 90 days (?days=, 30 to start).
const ranges=["7","30","90"] as const;
export type Range=(typeof ranges)[number];
export async function load({parent,url}){
 const raw=url.searchParams.get("days"),days:Range=ranges.find(r=>r===raw)??"30";
 const empty:Stats={days:[],top:[]};
 if(!(await parent()).signedIn)return {days,stats:empty};
 return {days,stats:await attempt(async()=>await parseResponse(dealsApi.stats.$get({query:{days}})),empty)};
}
