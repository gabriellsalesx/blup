import { queueAccountPurge,queueSitePurge } from "./site-cache-purge";
import { planNamed, ownerMatches } from "./plans";
import { ensureQuota } from "./quota";
import { createStripe } from "./stripe-client";
import { removeDomainProvider } from "./domain-provider";
import { sendDowngrade } from "./email";

export const DAY=86_400_000;
export async function accountAllowed(db:D1Database,userId:string) {
	const row=await db.prepare("SELECT u.id FROM user u LEFT JOIN resource_quota q ON q.user_id=u.id WHERE u.id=? AND u.banned=0 AND q.deleted_at IS NULL").bind(userId).first();
	return !!row;
}
type PlanState={lastPlan:string|null;graceUntil:number|null;storageGraceLimit:number|null;selectedSiteId:string|null;domainGraceUntil:number|null};
export async function readPlanState(db:D1Database,userId:string){
 return db.prepare("SELECT last_plan AS lastPlan,grace_until AS graceUntil,storage_grace_limit AS storageGraceLimit,selected_site_id AS selectedSiteId,domain_grace_until AS domainGraceUntil FROM resource_quota WHERE user_id=?").bind(userId).first<PlanState>();
}
export async function observePlan(db:D1Database,userId:string,name:string,now=Date.now()):Promise<PlanState|null> {
 await ensureQuota(db,userId);
 const plan=planNamed(name),limit=plan.limits.storageMB*1024*1024;
 for(let attempt=0;attempt<4;attempt++){
  const row=await db.prepare("SELECT last_plan AS lastPlan,grace_until AS graceUntil,storage_grace_limit AS storageGraceLimit,selected_site_id AS selectedSiteId,domain_grace_until AS domainGraceUntil,coalesce((SELECT storage_bytes FROM sync_account WHERE user_id=q.user_id),0)+coalesce((SELECT sum(bytes) FROM site WHERE user_id=q.user_id),0) AS used,(SELECT count(*) FROM site WHERE user_id=q.user_id) AS sites,(SELECT count(*) FROM site_domain d JOIN site s ON s.id=d.site_id WHERE s.user_id=q.user_id AND d.status<>'disabled') AS domains FROM resource_quota q WHERE user_id=? AND deleted_at IS NULL").bind(userId).first<PlanState&{used:number;sites:number;domains:number}>();
  if(!row)return null;
  const excessSites=plan.limits.sites>=0&&row.sites>plan.limits.sites,excessStorage=row.used>limit;
  const graceUntil=excessSites||excessStorage?(row.graceUntil??now+14*DAY):null;
  const storageGraceLimit=excessStorage?(row.storageGraceLimit??planNamed(row.lastPlan).limits.storageMB*1024*1024):null;
  const selectedSiteId=excessSites&&plan.limits.sites===1?row.selectedSiteId:null;
  const domainGraceUntil=!plan.limits.customDomain&&row.domains>0?(row.domainGraceUntil??now+14*DAY):null;
  if(row.lastPlan===plan.name&&row.graceUntil===graceUntil&&row.storageGraceLimit===storageGraceLimit&&row.selectedSiteId===selectedSiteId&&row.domainGraceUntil===domainGraceUntil)return row;
  const resetNotice=row.lastPlan!==plan.name||row.graceUntil!==graceUntil;
  const result=await db.batch([
   db.prepare("UPDATE resource_quota SET last_plan=?,grace_until=?,storage_grace_limit=?,selected_site_id=?,domain_grace_until=?,downgrade_notified=CASE WHEN ? THEN NULL ELSE downgrade_notified END,domain_downgrade_notified=CASE WHEN domain_grace_until IS NOT ? THEN NULL ELSE domain_downgrade_notified END WHERE user_id=? AND deleted_at IS NULL AND last_plan IS ? AND grace_until IS ? AND domain_grace_until IS ? AND storage_grace_limit IS ? AND selected_site_id IS ? RETURNING user_id").bind(plan.name,graceUntil,storageGraceLimit,selectedSiteId,domainGraceUntil,resetNotice?1:0,domainGraceUntil,userId,row.lastPlan,row.graceUntil,row.domainGraceUntil,row.storageGraceLimit,row.selectedSiteId),
   db.prepare("UPDATE site_domain SET disable_at=? WHERE site_id IN (SELECT id FROM site WHERE user_id=?) AND status<>'disabled' AND EXISTS(SELECT 1 FROM resource_quota WHERE user_id=? AND last_plan=? AND domain_grace_until IS ?) AND disable_at IS NOT ?").bind(domainGraceUntil,userId,userId,plan.name,domainGraceUntil,domainGraceUntil),
  ]);
  if(result[0]?.results.length)return {lastPlan:plan.name,graceUntil,storageGraceLimit,selectedSiteId,domainGraceUntil};
 }
 return readPlanState(db,userId);
}

