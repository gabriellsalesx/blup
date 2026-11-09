import {createMiddleware} from "hono/factory";
import {type Session} from "../auth";
import {requestSession} from "../request-session";
import {accountAllowed} from "../lifecycle";
import {pulseAccess,projectId} from "./limits";
export type PulseApp={Bindings:Cloudflare.Env;Variables:{session:Session;access:Awaited<ReturnType<typeof pulseAccess>>}};
export const pulseSession=createMiddleware<PulseApp>(async(c,next)=>{
 const session=await requestSession(c.req.raw);if(!session||!await accountAllowed(c.env.DB,session.user.id))return c.json({code:"UNAUTHORIZED"},401);
 c.set("session",session);c.set("access",await pulseAccess(c.env,session.user.id));await next();
});
export const pulseWrite=createMiddleware<PulseApp>(async(c,next)=>{if(!c.get("access").writable)return c.json({code:"PLAN"},403);await next();});
export async function ownedProject(env:Cloudflare.Env,owner:string,id:string){return env.DB.prepare("SELECT id,name,platform,object_name AS objectName,active,plan_disabled AS planDisabled,retire_at AS retireAt,created_at AS createdAt FROM pulse_project WHERE id=? AND user_id=? AND deleted_at IS NULL AND (retire_at IS NULL OR retire_at>?)").bind(id,owner,Date.now()).first<{id:string;name:string;platform:string;objectName:string;active:number;planDisabled:number;retireAt:number|null;createdAt:number}>();}
export const projectWrite=createMiddleware<PulseApp>(async(c,next)=>{
 if(!["POST","PUT","PATCH"].includes(c.req.method))return next();
 const parsed=projectId.safeParse(c.req.param("project"));if(!parsed.success)return c.json({code:"INVALID"},400);
 const project=await ownedProject(c.env,c.get("session").user.id,parsed.data);if(!project)return c.json({code:"NOT_FOUND"},404);if(project.planDisabled)return c.json({code:"PLAN"},403);await next();
});
