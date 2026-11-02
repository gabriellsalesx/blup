import { accountAllowed } from "./lifecycle";
import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, desc, eq, inArray, lt } from "drizzle-orm";
import { z } from "zod";

import { v7 as uuidv7 } from "uuid";
import { auth, db } from "./auth";
import { syncAccount, syncBlock, vault, vaultVersion } from "./db/schema";
import { planOf } from "./account";
import { reserve, release, settle } from "./quota";
import { validate, idParam, hashId, safeInt, boundedBody } from "./validation";

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
	if (!s || !await accountAllowed(c.env.DB,s.user.id)) return c.json({ code: "UNAUTHORIZED" }, 401);
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
	// The decrement binds the vault and account IDs, leaving 98 version numbers.
	const old = await db.select({ n: vaultVersion.n, key: vaultVersion.key, size: vaultVersion.size }).from(vaultVersion).where(where).limit(98);
	if (!old.length) return;
	await env.NOTES.delete(old.flatMap((o) => [`${o.key}/m`, `${o.key}/r`]));
	await env.DB.batch([
		env.DB.prepare(`UPDATE sync_account SET storage_bytes=max(0,storage_bytes-(SELECT coalesce(sum(size),0) FROM vault_version WHERE vault_id=? AND n IN (${old.map(()=>"?").join(",")}))) WHERE user_id=?`).bind(vaultId,...old.map(o=>o.n),userId),
		env.DB.prepare(`DELETE FROM vault_version WHERE vault_id = ? AND n IN (${old.map(() => "?").join(",")})`).bind(vaultId, ...old.map((o) => o.n)),
	]);
}

