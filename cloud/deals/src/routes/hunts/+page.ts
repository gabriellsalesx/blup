import {dealsApi,parseResponse,type Channel,type Hunt} from "../../lib/api";
import {attempt} from "../../lib/load";
export async function load({parent}){
 if(!(await parent()).signedIn)return {hunts:[] as Hunt[],channels:[] as Channel[]};
 const [hunts,channels]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.hunts.$get())).hunts,[] as Hunt[]),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
 ]);
 return {hunts,channels};
}
