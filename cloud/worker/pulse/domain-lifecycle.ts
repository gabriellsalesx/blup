import {DomainError,removeDomainProvider} from "../domain-provider";
import {releaseHostname} from "../hostname-claim";
import {siteLock,unlockSite} from "../quota";
import type {PulseDomain} from "./domains";
async function read(env:Cloudflare.Env,page:string){return env.DB.prepare("SELECT * FROM pulse_page_domain WHERE page_id=?").bind(page).first<PulseDomain>();}
export async function disablePulseDomain(env:Cloudflare.Env,page:string,remove=false,onlyIneligible=false){
 const lock=await siteLock(env.DB,`pulse-domain:${page}`,60_000);if(!lock)throw new DomainError("DOMAIN_PROVIDER");
 try{
  const row=await read(env,page);if(!row)return;
  if(onlyIneligible){
   const owner=(env.STAFF_OWNER_EMAIL??" ").trim().toLowerCase();
   const claimed=await env.DB.prepare("UPDATE pulse_page_domain SET status='disabled',ssl='pending' WHERE page_id=? AND EXISTS(SELECT 1 FROM pulse_page p JOIN user u ON u.id=p.user_id LEFT JOIN resource_quota q ON q.user_id=u.id WHERE p.id=? AND (p.plan_disabled=1 OR u.banned=1 OR q.deleted_at IS NOT NULL OR NOT(u.email_verified=1 AND lower(u.email)=?) AND NOT EXISTS(SELECT 1 FROM subscription s WHERE s.reference_id=u.id AND s.plan='ultra_pulse' AND s.status IN('active','trialing','past_due')))) RETURNING page_id").bind(page,page,owner).first();if(!claimed)return;
  }
  // Stop serving first. Preserve the provider ID until deletion succeeds.
  await env.DB.prepare("UPDATE pulse_page_domain SET status='disabled',ssl='pending' WHERE page_id=?").bind(page).run();
  await removeDomainProvider(env,{hostname:row.hostname,providerId:row.provider_id,verifiedAt:row.verified_at});
  await env.DB.batch([releaseHostname(env.DB,row.hostname,"pulse",page),remove?env.DB.prepare("DELETE FROM pulse_page_domain WHERE page_id=?").bind(page):env.DB.prepare("UPDATE pulse_page_domain SET provider_id=NULL,verified_at=NULL,records='[]',checked_at=NULL WHERE page_id=?").bind(page)]);
 }finally{await unlockSite(env.DB,`pulse-domain:${page}`,lock);}
}
export async function expirePulseDomains(env:Cloudflare.Env){
 const owner=(env.STAFF_OWNER_EMAIL??"").trim().toLowerCase();
 const rows=await env.DB.prepare("SELECT d.page_id AS id FROM pulse_page_domain d JOIN pulse_page p ON p.id=d.page_id JOIN user u ON u.id=p.user_id LEFT JOIN resource_quota q ON q.user_id=u.id WHERE (d.status<>'disabled' OR d.provider_id IS NOT NULL) AND (p.plan_disabled=1 OR u.banned=1 OR q.deleted_at IS NOT NULL OR NOT(u.email_verified=1 AND lower(u.email)=?) AND NOT EXISTS(SELECT 1 FROM subscription s WHERE s.reference_id=u.id AND s.plan='ultra_pulse' AND s.status IN('active','trialing','past_due'))) LIMIT 25").bind(owner).all<{id:string}>();
 for(const row of rows.results)await disablePulseDomain(env,row.id,false,true);
}
