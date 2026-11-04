import {error} from "@sveltejs/kit";
import {dealsApi,parseResponse,type Channel,type Hunt,type StoreState} from "../../../lib/api";
import {attempt} from "../../../lib/load";
// One hunt to edit; one that isn't the person's is a missing page.
export async function load({parent,params}){
 if(!(await parent()).signedIn)return {hunt:null as Hunt|null,channels:[] as Channel[],affiliates:[] as StoreState[]};
 const [hunts,channels,affiliates]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.hunts.$get())).hunts,null),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
  attempt(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores,[] as StoreState[]),
 ]);
 const hunt=hunts?.find(h=>h.id===params.id)??null;
 if(hunts&&!hunt)error(404,"NOT_FOUND");
 return {hunt,channels,affiliates};
}
