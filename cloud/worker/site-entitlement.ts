import type { Plan } from "./plans";
import { readPlanState } from "./lifecycle";

// One policy point for the fallback, separate from the saved user choice.
// A chosen site that was removed also falls back to the oldest remaining.
export async function retainedSiteIds(db:D1Database,userId:string,limit:number,chosen:string|null):Promise<string[]>{
	if(limit===0)return [];
	const rows=await db.prepare("SELECT id FROM site WHERE user_id=? ORDER BY CASE WHEN id=? THEN 0 ELSE 1 END,created_at,id LIMIT ?").bind(userId,chosen,limit<0?100:limit).all<{id:string}>();
	return rows.results.map(row=>row.id);
}
export async function siteDowngrade(db:D1Database,userId:string,plan:Plan,count:number,now=Date.now()){
	const state=await readPlanState(db,userId);
	if(!state?.graceUntil||plan.limits.sites<0||count<=plan.limits.sites)return null;
	const retained=await retainedSiteIds(db,userId,plan.limits.sites,state.selectedSiteId);
	return {limit:plan.limits.sites,deadline:new Date(state.graceUntil).toISOString(),selectedSiteId:state.selectedSiteId&&retained.includes(state.selectedSiteId)?state.selectedSiteId:null,retainedSiteIds:retained,selectionOpen:plan.limits.sites===1&&state.graceUntil>now};
}
