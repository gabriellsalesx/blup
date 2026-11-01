import { and, count, eq, gt } from "drizzle-orm";
import { db } from "./auth";
import { session, syncAccount, vault } from "./db/schema";

// Blup Studio's requests say who they are; its sessions are its devices.
export const STUDIO_AGENT = "Blup Studio";

// Blup Studio installs signed in to an account now.
export async function devicesOf(userId: string): Promise<number> {
	const [{ n }] = await db.select({ n: count() }).from(session).where(and(eq(session.userId, userId), eq(session.userAgent, STUDIO_AGENT), gt(session.expiresAt, new Date())));
	return n;
}

// What an account is using: storage of the synced notes, folders synced,
// sites published by Blup, Studio devices and the last sync. Only numbers
// (the notes themselves are encrypted on the person's computer). Sites stay
// 0 until publishing through the cloud exists. Both the
// account panel and the staff read this one place.

export type Usage = {
	storageBytes: number;
	vaults: number;
	sites: number;
	devices: number; // Blup Studio installs signed in
	lastSyncAt: string | null;
};

export async function usageOf(userId: string): Promise<Usage> {
	const [[acc], [{ n }], devices] = await Promise.all([
		db.select({ storageBytes: syncAccount.storageBytes, lastSyncAt: syncAccount.lastSyncAt }).from(syncAccount).where(eq(syncAccount.userId, userId)).limit(1),
		db.select({ n: count() }).from(vault).where(eq(vault.userId, userId)),
		devicesOf(userId),
	]);
	return {
		storageBytes: acc?.storageBytes ?? 0,
		vaults: n,
		sites: 0,
		devices,
		lastSyncAt: acc?.lastSyncAt ? acc.lastSyncAt.toISOString() : null,
	};
}
