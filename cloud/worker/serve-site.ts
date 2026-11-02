import { observePlan } from "./lifecycle";
import { planNamed,ownerMatches } from "./plans";
import { manifestSchema } from "./site-rules";
import { acceptsGzip, cacheControl, fileKey, manifestKey, resolvePath, type Manifest } from "./site-rules";

// Serving <slug>.getblup.site: the live deploy's manifest says which stored
// file each path is. To spend as little as possible, answers come from the
// data center's cache whenever they can (keyed by the deploy, so a new
// deploy never mixes with the old one), the site's live deploy is kept in
// memory for a few seconds and manifests stay in memory while the isolate
// lives. Text files are stored gzipped and sent as they are.

const manifests = new Map<string, Manifest>(); // by deploy
const MAX_MANIFESTS = 32;
// The browser's cache rule, kept beside the cache's own in the stored copy.
const BROWSER_CC = "x-blup-browser-cache";

async function liveDeploy(env:Cloudflare.Env,slug:string,customDomain:boolean) {
 const row=await env.DB.prepare("SELECT s.id,s.deploy,s.user_id AS owner,u.email,u.email_verified AS verified,q.grace_until AS grace,q.last_plan AS lastPlan,(SELECT plan FROM subscription WHERE reference_id=u.id AND status IN ('active','trialing','past_due') LIMIT 1) AS plan FROM site s JOIN user u ON u.id=s.user_id LEFT JOIN resource_quota q ON q.user_id=u.id WHERE s.slug=? AND u.banned=0 AND q.deleted_at IS NULL LIMIT 1").bind(slug).first<{id:string;deploy:string;owner:string;email:string;verified:number;grace:number|null;lastPlan:string|null;plan:string|null}>();
 if(!row)return null;
 const plan=planNamed(ownerMatches(env.STAFF_OWNER_EMAIL,row.email,!!row.verified)?"ultra":row.plan);
 if(customDomain&&!plan.limits.customDomain)return null;
 if(row.lastPlan!==plan.name && (plan.name!=="free"||row.grace===null)){
  await observePlan(env.DB,row.owner,plan.name);
  if(plan.name==="free")return {...row,grace:Date.now()+14*86_400_000};
 }
 if(plan.limits.sites===0&&(!row.grace||row.grace<=Date.now()))return null;
 return row;
}

async function manifestOf(env: Cloudflare.Env, siteId: string, deploy: string): Promise<Manifest | null> {
	const hit = manifests.get(deploy);
	if (hit) return hit;
	const obj = await env.SITES.get(manifestKey(siteId, deploy));
	if (!obj) return null;
	const parsed=manifestSchema.safeParse(await obj.json().catch(()=>null));
	if(!parsed.success)return null;
	const m=parsed.data;
	const oldest=manifests.keys().next().value;
	if (manifests.size >= MAX_MANIFESTS && oldest!==undefined) manifests.delete(oldest);
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

export async function serveSite(req: Request, env: Cloudflare.Env, ctx: ExecutionContext, slug: string, customDomain=false): Promise<Response> {
	if (req.method !== "GET" && req.method !== "HEAD") return new Response(null, { status: 405, headers: { allow: "GET, HEAD" } });
	const url = new URL(req.url);
	const live = await liveDeploy(env, slug,customDomain);
	if (!live?.deploy) return notFound();

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
