import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, count, desc, eq, inArray } from "drizzle-orm";
import { z } from "zod";
import { zValidator } from "@hono/zod-validator";
import { v7 as uuidv7 } from "uuid";
import { auth, db } from "./auth";
import { site, siteBlob } from "./db/schema";
import { storageUsed } from "./usage";
import { planOf } from "./account";
import { HASH, deploySchema, fileKey, manifestKey, slugAllowed, toManifest } from "./site-rules";

// Sites published through Blup Cloud (paid plans). The Studio builds the
// site with Hugo (minified), gzips the text files and sends only what this
// site doesn't have yet, by hash; a deploy then names every path's file and
// goes live at once. Another Worker serves them (worker/sites-entry.ts,
// wrangler.sites.jsonc), so the sites' traffic never touches the panel.
//
// R2 layout (SITES bucket), all under s/<site>/:
//   f/<sha256>   a file as stored (gzipped when the manifest says so)
//   d/<deploy>   the manifest of a deploy (JSON, path -> [hash, type, gz])
// Only the live deploy is kept; files no deploy uses go when the next one
// lands.

type App = { Bindings: Cloudflare.Env; Variables: { userId: string } };

const MB = 1024 * 1024;

const signedIn = createMiddleware<App>(async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (!s) return c.json({ code: "UNAUTHORIZED" }, 401);
	c.set("userId", s.user.id);
	await next();
});


async function ownSite(userId: string, id: string) {
	const [s] = await db.select().from(site).where(and(eq(site.id, id), eq(site.userId, userId))).limit(1);
	return s;
}


// The site's address, as the visitor types it (on this computer, the
// sites' Worker runs with npx wrangler dev -c wrangler.sites.jsonc --port 8788).
export const siteURL = (env: Cloudflare.Env, slug: string) => {
	const local = env.SITES_DOMAIN === "localhost";
	return `${local ? "http" : "https"}://${slug}.${env.SITES_DOMAIN}${local ? ":8788" : ""}/`;
};

// Every R2 object of a site, in pages of 1000 (list and delete both take that many).
async function deletePrefix(bucket: R2Bucket, prefix: string) {
	let cursor: string | undefined;
	do {
		const page = await bucket.list({ prefix, cursor, limit: 1000 });
		if (page.objects.length) await bucket.delete(page.objects.map((o) => o.key));
		cursor = page.truncated ? page.cursor : undefined;
	} while (cursor);
}

// Files of the site no deploy uses any more (the live one is `keep`).
async function dropUnused(env: Cloudflare.Env, siteId: string, keep: Set<string>) {
	const rows = await db.select({ hash: siteBlob.hash, size: siteBlob.size }).from(siteBlob).where(eq(siteBlob.siteId, siteId));
	const unused = rows.filter((r) => !keep.has(r.hash));
	for (let i = 0; i < unused.length; i += 1000) await env.SITES.delete(unused.slice(i, i + 1000).map((r) => fileKey(siteId, r.hash)));
	// D1 takes at most 100 bound values per statement.
	for (let i = 0; i < unused.length; i += 99) {
		const part = unused.slice(i, i + 99);
		await env.DB.batch([
			env.DB.prepare(`DELETE FROM site_blob WHERE site_id = ? AND hash IN (${part.map(() => "?").join(",")})`).bind(siteId, ...part.map((r) => r.hash)),
			env.DB.prepare("UPDATE site SET bytes = max(0, bytes - ?) WHERE id = ?").bind(part.reduce((t, r) => t + r.size, 0), siteId),
		]);
	}
}

const created = z.object({ slug: z.string().refine(slugAllowed, "slug"), title: z.string().trim().min(1).max(200) });

