import { env } from "cloudflare:workers";
import { accountPlan,observePlan } from "./lifecycle";
import { historyBytes } from "./sync-history";
import { and, count, eq, gt, sum } from "drizzle-orm";
import { db } from "./auth";
import { session, site, syncAccount, vault } from "./db/schema";
import { dealsSite } from "./db/deals-schema";

// Blup Studio's requests say who they are ("Blup Studio (Linux; my-pc)");
// its sessions are its devices.
export const isStudio = eq(session.studioDevice, true);

// Blup Studio installs signed in to an account now.
async function devicesOf(userId: string): Promise<number> {
	const [{ n } = { n: 0 }] = await db.select({ n: count() }).from(session).where(and(eq(session.userId, userId), isStudio, gt(session.expiresAt, new Date())));
	return n;
}

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
	const [[acc], [{ n } = { n: 0 }], [sites], [deals], devices] = await Promise.all([
		db.select({ storageBytes: syncAccount.storageBytes, lastSyncAt: syncAccount.lastSyncAt }).from(syncAccount).where(eq(syncAccount.userId, userId)).limit(1),
		db.select({ n: count() }).from(vault).where(eq(vault.userId, userId)),
		db.select({ n: count(), bytes: sum(site.bytes) }).from(site).where(eq(site.userId, userId)),
		// The Deals page counts as one of the plan's sites.
		db.select({ n: count() }).from(dealsSite).where(eq(dealsSite.userId, userId)),
		devicesOf(userId),
	]);
	const plan=await accountPlan(env,userId);
	const state=plan?await observePlan(env.DB,userId,plan.name):null;
	const history=await historyBytes(env,userId);
	return {
		storageLimitBytes:(plan?.limits.storageMB??0)*1024*1024,
		storageGraceUntil:state?.storageGraceLimit!==null&&state?.storageGraceLimit!==undefined&&state.graceUntil?new Date(state.graceUntil).toISOString():null,
		storageGraceLimitBytes:state?.storageGraceLimit??null,historyBytes:history,
		// Notes and sites share the plan's storage.
		storageBytes: (acc?.storageBytes ?? 0) + Number(sites?.bytes ?? 0),
		vaults: n,
		sites: (sites?.n ?? 0) + (deals?.n ?? 0),
		devices,
		lastSyncAt: acc?.lastSyncAt ? acc.lastSyncAt.toISOString() : null,
	};
}
