import {dealsApi,parseResponse,type Channel,type Hunt} from "../../lib/api";
import {early} from "../../lib/load";
export async function load({parent}){
 const [hunts,channels]=await Promise.all([
  early(async()=>(await parseResponse(dealsApi.hunts.$get())).hunts,parent,[] as Hunt[]),
  early(async()=>(await parseResponse(dealsApi.channels.$get())).channels,parent,[] as Channel[]),
 ]);
 return {hunts,channels};
}
