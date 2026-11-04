import {dealsApi,parseResponse,type Settings} from "../../lib/api";
import {attempt} from "../../lib/load";
// The account's settings: pace, opening lines, extra text and networks.
export async function load({parent}){
 if(!(await parent()).signedIn)return {settings:null};
 return {settings:await attempt<Settings|null>(async()=>await parseResponse(dealsApi.settings.$get()),null)};
}
