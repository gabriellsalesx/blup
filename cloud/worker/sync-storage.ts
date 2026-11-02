import { z } from "zod";
import { unlockSite } from "./quota";
import { readPlanState } from "./lifecycle";
import type { Plan } from "./plans";

export const syncLockKey=(userId:string)=>`sync:${userId}`;
const uploadPrefix=(userId:string)=>`sync-upload:${userId}:`;
export async function lockSync(db:D1Database,userId:string,ttl=120_000){
 const prefix=uploadPrefix(userId),token=crypto.randomUUID(),now=Date.now();
 const row=await db.prepare("INSERT INTO site_lease(site_id,token,expires_at) SELECT ?,?,? WHERE NOT EXISTS(SELECT 1 FROM site_lease WHERE site_id>=? AND site_id<? AND expires_at>?) ON CONFLICT(site_id) DO UPDATE SET token=excluded.token,expires_at=excluded.expires_at WHERE expires_at<=? RETURNING token").bind(syncLockKey(userId),token,now+ttl,prefix,prefix+"~",now,now).first();
 return row?token:null;
}
export async function lockUpload(db:D1Database,userId:string,id:string){
 const token=crypto.randomUUID(),now=Date.now(),key=uploadPrefix(userId)+id;
 const row=await db.prepare("INSERT INTO site_lease(site_id,token,expires_at) SELECT ?,?,? WHERE NOT EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND expires_at>?) ON CONFLICT(site_id) DO UPDATE SET token=excluded.token,expires_at=excluded.expires_at WHERE expires_at<=? RETURNING token").bind(key,token,now+120_000,syncLockKey(userId),now,now).first();
 return row?{key,token}:null;
}
export function uploadGuard(db:D1Database,userId:string,lock:{key:string;token:string}){
 return db.prepare("UPDATE resource_quota SET storage=CASE WHEN deleted_at IS NULL AND EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?) THEN storage ELSE NULL END WHERE user_id=?").bind(lock.key,lock.token,Date.now(),userId);
}
export const unlockUpload=(db:D1Database,lock:{key:string;token:string})=>unlockSite(db,lock.key,lock.token);
export const unlockSync=(db:D1Database,userId:string,token:string)=>unlockSite(db,syncLockKey(userId),token);
export class SyncError extends Error{
	constructor(readonly code:string,readonly status:400|404|409|410|413|503,readonly head?:number){super(code);}
}
export async function storageLimit(db:D1Database,userId:string,plan:Plan,now=Date.now()){
	const state=await readPlanState(db,userId),current=plan.limits.storageMB*1024*1024;
	return state?.graceUntil&&state.graceUntil>now&&state.storageGraceLimit!==null?Math.max(current,state.storageGraceLimit):current;
}
export type StoredVersion={vaultId:string;n:number;key:string;size:number;createdAt:number;head:number};
export type StoredBlock={id:string;size:number;objectKey:string;createdAt:number};
export const versionSchema=z.object({vaultId:z.string(),n:z.number().int(),key:z.string(),size:z.number().int().min(0),createdAt:z.number(),head:z.number().int()});
export const blockSchema=z.object({id:z.string().regex(/^[a-f0-9]{64}$/),size:z.number().int().min(0),objectKey:z.string(),createdAt:z.number()});
export const cleanupSchema=z.object({versions:z.array(versionSchema),blocks:z.array(blockSchema)});
export type CleanupPlan=z.infer<typeof cleanupSchema>;
export async function versionsOf(db:D1Database,userId:string){
	return (await db.prepare("SELECT vv.vault_id AS vaultId,vv.n,vv.key,vv.size,vv.created_at AS createdAt,v.head FROM vault_version vv JOIN vault v ON v.id=vv.vault_id WHERE v.user_id=?").bind(userId).all<StoredVersion>()).results;
}
export async function refsOf(env:Cloudflare.Env,key:string):Promise<string[]>{
	const obj=await env.NOTES.get(`${key}/r`);
	if(!obj)throw new SyncError("SYNC_METADATA_UNAVAILABLE",503);
	const text=await obj.text();
	return z.array(z.string().regex(/^[a-f0-9]{64}$/)).max(20_000).parse(text.split("\n").filter(Boolean));
}
export async function cleanupPlan(env:Cloudflare.Env,userId:string,drop:StoredVersion[],additionalRefs:string[]=[],now=Date.now()):Promise<CleanupPlan>{
	const all=await versionsOf(env.DB,userId),remove=new Set(drop.map(v=>v.key)),keep=new Set(additionalRefs),candidates=new Set<string>();
	for(const v of all){
		const obj=await env.NOTES.get(`${v.key}/r`);
		if(!obj){if(remove.has(v.key))continue;throw new SyncError("SYNC_METADATA_UNAVAILABLE",503);}
		const refs=(await obj.text()).split("\n").filter(Boolean);
		for(const id of refs)(remove.has(v.key)?candidates:keep).add(id);
	}
	const blocks:StoredBlock[]=[];let after="";
	for(;;){
		const page=await env.DB.prepare("SELECT id,size,coalesce(object_key,'u/'||user_id||'/b/'||id) AS objectKey,created_at AS createdAt FROM sync_block WHERE user_id=? AND id>? ORDER BY id LIMIT 500").bind(userId,after).all<StoredBlock>();
		if(!page.results.length)break;
		for(const b of page.results)if(!keep.has(b.id)&&(candidates.has(b.id)||b.createdAt<now-86_400_000))blocks.push(b);
		after=page.results[page.results.length-1]?.id??after;
	}
	return {versions:drop,blocks};
}

