import {dealsApi,parseResponse,type Channel,type Hunt,type StoreState} from "../../../lib/api";
import {attempt} from "../../../lib/load";
export async function load({parent}){
 if(!(await parent()).signedIn)return {hunts:[] as Hunt[],channels:[] as Channel[],affiliates:[] as StoreState[]};
 const [hunts,channels,affiliates]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.hunts.$get())).hunts,[] as Hunt[]),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
  attempt(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores,[] as StoreState[]),
 ]);
 return {hunts,channels,affiliates};
}
