import { and, asc, desc, eq, gt, isNull, lt, or } from "drizzle-orm";
import { db } from "./auth";
import { syncAccount, syncBlock, vault, vaultVersion } from "./db/schema";
import { planOf } from "./account";
import { thin, type Version } from "./history";

// The daily clean-up (Cron Trigger). For every account that synced:
//
// 1. Old versions thin out: every version of the last day stays, one per
//    hour of the last week, one per day after that, up to the plan's
//    history (none beyond the current one without history).
// 2. Blocks that no version uses any more are deleted (only blocks older
//    than a day: a sync in progress uploads blocks before its version).
//
// Only ids, sizes and dates are read here: the notes stay sealed.

const HOUR = 3_600_000;
const DAY = 24 * HOUR;

async function cleanAccount(env: Cloudflare.Env, userId: string, now: number, cleanedAt: Date | null) {
	const plan = await planOf(userId);
	const vaults = await db.select({ id: vault.id }).from(vault).where(eq(vault.userId, userId));
	let freed = 0;
	let changed = false;
	const used = new Set<string>();
	const kept: Version[] = [];
	for (const { id } of vaults) {
		const versions = await db.select({ n: vaultVersion.n, key: vaultVersion.key, size: vaultVersion.size, createdAt: vaultVersion.createdAt }).from(vaultVersion).where(eq(vaultVersion.vaultId, id)).orderBy(desc(vaultVersion.n));
		const keep = thin(versions, plan.limits.historyDays, now);
		const drop = versions.filter((v) => !keep.has(v.n));
		if (drop.length || versions.some((v) => !cleanedAt || v.createdAt > cleanedAt)) changed = true;
		for (let i = 0; i < drop.length; i += 50) {
			const part = drop.slice(i, i + 50);
			await env.NOTES.delete(part.flatMap((v) => [`${v.key}/m`, `${v.key}/r`]));
			await env.DB.prepare(`DELETE FROM vault_version WHERE vault_id = ? AND n IN (${part.map(() => "?").join(",")})`).bind(id, ...part.map((v) => v.n)).run();
			freed += part.reduce((t, v) => t + v.size, 0);
		}
		kept.push(...versions.filter((v) => keep.has(v.n)));
	}
	// Nothing new and nothing dropped: no block can have become unused.
	if (!changed) {
		await env.DB.prepare("UPDATE sync_account SET cleaned_at = ? WHERE user_id = ?").bind(now, userId).run();
		return;
	}
	for (const v of kept) {
		const refs = await env.NOTES.get(`${v.key}/r`);
		for (const b of (refs ? await refs.text() : "").split("\n")) if (b) used.add(b);
	}
	// Blocks no kept version uses, a page at a time (keyset on the id).
	const cutoff = new Date(now - DAY);
	let after = "";
	for (;;) {
		const page = await db
			.select({ id: syncBlock.id, size: syncBlock.size, createdAt: syncBlock.createdAt })
			.from(syncBlock)
			.where(and(eq(syncBlock.userId, userId), gt(syncBlock.id, after)))
			.orderBy(asc(syncBlock.id))
			.limit(500);
		if (!page.length) break;
		after = page[page.length - 1].id;
		const gone = page.filter((b) => !used.has(b.id) && b.createdAt < cutoff);
		for (let i = 0; i < gone.length; i += 90) {
			const part = gone.slice(i, i + 90);
			await env.NOTES.delete(part.map((b) => `u/${userId}/b/${b.id}`));
			await env.DB.prepare(`DELETE FROM sync_block WHERE user_id = ? AND id IN (${part.map(() => "?").join(",")})`).bind(userId, ...part.map((b) => b.id)).run();
			freed += part.reduce((t, b) => t + b.size, 0);
		}
	}
	await env.DB.prepare("UPDATE sync_account SET storage_bytes = max(0, storage_bytes - ?), cleaned_at = ? WHERE user_id = ?").bind(freed, now, userId).run();
}

// Accounts that synced since their last clean-up, or whose history may have
// aged (cleaned more than a day ago).
export async function cleanUp(env: Cloudflare.Env) {
	const now = Date.now();
	const stale = new Date(now - DAY + HOUR);
	let after = "";
	for (;;) {
		const page = await db
			.select({ userId: syncAccount.userId, cleanedAt: syncAccount.cleanedAt })
			.from(syncAccount)
			.where(and(gt(syncAccount.userId, after), or(isNull(syncAccount.cleanedAt), lt(syncAccount.cleanedAt, stale))))
			.orderBy(asc(syncAccount.userId))
			.limit(100);
		if (!page.length) return;
		after = page[page.length - 1].userId;
		for (const { userId, cleanedAt } of page) {
			try {
				await cleanAccount(env, userId, now, cleanedAt);
			} catch (e) {
				console.error("[blup] limpeza falhou", userId, (e as Error).message);
			}
		}
	}
}
