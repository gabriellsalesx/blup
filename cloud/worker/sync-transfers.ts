import { z } from "zod";
import { reserve,release,settle,siteLock,unlockSite } from "./quota";
import { cleanupPlan,cleanupSchema,cleanupWrites,drainSyncGarbage,ensureHeadReferences,headReferenceWrites,leaseGuard,lockSync,storageLimit,syncLockKey,SyncError,unlockSync,versionsOf } from "./sync-storage";
import { deletePrefix } from "./lifecycle";
import { hashId,safeInt } from "./validation";
import type { Plan } from "./plans";

const BLOCK_MAX=4*1024*1024+64,TTL=30*60_000;
// Small encrypted-manifest growth for a rename, not an arbitrary upload.
const MANIFEST_ALLOWANCE=4*1024;
const declaredBlock=z.object({id:hashId,size:z.number().int().min(0).max(BLOCK_MAX)}).strict();
const blockList=z.string().max(1_900_000).transform((value,ctx):unknown=>{
	try{return JSON.parse(value);}catch{ctx.addIssue({code:"custom",message:"blocks"});return z.NEVER;}
}).pipe(z.array(declaredBlock).max(20_000).refine(a=>new Set(a.map(b=>b.id)).size===a.length));
export const preparationSchema=z.object({parent:safeInt,blocks:blockList,manifest:z.instanceof(File).refine(f=>f.size>0&&f.size<=32*1024*1024)});
const transferPlanSchema=z.object({refs:z.array(hashId).max(20_000),cleanup:cleanupSchema,newBytes:z.number().int().min(0)});
type Transfer={id:string;user_id:string;vault_id:string;parent:number;key:string;size:number;reservation_id:string;reserved:number;lease:string;expires_at:number;created_at:number;completed_version:number|null};
async function owned(env:Cloudflare.Env,userId:string,id:string,allowExpired=false){
	const row=await env.DB.prepare("SELECT * FROM sync_transfer WHERE id=? AND user_id=?").bind(id,userId).first<Transfer>();
	if(!row)throw new SyncError("NOT_FOUND",404);
	if(!allowExpired&&(row.completed_version===0||row.completed_version===null&&row.expires_at<=Date.now()))throw new SyncError("TICKET_EXPIRED",410);
	return row;
}
async function renew(env:Cloudflare.Env,row:Transfer){
	const until=Math.min(Date.now()+TTL,row.created_at+24*3600_000);
	if(until<=Date.now())throw new SyncError("TICKET_EXPIRED",410);
	const res=await env.DB.batch([
		env.DB.prepare("UPDATE site_lease SET expires_at=? WHERE site_id=? AND token=? AND expires_at>? RETURNING token").bind(until,syncLockKey(row.user_id),row.lease,Date.now()),
		env.DB.prepare("UPDATE sync_transfer SET expires_at=? WHERE id=? AND completed_version IS NULL AND changes()>0").bind(until,row.id),
		env.DB.prepare("UPDATE resource_reservation SET expires_at=? WHERE id=? AND EXISTS(SELECT 1 FROM sync_transfer WHERE id=? AND expires_at=?)").bind(until,row.reservation_id,row.id,until),
	]);
	if(!res[0]?.results.length)throw new SyncError("TICKET_EXPIRED",410);
	return until;
}
export async function prepareTransfer(env:Cloudflare.Env,userId:string,vaultId:string,input:z.infer<typeof preparationSchema>,plan:Plan){
	const lease=await lockSync(env.DB,userId,TTL);if(!lease)throw new SyncError("SYNC_BUSY",409);
	const ticket=crypto.randomUUID(),key=`u/${userId}/t/${ticket}`,now=Date.now();
	let hold:Awaited<ReturnType<typeof reserve>>=null;
	try{
		const vault=await env.DB.prepare("SELECT head FROM vault WHERE id=? AND user_id=?").bind(vaultId,userId).first<{head:number}>();
		if(!vault)throw new SyncError("NOT_FOUND",404);
		if(vault.head!==input.parent)throw new SyncError("MOVED",409,vault.head);
		if(!await env.DB.prepare("SELECT user_id FROM sync_account WHERE user_id=?").bind(userId).first())throw new SyncError("NO_KEY",409);
		await ensureHeadReferences(env,userId);
		const existing=new Map<string,number>();
		for(let i=0;i<input.blocks.length;i+=99){const part=input.blocks.slice(i,i+99);const rows=await env.DB.prepare(`SELECT id,size FROM sync_block WHERE user_id=? AND id IN (${part.map(()=>"?").join(",")})`).bind(userId,...part.map(b=>b.id)).all<{id:string;size:number}>();for(const b of rows.results)existing.set(b.id,b.size);}
		const missing=input.blocks.filter(b=>!existing.has(b.id));
		if(missing.some(b=>b.size===0))throw new SyncError("INVALID",400);
		const refs=input.blocks.map(b=>b.id),refBytes=new TextEncoder().encode(refs.join("\n")),size=input.manifest.size+refBytes.byteLength;
		const versions=await versionsOf(env.DB,userId),cutoff=now-plan.limits.historyDays*86_400_000;
		const drop=versions.filter(v=>v.vaultId===vaultId&&(plan.limits.historyDays===0||v.createdAt<cutoff));
		const cleanup=await cleanupPlan(env,userId,drop,refs,now);
		const credit=cleanup.versions.reduce((n,v)=>n+v.size,0)+cleanup.blocks.reduce((n,b)=>n+b.size,0);
		const newBytes=size+missing.reduce((n,b)=>n+b.size,0);
		const limit=await storageLimit(env.DB,userId,plan);
		const used=await env.DB.prepare("SELECT coalesce((SELECT storage_bytes FROM sync_account WHERE user_id=?),0)+coalesce((SELECT sum(bytes) FROM site WHERE user_id=?),0) AS bytes").bind(userId,userId).first<{bytes:number}>();
		let amount=Math.max(0,newBytes-credit);
		if(missing.length===0&&(used?.bytes??0)>limit){
			const head=versions.find(v=>v.vaultId===vaultId&&v.n===vault.head);
			const previous=head?await env.NOTES.head(`${head.key}/m`):null;
			if(vault.head>0&&!previous)throw new SyncError("SYNC_METADATA_UNAVAILABLE",503);
			// Compare the sealed manifest alone, not combined manifest+refs or
			// a caller-supplied size. Removing references cannot inflate it.
			if(input.manifest.size>(previous?.size??0)+MANIFEST_ALLOWANCE)throw new SyncError("QUOTA",413);
			amount=0;
		}
		hold=await reserve(env.DB,userId,"storage",amount,limit,undefined,TTL);
		if(!hold)throw new SyncError("QUOTA",413);
		await env.NOTES.put(`${key}/m`,input.manifest.stream());
		await env.NOTES.put(`${key}/r`,refBytes);
		await env.NOTES.put(`${key}/p`,JSON.stringify({refs,cleanup,newBytes}));
		const writes=[leaseGuard(env.DB,userId,lease),env.DB.prepare("INSERT INTO sync_transfer(id,user_id,vault_id,parent,key,size,reservation_id,reserved,lease,expires_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)").bind(ticket,userId,vaultId,input.parent,key,size,hold.id,amount,lease,now+TTL,now)];
		for(let i=0;i<missing.length;i+=33){const part=missing.slice(i,i+33);writes.push(env.DB.prepare(`INSERT INTO sync_transfer_block(ticket_id,block_id,size) VALUES ${part.map(()=>"(?,?,?)").join(",")}`).bind(...part.flatMap(b=>[ticket,b.id,b.size])));}
		await env.DB.batch(writes);
		return {ticket,expiresAt:new Date(now+TTL).toISOString(),missing:missing.map(b=>b.id)};
	}catch(e){await Promise.allSettled([deletePrefix(env.NOTES,`${key}/`),...(hold?[release(env.DB,hold)]:[]),unlockSync(env.DB,userId,lease)]);throw e;}
}

