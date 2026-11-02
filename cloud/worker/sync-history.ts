import { cleanupPlan,cleanupWrites,drainSyncGarbage,ensureHeadReferences,leaseGuard,lockSync,SyncError,unlockSync,versionsOf } from "./sync-storage";
import { accountPlan,observePlan } from "./lifecycle";

async function storageBytes(env:Cloudflare.Env,userId:string){
	const row=await env.DB.prepare("SELECT coalesce((SELECT storage_bytes FROM sync_account WHERE user_id=?),0)+coalesce((SELECT sum(bytes) FROM site WHERE user_id=?),0) AS n").bind(userId,userId).first<{n:number}>();
	return row?.n??0;
}
export async function purgeVaultHistory(env:Cloudflare.Env,userId:string,vaultId:string){
	if(!await env.DB.prepare("SELECT id FROM vault WHERE id=? AND user_id=?").bind(vaultId,userId).first())throw new SyncError("NOT_FOUND",404);
	const lock=await lockSync(env.DB,userId,30*60_000);if(!lock)throw new SyncError("SYNC_BUSY",409);
	try{
		await ensureHeadReferences(env,userId);
		const versions=await versionsOf(env.DB,userId);
		const cleanup=await cleanupPlan(env,userId,versions.filter(v=>v.vaultId===vaultId&&v.n!==v.head));
		await env.DB.batch([leaseGuard(env.DB,userId,lock),...cleanupWrites(env.DB,userId,cleanup)]);
		const freedBytes=await drainSyncGarbage(env,userId);
		const after=await storageBytes(env,userId),plan=await accountPlan(env,userId);
		if(plan)await observePlan(env.DB,userId,plan.name);
		return {freedBytes,storageBytes:after};
	}finally{await unlockSync(env.DB,userId,lock);}
}
export async function historyBytes(env:Cloudflare.Env,userId:string):Promise<number|null>{
	const missing=await env.DB.prepare("SELECT id FROM vault WHERE user_id=? AND head_refs_ready=0 LIMIT 1").bind(userId).first();
	if(missing){
		const lock=await lockSync(env.DB,userId);if(!lock)return null;
		try{await ensureHeadReferences(env,userId);}catch{return null;}finally{await unlockSync(env.DB,userId,lock);}
	}
	const row=await env.DB.prepare("SELECT coalesce((SELECT sum(vv.size) FROM vault_version vv JOIN vault v ON v.id=vv.vault_id WHERE v.user_id=? AND vv.n<>v.head),0)+coalesce((SELECT sum(b.size) FROM sync_block b WHERE b.user_id=? AND NOT EXISTS(SELECT 1 FROM vault_head_block h JOIN vault v ON v.id=h.vault_id WHERE h.block_id=b.id AND v.user_id=b.user_id)),0)+coalesce((SELECT sum(bytes) FROM sync_garbage WHERE user_id=?),0) AS n").bind(userId,userId,userId).first<{n:number}>();
	return row?.n??0;
}
