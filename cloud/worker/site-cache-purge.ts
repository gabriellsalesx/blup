// Tags contain only server-generated identifiers, never a supplied host.
export function siteTags(siteId:string,deploy?:string){
	return [`site:${siteId}`,...(deploy?[`deploy:${siteId}:${deploy}`]:[])];
}
export function queueSitePurge(db:D1Database,siteId:string){
	return db.prepare("INSERT INTO site_cache_purge(site_id,generation,queued_at) VALUES (?,?,?) ON CONFLICT(site_id) DO UPDATE SET generation=excluded.generation,queued_at=excluded.queued_at").bind(siteId,crypto.randomUUID(),Date.now());
}
export function queueAccountPurge(db:D1Database,userId:string){
	return db.prepare("INSERT INTO site_cache_purge(site_id,generation,queued_at) SELECT id,?,? FROM site WHERE user_id=? ON CONFLICT(site_id) DO UPDATE SET generation=excluded.generation,queued_at=excluded.queued_at").bind(crypto.randomUUID(),Date.now(),userId);
}

// One global claim per minute, at most 100 tags: below the Free-tier
// limit (5/minute). Failed/limited purges stay queued. A concurrent event
// changes generation and cannot be acknowledged by an older purge.
export async function drainSitePurges(db:D1Database,cache:CacheContext|undefined,now=Date.now()){
	if(!cache)return;
	const rows=await db.prepare("SELECT site_id AS id,generation FROM site_cache_purge ORDER BY queued_at LIMIT 100").all<{id:string;generation:string}>();
	if(!rows.results.length)return;
	const won=await db.prepare("INSERT INTO site_cache_purge_state(id,next_at) VALUES ('global',?) ON CONFLICT(id) DO UPDATE SET next_at=excluded.next_at WHERE next_at<=? RETURNING id").bind(now+60_000,now).first();
	if(!won)return;
	try{
		const result=await cache.purge({tags:rows.results.map(r=>siteTags(r.id)[0])});
		if(!result.success){console.error("[blup] site_cache_purge_retry");return;}
		await db.batch(rows.results.map(r=>db.prepare("DELETE FROM site_cache_purge WHERE site_id=? AND generation=?").bind(r.id,r.generation)));
	}catch{console.error("[blup] site_cache_purge_retry");}
}

export async function expiredSitePurges(db:D1Database,now=Date.now()){
	const rows=await db.prepare("SELECT user_id AS id,grace_until AS deadline FROM resource_quota WHERE grace_until<=? AND site_cache_grace IS NOT grace_until AND deleted_at IS NULL ORDER BY grace_until LIMIT 25").bind(now).all<{id:string;deadline:number}>();
	for(const r of rows.results)await db.batch([
		queueAccountPurge(db,r.id),
		db.prepare("UPDATE resource_quota SET site_cache_grace=? WHERE user_id=? AND grace_until=?").bind(r.deadline,r.id,r.deadline),
	]);
}

export async function flushSitePurges(env:Cloudflare.Env){
	await expiredSitePurges(env.DB);
	if(!env.SITE_CACHE)return; // Local API tests may run without the other Worker.
	try{await env.SITE_CACHE.purgePending();}catch{console.error("[blup] site_cache_purge_retry");}
}
