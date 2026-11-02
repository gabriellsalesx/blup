import {pulseAccess} from "./limits";
import {keepSchema} from "../../shared/pulse-selection";

export async function observePulseEntitlements(env:Cloudflare.Env,userId:string,now=Date.now()){
 const access=await pulseAccess(env,userId,now);if(!access.writable||!access.plan||!access.limits){
  await env.DB.prepare("UPDATE pulse_selection SET applied_at=? WHERE user_id=? AND applied_at IS NULL AND effective_at<=? AND target_plan NOT IN('max_pulse','ultra_pulse') AND NOT EXISTS(SELECT 1 FROM subscription WHERE reference_id=? AND plan IN('max_pulse','ultra_pulse') AND status IN('active','trialing','past_due'))").bind(now,userId,now,userId).run();return access;
 }
 const selection=await env.DB.prepare("SELECT target_plan AS plan,effective_at AS at,keep,applied_at AS appliedAt FROM pulse_selection WHERE user_id=?").bind(userId).first<{plan:string;at:number;keep:string;appliedAt:number|null}>();
 const tables=[{name:"pulse_project",field:"projects"},{name:"pulse_monitor",field:"monitors"},{name:"pulse_page",field:"pages"}] as const;
  const current="EXISTS(SELECT 1 FROM pulse_account a JOIN user u ON u.id=a.user_id WHERE a.user_id=? AND a.last_plan=? AND a.resource_plan=? AND ((u.email_verified=1 AND lower(u.email)=?) OR EXISTS(SELECT 1 FROM subscription s WHERE s.reference_id=a.user_id AND s.plan=? AND s.status IN('active','trialing','past_due'))))",guard=[userId,access.plan,access.plan,(env.STAFF_OWNER_EMAIL??"").trim().toLowerCase(),access.plan];
 if(selection&&selection.plan===access.plan&&selection.at<=now&&selection.appliedAt===null){
  const keep=keepSchema.parse(JSON.parse(selection.keep));

  await env.DB.batch([
   ...tables.map(t=>env.DB.prepare(`UPDATE ${t.name} SET plan_disabled=CASE WHEN id IN(SELECT value FROM json_each(?)) THEN 0 ELSE 1 END,retire_at=CASE WHEN id IN(SELECT value FROM json_each(?)) THEN NULL ELSE coalesce(retire_at,?) END WHERE user_id=? AND ${current} AND EXISTS(SELECT 1 FROM pulse_selection WHERE user_id=? AND target_plan=? AND applied_at IS NULL)`).bind(JSON.stringify(keep[t.field]),JSON.stringify(keep[t.field]),now+14*86_400_000,userId,...guard,userId,access.plan)),
   env.DB.prepare(`UPDATE pulse_selection SET applied_at=? WHERE user_id=? AND target_plan=? AND applied_at IS NULL AND ${current}`).bind(now,userId,access.plan,...guard),
  ]);
 }else if(!(selection&&selection.plan===access.plan&&selection.appliedAt!==null)){
  // A paid upgrade restores resources only while their retention window
  // is still open, and only if all of them fit. Never revive expired data.
  for(const t of tables){
   await env.DB.prepare(`UPDATE ${t.name} SET plan_disabled=0,retire_at=NULL WHERE user_id=? AND plan_disabled=1 AND retire_at>? AND (SELECT count(*) FROM ${t.name} WHERE user_id=? AND (retire_at IS NULL OR retire_at>?))<=? AND NOT EXISTS(SELECT 1 FROM pulse_selection WHERE user_id=? AND applied_at IS NULL AND effective_at>?) AND ${current}`).bind(userId,now,userId,now,access.limits[t.field],userId,now,...guard).run();
  }
 }
 return access;
}
export {keepSchema as selectionBody} from "../../shared/pulse-selection";