// Logical removal and deletion queue are one D1 transaction. R2 bytes
// remain charged until deletion succeeds, even when R2 is unavailable.
export function cleanupWrites(db:D1Database,userId:string,plan:CleanupPlan,now=Date.now()):D1PreparedStatement[]{
	const writes:D1PreparedStatement[]=[];
	for(const v of plan.versions)writes.push(db.prepare("DELETE FROM vault_version WHERE vault_id=? AND n=? AND key=?").bind(v.vaultId,v.n,v.key));
	for(let i=0;i<plan.blocks.length;i+=99){const part=plan.blocks.slice(i,i+99);writes.push(db.prepare(`DELETE FROM sync_block WHERE user_id=? AND id IN (${part.map(()=>"?").join(",")})`).bind(userId,...part.map(b=>b.id)));}
	const objects=[...plan.versions.map(v=>({keys:[`${v.key}/m`,`${v.key}/r`],size:v.size})),...plan.blocks.map(b=>({keys:[b.objectKey],size:b.size}))];
	for(let i=0;i<objects.length;i+=250){const part=objects.slice(i,i+250);writes.push(db.prepare("INSERT INTO sync_garbage(id,user_id,keys,bytes,created_at) VALUES (?,?,?,?,?)").bind(crypto.randomUUID(),userId,JSON.stringify(part.flatMap(p=>p.keys)),part.reduce((n,p)=>n+p.size,0),now));}
	return writes;
}
export function leaseGuard(db:D1Database,userId:string,token:string){
	return db.prepare("UPDATE resource_quota SET storage=CASE WHEN deleted_at IS NULL AND EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?) THEN storage ELSE NULL END WHERE user_id=?").bind(syncLockKey(userId),token,Date.now(),userId);
}
export async function drainSyncGarbage(env:Cloudflare.Env,userId?:string){
	let freed=0;
	const rows=await env.DB.prepare(`SELECT id,user_id AS userId,keys,bytes FROM sync_garbage ${userId?"WHERE user_id=?":""} ORDER BY created_at LIMIT 25`).bind(...(userId?[userId]:[])).all<{id:string;userId:string;keys:string;bytes:number}>();
	for(const row of rows.results){
		try{
			const keys=z.array(z.string()).max(500).parse(JSON.parse(row.keys));
			await env.NOTES.delete(keys);
			const settled=await env.DB.batch([
				env.DB.prepare("DELETE FROM sync_garbage WHERE id=? RETURNING id").bind(row.id),
				env.DB.prepare("UPDATE sync_account SET storage_bytes=max(0,storage_bytes-?) WHERE user_id=? AND changes()>0").bind(row.bytes,row.userId),
			]);
			if(settled[0]?.meta.changes)freed+=row.bytes;
		}catch{console.error("[blup] sync_garbage_retry");}
	}
	return freed;
}
export function headReferenceWrites(db:D1Database,vaultId:string,refs:string[]):D1PreparedStatement[]{
	const writes=[db.prepare("DELETE FROM vault_head_block WHERE vault_id=?").bind(vaultId)];
	for(let i=0;i<refs.length;i+=49){const part=refs.slice(i,i+49);writes.push(db.prepare(`INSERT INTO vault_head_block(vault_id,block_id) VALUES ${part.map(()=>"(?,?)").join(",")}`).bind(...part.flatMap(id=>[vaultId,id])));}
	writes.push(db.prepare("UPDATE vault SET head_refs_ready=1 WHERE id=?").bind(vaultId));
	return writes;
}
export async function ensureHeadReferences(env:Cloudflare.Env,userId:string){
	const rows=await env.DB.prepare("SELECT v.id,v.head,vv.key FROM vault v LEFT JOIN vault_version vv ON vv.vault_id=v.id AND vv.n=v.head WHERE v.user_id=? AND v.head_refs_ready=0").bind(userId).all<{id:string;head:number;key:string|null}>();
	for(const row of rows.results){
		if(row.head>0&&!row.key)throw new SyncError("SYNC_METADATA_UNAVAILABLE",503);
		const refs=row.key?await refsOf(env,row.key):[];
		await env.DB.batch(headReferenceWrites(env.DB,row.id,refs));
	}
}