// Used by the public Worker too. No session or owner-supplied e-mail is
// trusted: owner gift and subscription are looked up in D1.
export async function accountPlan(env:Cloudflare.Env,userId:string) {
	const row=await env.DB.prepare("SELECT email,email_verified AS verified,(SELECT plan FROM subscription WHERE reference_id=user.id AND status IN ('active','trialing','past_due') LIMIT 1) AS plan FROM user WHERE id=? AND banned=0").bind(userId).first<{email:string;verified:number;plan:string|null}>();
	if(!row) return null;
	const plan=planNamed(ownerMatches(env.STAFF_OWNER_EMAIL,row.email,!!row.verified)?"ultra":row.plan);
	return plan;
}

export async function softDelete(env:Cloudflare.Env,userId:string,now=Date.now()) {
	await ensureQuota(env.DB,userId);
	await env.DB.batch([
		queueAccountPurge(env.DB,userId),
		env.DB.prepare("UPDATE resource_quota SET deleted_at=?,purge_after=?,billing_cancel_pending=1 WHERE user_id=? AND deleted_at IS NULL").bind(now,now+30*DAY,userId),
		env.DB.prepare("DELETE FROM session WHERE user_id=?").bind(userId),
		env.DB.prepare("UPDATE reminder SET sent_at=0,claim=NULL,claim_until=0 WHERE user_id=? AND sent_at IS NULL").bind(userId),
		env.DB.prepare("UPDATE reminder_month SET reserved=0 WHERE user_id=?").bind(userId),
		env.DB.prepare("DELETE FROM device_approval WHERE user_id=?").bind(userId),
		env.DB.prepare("UPDATE device_code SET status='denied' WHERE user_id=?").bind(userId),
	]);
	try{await env.SYNC_LIVE.getByName(userId).revoke();}catch{console.error("[blup] live_revocation_retry");}
}
export async function restoreAccount(db:D1Database,userId:string,now=Date.now()) {
	const row=await db.prepare("UPDATE resource_quota SET deleted_at=NULL,purge_after=NULL WHERE user_id=? AND deleted_at IS NOT NULL AND purge_after>? AND purge_started IS NULL RETURNING user_id").bind(userId,now).first();
	if(row)await queueAccountPurge(db,userId).run();
	return !!row;
}

export async function deletePrefix(bucket:R2Bucket,prefix:string) {
	let cursor:string|undefined;
	do{const page=await bucket.list({prefix,cursor,limit:1000});if(page.objects.length)await bucket.delete(page.objects.map(o=>o.key));cursor=page.truncated?page.cursor:undefined;}while(cursor);
}

// Only expired tombstones are selected by index. A purging account cannot
// be restored part-way through deletion. Failed R2 work is retried daily.
export async function purgeAccounts(env:Cloudflare.Env,now=Date.now()) {
	const due=await env.DB.prepare("SELECT user_id AS id FROM resource_quota WHERE purge_after<=? AND billing_cancel_pending=0 ORDER BY purge_after LIMIT 25").bind(now).all<{id:string}>();
	for(const {id} of due.results){
		const claimed=await env.DB.prepare("UPDATE resource_quota SET purge_started=? WHERE user_id=? AND deleted_at IS NOT NULL AND purge_after<=? RETURNING user_id").bind(now,id,now).first();
		if(!claimed)continue;
		await deletePrefix(env.NOTES,`u/${id}/`);
		const sites=await env.DB.prepare("SELECT id FROM site WHERE user_id=?").bind(id).all<{id:string}>();
		for(const s of sites.results){
			const domain=await env.DB.prepare("SELECT hostname,provider_id AS providerId,verified_at AS verifiedAt FROM site_domain WHERE site_id=?").bind(s.id).first<{hostname:string;providerId:string|null;verifiedAt:number|null}>();
			if(domain)await removeDomainProvider(env,domain);
			await queueSitePurge(env.DB,s.id).run();
			await deletePrefix(env.SITES,`s/${s.id}/`);
			await env.DB.batch(["site_blob","retired_deploy","site_domain","site_lease"].map(table=>env.DB.prepare(`DELETE FROM ${table} WHERE site_id=?`).bind(s.id)));
		}
		await env.DB.batch([
			env.DB.prepare("DELETE FROM vault_head_block WHERE vault_id IN (SELECT id FROM vault WHERE user_id=?)").bind(id),
			env.DB.prepare("DELETE FROM sync_transfer_block WHERE ticket_id IN (SELECT id FROM sync_transfer WHERE user_id=?)").bind(id),
			env.DB.prepare("DELETE FROM vault_version WHERE vault_id IN (SELECT id FROM vault WHERE user_id=?)").bind(id),
			...["vault","sync_block","sync_account","site","ai_usage","reminder","reminder_month","device_approval","resource_reservation","sync_transfer","sync_garbage","stripe_webhook_event"].map(table=>env.DB.prepare(`DELETE FROM ${table} WHERE user_id=?`).bind(id)),
			env.DB.prepare("DELETE FROM subscription WHERE reference_id=?").bind(id),
			env.DB.prepare("DELETE FROM device_code WHERE user_id=?").bind(id),
			env.DB.prepare("DELETE FROM billing_event WHERE user_id=?").bind(id),
			env.DB.prepare("DELETE FROM staff_log WHERE user_id=?").bind(id),
			env.DB.prepare("DELETE FROM user WHERE id=?").bind(id),
			env.DB.prepare("DELETE FROM resource_quota WHERE user_id=?").bind(id),
		]);
	}
}