export const sitesApi = new Hono<App>()
	.use(signedIn)

	// The account's sites and how many the plan allows (0 on Free).
	.get("/", async (c) => {
		const userId = c.get("userId");
		const [plan, rows] = await Promise.all([
			planOf(userId),
			db.select({ id: site.id, slug: site.slug, title: site.title, bytes: site.bytes, files: site.files, live: site.deploy, updatedAt: site.updatedAt }).from(site).where(eq(site.userId, userId)).orderBy(desc(site.updatedAt)),
		]);
		return c.json({
			limit: plan.limits.sites,
			sites: rows.map((r) => ({ ...r, live: r.live !== "", url: siteURL(c.env, r.slug) })),
		});
	})

	.post("/", zValidator("json", created), async (c) => {
		const userId = c.get("userId");
		const { slug, title } = c.req.valid("json");
		const [plan, [{ n }]] = await Promise.all([planOf(userId), db.select({ n: count() }).from(site).where(eq(site.userId, userId))]);
		if (plan.limits.sites === 0) return c.json({ code: "PLAN" }, 403);
		if (plan.limits.sites > 0 && n >= plan.limits.sites) return c.json({ code: "SITES_LIMIT", limit: plan.limits.sites }, 403);
		const id = uuidv7();
		const added = await db.insert(site).values({ id, userId, slug, title }).onConflictDoNothing().returning({ id: site.id });
		if (!added.length) return c.json({ code: "SLUG_TAKEN" }, 409);
		return c.json({ id, slug, title, url: siteURL(c.env, slug) }, 201);
	})

	// Taking a site down: its address stops answering and its files go.
	.delete("/:id", async (c) => {
		const s = await ownSite(c.get("userId"), c.req.param("id"));
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		await c.env.DB.batch([c.env.DB.prepare("DELETE FROM site WHERE id = ?").bind(s.id), c.env.DB.prepare("DELETE FROM site_blob WHERE site_id = ?").bind(s.id)]);
		c.executionCtx.waitUntil(deletePrefix(c.env.SITES, `s/${s.id}/`));
		return c.json({ ok: true });
	})

	// Which of these files the site doesn't have yet (at most 1000 per call).
	.post("/:id/missing", zValidator("json", z.object({ hashes: z.array(z.string().regex(HASH)).max(1000) })), async (c) => {
		const s = await ownSite(c.get("userId"), c.req.param("id"));
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		const hashes = [...new Set(c.req.valid("json").hashes)];
		const found = new Set<string>();
		for (let i = 0; i < hashes.length; i += 99) {
			const rows = await db.select({ hash: siteBlob.hash }).from(siteBlob).where(and(eq(siteBlob.siteId, s.id), inArray(siteBlob.hash, hashes.slice(i, i + 99))));
			for (const r of rows) found.add(r.hash);
		}
		return c.json({ missing: hashes.filter((h) => !found.has(h)) });
	})

	// One file, as stored; its name is the SHA-256 of these bytes.
	.put("/:id/files/:hash", async (c) => {
		const userId = c.get("userId");
		const hash = c.req.param("hash");
		if (!HASH.test(hash)) return c.json({ code: "BAD_REQUEST" }, 400);
		const s = await ownSite(userId, c.req.param("id"));
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		const plan = await planOf(userId);
		if (plan.limits.sites === 0) return c.json({ code: "PLAN" }, 403);
		const body = await c.req.arrayBuffer();
		if (body.byteLength === 0) return c.json({ code: "BAD_REQUEST" }, 400);
		if (body.byteLength > plan.limits.maxFileMB * MB) return c.json({ code: "TOO_LARGE", limitMB: plan.limits.maxFileMB }, 413);
		const digest = [...new Uint8Array(await crypto.subtle.digest("SHA-256", body))].map((b) => b.toString(16).padStart(2, "0")).join("");
		if (digest !== hash) return c.json({ code: "HASH_MISMATCH" }, 400);
		const [have] = await db.select({ hash: siteBlob.hash }).from(siteBlob).where(and(eq(siteBlob.siteId, s.id), eq(siteBlob.hash, hash))).limit(1);
		if (have) return c.json({ ok: true });
		if ((await storageUsed(userId)) + body.byteLength > plan.limits.storageMB * MB) return c.json({ code: "QUOTA", limitMB: plan.limits.storageMB }, 413);
		await c.env.SITES.put(fileKey(s.id, hash), body);
		const added = await db.insert(siteBlob).values({ siteId: s.id, hash, size: body.byteLength }).onConflictDoNothing().returning({ hash: siteBlob.hash });
		if (added.length) await c.env.DB.prepare("UPDATE site SET bytes = bytes + ? WHERE id = ?").bind(body.byteLength, s.id).run();
		return c.json({ ok: true }, 201);
	})

	// A deploy: every path and its file. Live as soon as it is saved.
	.post("/:id/deploys", zValidator("json", deploySchema), async (c) => {
		const userId = c.get("userId");
		const s = await ownSite(userId, c.req.param("id"));
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		if ((await planOf(userId)).limits.sites === 0) return c.json({ code: "PLAN" }, 403);
		const { files } = c.req.valid("json");
		const hashes = [...new Set(files.map((f) => f.hash))];
		const have = new Set<string>();
		for (let i = 0; i < hashes.length; i += 99) {
			const rows = await db.select({ hash: siteBlob.hash }).from(siteBlob).where(and(eq(siteBlob.siteId, s.id), inArray(siteBlob.hash, hashes.slice(i, i + 99))));
			for (const r of rows) have.add(r.hash);
		}
		const missing = hashes.filter((h) => !have.has(h));
		if (missing.length) return c.json({ code: "MISSING_FILES", missing: missing.slice(0, 1000) }, 409);

		const deploy = uuidv7();
		await c.env.SITES.put(manifestKey(s.id, deploy), JSON.stringify(toManifest(files)), { httpMetadata: { contentType: "application/json" } });
		await db.update(site).set({ deploy, files: files.length, updatedAt: new Date() }).where(eq(site.id, s.id));
		c.executionCtx.waitUntil(
			(async () => {
				if (s.deploy) await c.env.SITES.delete(manifestKey(s.id, s.deploy));
				await dropUnused(c.env, s.id, new Set(hashes));
			})(),
		);
		return c.json({ deploy, url: siteURL(c.env, s.slug) }, 201);
	});
