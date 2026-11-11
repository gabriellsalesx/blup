// D1 serializes conditional UPDATEs. Every reservation is made before an
// external operation and released together with its durable usage record.
type Resource = "storage" | "sites" | "vaults" | "ai" | "devices";
type Reservation = { id: string; userId: string; kind: Resource; amount: number };
const columns = { storage: "storage", sites: "sites", vaults: "vaults", ai: "ai", devices: "devices" } satisfies Record<Resource,string>;
const used = {
	storage: "coalesce((SELECT storage_bytes FROM sync_account WHERE user_id = resource_quota.user_id),0) + coalesce((SELECT sum(bytes) FROM site WHERE user_id = resource_quota.user_id),0)",
	// The Blup sites and the Deals page share the plan's sites.
	sites: "(SELECT count(*) FROM site WHERE user_id = resource_quota.user_id) + (SELECT count(*) FROM deals_site WHERE user_id = resource_quota.user_id)",
	vaults: "(SELECT count(*) FROM vault WHERE user_id = resource_quota.user_id)",
	devices: "(SELECT count(*) FROM session WHERE user_id = resource_quota.user_id AND studio_device = 1 AND expires_at > ?)",
};
export const ensureQuotaStatement = (db: D1Database, userId: string) => db.prepare("INSERT OR IGNORE INTO resource_quota (user_id) SELECT id FROM user WHERE id = ?").bind(userId);
export async function ensureQuota(db: D1Database, userId: string) {
	await ensureQuotaStatement(db,userId).run();
}

// A guard is SQL that must hold for the reservation to be made, checked in
// the same statement (for instance, that a lease is still this request's).
export type ReservationGuard = { sql: string; values: unknown[] };
// The reservation as statements, to run in a caller's batch: the quota row
// is made if missing, then the conditional UPDATE (index 1, RETURNING a row
// when reserved) and the receipt.
export function reservationStatements(db: D1Database, userId: string, kind: Resource, amount: number, limit: number, weekly?: number, ttl = 10 * 60_000, guard?: ReservationGuard) {
	if (!Number.isSafeInteger(amount) || amount < 0 || !Number.isSafeInteger(limit)) throw new Error("Invalid quota reservation");
	const now = Date.now(), column = columns[kind];
	const id = crypto.randomUUID();
	const month = new Date();
	const monthStart = Date.UTC(month.getUTCFullYear(),month.getUTCMonth(),1);
	const condition = kind === "ai"
		? "ai + ? + coalesce((SELECT sum(cost_micros) FROM ai_usage WHERE user_id = resource_quota.user_id AND created_at >= ?),0) <= ? AND ai + ? + coalesce((SELECT sum(cost_micros) FROM ai_usage WHERE user_id = resource_quota.user_id AND created_at >= ?),0) <= ?"
		: kind === "storage" && amount === 0 ? "1=1" : `${column} + ? + ${used[kind]} <= ?`;
	const values = kind === "ai" ? [amount,monthStart,limit,amount,now-7*86_400_000,weekly ?? limit] : kind === "storage" && amount === 0 ? [] : kind === "devices" ? [amount,now,limit] : [amount,limit];
	const reservation: Reservation = { id, userId, kind, amount };
	return { reservation, statements: [
		ensureQuotaStatement(db,userId),
		db.prepare(`UPDATE resource_quota SET ${column} = ${column} + ? WHERE user_id = ? AND deleted_at IS NULL AND ${condition}${guard ? ` AND (${guard.sql})` : ""} RETURNING user_id`).bind(amount,userId,...values,...(guard?.values ?? [])),
		db.prepare("INSERT INTO resource_reservation (id,user_id,kind,amount,expires_at) SELECT ?,?,?,?,? WHERE changes() > 0").bind(id,userId,kind,amount,now+ttl),
	] };
}
export async function reserve(db: D1Database, userId: string, kind: Resource, amount: number, limit: number, weekly?: number, ttl = 10 * 60_000): Promise<Reservation | null> {
	const { reservation, statements } = reservationStatements(db,userId,kind,amount,limit,weekly,ttl);
	const result = await db.batch(statements);
	return result[1]?.results.length ? reservation : null;
}

// The DELETE gates the decrement. Repeated release/settle cannot subtract
// a different request's reservation, even after the expiry cleaner ran.
export function releaseStatements(db: D1Database, r: Reservation): D1PreparedStatement[] {
	return [
		db.prepare("DELETE FROM resource_reservation WHERE id = ? RETURNING id").bind(r.id),
		db.prepare(`UPDATE resource_quota SET ${columns[r.kind]} = max(0, ${columns[r.kind]} - ?) WHERE user_id = ? AND changes() > 0`).bind(r.amount,r.userId),
	];
}
export async function release(db: D1Database, r: Reservation) { await db.batch(releaseStatements(db,r)); }
// `after`: statements committed together with the usage, after the
// reservation is released (for instance, the request's own unlock).
export async function settle(db: D1Database, r: Reservation, writes: D1PreparedStatement[], after: D1PreparedStatement[] = []) {
	// A request whose lease expired must never commit unreserved usage.
	await db.batch([
		db.prepare("UPDATE resource_quota SET storage=CASE WHEN EXISTS(SELECT 1 FROM resource_reservation WHERE id=? AND expires_at>?) THEN storage ELSE NULL END WHERE user_id=?").bind(r.id,Date.now(),r.userId),
		...writes,...releaseStatements(db,r),...after,
	]);
}

export async function expireReservations(db: D1Database, now = Date.now()) {
	const rows = await db.prepare("SELECT id,user_id AS userId,kind,amount FROM resource_reservation WHERE expires_at <= ? ORDER BY expires_at LIMIT 50").bind(now).all<Reservation>();
	for (const r of rows.results) await release(db,r);
	await db.prepare("DELETE FROM site_lease WHERE site_id IN (SELECT site_id FROM site_lease WHERE expires_at<=? ORDER BY expires_at LIMIT 100)").bind(now).run();
	await db.prepare("DELETE FROM device_approval WHERE code_hash IN (SELECT code_hash FROM device_approval WHERE expires_at<=? ORDER BY expires_at LIMIT 100)").bind(now).run();
	await db.prepare("DELETE FROM device_pending WHERE code_hash IN (SELECT code_hash FROM device_pending WHERE expires_at<=? ORDER BY expires_at LIMIT 100)").bind(now).run();
}

// The lock as a statement for a caller's batch: RETURNING a row when taken.
export const siteLockStatement = (db: D1Database, id: string, token: string, ttl = 120_000) => db.prepare("INSERT INTO site_lease (site_id,token,expires_at) VALUES (?,?,?) ON CONFLICT(site_id) DO UPDATE SET token = excluded.token, expires_at = excluded.expires_at WHERE expires_at <= ? RETURNING token").bind(id,token,Date.now()+ttl,Date.now());
export async function siteLock(db: D1Database, id: string, ttl = 120_000) {
	const token = crypto.randomUUID();
	const row = await siteLockStatement(db,id,token,ttl).first();
	return row ? token : null;
}
export const unlockStatement = (db: D1Database, id: string, token: string) => db.prepare("DELETE FROM site_lease WHERE site_id = ? AND token = ?").bind(id,token);
export async function unlockSite(db: D1Database, id: string, token: string) {
	await unlockStatement(db,id,token).run();
}
