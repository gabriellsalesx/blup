import { planNamed, ownerMatches } from "./plans";
import { ensureQuota } from "./quota";
import Stripe from "stripe";
import { removeDomainProvider } from "./domain-provider";
import { sendDowngrade } from "./email";

export const DAY=86_400_000;
export async function accountAllowed(db:D1Database,userId:string) {
	const row=await db.prepare("SELECT u.id FROM user u LEFT JOIN resource_quota q ON q.user_id=u.id WHERE u.id=? AND u.banned=0 AND q.deleted_at IS NULL").bind(userId).first();
	return !!row;
}
export async function observePlan(db:D1Database,userId:string,plan:string,now=Date.now()) {
	await ensureQuota(db,userId);
	if(plan!=="free") {
		await db.prepare("UPDATE resource_quota SET last_plan=?,grace_until=NULL,downgrade_notified=NULL WHERE user_id=? AND (last_plan IS NOT ? OR grace_until IS NOT NULL)").bind(plan,userId,plan).run();
	}else{
		await db.prepare("UPDATE resource_quota SET last_plan='free',grace_until=coalesce(grace_until,?),downgrade_notified=NULL WHERE user_id=? AND (last_plan IS NULL OR last_plan<>'free') AND EXISTS(SELECT 1 FROM site WHERE user_id=?)").bind(now+14*DAY,userId,userId).run();
	}
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
			await deletePrefix(env.SITES,`s/${s.id}/`);
			await env.DB.batch(["site_blob","retired_deploy","site_domain","site_lease"].map(table=>env.DB.prepare(`DELETE FROM ${table} WHERE site_id=?`).bind(s.id)));
		}
		await env.DB.batch([
			env.DB.prepare("DELETE FROM vault_version WHERE vault_id IN (SELECT id FROM vault WHERE user_id=?)").bind(id),
			...["vault","sync_block","sync_account","site","ai_usage","reminder","reminder_month","device_approval","resource_reservation"].map(table=>env.DB.prepare(`DELETE FROM ${table} WHERE user_id=?`).bind(id)),
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
				const stripe=new Stripe(env.STRIPE_SECRET_KEY,{httpClient:Stripe.createFetchHttpClient()});
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
	const rows=await env.DB.prepare("SELECT q.user_id AS id,u.email FROM resource_quota q JOIN user u ON u.id=q.user_id WHERE q.grace_until IS NOT NULL AND q.downgrade_notified IS NULL AND q.deleted_at IS NULL ORDER BY q.grace_until LIMIT 25").all<{id:string;email:string}>();
	for(const row of rows.results){
		const plan=await accountPlan(env,row.id);if(!plan||plan.name!=="free")continue;
		// Claim before sending so overlapping crons never send twice. A send
		// failure releases the claim for the next run.
		const claim=Date.now();
		const won=await env.DB.prepare("UPDATE resource_quota SET downgrade_notified=? WHERE user_id=? AND downgrade_notified IS NULL RETURNING user_id").bind(claim,row.id).first();
		if(!won)continue;
		if(!await sendDowngrade(env,row.email))await env.DB.prepare("UPDATE resource_quota SET downgrade_notified=NULL WHERE user_id=? AND downgrade_notified=?").bind(row.id,claim).run();
	}
}
