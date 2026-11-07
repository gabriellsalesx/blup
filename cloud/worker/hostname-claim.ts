// A single atomic namespace covers Hugo sites, Pulse pages and Deals
// pages. Pending TXT challenges do not reserve somebody else's domain.
export type HostnameKind="site"|"pulse"|"deals";
export async function claimHostname(db:D1Database,hostname:string,kind:HostnameKind,resourceId:string){
 await db.prepare("INSERT OR IGNORE INTO hostname_claim(hostname,kind,resource_id) SELECT ?1,?2,?3 WHERE NOT EXISTS(SELECT 1 FROM site_domain WHERE hostname=?1 AND verified_at IS NOT NULL AND NOT(?2='site' AND site_id=?3)) AND NOT EXISTS(SELECT 1 FROM pulse_page_domain WHERE hostname=?1 AND verified_at IS NOT NULL AND NOT(?2='pulse' AND page_id=?3)) AND NOT EXISTS(SELECT 1 FROM deals_site_domain WHERE hostname=?1 AND verified_at IS NOT NULL AND NOT(?2='deals' AND site_id=?3))").bind(hostname,kind,resourceId).run();
 const claim=await db.prepare("SELECT kind,resource_id AS resourceId FROM hostname_claim WHERE hostname=?").bind(hostname).first<{kind:string;resourceId:string}>();return claim?.kind===kind&&claim.resourceId===resourceId;
}
export function releaseHostname(db:D1Database,hostname:string,kind:HostnameKind,resourceId:string){return db.prepare("DELETE FROM hostname_claim WHERE hostname=? AND kind=? AND resource_id=?").bind(hostname,kind,resourceId);}
