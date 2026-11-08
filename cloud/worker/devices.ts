import { z } from "zod";
import { reserve,release,releaseStatements } from "./quota";
export const approvalBody=z.strictObject({userCode:z.string().regex(/^[A-Za-z0-9-]{6,32}$/)});
const tokenBody=z.looseObject({device_code:z.string().min(1).max(512)});
const hash=async(code:string)=>[...new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(code)))].map(n=>n.toString(16).padStart(2,"0")).join("");

export async function approveDevice(db:D1Database,userId:string,code:string,limit:number){
	const normalized=code.replace(/[^a-zA-Z0-9]/g,"").toUpperCase();
	const row=await db.prepare("SELECT id,device_code AS code,expires_at AS expires FROM device_code WHERE user_code IN (?,?) AND user_id=? AND status='pending' AND client_id='blup-studio' AND expires_at>? LIMIT 1").bind(code,normalized,userId,Date.now()).first<{id:string;code:string;expires:number}>();
	if(!row)return "INVALID";
	const hold=await reserve(db,userId,"devices",1,limit<0?Number.MAX_SAFE_INTEGER:limit,undefined,Math.max(1,row.expires-Date.now()));
	if(!hold)return "DEVICES_LIMIT";
	try{
		const result=await db.batch([
			db.prepare("UPDATE device_code SET status='approved' WHERE id=? AND user_id=? AND status='pending' AND expires_at>? RETURNING id").bind(row.id,userId,Date.now()),
			db.prepare("INSERT INTO device_approval(code_hash,user_id,reservation_id,expires_at) SELECT ?,?,?,? WHERE changes()>0").bind(await hash(row.code),userId,hold.id,row.expires),
		]);
		if(!result[0]?.results.length){await release(db,hold);return "INVALID";}
		return "OK";
	}catch(e){await release(db,hold);throw e;}
}
export async function deviceMark(db:D1Database,userId:string,body:unknown){
	const input=tokenBody.safeParse(body);if(!input.success)return false;
	const row=await db.prepare("SELECT d.code_hash FROM device_approval d JOIN resource_reservation r ON r.id=d.reservation_id WHERE d.code_hash=? AND d.user_id=? AND d.expires_at>? AND r.expires_at>?").bind(await hash(input.data.device_code),userId,Date.now(),Date.now()).first();
	return !!row;
}
export async function finishDevice(db:D1Database,userId:string,body:unknown){
	const input=tokenBody.safeParse(body);if(!input.success)return;
	const codeHash=await hash(input.data.device_code);
	const row=await db.prepare("SELECT r.id,r.amount FROM device_approval d JOIN resource_reservation r ON r.id=d.reservation_id WHERE d.code_hash=? AND d.user_id=?").bind(codeHash,userId).first<{id:string;amount:number}>();
	if(!row)return;
	await db.batch([db.prepare("DELETE FROM device_approval WHERE code_hash=?").bind(codeHash),...releaseStatements(db,{id:row.id,userId,kind:"devices",amount:row.amount})]);
}

// A plan that dropped (downgrade, end of the subscription or of the past_due
// grace) keeps only the devices it allows: the most recently used Studio
// and Notes sessions stay, the oldest by last use (updated_at) leave. The
// panel's browser session is not a device and never counts. Approvals still
// waiting for their token are cancelled when they would pass the limit.
// Returns how many sessions were removed (the caller closes live sync).
export async function disconnectExcessDevices(db:D1Database,userId:string,limit:number,now=Date.now()){
	if(!Number.isSafeInteger(limit)||limit<0)return 0;
	const removed=await db.prepare("DELETE FROM session WHERE id IN (SELECT id FROM session WHERE user_id=? AND studio_device=1 AND expires_at>? ORDER BY updated_at DESC,created_at DESC,id DESC LIMIT -1 OFFSET ?) RETURNING id").bind(userId,now,limit).all<{id:string}>();
	const kept=await db.prepare("SELECT count(*) AS n FROM session WHERE user_id=? AND studio_device=1 AND expires_at>?").bind(userId,now).first<{n:number}>();
	const pending=await db.prepare("SELECT d.code_hash AS codeHash,r.id,r.amount FROM device_approval d JOIN resource_reservation r ON r.id=d.reservation_id WHERE d.user_id=? ORDER BY d.expires_at").bind(userId).all<{codeHash:string;id:string;amount:number}>();
	const room=Math.max(0,limit-(kept?.n??0)),cancel=pending.results.slice(room);
	if(cancel.length)await db.batch(cancel.flatMap(p=>[db.prepare("DELETE FROM device_approval WHERE code_hash=? AND user_id=?").bind(p.codeHash,userId),...releaseStatements(db,{id:p.id,userId,kind:"devices",amount:p.amount})]));
	return removed.results.length;
}
