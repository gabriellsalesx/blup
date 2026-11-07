import {accountAllowed} from "../lifecycle";
import {ownerMatches,GIFT_PLAN} from "../plans";
import {dealsLimits,hasDeals,planNameSchema,type DealsPlan,type DealsLimits} from "../../shared/pulse-plans";
import {startReadOnly} from "./purge";

// Blup Deals comes only with Ultra + Deals (with or without Pulse). The
// owner's gift has it. A subscription keeps it to the end of the period it
// paid for (Stripe keeps it active until then). When the plan goes, the
// account reads for READ_ONLY_DAYS more (nothing is sent), then its
// private Deals data is purged (purge.ts, daily).
export type DealsAccess={plan:DealsPlan|null;limits:DealsLimits|null;readOnly:boolean;readable:boolean;readOnlyUntil:number|null};
const none:DealsAccess={plan:null,limits:null,readOnly:true,readable:false,readOnlyUntil:null};
export async function dealsAccess(env:Cloudflare.Env,userId:string,now=Date.now()):Promise<DealsAccess>{
 if(!await accountAllowed(env.DB,userId))return none;
 const row=await env.DB.prepare("SELECT email,email_verified AS verified,(SELECT plan FROM subscription WHERE reference_id=user.id AND status IN ('active','trialing','past_due') LIMIT 1) AS plan FROM user WHERE id=?").bind(userId).first<{email:string;verified:number;plan:string|null}>();
 if(!row)return none;
 const plan=planNameSchema.safeParse(ownerMatches(env.STAFF_OWNER_EMAIL,row.email,!!row.verified)?GIFT_PLAN:row.plan);
 if(plan.success&&hasDeals(plan.data)){
  // Back on the plan before the purge: everything is as it was.
  await env.DB.prepare("INSERT INTO deals_account(user_id,last_plan) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET last_plan=excluded.last_plan,read_only_until=NULL WHERE purge_started IS NULL AND (last_plan<>excluded.last_plan OR read_only_until IS NOT NULL)").bind(userId,plan.data).run();
  const state=await env.DB.prepare("SELECT purge_started AS purging FROM deals_account WHERE user_id=?").bind(userId).first<{purging:number|null}>();
  if(state?.purging!=null)return none;
  return {plan:plan.data,limits:dealsLimits,readOnly:false,readable:true,readOnlyUntil:null};
 }
 const until=await startReadOnly(env.DB,userId,now);
 return {...none,readable:until!==null&&until>now,readOnlyUntil:until};
}

