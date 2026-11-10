import {Hono} from "hono";
import {createMiddleware} from "hono/factory";
import {z} from "zod";
import type {Session} from "./auth";
import {requestSession} from "./request-session";
import {accountAllowed,settlePlan} from "./lifecycle";
import {validate} from "./validation";
import {clientKey} from "./security";
import {APPSUMO_MAX,appsumoLevels,LIFETIME_ID_SQL,type AppsumoPlan} from "../shared/appsumo-levels";

// AppSumo's lifetime deal ("Codes" model, stacking): each code redeemed
// moves the account up one level, to APPSUMO_MAX. The plan is a row of the
// subscription table (id appsumo-<user>, no Stripe ids, no end), with the
// same limits as the subscription plan of that name; every reader picks
// the highest plan when Stripe's row is there too (planOrder). The level
// is always the count of the account's redeemed codes, never a counter, so
// a refund or two redemptions at once leave it right. The ladder and the
// test of a lifetime id: shared/appsumo-levels.ts.
const levelPlan=(n:number):AppsumoPlan|"free"=>n<1?"free":appsumoLevels[Math.min(n,APPSUMO_MAX)-1]??"free";
// The level's plan in SQL, from the constants above (never input).
const LEVEL_CASE=`CASE min(count(*),${APPSUMO_MAX}) ${appsumoLevels.map((p,i)=>`WHEN ${i+1} THEN '${p}'`).join(" ")} END`;

