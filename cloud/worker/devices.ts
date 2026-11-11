import { z } from "zod";
import { reserve,release,releaseStatements } from "./quota";
export const approvalBody=z.strictObject({userCode:z.string().regex(/^[A-Za-z0-9-]{6,32}$/)});
const tokenBody=z.looseObject({device_code:z.string().min(1).max(512)});
const hash=async(code:string)=>[...new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(code)))].map(n=>n.toString(16).padStart(2,"0")).join("");

// The device asking for a sign-in code: the HMAC Blup Studio and Blup Notes
// send (internal/blupcloud/device.go), 64 hex characters. Anything else is
// a device without a mark, as before.
export const DEVICE_HEADER="x-blup-device";
const clientDevice=z.string().regex(/^[0-9a-f]{64}$/);

// The device as one account knows it: the same phone or computer gives
// another id in another account, so sessions never link accounts.
export const accountDevice=async(userId:string,device:string)=>hash(`${userId}:${device}`);

// Kept when a code is made (/device/code) until it is redeemed or expires.
export async function rememberDevice(db:D1Database,deviceCode:unknown,device:string|null|undefined,expiresInSeconds:unknown){
	const mark=clientDevice.safeParse(device);
	const code=z.string().min(1).max(512).safeParse(deviceCode);
	const ttl=z.number().int().positive().max(3600).safeParse(expiresInSeconds);
	if(!mark.success||!code.success||!ttl.success)return;
	await db.prepare("INSERT INTO device_pending(code_hash,device,expires_at) VALUES (?,?,?) ON CONFLICT(code_hash) DO NOTHING").bind(await hash(code.data),mark.data,Date.now()+ttl.data*1000).run();
}

// The account's id of the device behind a code (null: no mark).
async function codeDevice(db:D1Database,userId:string,deviceCode:string){
	const row=await db.prepare("SELECT device FROM device_pending WHERE code_hash=? AND expires_at>?").bind(await hash(deviceCode),Date.now()).first<{device:string}>();
	return row?await accountDevice(userId,row.device):null;
}

export async function approveDevice(db:D1Database,userId:string,code:string,limit:number){
	const normalized=code.replace(/[^a-zA-Z0-9]/g,"").toUpperCase();
	const row=await db.prepare("SELECT id,device_code AS code,expires_at AS expires FROM device_code WHERE user_code IN (?,?) AND user_id=? AND status='pending' AND client_id='blup-studio' AND expires_at>? LIMIT 1").bind(code,normalized,userId,Date.now()).first<{id:string;code:string;expires:number}>();
	if(!row)return "INVALID";
	// The same device signing in again (installed again, signed out) takes
	// its old sessions' place when its token comes (finishDevice): they
	// don't count against the limit now.
	const device=await codeDevice(db,userId,row.code);
	const same=device?(await db.prepare("SELECT count(*) AS n FROM session WHERE user_id=? AND studio_device=1 AND device_id=? AND expires_at>?").bind(userId,device,Date.now()).first<{n:number}>())?.n??0:0;
	const hold=await reserve(db,userId,"devices",1,limit<0?Number.MAX_SAFE_INTEGER:limit+same,undefined,Math.max(1,row.expires-Date.now()));
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
// The approved code's session about to be made: whether the approval still
// holds its place, and the account's id of the device (null: no mark).
export type DeviceGrant={deviceId:string|null};
export async function deviceMark(db:D1Database,userId:string,body:unknown):Promise<DeviceGrant|null>{
	const input=tokenBody.safeParse(body);if(!input.success)return null;
	const row=await db.prepare("SELECT d.code_hash FROM device_approval d JOIN resource_reservation r ON r.id=d.reservation_id WHERE d.code_hash=? AND d.user_id=? AND d.expires_at>? AND r.expires_at>?").bind(await hash(input.data.device_code),userId,Date.now(),Date.now()).first();
	if(!row)return null;
	return {deviceId:await codeDevice(db,userId,input.data.device_code)};
}
// The session was made: the approval's place is released (the session
// counts now), the code's device forgotten, and the same device's older
// sessions removed. Returns how many were removed (the caller closes the
// account's live sync, as when a device is disconnected in the panel).
export async function finishDevice(db:D1Database,userId:string,body:unknown,session?:{id:string;deviceId?:string|null}){
	const input=tokenBody.safeParse(body);if(!input.success)return 0;
	const codeHash=await hash(input.data.device_code);
	const row=await db.prepare("SELECT r.id,r.amount FROM device_approval d JOIN resource_reservation r ON r.id=d.reservation_id WHERE d.code_hash=? AND d.user_id=?").bind(codeHash,userId).first<{id:string;amount:number}>();
	const device=session?.deviceId?clientDevice.safeParse(session.deviceId):null;
	const replace=session&&device?.success?[db.prepare("DELETE FROM session WHERE user_id=? AND studio_device=1 AND device_id=? AND id<>? RETURNING id").bind(userId,device.data,session.id)]:[];
	const results=await db.batch([
		db.prepare("DELETE FROM device_pending WHERE code_hash=?").bind(codeHash),
		...(row?[db.prepare("DELETE FROM device_approval WHERE code_hash=?").bind(codeHash),...releaseStatements(db,{id:row.id,userId,kind:"devices",amount:row.amount})]:[]),
		...replace,
	]);
	return replace.length?results.at(-1)?.results.length??0:0;
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
