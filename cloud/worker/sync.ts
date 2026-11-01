import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, desc, eq, inArray, lt } from "drizzle-orm";
import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import { v7 as uuidv7 } from "uuid";
import { auth, db } from "./auth";
import { syncAccount, syncBlock, vault, vaultVersion } from "./db/schema";
import { planOf } from "./account";

// Sync of the notes of each site (a vault), end-to-end encrypted by the
// Studio (internal/cloudcrypt): this side only ever sees sealed blocks,
// sealed manifests and sealed vault names, plus the ids of blocks.
//
// R2 layout, all under u/<user>/:
//   b/<block id>        a sealed piece of a file
//   v/<vault>/<n>-<id>/m  the sealed manifest of version n
//   v/<vault>/<n>-<id>/r  the ids of the blocks it uses (for the clean-up)
//
// A version only moves forward from the one it was made from; when
// another computer got there first the answer is 409 and the Studio
// compares again.

type App = { Bindings: Cloudflare.Env; Variables: { userId: string } };

const BLOCK_ID = /^[0-9a-f]{64}$/;
// A block holds at most 4 MiB plus the seal (cloudcrypt.BlockSize).
const MAX_BLOCK = 4 * 1024 * 1024 + 64;
const MAX_MANIFEST = 32 * 1024 * 1024;
const MB = 1024 * 1024;

const signedIn = createMiddleware<App>(async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (!s) return c.json({ code: "UNAUTHORIZED" }, 401);
	c.set("userId", s.user.id);
	await next();
});

const b64 = z.string().regex(/^[A-Za-z0-9+/]*={0,2}$/).max(1024);
const wrappedKey = z.object({
	params: z.object({ time: z.number().int().positive(), memory_kib: z.number().int().positive(), threads: z.number().int().positive(), salt: b64 }),
	by_password: b64,
	by_recovery: b64,
});

const blockKey = (user: string, id: string) => `u/${user}/b/${id}`;

async function ownVault(userId: string, id: string) {
	const [v] = await db.select({ id: vault.id, name: vault.name, head: vault.head, updatedAt: vault.updatedAt }).from(vault).where(and(eq(vault.id, id), eq(vault.userId, userId))).limit(1);
	return v;
}

async function account(userId: string) {
	const [a] = await db.select().from(syncAccount).where(eq(syncAccount.userId, userId)).limit(1);
	return a;
}

// Old versions past the plan's history (all but the head on a plan without
// history). Their blocks go with the daily clean-up, once nothing uses them.
async function dropOldVersions(env: Cloudflare.Env, userId: string, vaultId: string, head: number, historyDays: number) {
	const cutoff = new Date(Date.now() - historyDays * 86_400_000);
	const where = historyDays > 0 ? and(eq(vaultVersion.vaultId, vaultId), lt(vaultVersion.n, head), lt(vaultVersion.createdAt, cutoff)) : and(eq(vaultVersion.vaultId, vaultId), lt(vaultVersion.n, head));
	const old = await db.select({ n: vaultVersion.n, key: vaultVersion.key, size: vaultVersion.size }).from(vaultVersion).where(where).limit(100);
	if (!old.length) return;
	await env.NOTES.delete(old.flatMap((o) => [`${o.key}/m`, `${o.key}/r`]));
	const freed = old.reduce((t, o) => t + o.size, 0);
	await env.DB.batch([
		env.DB.prepare(`DELETE FROM vault_version WHERE vault_id = ? AND n IN (${old.map(() => "?").join(",")})`).bind(vaultId, ...old.map((o) => o.n)),
		env.DB.prepare("UPDATE sync_account SET storage_bytes = max(0, storage_bytes - ?) WHERE user_id = ?").bind(freed, userId),
	]);
}

