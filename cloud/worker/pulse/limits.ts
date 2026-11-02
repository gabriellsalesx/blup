import {z} from "zod";
import {pulseLimits,pulsePlanSchema,type PulsePlan} from "../../shared/pulse-plans";
import {ownerMatches} from "../plans";
import {accountAllowed,DAY} from "../lifecycle";

export const projectId=z.string().regex(/^[1-9][0-9]{0,15}$/).refine(v=>Number.isSafeInteger(Number(v)));
export const issueState=z.enum(["open","resolved","ignored"]);
export {pulseLimits,pulsePlanSchema,type PulsePlan} from "../../shared/pulse-plans";
export const pulseIdParam=z.object({project:projectId});
export const monthOf=(now=Date.now())=>new Date(now).toISOString().slice(0,7);
export const monthWait=(now=Date.now())=>Math.max(1,Math.ceil((Date.UTC(new Date(now).getUTCFullYear(),new Date(now).getUTCMonth()+1,1)-now)/1000));
export type PulseAccess={plan:PulsePlan|null;limits:typeof pulseLimits[PulsePlan]|null;readOnlyUntil:number|null;readable:boolean;writable:boolean};
export async function pulseAccess(env:Cloudflare.Env,userId:string,now=Date.now()):Promise<PulseAccess>{
 const none:PulseAccess={plan:null,limits:null,readOnlyUntil:null,readable:false,writable:false};
 if(!await accountAllowed(env.DB,userId))return none;
 const row=await env.DB.prepare("SELECT email,email_verified AS verified,(SELECT plan FROM subscription WHERE reference_id=user.id AND status IN ('active','trialing','past_due') LIMIT 1) AS plan FROM user WHERE id=?").bind(userId).first<{email:string;verified:number;plan:string|null}>();
 if(!row)return none;
 const parsed=pulsePlanSchema.safeParse(ownerMatches(env.STAFF_OWNER_EMAIL,row.email,!!row.verified)?"ultra_pulse":row.plan);
 if(parsed.success){
  const previous=await env.DB.prepare("SELECT read_only_until AS until,purge_started AS purging FROM pulse_account WHERE user_id=?").bind(userId).first<{until:number|null;purging:number|null}>();
  if(previous&&(previous.purging!==null||previous.until!==null&&previous.until<=now)){await env.DB.prepare("UPDATE pulse_account SET purge_started=coalesce(purge_started,?) WHERE user_id=? AND read_only_until<=?").bind(now,userId,now).run();return none;}
  const eligible="EXISTS(SELECT 1 FROM user u WHERE u.id=? AND (u.email_verified=1 AND lower(u.email)=? OR EXISTS(SELECT 1 FROM subscription s WHERE s.reference_id=u.id AND s.plan=? AND s.status IN('active','trialing','past_due'))))",owner=(env.STAFF_OWNER_EMAIL??"").trim().toLowerCase();
  await env.DB.prepare(`INSERT INTO pulse_account(user_id,last_plan,paid_until) SELECT ?,?,CASE WHEN EXISTS(SELECT 1 FROM user WHERE id=? AND email_verified=1 AND lower(email)=?) THEN NULL ELSE (SELECT period_end FROM subscription WHERE reference_id=? AND plan=? AND status IN('active','trialing','past_due') LIMIT 1) END WHERE ${eligible} ON CONFLICT(user_id) DO UPDATE SET last_plan=excluded.last_plan,paid_until=excluded.paid_until,read_only_until=NULL WHERE purge_started IS NULL AND ${eligible} AND (last_plan<>excluded.last_plan OR read_only_until IS NOT NULL OR paid_until IS NOT excluded.paid_until)`).bind(userId,parsed.data,userId,owner,userId,parsed.data,userId,owner,parsed.data,userId,owner,parsed.data).run();
  const state=await env.DB.prepare(`SELECT purge_started AS purging,last_plan AS lastPlan,resource_plan AS resourcePlan FROM pulse_account WHERE user_id=? AND ${eligible}`).bind(userId,userId,owner,parsed.data).first<{purging:number|null;lastPlan:string;resourcePlan:string|null}>();
  if(!state||state.purging!==null||state.lastPlan!==parsed.data)return none;
  if(state.resourcePlan!==parsed.data){
   const limits=pulseLimits[parsed.data],tables=[{name:"pulse_project",limit:limits.projects},{name:"pulse_monitor",limit:limits.monitors},{name:"pulse_page",limit:limits.pages}];
   await env.DB.batch([...tables.map(t=>env.DB.prepare(`UPDATE ${t.name} SET plan_disabled=1,retire_at=coalesce(retire_at,?) WHERE user_id=? AND (SELECT count(*) FROM ${t.name} WHERE user_id=? AND plan_disabled=0)>? AND EXISTS(SELECT 1 FROM pulse_account WHERE user_id=? AND last_plan=? AND resource_plan IS ?)` ).bind(now+14*DAY,userId,userId,t.limit,userId,parsed.data,state.resourcePlan)),env.DB.prepare("UPDATE pulse_account SET resource_plan=? WHERE user_id=? AND last_plan=? AND resource_plan IS ?").bind(parsed.data,userId,parsed.data,state.resourcePlan)]);
  }
  return {plan:parsed.data,limits:pulseLimits[parsed.data],readOnlyUntil:null,readable:true,writable:true};
 }
 await env.DB.prepare("UPDATE pulse_account SET read_only_until=min(?,coalesce(paid_until,?))+? WHERE user_id=? AND read_only_until IS NULL AND NOT EXISTS(SELECT 1 FROM subscription WHERE reference_id=? AND plan IN('max_pulse','ultra_pulse') AND status IN('active','trialing','past_due')) AND NOT EXISTS(SELECT 1 FROM user WHERE id=? AND email_verified=1 AND lower(email)=?)").bind(now,now,14*DAY,userId,userId,userId,(env.STAFF_OWNER_EMAIL??"").trim().toLowerCase()).run();
 const state=await env.DB.prepare("SELECT read_only_until AS until,purge_started AS purging,last_plan AS plan FROM pulse_account WHERE user_id=?").bind(userId).first<{until:number|null;purging:number|null;plan:string}>();
 const old=pulsePlanSchema.safeParse(state?.plan);
 return {...none,plan:old.success?old.data:null,limits:old.success?pulseLimits[old.data]:null,readOnlyUntil:state?.until??null,readable:state?.purging===null&&state.until!==null&&state.until>now};
}
export function numericId(){const bytes=crypto.getRandomValues(new Uint8Array(7));bytes[0]&=31;let n=0;for(const b of bytes)n=n*256+b;return String(n||1);}
export async function sha256(value:string){const bytes=new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(value)));return Array.from(bytes,b=>b.toString(16).padStart(2,"0")).join("");}
export function publicKey(){return Array.from(crypto.getRandomValues(new Uint8Array(24)),b=>b.toString(16).padStart(2,"0")).join("");}
