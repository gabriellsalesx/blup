import {createMiddleware} from "hono/factory";
import {auth,type Session} from "../auth";
import {accountAllowed} from "../lifecycle";
import {ownerMatches,GIFT_PLAN} from "../plans";
import {dealsLimits,hasDeals,planNameSchema,type DealsPlan,type DealsLimits} from "../../shared/pulse-plans";

// Blup Deals comes only with Ultra + Deals (with or without Pulse). The
// owner's gift has it. Without the plan the panel still opens, read only.
export type DealsAccess={plan:DealsPlan|null;limits:DealsLimits|null;readOnly:boolean};
const none:DealsAccess={plan:null,limits:null,readOnly:true};
export async function dealsAccess(env:Cloudflare.Env,userId:string):Promise<DealsAccess>{
 if(!await accountAllowed(env.DB,userId))return none;
 const row=await env.DB.prepare("SELECT email,email_verified AS verified,(SELECT plan FROM subscription WHERE reference_id=user.id AND status IN ('active','trialing','past_due') LIMIT 1) AS plan FROM user WHERE id=?").bind(userId).first<{email:string;verified:number;plan:string|null}>();
 if(!row)return none;
 const plan=planNameSchema.safeParse(ownerMatches(env.STAFF_OWNER_EMAIL,row.email,!!row.verified)?GIFT_PLAN:row.plan);
 if(!plan.success||!hasDeals(plan.data))return none;
 return {plan:plan.data,limits:dealsLimits,readOnly:false};
}

export type DealsApp={Bindings:Cloudflare.Env;Variables:{session:Session;access:DealsAccess}};
export const dealsSession=createMiddleware<DealsApp>(async(c,next)=>{
 const session=await auth.api.getSession({headers:c.req.raw.headers});if(!session||!await accountAllowed(c.env.DB,session.user.id))return c.json({code:"UNAUTHORIZED"},401);
 c.set("session",session);c.set("access",await dealsAccess(c.env,session.user.id));await next();
});
// Changes need the plan: without it, the code PLAN (like Pulse).
export const dealsWrite=createMiddleware<DealsApp>(async(c,next)=>{if(c.get("access").readOnly)return c.json({code:"PLAN"},403);await next();});