export async function putPreparedBlock(env:Cloudflare.Env,userId:string,ticket:string,id:string,body:Uint8Array<ArrayBuffer>){
 const lock=await siteLock(env.DB,`sync-block:${ticket}:${id}`);if(!lock)throw new SyncError("SYNC_BUSY",409);
 let physical:string|undefined,adopted=false,unknownAdoption=false;
 try{
  // A delayed duplicate may acquire the lock after another PUT committed.
  // Read state only after acquiring the block lock.
  const row=await owned(env,userId,ticket);
  if(row.completed_version!==null){
   if(await env.DB.prepare("SELECT id FROM sync_block WHERE user_id=? AND id=?").bind(userId,id).first())return {ok:true};
   throw new SyncError("NOT_FOUND",404);
  }
  const block=await env.DB.prepare("SELECT size,ready FROM sync_transfer_block WHERE ticket_id=? AND block_id=?").bind(ticket,id).first<{size:number;ready:number}>();
  if(!block||block.size!==body.byteLength)throw new SyncError("INVALID",400);
  await renew(env,row);
  if(block.ready)return {ok:true};
  // A lease can expire during R2 I/O. Never write to an already adopted
  // key: each attempt has its own key, and D1 elects exactly one receipt.
  physical=`${row.key}/b/${id}-${crypto.randomUUID()}`;
  await env.NOTES.put(physical,body);
  unknownAdoption=true;
  const result=await env.DB.batch([leaseGuard(env.DB,userId,row.lease),env.DB.prepare("UPDATE sync_transfer_block SET ready=1,object_key=? WHERE ticket_id=? AND block_id=? AND ready=0 RETURNING block_id").bind(physical,ticket,id)]);
  adopted=!!result[1]?.results.length;unknownAdoption=false;
  return {ok:true};
 }finally{
  // A lost D1 acknowledgement may already have adopted the object.
  // Keep it for ticket cancellation/expiry rather than deleting live bytes.
  if(physical&&!adopted&&!unknownAdoption)await env.NOTES.delete(physical).catch(()=>{});
  await unlockSite(env.DB,`sync-block:${ticket}:${id}`,lock);
 }
}
export async function commitTransfer(env:Cloudflare.Env,userId:string,vaultId:string,ticket:string){
	const commitLock=await siteLock(env.DB,`sync-commit:${ticket}`);if(!commitLock)throw new SyncError("SYNC_BUSY",409);
	try{
		const row=await owned(env,userId,ticket);
		if(row.vault_id!==vaultId)throw new SyncError("NOT_FOUND",404);
		if(row.completed_version!==null){await unlockSync(env.DB,userId,row.lease);return {n:row.completed_version};}
		await renew(env,row);
		const vault=await env.DB.prepare("SELECT head FROM vault WHERE id=? AND user_id=?").bind(vaultId,userId).first<{head:number}>();
		if(!vault)throw new SyncError("NOT_FOUND",404);
		if(vault.head!==row.parent)throw new SyncError("MOVED",409,vault.head);
		const notReady=await env.DB.prepare("SELECT block_id FROM sync_transfer_block WHERE ticket_id=? AND ready=0 LIMIT 1").bind(ticket).first();
		if(notReady)throw new SyncError("MISSING_BLOCKS",409);
		const stored=await env.NOTES.get(`${row.key}/p`);if(!stored)throw new SyncError("TICKET_EXPIRED",410);
		const plan=transferPlanSchema.parse(await stored.json());
		const blocks=await env.DB.prepare("SELECT block_id AS id,size,object_key AS objectKey FROM sync_transfer_block WHERE ticket_id=?").bind(ticket).all<{id:string;size:number;objectKey:string|null}>();
		if(blocks.results.some(b=>!b.objectKey))throw new SyncError("SYNC_METADATA_UNAVAILABLE",503);
		const n=row.parent+1,now=Date.now();
		const writes=[leaseGuard(env.DB,userId,row.lease),
			env.DB.prepare("UPDATE vault SET head=CASE WHEN head=? THEN head ELSE NULL END WHERE id=? AND user_id=?").bind(row.parent,vaultId,userId),
			env.DB.prepare("INSERT INTO vault_version(vault_id,n,key,size,created_at) VALUES (?,?,?,?,?)").bind(vaultId,n,row.key,row.size,now)];
		for(let i=0;i<blocks.results.length;i+=24){const part=blocks.results.slice(i,i+24);writes.push(env.DB.prepare(`INSERT INTO sync_block(user_id,id,size,object_key) VALUES ${part.map(()=>"(?,?,?,?)").join(",")}`).bind(...part.flatMap(b=>[userId,b.id,b.size,b.objectKey])));}
		writes.push(...cleanupWrites(env.DB,userId,plan.cleanup,now),...headReferenceWrites(env.DB,vaultId,plan.refs),
			env.DB.prepare("UPDATE vault SET head=?,updated_at=? WHERE id=?").bind(n,now,vaultId),
			env.DB.prepare("UPDATE sync_account SET storage_bytes=storage_bytes+?,last_sync_at=?,updated_at=? WHERE user_id=?").bind(plan.newBytes,now,now,userId),
			env.DB.prepare("UPDATE sync_transfer SET completed_version=?,expires_at=? WHERE id=?").bind(n,now+86_400_000,ticket),
			env.DB.prepare("DELETE FROM sync_transfer_block WHERE ticket_id=?").bind(ticket));
		await settle(env.DB,{id:row.reservation_id,userId,kind:"storage",amount:row.reserved},writes);
		await drainSyncGarbage(env,userId);
		await env.NOTES.delete(`${row.key}/p`).catch(()=>{});
		await unlockSync(env.DB,userId,row.lease);
		return {n};
	}finally{await unlockSite(env.DB,`sync-commit:${ticket}`,commitLock);}
}
export async function cancelTransfer(env:Cloudflare.Env,userId:string,ticket:string){
	const lock=await siteLock(env.DB,`sync-commit:${ticket}`);if(!lock)throw new SyncError("SYNC_BUSY",409);
	try{
		const row=await env.DB.prepare("SELECT * FROM sync_transfer WHERE id=? AND user_id=?").bind(ticket,userId).first<Transfer>();
		if(!row)throw new SyncError("NOT_FOUND",404);
		if(row.completed_version!==null&&row.completed_version>0)return;
		await env.DB.batch([env.DB.prepare("UPDATE sync_transfer SET completed_version=0,expires_at=? WHERE id=?").bind(Date.now()-1,ticket),env.DB.prepare("DELETE FROM sync_transfer_block WHERE ticket_id=?").bind(ticket)]);
		const cleanup=await Promise.allSettled([deletePrefix(env.NOTES,`${row.key}/`),release(env.DB,{id:row.reservation_id,userId,kind:"storage",amount:row.reserved}),unlockSync(env.DB,userId,row.lease)]);
		if(cleanup.some(r=>r.status==="rejected"))throw new SyncError("SYNC_CLEANUP_RETRY",503);
	}finally{await unlockSite(env.DB,`sync-commit:${ticket}`,lock);}
}
export async function expireTransfers(env:Cloudflare.Env,now=Date.now()){
	const rows=await env.DB.prepare("SELECT id,user_id AS userId,completed_version AS n,created_at AS createdAt FROM sync_transfer WHERE expires_at<=? ORDER BY expires_at LIMIT 25").bind(now).all<{id:string;userId:string;n:number|null;createdAt:number}>();
	for(const row of rows.results){
		if(row.n!==null&&row.n>0){await env.DB.prepare("DELETE FROM sync_transfer WHERE id=?").bind(row.id).run();continue;}
		try{
			await cancelTransfer(env,row.userId,row.id);
			if(row.createdAt<now-86_400_000)await env.DB.prepare("DELETE FROM sync_transfer WHERE id=? AND completed_version=0").bind(row.id).run();
			else await env.DB.prepare("UPDATE sync_transfer SET expires_at=? WHERE id=? AND completed_version=0").bind(now+300_000,row.id).run();
		}catch{
			await env.DB.prepare("UPDATE sync_transfer SET expires_at=? WHERE id=? AND completed_version=0").bind(now+300_000,row.id).run();
			console.error("[blup] sync_transfer_cleanup_retry");
		}
	}
}