export const syncApi = new Hono<App>()
	.use(signedIn)

	// The live room (plans with live sync): a WebSocket that says when a
	// vault moved on another computer.
	.get("/live", async (c) => {
		if (c.req.header("upgrade") !== "websocket") return c.json({ code: "UPGRADE_REQUIRED" }, 426);
		const userId = c.get("userId");
		if (!(await planOf(userId)).limits.live) return c.json({ code: "NOT_LIVE" }, 403);
		return c.env.SYNC_LIVE.getByName(userId).fetch(c.req.raw);
	})

	// The master key, wrapped by the notes password and the recovery key.
	.get("/key", async (c) => {
		const a = await account(c.get("userId"));
		if (!a) return c.json({ code: "NO_KEY" }, 404);
		const parsed=wrappedKey.safeParse(JSON.parse(a.wrappedKey));
		if(!parsed.success)return c.json({code:"INTERNAL"},500);
		return c.json({wrapped:parsed.data});
	})
	// Created once; afterwards only the password wrapping may change (a new
	// notes password), never the recovery key's.
	.put("/key", validate("json", z.object({ wrapped: wrappedKey })), async (c) => {
		const userId = c.get("userId");
		const { wrapped } = c.req.valid("json");
		const a = await account(userId);
		if (!a) {
			await db.insert(syncAccount).values({ userId, wrappedKey: JSON.stringify(wrapped) }).onConflictDoNothing();
			return c.json({ ok: true }, 201);
		}
		const previous=wrappedKey.safeParse(JSON.parse(a.wrappedKey));
		if(!previous.success)return c.json({code:"INTERNAL"},500);
		if (previous.data.by_recovery !== wrapped.by_recovery) return c.json({ code: "RECOVERY_KEY_FIXED" }, 409);
		await db.update(syncAccount).set({ wrappedKey: JSON.stringify(wrapped), previousKey: a.wrappedKey, updatedAt: new Date() }).where(eq(syncAccount.userId, userId));
		return c.json({ ok: true });
	})

	.get("/vaults", async (c) => {
		const rows = await db.select({ id: vault.id, name: vault.name, head: vault.head, updatedAt: vault.updatedAt }).from(vault).where(eq(vault.userId, c.get("userId"))).orderBy(desc(vault.id)).limit(500);
		return c.json({ vaults: rows });
	})
	.post("/vaults", validate("json", z.object({ name: b64.min(1) })), async (c) => {
		const userId = c.get("userId");
		if (!(await account(userId))) return c.json({ code: "NO_KEY" }, 409);
		const id = uuidv7();
		const hold=await reserve(c.env.DB,userId,"vaults",1,50);
		if(!hold) return c.json({code:"VAULTS_LIMIT",limit:50},403);
		try{await settle(c.env.DB,hold,[c.env.DB.prepare("INSERT INTO vault(id,user_id,name) VALUES (?,?,?)").bind(id,userId,c.req.valid("json").name)]);}
		finally{await release(c.env.DB,hold);}
		return c.json({ id, head: 0 }, 201);
	})
	.get("/vaults/:id", validate("param",idParam), async (c) => {
		const v = await ownVault(c.get("userId"), c.req.param("id"));
		return v ? c.json(v) : c.json({ code: "NOT_FOUND" }, 404);
	})

	// The versions kept, newest first (the history).
	.get("/vaults/:id/versions", validate("param",idParam), validate("query", z.object({ before: safeInt.pipe(z.number().positive()).optional() })), async (c) => {
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
	.get("/vaults/:id/versions/:n", validate("param",idParam.extend({n:safeInt.pipe(z.number().positive())})), async (c) => {
		const id = c.req.param("id");
		if (!(await ownVault(c.get("userId"), id))) return c.json({ code: "NOT_FOUND" }, 404);
		const [row] = await db.select({ key: vaultVersion.key }).from(vaultVersion).where(and(eq(vaultVersion.vaultId, id), eq(vaultVersion.n, c.req.valid("param").n))).limit(1);
		const obj = row && (await c.env.NOTES.get(`${row.key}/m`));
		if (!obj) return c.json({ code: "NOT_FOUND" }, 404);
		return new Response(obj.body, { headers: { "content-type": "application/octet-stream" } });
	})

	// A new version, made from parent: multipart with "parent", "blocks"
	// (the ids it uses, one per line) and "manifest" (sealed).
	.post("/vaults/:id/versions", validate("param",idParam), async (c) => {
		const userId = c.get("userId");
		const id = c.req.param("id");
		const v = await ownVault(userId, id);
		if (!v) return c.json({ code: "NOT_FOUND" }, 404);
		const form=await c.req.formData().catch(()=>null);
		if(!form) return c.json({code:"INVALID"},400);
		const parsed=z.object({parent:safeInt,blocks:z.string().max(1_500_000).transform(s=>s.split("\n").filter(Boolean)).pipe(z.array(hashId).max(20_000)),manifest:z.instanceof(File).refine(f=>f.size>0&&f.size<=MAX_MANIFEST)}).safeParse({parent:form.get("parent"),blocks:form.get("blocks"),manifest:form.get("manifest")});
		if(!parsed.success || [...form.keys()].some(k=>form.getAll(k).length!==1)) return c.json({code:"INVALID"},400);
		const {parent,blocks,manifest}=parsed.data;
		if (parent !== v.head) return c.json({ code: "MOVED", head: v.head }, 409);
		const [plan, acc] = await Promise.all([planOf(userId), account(userId)]);
		const refs = new TextEncoder().encode([...new Set(blocks)].join("\n"));
		const size = manifest.size + refs.byteLength;
		if (!acc) return c.json({ code: "NO_KEY" }, 409);
		const hold=await reserve(c.env.DB,userId,"storage",size,plan.limits.storageMB*MB);
		if(!hold) return c.json({code:"QUOTA",limitMB:plan.limits.storageMB},413);

		const n = parent + 1;
		const key = `u/${userId}/v/${id}/${n}-${uuidv7()}`;
		try {
		await Promise.all([c.env.NOTES.put(`${key}/m`, manifest.stream()), c.env.NOTES.put(`${key}/r`, refs)]);
		const now = Date.now();
		const res = await c.env.DB.batch([
			c.env.DB.prepare("INSERT INTO vault_version (vault_id, n, key, size, created_at) SELECT id, ?, ?, ?, ? FROM vault WHERE id = ? AND user_id = ? AND head = ?").bind(n, key, size, now, id, userId, parent),
			c.env.DB.prepare("UPDATE vault SET head = ?, updated_at = ? WHERE id = ? AND user_id = ? AND head = ?").bind(n, now, id, userId, parent),
			c.env.DB.prepare("UPDATE sync_account SET storage_bytes=storage_bytes+?,last_sync_at=?,updated_at=? WHERE user_id=? AND changes()>0").bind(size,now,now,userId),
		]);
		if (!res[1].meta.changes) {
			await c.env.NOTES.delete([`${key}/m`, `${key}/r`]);
			const now = await ownVault(userId, id);
			return c.json({ code: "MOVED", head: now?.head ?? 0 }, 409);
		}

		c.executionCtx.waitUntil(dropOldVersions(c.env, userId, id, n, plan.limits.historyDays));
		// The account's other computers hear of it at once (live plans).
		if (plan.limits.live) c.executionCtx.waitUntil(c.env.SYNC_LIVE.getByName(userId).notify(id, n));
		return c.json({ n }, 201);
		}catch(e){await c.env.NOTES.delete([`${key}/m`,`${key}/r`]);throw e;}
		finally{await release(c.env.DB,hold);}
	})

	// Which of these blocks are not stored yet (at most 1000 per call).
	.post("/blocks/missing", validate("json", z.object({ ids: z.array(z.string().regex(BLOCK_ID)).max(1000) })), async (c) => {
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
	.put("/blocks/:id", validate("param",z.object({id:hashId})), async (c) => {
		const userId = c.get("userId");
		const id = c.req.param("id");
		if (!BLOCK_ID.test(id)) return c.json({ code: "BAD_REQUEST" }, 400);
		const body = await boundedBody(c.req.raw,MAX_BLOCK);
		if (body.byteLength === 0 || body.byteLength > MAX_BLOCK) return c.json({ code: "BAD_REQUEST" }, 400);
		const [have] = await db.select({ id: syncBlock.id }).from(syncBlock).where(and(eq(syncBlock.userId, userId), eq(syncBlock.id, id))).limit(1);
		if (have) return c.json({ ok: true });
		const [plan, acc] = await Promise.all([planOf(userId), account(userId)]);
		if (!acc) return c.json({ code: "NO_KEY" }, 409);
		const hold=await reserve(c.env.DB,userId,"storage",body.byteLength,plan.limits.storageMB*MB);
		if(!hold) return c.json({code:"QUOTA",limitMB:plan.limits.storageMB},413);
		try{
			await c.env.NOTES.put(blockKey(userId,id),body);
			await settle(c.env.DB,hold,[
				c.env.DB.prepare("INSERT INTO sync_block(user_id,id,size) VALUES (?,?,?) ON CONFLICT DO NOTHING").bind(userId,id,body.byteLength),
				c.env.DB.prepare("UPDATE sync_account SET storage_bytes=storage_bytes+? WHERE user_id=? AND changes()>0").bind(body.byteLength,userId),
			]);
			return c.json({ok:true},201);
		}finally{await release(c.env.DB,hold);}

	})
	.get("/blocks/:id", validate("param",z.object({id:hashId})), async (c) => {
		const id = c.req.param("id");
		if (!BLOCK_ID.test(id)) return c.json({ code: "BAD_REQUEST" }, 400);
		const obj = await c.env.NOTES.get(blockKey(c.get("userId"), id));
		if (!obj) return c.json({ code: "NOT_FOUND" }, 404);
		return new Response(obj.body, { headers: { "content-type": "application/octet-stream", "cache-control": "private, max-age=31536000, immutable" } });
	});
