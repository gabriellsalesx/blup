import {monthOf,monthWait} from "./limits";
import {IngestError} from "./protocol";

// A grant is never refunded on network timeout. Its deterministic ID is
// imported once by the DO. Unused slots remain reserved for this month.
export async function reserveGrant(db:D1Database,userId:string,project:string,sequence:number,limit:number,now=Date.now()){
 const month=monthOf(now),id=`${project}:${month}:${sequence}`;
 await db.batch([
  db.prepare("INSERT OR IGNORE INTO pulse_month(user_id,month,reserved) VALUES(?,?,0)").bind(userId,month),
  db.prepare("INSERT OR IGNORE INTO pulse_grant(id,user_id,project_id,month,amount,created_at) SELECT ?,?,?,?,min(32,?-reserved),? FROM pulse_month WHERE user_id=? AND month=? AND reserved<?").bind(id,userId,project,month,limit,now,userId,month,limit),
  db.prepare("UPDATE pulse_month SET reserved=(SELECT coalesce(sum(amount),0) FROM pulse_grant WHERE user_id=? AND month=?) WHERE user_id=? AND month=?").bind(userId,month,userId,month),
 ]);
 const grant=await db.prepare("SELECT amount FROM pulse_grant WHERE id=? AND user_id=? AND project_id=?").bind(id,userId,project).first<{amount:number}>();
 if(!grant)throw new IngestError(429,"MONTHLY",monthWait(now));return {id,amount:grant.amount};
}
