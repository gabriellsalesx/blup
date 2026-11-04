import {error} from "@sveltejs/kit";
import {dealsApi,parseResponse,type StoreState} from "../../../lib/api";
import {attempt} from "../../../lib/load";
import {stores} from "../../../lib/catalog";
// One store's page: only the stores the panel knows; its saved state (never
// the IDs or keys themselves).
export async function load({parent,params}){
 const info=stores.find(s=>s.id===params.store);if(!info)error(404);
 if(!(await parent()).signedIn)return {store:info.id,saved:undefined as StoreState|undefined};
 const all=await attempt(async()=>(await parseResponse(dealsApi.affiliates.$get())).stores,[] as StoreState[]);
 return {store:info.id,saved:all.find(s=>s.store===info.id)};
}
