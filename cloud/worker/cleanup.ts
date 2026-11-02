import { and, asc, eq, gt, isNull, lt, or } from "drizzle-orm";
import { db } from "./auth";
import { syncAccount, resourceQuota } from "./db/schema";
import { planOf } from "./account";
import { accountAllowed } from "./lifecycle";
import { thin } from "./history";
import { cleanupPlan,cleanupWrites,drainSyncGarbage,ensureHeadReferences,leaseGuard,lockSync,unlockSync,versionsOf } from "./sync-storage";

// Keep the plan's history policy; never prune merely for exceeding quota.
// The same account lease protects prepare/commit, manual and daily cleanup.
async function cleanAccount(env:Cloudflare.Env,userId:string,now:number){
 if(!await accountAllowed(env.DB,userId))return;
 const lock=await lockSync(env.DB,userId,30*60_000);if(!lock)return;
 try{
  const plan=await planOf(userId),versions=await versionsOf(env.DB,userId);
  const keep=new Set<string>();
  for(const vaultId of new Set(versions.map(v=>v.vaultId))){
   const rows=versions.filter(v=>v.vaultId===vaultId);
   const kept=thin(rows.map(v=>({...v,createdAt:new Date(v.createdAt)})),plan.limits.historyDays,now);
   for(const v of rows)if(v.n===v.head||kept.has(v.n))keep.add(v.key);
  }
  const cleanup=await cleanupPlan(env,userId,versions.filter(v=>!keep.has(v.key)),[],now);
  await env.DB.batch([leaseGuard(env.DB,userId,lock),...cleanupWrites(env.DB,userId,cleanup,now),env.DB.prepare("UPDATE sync_account SET cleaned_at=? WHERE user_id=?").bind(now,userId)]);
  await drainSyncGarbage(env,userId);
  await ensureHeadReferences(env,userId);
 }finally{await unlockSync(env.DB,userId,lock);}
}
export async function cleanUp(env:Cloudflare.Env){
 const now=Date.now(),stale=new Date(now-23*3600_000);let after="";
 for(;;){
  const page=await db.select({userId:syncAccount.userId}).from(syncAccount).leftJoin(resourceQuota,eq(resourceQuota.userId,syncAccount.userId))
   .where(and(isNull(resourceQuota.deletedAt),gt(syncAccount.userId,after),or(isNull(syncAccount.cleanedAt),lt(syncAccount.cleanedAt,stale)))).orderBy(asc(syncAccount.userId)).limit(100);
  if(!page.length)return;
  after=page[page.length-1]?.userId??after;
  for(const row of page){try{await cleanAccount(env,row.userId,now);}catch{console.error("[blup] cleanup_failed");}}
 }
}
