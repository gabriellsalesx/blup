import {plansWithPulse,sqlPlans} from "../../shared/pulse-plans";
import {dropSourcemaps,SOURCEMAP_KEEP_MS} from "./sourcemap-store";
import {dropPageAssets} from "./page-asset-store";
import {observePulseEntitlements} from "./entitlements";
import {disablePulseDomain,expirePulseDomains} from "./domain-lifecycle";
import {releaseHostname} from "../hostname-claim";
import {removeDomainProvider} from "../domain-provider";

export async function purgeProject(env:Cloudflare.Env,id:string,objectName:string){
 await env.PULSE_PROJECT.getByName(objectName).purge();
 await dropSourcemaps(env,{project:id});
 await env.DB.batch([
  env.DB.prepare("DELETE FROM pulse_component WHERE monitor_id IN(SELECT id FROM pulse_monitor WHERE project_id=?)").bind(id),
  env.DB.prepare("DELETE FROM pulse_monitor WHERE project_id=?").bind(id),
  env.DB.prepare("DELETE FROM pulse_outbox WHERE project_id=? OR channel_id IN(SELECT channel_id FROM pulse_project_channel WHERE project_id=?)").bind(id,id),
  // The project's destinations leave with it (each belongs to one project).
  env.DB.prepare("DELETE FROM pulse_cooldown WHERE substr(key,1,instr(key,':')-1) IN(SELECT channel_id FROM pulse_project_channel WHERE project_id=?)").bind(id),
  env.DB.prepare("DELETE FROM pulse_channel WHERE id IN(SELECT channel_id FROM pulse_project_channel WHERE project_id=?)").bind(id),
  env.DB.prepare("DELETE FROM pulse_project_channel WHERE project_id=?").bind(id),
  env.DB.prepare("DELETE FROM pulse_key WHERE project_id=?").bind(id),
  env.DB.prepare("DELETE FROM pulse_upload_token WHERE project_id=?").bind(id),
  env.DB.prepare("DELETE FROM pulse_project WHERE id=?").bind(id),
 ]);
}
export async function purgePulseAccount(env:Cloudflare.Env,userId:string,keepAdmission=false){
 const projects=await env.DB.prepare("SELECT id,object_name AS objectName FROM pulse_project WHERE user_id=?").bind(userId).all<{id:string;objectName:string}>();
 for(const p of projects.results)await purgeProject(env,p.id,p.objectName);
 const domains=await env.DB.prepare("SELECT d.page_id AS pageId,d.hostname,d.provider_id AS providerId,d.verified_at AS verifiedAt FROM pulse_page_domain d JOIN pulse_page p ON p.id=d.page_id WHERE p.user_id=?").bind(userId).all<{pageId:string;hostname:string;providerId:string|null;verifiedAt:number|null}>();
 for(const domain of domains.results){await removeDomainProvider(env,domain);await releaseHostname(env.DB,domain.hostname,"pulse",domain.pageId).run();}
 for(const page of (await env.DB.prepare("SELECT id FROM pulse_page WHERE user_id=?").bind(userId).all<{id:string}>()).results)await dropPageAssets(env,page.id);
 await env.DB.batch([
  env.DB.prepare("DELETE FROM pulse_incident_update WHERE incident_id IN(SELECT i.id FROM pulse_incident i JOIN pulse_page p ON p.id=i.page_id WHERE p.user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_incident_component WHERE incident_id IN(SELECT i.id FROM pulse_incident i JOIN pulse_page p ON p.id=i.page_id WHERE p.user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_incident WHERE page_id IN(SELECT id FROM pulse_page WHERE user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_maintenance_component WHERE maintenance_id IN(SELECT m.id FROM pulse_maintenance m JOIN pulse_page p ON p.id=m.page_id WHERE p.user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_maintenance WHERE page_id IN(SELECT id FROM pulse_page WHERE user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_component WHERE page_id IN(SELECT id FROM pulse_page WHERE user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_page_domain WHERE page_id IN(SELECT id FROM pulse_page WHERE user_id=?)").bind(userId),
  env.DB.prepare("DELETE FROM pulse_cooldown WHERE key IN(SELECT cd.key FROM pulse_cooldown cd JOIN pulse_channel ch ON cd.key LIKE ch.id||':%' WHERE ch.user_id=?)").bind(userId),
  ...["pulse_channel","pulse_telegram_link","pulse_outbox","pulse_page",...(keepAdmission?[]:["pulse_month","pulse_grant"]),"pulse_account"].map(table=>env.DB.prepare(`DELETE FROM ${table} WHERE user_id=?`).bind(userId)),
 ]);
}
export async function pulseMaintenance(env:Cloudflare.Env,daily=false,now=Date.now()){
 // Recover missed application work without rescanning all event data.
 const changes=await env.DB.prepare(`SELECT user_id AS id FROM pulse_selection WHERE applied_at IS NULL AND effective_at<=? UNION SELECT a.user_id AS id FROM pulse_account a WHERE a.read_only_until IS NULL AND NOT EXISTS(SELECT 1 FROM subscription s WHERE s.reference_id=a.user_id AND s.plan IN(${sqlPlans(plansWithPulse())}) AND s.status IN('active','trialing','past_due')) LIMIT 25`).bind(now).all<{id:string}>();
 for(const account of changes.results)await observePulseEntitlements(env,account.id,now);
 await expirePulseDomains(env);
 const retired=await env.DB.prepare("SELECT id,object_name AS objectName FROM pulse_project WHERE retire_at<=? ORDER BY retire_at LIMIT 25").bind(now).all<{id:string;objectName:string}>();
 for(const project of retired.results){await env.DB.prepare("UPDATE pulse_project SET deleted_at=? WHERE id=? AND retire_at<=?").bind(now,project.id,now).run();await purgeProject(env,project.id,project.objectName);}
 const pages=await env.DB.prepare("SELECT id FROM pulse_page WHERE retire_at<=? ORDER BY retire_at LIMIT 25").bind(now).all<{id:string}>();
 for(const page of pages.results){await disablePulseDomain(env,page.id,true);await dropPageAssets(env,page.id);await env.DB.batch([env.DB.prepare("DELETE FROM pulse_incident_update WHERE incident_id IN(SELECT id FROM pulse_incident WHERE page_id=?)").bind(page.id),env.DB.prepare("DELETE FROM pulse_incident_component WHERE incident_id IN(SELECT id FROM pulse_incident WHERE page_id=?)").bind(page.id),env.DB.prepare("DELETE FROM pulse_maintenance_component WHERE maintenance_id IN(SELECT id FROM pulse_maintenance WHERE page_id=?)").bind(page.id),...["pulse_incident","pulse_maintenance","pulse_component"].map(table=>env.DB.prepare(`DELETE FROM ${table} WHERE page_id=?`).bind(page.id)),env.DB.prepare("DELETE FROM pulse_page WHERE id=? AND retire_at<=?").bind(page.id,now)]);}
 const monitors=await env.DB.prepare("SELECT id FROM pulse_monitor WHERE retire_at<=? ORDER BY retire_at LIMIT 50").bind(now).all<{id:string}>();
 for(const monitor of monitors.results)await env.DB.batch([env.DB.prepare("DELETE FROM pulse_component WHERE monitor_id=?").bind(monitor.id),env.DB.prepare("DELETE FROM pulse_monitor WHERE id=? AND retire_at<=?").bind(monitor.id,now)]);
 const due=await env.DB.prepare("SELECT user_id AS id FROM pulse_account WHERE read_only_until<=? ORDER BY read_only_until LIMIT 10").bind(now).all<{id:string}>();
 for(const account of due.results){const claimed=await env.DB.prepare("UPDATE pulse_account SET purge_started=? WHERE user_id=? AND read_only_until<=? RETURNING user_id").bind(now,account.id,now).first();if(claimed)await purgePulseAccount(env,account.id,true);}
 if(daily)await dropSourcemaps(env,{before:now-SOURCEMAP_KEEP_MS});
 const deleted=await env.DB.prepare("SELECT id,object_name AS objectName FROM pulse_project WHERE deleted_at IS NOT NULL ORDER BY deleted_at LIMIT 25").all<{id:string;objectName:string}>();
 for(const p of deleted.results)await purgeProject(env,p.id,p.objectName);
 {
  const projects=await env.DB.prepare("SELECT id,object_name AS objectName FROM pulse_project WHERE cleaned_at<? AND deleted_at IS NULL ORDER BY cleaned_at LIMIT ?").bind(now-23*60*60_000,daily?100:25).all<{id:string;objectName:string}>();
  for(const project of projects.results){await env.PULSE_PROJECT.getByName(project.objectName).cleanup(project.id);await env.DB.prepare("UPDATE pulse_project SET cleaned_at=? WHERE id=?").bind(now,project.id).run();}
  const cutoff=new Date(now-95*86_400_000).toISOString().slice(0,7);
  // Bound old configuration/admission receipts too. Never refund a grant.
  await env.DB.batch([
   env.DB.prepare("DELETE FROM pulse_billing_ticket WHERE id IN(SELECT id FROM pulse_billing_ticket WHERE expires_at<? LIMIT 500)").bind(now-400*86_400_000),
   env.DB.prepare("DELETE FROM billing_checkout WHERE user_id IN(SELECT user_id FROM billing_checkout WHERE expires_at<=? LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_month WHERE (user_id,month) IN(SELECT user_id,month FROM pulse_month WHERE month<? LIMIT 500)").bind(cutoff),
   env.DB.prepare("DELETE FROM pulse_outbox WHERE id IN(SELECT id FROM pulse_outbox WHERE expires_at<=? LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_telegram_link WHERE id IN(SELECT id FROM pulse_telegram_link WHERE expires_at<=? LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_telegram_update WHERE id IN(SELECT id FROM pulse_telegram_update WHERE received_at<=? LIMIT 500)").bind(now-86_400_000),
   env.DB.prepare("DELETE FROM pulse_cooldown WHERE key IN(SELECT key FROM pulse_cooldown WHERE until<=? LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_grant WHERE id IN(SELECT id FROM pulse_grant WHERE month<? LIMIT 500)").bind(cutoff),
   env.DB.prepare("DELETE FROM pulse_incident_update WHERE id IN(SELECT id FROM pulse_incident_update WHERE expires_at<=? LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_incident_component WHERE incident_id IN(SELECT id FROM pulse_incident WHERE expires_at<=? ORDER BY id LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_incident WHERE id IN(SELECT id FROM pulse_incident WHERE expires_at<=? ORDER BY id LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_maintenance_component WHERE maintenance_id IN(SELECT id FROM pulse_maintenance WHERE expires_at<=? ORDER BY id LIMIT 500)").bind(now),
   env.DB.prepare("DELETE FROM pulse_maintenance WHERE id IN(SELECT id FROM pulse_maintenance WHERE expires_at<=? ORDER BY id LIMIT 500)").bind(now),
  ]);
 }
}
