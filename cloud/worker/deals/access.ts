import {createMiddleware} from "hono/factory";
import {type Session} from "../auth";
import {requestSession} from "../request-session";
import {accountAllowed} from "../lifecycle";

// Who has Deals and how (entitlement.ts, kept apart from the session so
// the public pages' worker does not load the sign-in).
export {dealsAccess,type DealsAccess} from "./entitlement";
import {dealsAccess,type DealsAccess} from "./entitlement";

export type DealsApp={Bindings:Cloudflare.Env;Variables:{session:Session;access:DealsAccess}};
export const dealsSession=createMiddleware<DealsApp>(async(c,next)=>{
 const session=await requestSession(c.req.raw);if(!session||!await accountAllowed(c.env.DB,session.user.id))return c.json({code:"UNAUTHORIZED"},401);
 c.set("session",session);c.set("access",await dealsAccess(c.env,session.user.id));await next();
});
// Changes need the plan: without it, the code PLAN (like Pulse).
export const dealsWrite=createMiddleware<DealsApp>(async(c,next)=>{if(c.get("access").readOnly)return c.json({code:"PLAN"},403);await next();});
