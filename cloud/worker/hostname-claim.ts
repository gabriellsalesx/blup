// A single atomic namespace covers both Hugo sites and Pulse pages.
// Pending TXT challenges do not reserve somebody else's domain.
export async function claimHostname(db:D1Database,hostname:string,kind:"site"|"pulse",resourceId:string){
 await db.prepare("INSERT OR IGNORE INTO hostname_claim(hostname,kind,resource_id) SELECT ?,?,? WHERE NOT EXISTS(SELECT 1 FROM site_domain WHERE hostname=? AND verified_at IS NOT NULL AND NOT(?='site' AND site_id=?)) AND NOT EXISTS(SELECT 1 FROM pulse_page_domain WHERE hostname=? AND verified_at IS NOT NULL AND NOT(?='pulse' AND page_id=?))").bind(hostname,kind,resourceId,hostname,kind,resourceId,hostname,kind,resourceId).run();
 const claim=await db.prepare("SELECT kind,resource_id AS resourceId FROM hostname_claim WHERE hostname=?").bind(hostname).first<{kind:string;resourceId:string}>();return claim?.kind===kind&&claim.resourceId===resourceId;
}
export function releaseHostname(db:D1Database,hostname:string,kind:"site"|"pulse",resourceId:string){return db.prepare("DELETE FROM hostname_claim WHERE hostname=? AND kind=? AND resource_id=?").bind(hostname,kind,resourceId);}
