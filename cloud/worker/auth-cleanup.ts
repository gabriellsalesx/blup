// Better Auth drops an expired session only when someone presents it, and
// a code or passkey challenge only when it is used: the rest stays in D1.
// Studio sign in codes (device_code, 15 minutes) that nobody finished too.
// A day past expiry, in bounded batches (no index on expires_at: these
// generated tables are small and the scan runs once a day).
export async function cleanExpiredAuth(db:D1Database,now=Date.now()){
 const before=now-86_400_000;
 for(const table of ["session","verification","device_code"] as const){
  for(let round=0;round<20;round++){
   const done=await db.prepare(`DELETE FROM ${table} WHERE id IN (SELECT id FROM ${table} WHERE expires_at<? LIMIT 500)`).bind(before).run();
   if((done.meta.changes??0)<500)break;
  }
 }
}
