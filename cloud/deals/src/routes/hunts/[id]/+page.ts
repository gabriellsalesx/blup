import {error} from "@sveltejs/kit";
import {dealsApi,parseResponse,type Channel,type Hunt} from "../../../lib/api";
import {attempt} from "../../../lib/load";
// One hunt to edit; one that isn't the person's is a missing page.
export async function load({parent,params}){
 if(!(await parent()).signedIn)return {hunt:null as Hunt|null,channels:[] as Channel[]};
 const [hunts,channels]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.hunts.$get())).hunts,null),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
 ]);
 const hunt=hunts?.find(h=>h.id===params.id)??null;
 if(hunts&&!hunt)error(404,"NOT_FOUND");
 return {hunt,channels};
}