// What a person types: any case, with spaces or hyphens. Our codes are 16
// letters and digits; the bounds leave room for another format.
export function normalizeCode(input:string){
 const code=input.toUpperCase().replace(/[^A-Z0-9]/g,"");
 return code.length>=8&&code.length<=64?code:null;
}
// Codes are secrets: only HMAC-SHA256(APPSUMO_CODE_PEPPER, code) is stored
// and looked up, so the table alone gives no usable code.
export async function codeHasher(pepper:string){
 const key=await crypto.subtle.importKey("raw",new TextEncoder().encode(pepper),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
 return async(code:string)=>Array.from(new Uint8Array(await crypto.subtle.sign("HMAC",key,new TextEncoder().encode(code))),b=>b.toString(16).padStart(2,"0")).join("");
}
// The pepper, or null when it is missing or too short to be a secret (at
// least 32 random characters). Never rotated: every imported code is a
// hash made with it.
export const pepperOf=(env:Cloudflare.Env)=>{const pepper=env.APPSUMO_CODE_PEPPER;return pepper&&pepper.length>=32?pepper:null;};
// The display form of a code: its last 4 characters.
export const last4Of=(code:string)=>code.slice(-4);

// The lifetime rows of the accounts that `users` (a SELECT of user ids)
// names, made to match their redeemed codes: upserted at the level of the
// count, deleted at 0. Goes in the same batch as the change of the codes.
// `guard`: an SQL condition (AND ...) that must hold for them to run at
// all, with its own bindings after the users'.
export function lifetimeWrites(db:D1Database,users:string,binds:unknown[],guard="",guardBinds:unknown[]=[]){
 return [
  db.prepare(`INSERT INTO subscription(id,plan,reference_id,status,period_start) SELECT 'appsumo-'||user_id,${LEVEL_CASE},user_id,'active',min(redeemed_at) FROM appsumo_code WHERE status='redeemed' AND user_id IN (${users}) ${guard} GROUP BY user_id ON CONFLICT(id) DO UPDATE SET plan=excluded.plan,status='active',period_start=excluded.period_start,period_end=NULL`).bind(...binds,...guardBinds),
  db.prepare(`DELETE FROM subscription WHERE ${LIFETIME_ID_SQL} AND reference_id IN (${users}) AND NOT EXISTS(SELECT 1 FROM appsumo_code c WHERE c.user_id=subscription.reference_id AND c.status='redeemed') ${guard}`).bind(...binds,...guardBinds),
 ];
}
const oneUser="SELECT ?";

// Plan changes the way a Stripe downgrade does (plan state, Pulse, devices
// over the new limit). Failing here only delays it: every request observes
// the plan again.
export async function settleLifetime(env:Cloudflare.Env,userId:string){
 try{await settlePlan(env,userId);}catch{console.error("[blup] plan_settle_retry");}
}
// Many accounts (a refund): 10 at a time, each failure on its own.
export async function settleLifetimes(env:Cloudflare.Env,userIds:readonly string[]){
 for(let i=0;i<userIds.length;i+=10)await Promise.allSettled(userIds.slice(i,i+10).map(id=>settleLifetime(env,id)));
}
// The next request of an account observes its plan state (account.ts) and
// the Pulse cron its Pulse, but only settlePlan disconnects the devices
// over the new limit. So the daily cron settles again the accounts with a
// code revoked in the last 25 hours: a refund whose settling failed or was
// cut short (waitUntil) is settled within a day. Settling twice is harmless.
export async function settleRevoked(env:Cloudflare.Env,now=Date.now()){
 const rows=await env.DB.prepare("SELECT DISTINCT user_id AS id FROM appsumo_code WHERE status='revoked' AND revoked_at>? AND user_id IS NOT NULL LIMIT 500").bind(now-25*3_600_000).all<{id:string}>();
 await settleLifetimes(env,rows.results.map(r=>r.id));
}

const codeRows=z.array(z.object({id:z.string(),last4:z.string(),redeemedAt:z.number()}));
// What the panel shows: the level, the codes in it (last 4 only), whether
// a Stripe subscription still renews (Option A: nothing is cancelled for
// the person; the panel tells them to cancel it on the Plan page) and
// whether redeeming works here at all (the pepper is set). A code's id is
// the start of its hash (as in the staff's detail), a stable key.
export async function redeemState(env:Cloudflare.Env,userId:string){
 const [codes,stripe]=await env.DB.batch<unknown>([
  env.DB.prepare("SELECT substr(code_hash,1,16) AS id,last4,redeemed_at AS redeemedAt FROM appsumo_code WHERE user_id=? AND status='redeemed' ORDER BY redeemed_at").bind(userId),
  env.DB.prepare("SELECT 1 AS n FROM subscription WHERE reference_id=? AND stripe_subscription_id IS NOT NULL AND status IN('active','trialing','past_due') AND coalesce(cancel_at_period_end,0)=0 AND cancel_at IS NULL LIMIT 1").bind(userId),
 ]);
 const list=codeRows.parse(codes?.results??[]),level=Math.min(list.length,APPSUMO_MAX);
 return {level,max:APPSUMO_MAX,plan:levelPlan(level),codes:list,stripeActive:!!stripe?.results.length,available:!!pepperOf(env)};
}

// 10 wrong codes in an hour lock the account out of redeeming until the
// hour ends (the Workers rate limit is per location and approximate; this
// one is exact). A redemption clears it. lockedFor turns a locked account
// away early; the failure itself is counted and read back in one
// statement, so requests that passed lockedFor together still stop at 10.
const HOUR=3_600_000,MAX_FAILURES=10;
const waitOf=(start:number,now:number)=>Math.max(1,Math.ceil((start+HOUR-now)/1000));
async function lockedFor(db:D1Database,userId:string,now:number){
 const row=await db.prepare("SELECT window_start AS start,failures FROM appsumo_attempt WHERE user_id=?").bind(userId).first<{start:number;failures:number}>();
 return row&&row.start>now-HOUR&&row.failures>=MAX_FAILURES?waitOf(row.start,now):0;
}
// The failures of the window after this one (a new window when the last
// one is over): past MAX_FAILURES, the answer is the lockout.
async function failure(db:D1Database,userId:string,now:number){
 const row=await db.prepare("INSERT INTO appsumo_attempt(user_id,window_start,failures) VALUES(?,?,1) ON CONFLICT(user_id) DO UPDATE SET failures=CASE WHEN window_start<=?-? THEN 1 ELSE failures+1 END,window_start=CASE WHEN window_start<=?-? THEN excluded.window_start ELSE window_start END RETURNING window_start AS start,failures").bind(userId,now,now,HOUR,now,HOUR).first<{start:number;failures:number}>();
 return row&&row.failures>MAX_FAILURES?waitOf(row.start,now):0;
}

const foundCode=z.object({userId:z.string().nullable(),status:z.string()}).optional(),redeemedCount=z.object({n:z.number()});

type App={Bindings:Cloudflare.Env;Variables:{session:Session}};
const signedIn=createMiddleware<App>(async(c,next)=>{
 const s=await requestSession(c.req.raw);
 if(!s||!await accountAllowed(c.env.DB,s.user.id))return c.json({code:"UNAUTHORIZED"},401);
 c.set("session",s);await next();
});

// Errors: INVALID (400, not a code's shape), INVALID_CODE (400: unknown,
// revoked or someone else's, the same answer so nothing tells them apart),
// ALREADY_REDEEMED (409, this account's), MAX_LEVEL (409),
// TOO_MANY_REQUESTS (429, Retry-After), UNAVAILABLE (503, no pepper here
// or one under 32 characters).
export const redeemApi=new Hono<App>().use("*",signedIn)
 .get("/",async c=>c.json(await redeemState(c.env,c.get("session").user.id)))
 .post("/",validate("json",z.strictObject({code:z.string().max(200)})),async c=>{
  const userId=c.get("session").user.id,pepper=pepperOf(c.env);
  if(!pepper)return c.json({code:"UNAVAILABLE"},503);
  const hits=await Promise.all([`redeem:${userId}`,`redeem-ip:${clientKey(c.req.header("cf-connecting-ip"))}`].map(key=>c.env.REDEEM_LIMIT.limit({key})));
  if(hits.some(r=>!r.success)){c.header("Retry-After","60");return c.json({code:"TOO_MANY_REQUESTS"},429);}
  const code=normalizeCode(c.req.valid("json").code);
  if(!code)return c.json({code:"INVALID"},400);
  const now=Date.now(),wait=await lockedFor(c.env.DB,userId,now);
  if(wait){c.header("Retry-After",String(wait));return c.json({code:"TOO_MANY_REQUESTS"},429);}
  const hash=await(await codeHasher(pepper))(code);
  // One transaction: the code is taken only while available and only
  // below the top level (both checked by the UPDATE itself, so two at once
  // cannot pass 5), then the plan follows the new count. The writes after
  // it run only when this claim took the code (`claimed`).
  const claimed="AND EXISTS(SELECT 1 FROM appsumo_code WHERE code_hash=? AND user_id=? AND status='redeemed' AND redeemed_at=?)";
  const [claim]=await c.env.DB.batch([
   c.env.DB.prepare(`UPDATE appsumo_code SET status='redeemed',user_id=?,redeemed_at=? WHERE code_hash=? AND status='available' AND (SELECT count(*) FROM appsumo_code WHERE user_id=? AND status='redeemed')<${APPSUMO_MAX}`).bind(userId,now,hash,userId),
   c.env.DB.prepare(`DELETE FROM appsumo_attempt WHERE user_id=? ${claimed}`).bind(userId,hash,userId,now),
   ...lifetimeWrites(c.env.DB,oneUser,[userId],claimed,[hash,userId,now]),
  ]);
  if(claim?.meta.changes){
   await settleLifetime(c.env,userId);
   return c.json({...await redeemState(c.env,userId),redeemed:{last4:last4Of(code)}},200);
  }
  const [row,count]=await c.env.DB.batch([
   c.env.DB.prepare("SELECT user_id AS userId,status FROM appsumo_code WHERE code_hash=?").bind(hash),
   c.env.DB.prepare("SELECT count(*) AS n FROM appsumo_code WHERE user_id=? AND status='redeemed'").bind(userId),
  ]);
  const found=foundCode.parse(row?.results[0]),redeemed=redeemedCount.parse(count?.results[0]).n;
  if(found?.userId===userId&&found.status==="redeemed")return c.json({code:"ALREADY_REDEEMED"},409);
  if(redeemed>=APPSUMO_MAX)return c.json({code:"MAX_LEVEL"},409);
  const locked=await failure(c.env.DB,userId,now);
  if(locked){c.header("Retry-After",String(locked));return c.json({code:"TOO_MANY_REQUESTS"},429);}
  return c.json({code:"INVALID_CODE"},400);
 });

// The staff's side (staff.ts adds the roles and the log).
export const codeList=z.array(z.string().max(200)).min(1).max(2500);
// Normalized and hashed, each code once; what is not a code's shape is
// left out (counted as skipped).
async function hashedCodes(pepper:string,codes:string[]){
 const hash=await codeHasher(pepper),seen=new Map<string,string>();
 for(const raw of codes){const code=normalizeCode(raw);if(code&&!seen.has(code))seen.set(code,last4Of(code));}
 return Promise.all([...seen].map(async([code,last4])=>[await hash(code),last4] as const));
}
// INSERT OR IGNORE: a code already there (even used or refunded) stays as
// it is. One statement whatever the count (json_each, not 2500 bindings).
export async function importCodes(env:Cloudflare.Env,pepper:string,codes:string[],staffEmail:string){
 const rows=await hashedCodes(pepper,codes),now=Date.now();
 const [inserted]=await env.DB.batch([
  env.DB.prepare("INSERT OR IGNORE INTO appsumo_code(code_hash,last4,imported_at) SELECT json_extract(value,'$[0]'),json_extract(value,'$[1]'),? FROM json_each(?)").bind(now,JSON.stringify(rows)),
  env.DB.prepare("INSERT INTO staff_log(staff_email,action,detail) VALUES(?,'appsumo.import',?)").bind(staffEmail,`${rows.length} códigos, ${codes.length-rows.length} inválidos ou repetidos`),
 ]);
 const added=inserted?.meta.changes??0;
 return {added,skipped:codes.length-added};
}
// AppSumo's refunds: each code still available or redeemed is revoked for
// good, its account goes down to the codes it still has, and each account
// gets one log entry with the last 4 of its codes (never a code), and the
// refund one more with its counts only, so codes never redeemed or whose
// account is gone are in the log too. All in one transaction; the plans
// settle after it.
export async function refundCodes(env:Cloudflare.Env,pepper:string,codes:string[],staffEmail:string){
 const rows=await hashedCodes(pepper,codes),now=Date.now(),hashes=JSON.stringify(rows.map(([h])=>h));
 const users="SELECT DISTINCT user_id FROM appsumo_code WHERE code_hash IN (SELECT value FROM json_each(?)) AND user_id IS NOT NULL";
 const [revoked]=await env.DB.batch<{userId:string|null}>([
  env.DB.prepare("UPDATE appsumo_code SET status='revoked',revoked_at=?,revoke_reason='refund' WHERE code_hash IN (SELECT value FROM json_each(?)) AND status IN('available','redeemed') RETURNING user_id AS userId").bind(now,hashes),
  env.DB.prepare("INSERT INTO staff_log(staff_email,action,user_id,detail) SELECT ?,'appsumo.refund',user_id,'…'||group_concat(last4,' …') FROM appsumo_code WHERE code_hash IN (SELECT value FROM json_each(?)) AND status='revoked' AND revoked_at=? AND revoke_reason='refund' AND user_id IS NOT NULL GROUP BY user_id").bind(staffEmail,hashes,now),
  env.DB.prepare("INSERT INTO staff_log(staff_email,action,user_id,detail) SELECT ?,'appsumo.refund',NULL,count(*)||' revogados, '||(CAST(? AS INTEGER)-count(*))||' não encontrados ou já revogados, '||count(DISTINCT user_id)||' contas' FROM appsumo_code WHERE code_hash IN (SELECT value FROM json_each(?)) AND status='revoked' AND revoked_at=? AND revoke_reason='refund'").bind(staffEmail,rows.length,hashes,now),
  ...lifetimeWrites(env.DB,users,[hashes]),
 ]);
 const affected=[...new Set((revoked?.results??[]).flatMap(r=>r.userId?[r.userId]:[]))];
 return {revoked:revoked?.results.length??0,notFound:rows.length-(revoked?.results.length??0),users:affected.length,affected};
}
// One code of an account, by the start of its hash (the staff never sees
// the code). Logged with its last 4 in the same transaction.
export async function revokeCode(env:Cloudflare.Env,userId:string,id:string,staffEmail:string){
 const match="FROM appsumo_code WHERE user_id=? AND substr(code_hash,1,16)=? AND status='redeemed'";
 const [,gone]=await env.DB.batch([
  env.DB.prepare(`INSERT INTO staff_log(staff_email,action,user_id,detail) SELECT ?,'appsumo.revoke',user_id,'…'||last4 ${match}`).bind(staffEmail,userId,id),
  env.DB.prepare("UPDATE appsumo_code SET status='revoked',revoked_at=?,revoke_reason='staff' WHERE user_id=? AND substr(code_hash,1,16)=? AND status='redeemed'").bind(Date.now(),userId,id),
  ...lifetimeWrites(env.DB,oneUser,[userId]),
 ]);
 return !!gone?.meta.changes;
}
export async function appsumoStats(env:Cloudflare.Env){
 const [status,levels]=await env.DB.batch<{key:string|number;n:number}>([
  env.DB.prepare("SELECT status AS key,count(*) AS n FROM appsumo_code GROUP BY status"),
  env.DB.prepare(`SELECT level AS key,count(*) AS n FROM (SELECT min(count(*),${APPSUMO_MAX}) AS level FROM appsumo_code WHERE status='redeemed' AND user_id IS NOT NULL GROUP BY user_id) GROUP BY level`),
 ]);
 const counts={available:0,redeemed:0,revoked:0};
 for(const r of status?.results??[])if(r.key==="available"||r.key==="redeemed"||r.key==="revoked")counts[r.key]=r.n;
 const perLevel=Array.from({length:APPSUMO_MAX},(_,i)=>({level:i+1,plan:levelPlan(i+1),users:(levels?.results??[]).find(r=>r.key===i+1)?.n??0}));
 return {codes:counts,levels:perLevel};
}
const accountCodes=z.array(z.object({id:z.string(),last4:z.string(),status:z.enum(["available","redeemed","revoked"]),importedAt:z.number(),redeemedAt:z.number().nullable(),revokedAt:z.number().nullable(),revokeReason:z.enum(["refund","staff"]).nullable()}));
export async function codesOf(env:Cloudflare.Env,userId:string){
 const rows=await env.DB.prepare("SELECT substr(code_hash,1,16) AS id,last4,status,imported_at AS importedAt,redeemed_at AS redeemedAt,revoked_at AS revokedAt,revoke_reason AS revokeReason FROM appsumo_code WHERE user_id=? ORDER BY coalesce(redeemed_at,imported_at)").bind(userId).all();
 return accountCodes.parse(rows.results);
}
