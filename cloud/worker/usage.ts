import { env } from "cloudflare:workers";
import { observePlan,planStateRead,PAID_SUBSCRIPTION,PAST_DUE_GRACE,type PlanStateRow } from "./lifecycle";
import { historyBytes } from "./sync-history";
import { GIFT_PLAN,ownerMatches,planNamed } from "./plans";
import { planOrder } from "../shared/pulse-plans";
import { eq } from "drizzle-orm";
import { session } from "./db/schema";

// Blup Studio's requests say who they are ("Blup Studio (Linux; my-pc)");
// its sessions are its devices.
export const isStudio = eq(session.studioDevice, true);

// The rows of one statement of a batch (D1 types a batch by one row type).
const rowsOf = <T,>(result: D1Result<unknown> | undefined): T[] => (result?.results ?? []) as T[];

// What an account is using: storage of the synced notes, folders synced,
// sites published by Blup, Studio devices and the last sync. Only numbers
// (the notes themselves are encrypted on the person's computer). Both the
// account panel and the staff read this one place.

export type Usage = {
	storageBytes: number;
	storageLimitBytes:number;
	storageGraceUntil:string|null;
	storageGraceLimitBytes:number|null;
	historyBytes:number|null;
	vaults: number;
	sites: number;
	devices: number; // Blup Studio installs signed in
	lastSyncAt: string | null;
};

export async function usageOf(userId: string): Promise<Usage> {
	const now = Date.now();
	// Everything it reads in one round trip (a batch, in order): the counts
	// (storage, folders, sites, the Deals page, Studio installs signed in
	// now), the plan (accountPlan's query), the quota row made if missing
	// (as observePlan does, only for an account that isn't blocked), the
	// plan state (observePlan's read) and the old versions' bytes
	// (historyBytes, unless a folder still needs its references, which
	// goes the long way).
	const d1 = env.DB;
	const read = await d1.batch<unknown>([
		d1.prepare("SELECT storage_bytes AS storageBytes,last_sync_at AS lastSyncAt FROM sync_account WHERE user_id=? LIMIT 1").bind(userId),
		d1.prepare("SELECT count(*) AS n FROM vault WHERE user_id=?").bind(userId),
		d1.prepare("SELECT count(*) AS n,sum(bytes) AS bytes FROM site WHERE user_id=?").bind(userId),
		// The Deals page counts as one of the plan's sites.
		d1.prepare("SELECT count(*) AS n FROM deals_site WHERE user_id=?").bind(userId),
		d1.prepare("SELECT count(*) AS n FROM session WHERE user_id=? AND studio_device=1 AND expires_at>?").bind(userId, now),
		d1.prepare(`SELECT email,email_verified AS verified,(SELECT s.plan FROM subscription s WHERE s.reference_id=user.id AND ${PAID_SUBSCRIPTION} ORDER BY ${planOrder("s")} LIMIT 1) AS plan FROM user WHERE id=? AND banned=0`).bind(now - PAST_DUE_GRACE, userId),
		d1.prepare("INSERT OR IGNORE INTO resource_quota (user_id) SELECT id FROM user WHERE id = ? AND banned=0").bind(userId),
		planStateRead(d1, userId),
		d1.prepare("SELECT id FROM vault WHERE user_id=? AND head_refs_ready=0 LIMIT 1").bind(userId),
		d1.prepare("SELECT coalesce((SELECT sum(vv.size) FROM vault_version vv JOIN vault v ON v.id=vv.vault_id WHERE v.user_id=? AND vv.n<>v.head),0)+coalesce((SELECT sum(b.size) FROM sync_block b WHERE b.user_id=? AND NOT EXISTS(SELECT 1 FROM vault_head_block h JOIN vault v ON v.id=h.vault_id WHERE h.block_id=b.id AND v.user_id=b.user_id)),0)+coalesce((SELECT sum(bytes) FROM sync_garbage WHERE user_id=?),0) AS n").bind(userId, userId, userId),
	]);
	const [acc] = rowsOf<{ storageBytes: number; lastSyncAt: number | null }>(read[0]);
	const n = rowsOf<{ n: number }>(read[1])[0]?.n ?? 0;
	const [sites] = rowsOf<{ n: number; bytes: number | null }>(read[2]);
	const [deals] = rowsOf<{ n: number }>(read[3]);
	const devices = rowsOf<{ n: number }>(read[4])[0]?.n ?? 0;
	const [account] = rowsOf<{ email: string; verified: number; plan: string | null }>(read[5]);
	const [planRow] = rowsOf<PlanStateRow>(read[7]);
	const [missing] = rowsOf<{ id: string }>(read[8]);
	const [history] = rowsOf<{ n: number }>(read[9]);
	const plan = account ? planNamed(ownerMatches(env.STAFF_OWNER_EMAIL, account.email, !!account.verified) ? GIFT_PLAN : account.plan) : null;
	const state = plan ? await observePlan(env.DB, userId, plan.name, now, planRow ?? null) : null;
	const historyTotal = missing ? await historyBytes(env, userId) : (history?.n ?? 0);
	return {
		storageLimitBytes:(plan?.limits.storageMB??0)*1024*1024,
		storageGraceUntil:state?.storageGraceLimit!==null&&state?.storageGraceLimit!==undefined&&state.graceUntil?new Date(state.graceUntil).toISOString():null,
		storageGraceLimitBytes:state?.storageGraceLimit??null,historyBytes:historyTotal,
		// Notes and sites share the plan's storage.
		storageBytes: (acc?.storageBytes ?? 0) + Number(sites?.bytes ?? 0),
		vaults: n,
		sites: (sites?.n ?? 0) + (deals?.n ?? 0),
		devices,
		lastSyncAt: acc?.lastSyncAt ? new Date(acc.lastSyncAt).toISOString() : null,
	};
}
