import {dealsApi,parseResponse,type Channel,type Hunt,type Store} from "../../../lib/api";
import {attempt} from "../../../lib/load";
export async function load({parent}){
 if(!(await parent()).signedIn)return {hunts:[] as Hunt[],channels:[] as Channel[],configured:[] as Store[]};
 const [hunts,channels,configured]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.hunts.$get())).hunts,[] as Hunt[]),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
  attempt(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores.filter(s=>s.configured).map(s=>s.store),[] as Store[]),
 ]);
 return {hunts,channels,configured};
}
