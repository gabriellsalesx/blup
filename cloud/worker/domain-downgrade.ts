import { queueSitePurge } from "./site-cache-purge";
import { accountPlan, observePlan } from "./lifecycle";
import { siteLock, unlockSite } from "./quota";
import { removeDomainProvider } from "./domain-provider";

export async function disableExpiredDomain(env:Cloudflare.Env,siteId:string,now=Date.now()){
	const lock=await siteLock(env.DB,`domain:${siteId}`,60_000);
	if(!lock)return false;
	try{
		const row=await env.DB.prepare("SELECT d.hostname,d.provider_id AS providerId,d.verified_at AS verifiedAt,d.disable_at AS disableAt,d.status,s.user_id AS owner FROM site_domain d JOIN site s ON s.id=d.site_id WHERE d.site_id=?").bind(siteId).first<{hostname:string;providerId:string|null;verifiedAt:number|null;disableAt:number|null;status:string;owner:string}>();
		if(!row||row.disableAt===null||row.disableAt>now)return false;
		const plan=await accountPlan(env,row.owner);
		if(plan?.limits.customDomain&&row.status!=="disabled"){
			await observePlan(env.DB,row.owner,plan.name,now);
			return false;
		}
		const claimed=await env.DB.prepare("UPDATE site_domain SET status='disabled' WHERE site_id=? AND disable_at<=? RETURNING hostname,provider_id AS providerId,verified_at AS verifiedAt").bind(siteId,now).first<{hostname:string;providerId:string|null;verifiedAt:number|null}>();
		if(!claimed)return false;
		await queueSitePurge(env.DB,siteId).run();
		// Keep the ID/proof until this succeeds, including a lost provider ID.
		await removeDomainProvider(env,claimed);
		await env.DB.prepare("UPDATE site_domain SET provider_id=NULL,verified_at=NULL,ssl='pending',records='[]',checked_at=NULL WHERE site_id=? AND status='disabled'").bind(siteId).run();
		return true;
	}finally{await unlockSite(env.DB,`domain:${siteId}`,lock);}
}

export async function expireDomains(env:Cloudflare.Env,now=Date.now()){
	const rows=await env.DB.prepare("SELECT site_id AS id FROM site_domain WHERE disable_at<=? AND (status<>'disabled' OR provider_id IS NOT NULL OR verified_at IS NOT NULL) ORDER BY disable_at LIMIT 25").bind(now).all<{id:string}>();
	for(const row of rows.results){
		try{await disableExpiredDomain(env,row.id,now);}
		catch{console.error("[blup] domain_disable_retry");}
	}
}