// Cancellation is retried independently of immediate access revocation.
// The tombstone is retained if Stripe fails; no subscription is restarted
// by support restoration.
export async function cancelDeletedBilling(env:Cloudflare.Env){
	const rows=await env.DB.prepare("SELECT q.user_id AS id,u.stripe_customer_id AS customer FROM resource_quota q JOIN user u ON u.id=q.user_id WHERE q.billing_cancel_pending=1 LIMIT 25").all<{id:string;customer:string|null}>();
	for(const row of rows.results){
		try{
			if(row.customer){
				const stripe=createStripe(env.STRIPE_SECRET_KEY);
				for await(const sub of stripe.subscriptions.list({customer:row.customer,status:"all"})){
					if(!["canceled","incomplete_expired"].includes(sub.status))await stripe.subscriptions.cancel(sub.id);
				}
			}
			await env.DB.batch([
				env.DB.prepare("UPDATE subscription SET status='canceled' WHERE reference_id=?").bind(row.id),
				env.DB.prepare("UPDATE resource_quota SET billing_cancel_pending=0 WHERE user_id=?").bind(row.id),
			]);
		}catch{console.error("[blup] billing_cancellation_retry");}
	}
}

export async function downgradeNotices(env:Cloudflare.Env){
 const rows=await env.DB.prepare("SELECT q.user_id AS id,u.email FROM resource_quota q JOIN user u ON u.id=q.user_id WHERE q.grace_until IS NOT NULL AND q.downgrade_notified IS NULL AND q.deleted_at IS NULL UNION SELECT q.user_id AS id,u.email FROM resource_quota q JOIN user u ON u.id=q.user_id WHERE q.domain_grace_until IS NOT NULL AND q.domain_downgrade_notified IS NULL AND q.deleted_at IS NULL LIMIT 25").all<{id:string;email:string}>();
 for(const row of rows.results){
  const plan=await accountPlan(env,row.id);if(!plan)continue;
  const state=await observePlan(env.DB,row.id,plan.name);if(!state)continue;
  const pending=await env.DB.prepare("SELECT downgrade_notified AS general,domain_downgrade_notified AS domain FROM resource_quota WHERE user_id=?").bind(row.id).first<{general:number|null;domain:number|null}>();
  const general=state.graceUntil!==null&&pending?.general===null,domain=state.domainGraceUntil!==null&&pending?.domain===null;
  if(!general&&!domain)continue;
  const use=await env.DB.prepare("SELECT coalesce((SELECT storage_bytes FROM sync_account WHERE user_id=?),0)+coalesce((SELECT sum(bytes) FROM site WHERE user_id=?),0) AS bytes,(SELECT count(*) FROM site WHERE user_id=?) AS sites").bind(row.id,row.id,row.id).first<{bytes:number;sites:number}>();
  const claim=Date.now();
  const won=await env.DB.prepare("UPDATE resource_quota SET downgrade_notified=CASE WHEN ? THEN ? ELSE downgrade_notified END,domain_downgrade_notified=CASE WHEN ? THEN ? ELSE domain_downgrade_notified END WHERE user_id=? AND grace_until IS ? AND domain_grace_until IS ? AND (?=0 OR downgrade_notified IS NULL) AND (?=0 OR domain_downgrade_notified IS NULL) RETURNING user_id").bind(general?1:0,claim,domain?1:0,claim,row.id,state.graceUntil,state.domainGraceUntil,general?1:0,domain?1:0).first();
  if(!won)continue;
  const limit=plan.limits.storageMB*1024*1024;
  const sent=await sendDowngrade(env,row.email,{
   ...(general&&state.graceUntil&&use&&use.sites>plan.limits.sites?{sites:{limit:plan.limits.sites,deadline:state.graceUntil}}:{}),
   ...(general&&state.graceUntil&&use&&use.bytes>limit?{storage:{usedBytes:use.bytes,limitBytes:limit,deadline:state.graceUntil}}:{}),
   ...(domain&&state.domainGraceUntil?{domainDisableAt:state.domainGraceUntil}:{}),
  });
  if(!sent)await env.DB.prepare("UPDATE resource_quota SET downgrade_notified=CASE WHEN downgrade_notified=? THEN NULL ELSE downgrade_notified END,domain_downgrade_notified=CASE WHEN domain_downgrade_notified=? THEN NULL ELSE domain_downgrade_notified END WHERE user_id=?").bind(claim,claim,row.id).run();
 }
}
