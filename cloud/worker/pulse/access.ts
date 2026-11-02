import {createMiddleware} from "hono/factory";
import {auth,type Session} from "../auth";
import {accountAllowed} from "../lifecycle";
import {pulseAccess} from "./limits";
export type PulseApp={Bindings:Cloudflare.Env;Variables:{session:Session;access:Awaited<ReturnType<typeof pulseAccess>>}};
export const pulseSession=createMiddleware<PulseApp>(async(c,next)=>{
 const session=await auth.api.getSession({headers:c.req.raw.headers});if(!session||!await accountAllowed(c.env.DB,session.user.id))return c.json({code:"UNAUTHORIZED"},401);
 c.set("session",session);c.set("access",await pulseAccess(c.env,session.user.id));await next();
});
export const pulseWrite=createMiddleware<PulseApp>(async(c,next)=>{if(!c.get("access").writable)return c.json({code:"PLAN"},403);await next();});
export async function ownedProject(env:Cloudflare.Env,owner:string,id:string){return env.DB.prepare("SELECT id,name,platform,object_name AS objectName,active,created_at AS createdAt FROM pulse_project WHERE id=? AND user_id=? AND deleted_at IS NULL").bind(id,owner).first<{id:string;name:string;platform:string;objectName:string;active:number;createdAt:number}>();}
