import {dealsApi,parseResponse,type StoreState} from "../../lib/api";
import {attempt} from "../../lib/load";
// Which stores have the person's IDs (never the IDs or keys themselves).
export async function load({parent}){
 if(!(await parent()).signedIn)return {stores:[] as StoreState[]};
 return {stores:await attempt(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores,[] as StoreState[])};
}
