import {sqlPlans,plansWithDeals} from "../../shared/pulse-plans";
import {forgetWhatsapp} from "./bridge";
import {removeSiteDomain} from "./site-domain";
import {siteLogoKey} from "./site-rules";

export const READ_ONLY_DAYS=14;
const DAY=86_400_000;
// The account may have Deals: a subscription with Deals, or the owner.
const entitled=`(EXISTS(SELECT 1 FROM subscription s WHERE s.reference_id=deals_account.user_id AND s.plan IN(${sqlPlans(plansWithDeals)}) AND s.status IN('active','trialing','past_due')) OR EXISTS(SELECT 1 FROM user u WHERE u.id=deals_account.user_id AND u.email_verified=1 AND lower(u.email)=?))`;
// The first time an account that had Deals is seen without it: read only
// for READ_ONLY_DAYS, and nothing more goes out (the queue is closed, the
// hunts stop feeding through dealsAccess). Returns the end of that window,
// or null when the account never had Deals.
export async function startReadOnly(db:D1Database,userId:string,now=Date.now()){
 const [,row]=await db.batch([
  db.prepare("UPDATE deals_account SET read_only_until=? WHERE user_id=? AND read_only_until IS NULL AND purge_started IS NULL").bind(now+READ_ONLY_DAYS*DAY,userId),
  db.prepare("SELECT read_only_until AS until FROM deals_account WHERE user_id=?").bind(userId),
  db.prepare("UPDATE deals_post SET state='failed',error='PLAN',claim=NULL,claim_until=0 WHERE user_id=? AND state='queued' AND EXISTS(SELECT 1 FROM deals_account WHERE user_id=? AND read_only_until IS NOT NULL)").bind(userId,userId),
 ]);
 return (row?.results[0] as {until:number|null}|undefined)?.until??null;
}
// Daily: accounts that lost Deals without opening the panel start their
// read-only window; those past it are purged (unless the plan came back).
export async function dealsLifecycle(env:Cloudflare.Env,now=Date.now()){
 const owner=(env.STAFF_OWNER_EMAIL??"").trim().toLowerCase();
 await env.DB.batch([
  env.DB.prepare(`UPDATE deals_account SET read_only_until=? WHERE read_only_until IS NULL AND purge_started IS NULL AND NOT ${entitled}`).bind(now+READ_ONLY_DAYS*DAY,owner),
  env.DB.prepare(`UPDATE deals_account SET read_only_until=NULL WHERE read_only_until IS NOT NULL AND purge_started IS NULL AND ${entitled}`).bind(owner),
  env.DB.prepare("UPDATE deals_post SET state='failed',error='PLAN',claim=NULL,claim_until=0 WHERE state='queued' AND user_id IN(SELECT user_id FROM deals_account WHERE read_only_until IS NOT NULL)"),
 ]);
 const due=await env.DB.prepare("SELECT user_id AS id FROM deals_account WHERE read_only_until<=? ORDER BY read_only_until LIMIT 25").bind(now).all<{id:string}>();
 for(const {id} of due.results){
  const claimed=await env.DB.prepare(`UPDATE deals_account SET purge_started=coalesce(purge_started,?) WHERE user_id=? AND read_only_until<=? AND NOT ${entitled} RETURNING user_id`).bind(now,id,now,owner).first();
  if(claimed)await purgeDealsAccount(env,id);
 }
}
// The account's own Deals data, when the account is purged for good: store
// IDs and keys, the public page, channels, hunts, posts, clicks, Telegram links and the
// WhatsApp connection (logged out at the Bridge first). The
// shared catalog stays; only pasted products and videos nobody else posted go (and
// their prices), since they came from this account alone.
export async function purgeDealsAccount(env:Cloudflare.Env,userId:string){
 await forgetWhatsapp(env,userId);
 // The public page: its domain, its logos and the row.
 const page=await env.DB.prepare("SELECT id FROM deals_site WHERE user_id=?").bind(userId).first<{id:string}>();
 if(page){await removeSiteDomain(env,page.id);await env.DEALS_ASSETS.delete([siteLogoKey(page.id,"light"),siteLogoKey(page.id,"dark")]);await env.DB.prepare("DELETE FROM deals_site WHERE id=?").bind(page.id).run();}
 const orphan="kind IN('product','video') AND id IN(SELECT offer_id FROM deals_post WHERE user_id=?) AND NOT EXISTS(SELECT 1 FROM deals_post p WHERE p.offer_id=deals_offer.id AND p.user_id<>?)";
 await env.DB.batch([
  env.DB.prepare(`DELETE FROM deals_price WHERE offer_id IN(SELECT id FROM deals_offer WHERE ${orphan})`).bind(userId,userId),
  env.DB.prepare(`DELETE FROM deals_offer WHERE ${orphan}`).bind(userId,userId),
  env.DB.prepare("DELETE FROM deals_click_day WHERE user_id=? OR post_id IN(SELECT id FROM deals_post WHERE user_id=?)").bind(userId,userId),
  env.DB.prepare("DELETE FROM deals_post WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_hunt WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_channel WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_affiliate WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_telegram_link WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_whatsapp WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_setting WHERE user_id=?").bind(userId),
  env.DB.prepare("DELETE FROM deals_account WHERE user_id=?").bind(userId),
 ]);
}
