import {dealsApi,parseResponse,type Channel,type Post,type PostState} from "../lib/api";
import {attempt} from "../lib/load";
// The deals of one state (?state=queued|sent|failed, queued to start) and
// the channels, to name where each went.
const states=["queued","sent","failed"] as const satisfies readonly PostState[];
export async function load({parent,url}){
 const raw=url.searchParams.get("state"),state:PostState=states.find(s=>s===raw)??"queued";
 if(!(await parent()).signedIn)return {state,posts:[] as Post[],channels:[] as Channel[]};
 const [posts,channels]=await Promise.all([
  attempt(async()=>(await parseResponse(dealsApi.offers.$get({query:{state}}))).posts,[] as Post[]),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
 ]);
 return {state,posts,channels};
}
