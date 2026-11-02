import { eq } from "drizzle-orm";
import { drizzle } from "drizzle-orm/d1";
import { site } from "./db/blup-schema";
import { acceptsGzip, cacheControl, fileKey, manifestKey, resolvePath, type Manifest } from "./site-rules";

// Serving <slug>.getblup.site: the live deploy's manifest says which stored
// file each path is. To spend as little as possible, answers come from the
// data center's cache whenever they can (keyed by the deploy, so a new
// deploy never mixes with the old one), the site's live deploy is kept in
// memory for a few seconds and manifests stay in memory while the isolate
// lives. Text files are stored gzipped and sent as they are.

const HEAD_TTL = 30_000; // how long a new deploy may take to show (per isolate)
const MAX_HEADS = 1024;
const heads = new Map<string, { id: string; deploy: string; at: number }>();
const manifests = new Map<string, Manifest>(); // by deploy
const MAX_MANIFESTS = 32;
// The browser's cache rule, kept beside the cache's own in the stored copy.
const BROWSER_CC = "x-blup-browser-cache";

async function liveDeploy(env: Cloudflare.Env, slug: string) {
	const now = Date.now();
	const hit = heads.get(slug);
	if (hit && now - hit.at < HEAD_TTL) return hit;
	const [row] = await drizzle(env.DB).select({ id: site.id, deploy: site.deploy }).from(site).where(eq(site.slug, slug)).limit(1);
	const head = { id: row?.id ?? "", deploy: row?.deploy ?? "", at: now };
	if (heads.size >= MAX_HEADS) heads.delete(heads.keys().next().value!);
	heads.set(slug, head);
	return head;
}

async function manifestOf(env: Cloudflare.Env, siteId: string, deploy: string): Promise<Manifest | null> {
	const hit = manifests.get(deploy);
	if (hit) return hit;
	const obj = await env.SITES.get(manifestKey(siteId, deploy));
	if (!obj) return null;
	const m = (await obj.json()) as Manifest;
	if (manifests.size >= MAX_MANIFESTS) manifests.delete(manifests.keys().next().value!);
	manifests.set(deploy, m);
	return m;
}

const notFound = () => new Response("Not found", { status: 404, headers: { "content-type": "text/plain; charset=utf-8", "cache-control": "no-store" } });

async function fileResponse(env: Cloudflare.Env, req: Request, siteId: string, path: string, entry: Manifest[string], status: number) {
	const [hash, type, gz] = entry;
	const etag = `"${hash}"`;
	const headers = new Headers({ "content-type": type, etag, "cache-control": status === 200 ? cacheControl(path, type) : "no-store", "x-content-type-options": "nosniff", vary: "accept-encoding" });
	if (status === 200 && req.headers.get("if-none-match") === etag) return new Response(null, { status: 304, headers });
	const obj = await env.SITES.get(fileKey(siteId, hash));
	if (!obj) return notFound();
	const head = req.method === "HEAD";
	if (!gz) return new Response(head ? null : obj.body, { status, headers });
	// Almost every client takes gzip; the rare one that doesn't gets it
	// unpacked here.
	if (!acceptsGzip(req.headers.get("accept-encoding"))) {
		return new Response(head ? null : obj.body.pipeThrough(new DecompressionStream("gzip")), { status, headers });
	}
	headers.set("content-encoding", "gzip");
	return new Response(head ? null : obj.body, { status, headers, encodeBody: "manual" });
}

export async function serveSite(req: Request, env: Cloudflare.Env, ctx: ExecutionContext, slug: string): Promise<Response> {
	if (req.method !== "GET" && req.method !== "HEAD") return new Response(null, { status: 405, headers: { allow: "GET, HEAD" } });
	const url = new URL(req.url);
	const live = await liveDeploy(env, slug);
	if (!live.deploy) return notFound();

	// The same path of the same deploy is the same answer: kept in this data
	// center's cache (gzip clients only, which are nearly all).
	const gzipOK = acceptsGzip(req.headers.get("accept-encoding"));
	const cacheKey = new Request(`https://site-cache.internal/${slug}/${live.deploy}${url.pathname}`);
	const cache = caches.default;
	if (gzipOK) {
		const hit = await cache.match(cacheKey);
		if (hit) {
			// Passed on as it came (still gzipped), with the browser's own rule.
			const res = new Response(req.method === "HEAD" ? null : hit.body, { status: hit.status, headers: hit.headers, encodeBody: "manual" });
			res.headers.set("cache-control", hit.headers.get(BROWSER_CC) ?? "public, max-age=0, must-revalidate");
			res.headers.delete(BROWSER_CC);
			if (req.headers.get("if-none-match") === res.headers.get("etag")) return new Response(null, { status: 304, headers: res.headers });
			return res;
		}
	}

	const manifest = await manifestOf(env, live.id, live.deploy);
	if (!manifest) return notFound();
	const found = resolvePath(manifest, url.pathname);
	if (found && "redirect" in found) return new Response(null, { status: 301, headers: { location: found.redirect + url.search } });
	const path = found?.file ?? "/404.html";
	const entry = manifest[path];
	if (!entry) return notFound();
	const res = await fileResponse(env, req, live.id, path, entry, found ? 200 : 404);
	if (gzipOK && req.method === "GET" && res.status === 200) {
		// The cache entry lives as long as the deploy (its key changes with
		// the next one); the browser still gets the file's own rule.
		const stored = res.clone();
		const forCache = new Response(stored.body, { status: 200, headers: new Headers(stored.headers), encodeBody: "manual" });
		forCache.headers.set(BROWSER_CC, stored.headers.get("cache-control") ?? "");
		forCache.headers.set("cache-control", "public, max-age=31536000");
		ctx.waitUntil(cache.put(cacheKey, forCache).then(() => undefined));
		return res;
	}
	return res;
}
