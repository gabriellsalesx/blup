import { and, count, eq, gt, like, sum } from "drizzle-orm";
import { db } from "./auth";
import { session, site, syncAccount, vault } from "./db/schema";

// Blup Studio's requests say who they are ("Blup Studio (Linux; my-pc)");
// its sessions are its devices.
export const STUDIO_AGENT = "Blup Studio";
export const isStudio = like(session.userAgent, `${STUDIO_AGENT}%`);

// Blup Studio installs signed in to an account now.
export async function devicesOf(userId: string): Promise<number> {
	const [{ n }] = await db.select({ n: count() }).from(session).where(and(eq(session.userId, userId), isStudio, gt(session.expiresAt, new Date())));
	return n;
}

// What an account is using: storage of the synced notes, folders synced,
// sites published by Blup, Studio devices and the last sync. Only numbers
// (the notes themselves are encrypted on the person's computer). Both the
// account panel and the staff read this one place.

export type Usage = {
	storageBytes: number;
	vaults: number;
	sites: number;
	devices: number; // Blup Studio installs signed in
	lastSyncAt: string | null;
};

export async function usageOf(userId: string): Promise<Usage> {
	const [[acc], [{ n }], [sites], devices] = await Promise.all([
		db.select({ storageBytes: syncAccount.storageBytes, lastSyncAt: syncAccount.lastSyncAt }).from(syncAccount).where(eq(syncAccount.userId, userId)).limit(1),
		db.select({ n: count() }).from(vault).where(eq(vault.userId, userId)),
		db.select({ n: count(), bytes: sum(site.bytes) }).from(site).where(eq(site.userId, userId)),
		devicesOf(userId),
	]);
	return {
		// Notes and sites share the plan's storage.
		storageBytes: (acc?.storageBytes ?? 0) + Number(sites?.bytes ?? 0),
		vaults: n,
		sites: sites?.n ?? 0,
		devices,
		lastSyncAt: acc?.lastSyncAt ? acc.lastSyncAt.toISOString() : null,
	};
}

// Bytes the account stores: its notes and its sites share the plan's
// storage.
export async function storageUsed(userId: string): Promise<number> {
	const [[acc], [sites]] = await Promise.all([
		db.select({ bytes: syncAccount.storageBytes }).from(syncAccount).where(eq(syncAccount.userId, userId)).limit(1),
		db.select({ bytes: sum(site.bytes) }).from(site).where(eq(site.userId, userId)),
	]);
	return (acc?.bytes ?? 0) + Number(sites?.bytes ?? 0);
}
