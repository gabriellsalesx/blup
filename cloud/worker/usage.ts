import { count, eq } from "drizzle-orm";
import { db } from "./auth";
import { syncAccount, vault } from "./db/schema";

// What an account is using: storage of the synced notes, sites synced
// through the cloud, AI credits this month, Studio devices and the last
// sync. Only numbers (the notes themselves are encrypted on the person's
// computer). AI credits and devices stay 0 until those exist. Both the
// account panel and the staff read this one place.

export type Usage = {
	storageBytes: number;
	sites: number;
	aiCreditsUsed: number;
	aiCreditsResetAt: string | null; // when this month's credits start over
	devices: number; // Blup Studio installs signed in
	lastSyncAt: string | null;
};

export async function usageOf(userId: string, periodEnd?: Date | null): Promise<Usage> {
	const [[acc], [{ n }]] = await Promise.all([
		db.select({ storageBytes: syncAccount.storageBytes, lastSyncAt: syncAccount.lastSyncAt }).from(syncAccount).where(eq(syncAccount.userId, userId)).limit(1),
		db.select({ n: count() }).from(vault).where(eq(vault.userId, userId)),
	]);
	return {
		storageBytes: acc?.storageBytes ?? 0,
		sites: n,
		aiCreditsUsed: 0,
		aiCreditsResetAt: periodEnd ? periodEnd.toISOString() : null,
		devices: 0,
		lastSyncAt: acc?.lastSyncAt ? acc.lastSyncAt.toISOString() : null,
	};
}
