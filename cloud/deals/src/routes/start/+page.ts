import {dealsApi,parseResponse,type Channel,type StoreState} from "../../lib/api";
import {attempt} from "../../lib/load";
// The channels and the stores already set, and where the assistant was
// when it sent the person to link a channel (?step=&topic=).
export async function load({parent,url}){
 const step=Math.min(4,Math.max(0,Number(url.searchParams.get("step"))||0)),topic=url.searchParams.get("topic")==="other"?"other" as const:"games" as const;
 if(!(await parent()).signedIn)return {step,topic,channels:[] as Channel[],stores:[] as StoreState[]};
 const [channels,stores]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
  attempt(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores,[] as StoreState[]),
 ]);
 return {step,topic,channels,stores};
}
