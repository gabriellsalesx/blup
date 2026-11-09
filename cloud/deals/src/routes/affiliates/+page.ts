import {dealsApi,parseResponse,type StoreState} from "../../lib/api";
import {early} from "../../lib/load";
// Which stores have the person's IDs (never the IDs or keys themselves).
export async function load({parent}){
 return {stores:await early(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores,parent,[] as StoreState[])};
}
