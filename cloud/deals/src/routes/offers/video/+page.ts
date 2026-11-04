import {dealsApi,parseResponse,type Channel} from "../../../lib/api";
import {attempt} from "../../../lib/load";
// The channels a video's news can go to.
export async function load({parent}){
 if(!(await parent()).signedIn)return {channels:[] as Channel[]};
 return {channels:await attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[])};
}
