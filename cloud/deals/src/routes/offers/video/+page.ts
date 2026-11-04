import {dealsApi,parseResponse,type Channel,type Settings} from "../../../lib/api";
import {attempt} from "../../../lib/load";
// The channels a video's news can go to the account's pace (for the queue) and post layout.
export async function load({parent}){
 if(!(await parent()).signedIn)return {channels:[] as Channel[],pace:0,style:"classic" as const};
 const [channels,settings]=await Promise.all([attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),attempt<Settings|null>(async()=>await parseResponse(dealsApi.settings.$get()),null)]);
 return {channels,pace:settings?.pace??0,style:settings?.style??"classic"};
}
