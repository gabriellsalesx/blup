import {dealsApi,parseResponse,type Post,type PostState} from "./api";
import type {Page} from "#lib/pager.svelte.ts";
// One page of the posts of a state, for the pager.
export const postsPage=async(state:PostState,cursor:string|null):Promise<Page<Post>>=>{
 const r=await parseResponse(dealsApi.offers.$get({query:{state,...(cursor?{cursor}:{})}}));
 return {items:r.posts,nextCursor:r.nextCursor};
};
