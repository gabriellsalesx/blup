import {dealsApi,parseResponse,type Channel,type Post,type PostState,type Settings} from "../lib/api";
import {attempt} from "../lib/load";
import type {Page} from "#lib/pager.svelte.ts";
import {postsPage} from "../lib/posts";
// The deals of one state (?state=queued|sent|failed, queued to start), a
// page at a time, the channels (to name where each went) and the pace.
const states=["queued","sent","failed"] as const satisfies readonly PostState[];
export async function load({parent,url}){
 const raw=url.searchParams.get("state"),state:PostState=states.find(s=>s===raw)??"queued";
 if(!(await parent()).signedIn)return {state,first:{items:[],nextCursor:null} as Page<Post>,channels:[] as Channel[],settings:null};
 const [first,channels,settings]=await Promise.all([
  attempt<Page<Post>|null>(()=>postsPage(state,null),null),
  attempt(async()=>(await parseResponse(dealsApi.channels.$get())).channels,[] as Channel[]),
  attempt<Settings|null>(async()=>await parseResponse(dealsApi.settings.$get()),null),
 ]);
 return {state,first,channels,settings};
}
