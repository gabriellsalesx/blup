import {error} from "@sveltejs/kit";
import {dealsApi,parseResponse,type Channel} from "../../../lib/api";
import {attempt} from "../../../lib/load";
// One channel of the account, by its id: its details and examples.
export async function load({parent,params}){
 if(!/^[0-9a-f-]{36}$/.test(params.channel))error(404);
 if(!(await parent()).signedIn)return {channel:null as Channel|null};
 const found=await attempt(async()=>({channel:(await parseResponse(dealsApi.channels.$get())).channels.find(c=>c.id===params.channel)??null,loaded:true}),{channel:null as Channel|null,loaded:false});
 if(found.loaded&&!found.channel)error(404);
 return {channel:found.channel};
}