export const syncApi = new Hono<App>()
	.use(signedIn)

	// The master key, wrapped by the notes password and the recovery key.
	.get("/key", async (c) => {
		const a = await account(c.get("userId"));
		if (!a) return c.json({ code: "NO_KEY" }, 404);
		return c.json({ wrapped: JSON.parse(a.wrappedKey) });
	})
	// Created once; afterwards only the password wrapping may change (a new
	// notes password), never the recovery key's.
	.put("/key", zValidator("json", z.object({ wrapped: wrappedKey })), async (c) => {
		const userId = c.get("userId");
		const { wrapped } = c.req.valid("json");
		const a = await account(userId);
		if (!a) {
			await db.insert(syncAccount).values({ userId, wrappedKey: JSON.stringify(wrapped) }).onConflictDoNothing();
			return c.json({ ok: true }, 201);
		}
		if (JSON.parse(a.wrappedKey).by_recovery !== wrapped.by_recovery) return c.json({ code: "RECOVERY_KEY_FIXED" }, 409);
		await db.update(syncAccount).set({ wrappedKey: JSON.stringify(wrapped), previousKey: a.wrappedKey, updatedAt: new Date() }).where(eq(syncAccount.userId, userId));
		return c.json({ ok: true });
	})

	.get("/vaults", async (c) => {
		const rows = await db.select({ id: vault.id, name: vault.name, head: vault.head, updatedAt: vault.updatedAt }).from(vault).where(eq(vault.userId, c.get("userId"))).orderBy(desc(vault.id)).limit(500);
		return c.json({ vaults: rows });
	})
	.post("/vaults", zValidator("json", z.object({ name: b64.min(1) })), async (c) => {
		const userId = c.get("userId");
		if (!(await account(userId))) return c.json({ code: "NO_KEY" }, 409);
		const id = uuidv7();
		await db.insert(vault).values({ id, userId, name: c.req.valid("json").name });
		return c.json({ id, head: 0 }, 201);
	})
	.get("/vaults/:id", async (c) => {
		const v = await ownVault(c.get("userId"), c.req.param("id"));
		return v ? c.json(v) : c.json({ code: "NOT_FOUND" }, 404);
	})

	// The versions kept, newest first (the history).
	.get("/vaults/:id/versions", zValidator("query", z.object({ before: z.coerce.number().int().positive().optional() })), async (c) => {
		const id = c.req.param("id");
		if (!(await ownVault(c.get("userId"), id))) return c.json({ code: "NOT_FOUND" }, 404);
		const { before } = c.req.valid("query");
		const rows = await db
			.select({ n: vaultVersion.n, size: vaultVersion.size, createdAt: vaultVersion.createdAt })
			.from(vaultVersion)
			.where(before ? and(eq(vaultVersion.vaultId, id), lt(vaultVersion.n, before)) : eq(vaultVersion.vaultId, id))
			.orderBy(desc(vaultVersion.n))
			.limit(50);
		return c.json({ versions: rows });
	})
	.get("/vaults/:id/versions/:n{[0-9]+}", async (c) => {
		const id = c.req.param("id");
		if (!(await ownVault(c.get("userId"), id))) return c.json({ code: "NOT_FOUND" }, 404);
		const [row] = await db.select({ key: vaultVersion.key }).from(vaultVersion).where(and(eq(vaultVersion.vaultId, id), eq(vaultVersion.n, Number(c.req.param("n"))))).limit(1);
		const obj = row && (await c.env.NOTES.get(`${row.key}/m`));
		if (!obj) return c.json({ code: "NOT_FOUND" }, 404);
		return new Response(obj.body, { headers: { "content-type": "application/octet-stream" } });
	})

	// A new version, made from parent: multipart with "parent", "blocks"
	// (the ids it uses, one per line) and "manifest" (sealed).
	.post("/vaults/:id/versions", async (c) => {
		const userId = c.get("userId");
		const id = c.req.param("id");
		const v = await ownVault(userId, id);
		if (!v) return c.json({ code: "NOT_FOUND" }, 404);
		const form = await c.req.formData();
		const parent = Number(form.get("parent"));
		const manifest = form.get("manifest");
		const blocks = String(form.get("blocks") ?? "").split("\n").filter(Boolean);
		if (!Number.isInteger(parent) || parent < 0 || !(manifest instanceof File) || manifest.size > MAX_MANIFEST || !blocks.every((b) => BLOCK_ID.test(b))) {
			return c.json({ code: "BAD_REQUEST" }, 400);
		}
		if (parent !== v.head) return c.json({ code: "MOVED", head: v.head }, 409);
		const [plan, acc] = await Promise.all([planOf(userId), account(userId)]);
		const refs = new TextEncoder().encode([...new Set(blocks)].join("\n"));
		const size = manifest.size + refs.byteLength;
		if (!acc) return c.json({ code: "NO_KEY" }, 409);
		if (acc.storageBytes + size > plan.limits.storageMB * MB) return c.json({ code: "QUOTA", limitMB: plan.limits.storageMB }, 413);

		const n = parent + 1;
		const key = `u/${userId}/v/${id}/${n}-${uuidv7()}`;
		await Promise.all([c.env.NOTES.put(`${key}/m`, await manifest.arrayBuffer()), c.env.NOTES.put(`${key}/r`, refs)]);
		const now = Date.now();
		const res = await c.env.DB.batch([
			c.env.DB.prepare("INSERT INTO vault_version (vault_id, n, key, size, created_at) SELECT id, ?, ?, ?, ? FROM vault WHERE id = ? AND user_id = ? AND head = ?").bind(n, key, size, now, id, userId, parent),
			c.env.DB.prepare("UPDATE vault SET head = ?, updated_at = ? WHERE id = ? AND user_id = ? AND head = ?").bind(n, now, id, userId, parent),
		]);
		if (!res[1].meta.changes) {
			await c.env.NOTES.delete([`${key}/m`, `${key}/r`]);
			const now = await ownVault(userId, id);
			return c.json({ code: "MOVED", head: now?.head ?? 0 }, 409);
		}
		await c.env.DB.prepare("UPDATE sync_account SET storage_bytes = storage_bytes + ?, last_sync_at = ?, updated_at = ? WHERE user_id = ?").bind(size, now, now, userId).run();
		c.executionCtx.waitUntil(dropOldVersions(c.env, userId, id, n, plan.limits.historyDays));
		return c.json({ n }, 201);
	})

	// Which of these blocks are not stored yet (at most 1000 per call).
	.post("/blocks/missing", zValidator("json", z.object({ ids: z.array(z.string().regex(BLOCK_ID)).max(1000) })), async (c) => {
		const userId = c.get("userId");
		const ids = [...new Set(c.req.valid("json").ids)];
		const found = new Set<string>();
		// D1 takes at most 100 bound values per statement.
		for (let i = 0; i < ids.length; i += 99) {
			const rows = await db.select({ id: syncBlock.id }).from(syncBlock).where(and(eq(syncBlock.userId, userId), inArray(syncBlock.id, ids.slice(i, i + 99))));
			for (const r of rows) found.add(r.id);
		}
		return c.json({ missing: ids.filter((id) => !found.has(id)) });
	})
	.put("/blocks/:id", async (c) => {
		const userId = c.get("userId");
		const id = c.req.param("id");
		if (!BLOCK_ID.test(id)) return c.json({ code: "BAD_REQUEST" }, 400);
		const body = await c.req.arrayBuffer();
		if (body.byteLength === 0 || body.byteLength > MAX_BLOCK) return c.json({ code: "BAD_REQUEST" }, 400);
		const [have] = await db.select({ id: syncBlock.id }).from(syncBlock).where(and(eq(syncBlock.userId, userId), eq(syncBlock.id, id))).limit(1);
		if (have) return c.json({ ok: true });
		const [plan, acc] = await Promise.all([planOf(userId), account(userId)]);
		if (!acc) return c.json({ code: "NO_KEY" }, 409);
		if (acc.storageBytes + body.byteLength > plan.limits.storageMB * MB) return c.json({ code: "QUOTA", limitMB: plan.limits.storageMB }, 413);
		await c.env.NOTES.put(blockKey(userId, id), body);
		const added = await db.insert(syncBlock).values({ userId, id, size: body.byteLength }).onConflictDoNothing().returning({ id: syncBlock.id });
		if (added.length) await c.env.DB.prepare("UPDATE sync_account SET storage_bytes = storage_bytes + ? WHERE user_id = ?").bind(body.byteLength, userId).run();
		return c.json({ ok: true }, 201);
	})
	.get("/blocks/:id", async (c) => {
		const id = c.req.param("id");
		if (!BLOCK_ID.test(id)) return c.json({ code: "BAD_REQUEST" }, 400);
		const obj = await c.env.NOTES.get(blockKey(c.get("userId"), id));
		if (!obj) return c.json({ code: "NOT_FOUND" }, 404);
		return new Response(obj.body, { headers: { "content-type": "application/octet-stream", "cache-control": "private, max-age=31536000, immutable" } });
	});
