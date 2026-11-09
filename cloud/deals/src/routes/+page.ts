import {dealsApi,parseResponse,type Channel,type Post,type PostState,type Settings} from "../lib/api";
import {early} from "../lib/load";
import type {Page} from "#lib/pager.svelte.ts";
import {postsPage} from "../lib/posts";
// The deals of one state (?state=queued|sent|failed, queued to start), a
// page at a time, the channels (to name where each went) and the pace, all
// leaving together with the layout's request.
const states=["queued","sent","failed"] as const satisfies readonly PostState[];
export async function load({parent,url}){
 const raw=url.searchParams.get("state"),state:PostState=states.find(s=>s===raw)??"queued";
 const [first,channels,settings]=await Promise.all([
  early<Page<Post>|null>(()=>postsPage(state,null),parent,null),
  early(async()=>(await parseResponse(dealsApi.channels.$get())).channels,parent,[] as Channel[]),
  early<Settings|null>(async()=>await parseResponse(dealsApi.settings.$get()),parent,null),
 ]);
 return {state,first,channels,settings};
}
