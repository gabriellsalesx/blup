import {dealsApi,parseResponse,type Settings} from "../../lib/api";
import {early} from "../../lib/load";
// The account's settings: pace, opening lines, extra text and networks.
export async function load({parent}){
 return {settings:await early<Settings|null>(async()=>await parseResponse(dealsApi.settings.$get()),parent,null)};
}
