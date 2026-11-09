import {dealsApi,parseResponse,type Channel} from "../../lib/api";
import {early} from "../../lib/load";
export async function load({parent}){
 return {channels:await early(async()=>(await parseResponse(dealsApi.channels.$get())).channels,parent,[] as Channel[])};
}
