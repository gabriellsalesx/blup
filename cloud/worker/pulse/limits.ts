import {z} from "zod";
import {ownerMatches} from "../plans";
import {accountAllowed,DAY} from "../lifecycle";

export const projectId=z.string().regex(/^[1-9][0-9]{0,15}$/).refine(v=>Number.isSafeInteger(Number(v)));
export const issueState=z.enum(["open","resolved","ignored"]);
export const pulseLimits={
 max_pulse:{projects:3,events:50_000,retentionDays:30,monitors:10,intervalMinutes:5,pages:1,customDomain:false,projectBytes:256*1024*1024,burst:60},
 ultra_pulse:{projects:10,events:250_000,retentionDays:90,monitors:50,intervalMinutes:2,pages:3,customDomain:true,projectBytes:1024*1024*1024,burst:300},
} as const;
export type PulsePlan=keyof typeof pulseLimits;
export const pulsePlanSchema=z.enum(["max_pulse","ultra_pulse"]);
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
  await env.DB.prepare("INSERT INTO pulse_account(user_id,last_plan) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET last_plan=excluded.last_plan,read_only_until=NULL WHERE purge_started IS NULL AND (last_plan<>excluded.last_plan OR read_only_until IS NOT NULL)").bind(userId,parsed.data).run();
  const state=await env.DB.prepare("SELECT purge_started AS purging FROM pulse_account WHERE user_id=?").bind(userId).first<{purging:number|null}>();
  if(state?.purging!==null)return none;
  return {plan:parsed.data,limits:pulseLimits[parsed.data],readOnlyUntil:null,readable:true,writable:true};
 }
 await env.DB.prepare("UPDATE pulse_account SET read_only_until=? WHERE user_id=? AND read_only_until IS NULL").bind(now+14*DAY,userId).run();
 const state=await env.DB.prepare("SELECT read_only_until AS until,purge_started AS purging,last_plan AS plan FROM pulse_account WHERE user_id=?").bind(userId).first<{until:number|null;purging:number|null;plan:string}>();
 const old=pulsePlanSchema.safeParse(state?.plan);
 return {...none,plan:old.success?old.data:null,limits:old.success?pulseLimits[old.data]:null,readOnlyUntil:state?.until??null,readable:state?.purging===null&&state.until!==null&&state.until>now};
}
export function numericId(){const bytes=crypto.getRandomValues(new Uint8Array(7));bytes[0]&=31;let n=0;for(const b of bytes)n=n*256+b;return String(n||1);}
export async function sha256(value:string){const bytes=new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(value)));return Array.from(bytes,b=>b.toString(16).padStart(2,"0")).join("");}
export function publicKey(){return Array.from(crypto.getRandomValues(new Uint8Array(24)),b=>b.toString(16).padStart(2,"0")).join("");}
