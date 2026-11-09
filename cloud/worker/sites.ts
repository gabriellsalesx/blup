import { queueSitePurge } from "./site-cache-purge";
import { deletePrefix } from "./lifecycle";
import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, desc, eq, inArray } from "drizzle-orm";
import { z } from "zod";

import { v7 as uuidv7 } from "uuid";
import { db } from "./auth";
import { requestSession } from "./request-session";
import { site } from "./db/schema";
import { reserve, release, releaseStatements, reservationStatements, settle, siteLock, unlockSite, unlockStatement } from "./quota";
import { idParam, hashId, validate, safeInt } from "./validation";
import { removeDomainProvider,DomainError } from "./domain-provider";
import { storageLimitOf } from "./sync-storage";
import { siteDowngrade } from "./site-entitlement";
import { opaqueId } from "./validation";
import { domainsApi } from "./domains";
import { receiveFile } from "./upload";
import { accountAccess, type Access } from "./account";
import { ownSite as ownDealsPage, siteAddress as dealsPageAddress } from "./deals/sites";
import { HASH, deploySchema, fileKey, manifestKey, slugAllowed, slugCandidates, slugProblem, toManifest } from "./site-rules";

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

type App = { Bindings: Cloudflare.Env; Variables: { userId: string; access: Access } };

const MB = 1024 * 1024;

// The session (shared with the rate limit), then the account and its plan
// in one trip to D1.
const signedIn = createMiddleware<App>(async (c, next) => {
	const s = await requestSession(c.req.raw);
	const access = s ? await accountAccess(c.env.DB, s.user.id) : null;
	if (!s || !access?.allowed) return c.json({ code: "UNAUTHORIZED" }, 401);
	c.set("userId", s.user.id);
	c.set("access", access);
	await next();
});


const ownedSite=(d1:D1Database,userId:string,id:string)=>d1.prepare("SELECT id FROM site WHERE id=? AND user_id=?").bind(id,userId);
const blobRead=(d1:D1Database,siteId:string,hash:string)=>d1.prepare("SELECT hash FROM site_blob WHERE site_id=? AND hash=?").bind(siteId,hash);
// Which of these hashes the site stores, in chunks of 99 (D1's bound
// values) sent in one batch.
async function storedHashes(d1:D1Database,siteId:string,hashes:string[]){
	const lookups:D1PreparedStatement[]=[];
	for(let i=0;i<hashes.length;i+=99){const part=hashes.slice(i,i+99);lookups.push(d1.prepare(`SELECT hash FROM site_blob WHERE site_id=? AND hash IN (${part.map(()=>"?").join(",")})`).bind(siteId,...part));}
	const pages=lookups.length?await d1.batch<{hash:string}>(lookups):[];
	return new Set(pages.flatMap(page=>page.results.map(r=>r.hash)));
}

async function ownSite(userId: string, id: string) {
	const [s] = await db.select().from(site).where(and(eq(site.id, id), eq(site.userId, userId))).limit(1);
	return s;
}


// The site's address, as the visitor types it (on this computer, the
// sites' Worker runs with npx wrangler dev -c wrangler.sites.jsonc --port 8788).
const siteURL = (env: Cloudflare.Env, slug: string) => {
	const local = env.SITES_DOMAIN === "localhost";
	return `${local ? "http" : "https"}://${slug}.${env.SITES_DOMAIN}${local ? ":8788" : ""}/`;
};

// Every R2 object of a site, in pages of 1000 (list and delete both take that many).


const created = z.strictObject({ slug: z.string().refine(slugAllowed, "slug"), title: z.string().trim().min(1).max(200) });

export const sitesApi = new Hono<App>()
	.use(signedIn)
	.route("/:id/domain",domainsApi)

	// The account's sites and how many the plan allows (0 on Free), with
	// the Deals page, which counts as one.
	.get("/", async (c) => {
		const userId = c.get("userId");
		const {plan}=c.get("access");
		const [rows, deals] = await Promise.all([
			db.select({ id: site.id, slug: site.slug, title: site.title, bytes: site.bytes, files: site.files, live: site.deploy, updatedAt: site.updatedAt }).from(site).where(eq(site.userId, userId)).orderBy(desc(site.updatedAt)),
			ownDealsPage(c.env,userId),
		]);
		const downgrade=await siteDowngrade(c.env.DB,userId,plan,rows.length);
		const expired=downgrade!==null&&Date.parse(downgrade.deadline)<=Date.now();
		return c.json({
			limit: plan.limits.sites,downgrade,
			// The Deals page takes one of the plan's sites too (made and
			// edited in the Deals panel).
			deals: deals?{title:deals.title,channelId:deals.channelId,url:dealsPageAddress(c.env,deals.slug)}:null,
			sites: rows.map((r) => ({ ...r,published:r.live!=="",live:r.live!==""&&(!expired||downgrade.retainedSiteIds.includes(r.id)), url: siteURL(c.env, r.slug) })),
		});
	})

	.put("/retained",validate("json",z.strictObject({siteId:opaqueId})),async c=>{
		const userId=c.get("userId"),{siteId}=c.req.valid("json");
		if(!await ownSite(userId,siteId))return c.json({code:"NOT_FOUND"},404);
		const {plan}=c.get("access");
		const count=await c.env.DB.prepare("SELECT count(*) AS n FROM site WHERE user_id=?").bind(userId).first<{n:number}>();
		if(plan.limits.sites!==1||!count||count.n<=1)return c.json({code:"SELECTION_CLOSED"},409);
		const won=await c.env.DB.prepare("UPDATE resource_quota SET selected_site_id=? WHERE user_id=? AND grace_until>? AND deleted_at IS NULL AND EXISTS(SELECT 1 FROM site WHERE id=? AND user_id=?) RETURNING user_id").bind(siteId,userId,Date.now(),siteId,userId).first();
		return won?c.json({ok:true}):c.json({code:"SELECTION_CLOSED"},409);
	})

	// Whether a name can be the address of a new site, while it is typed:
	// free, the person's own, taken (with names close to it that are free),
	// kept by Blup, or not a valid name.
	.get("/check", validate("query", z.object({ slug: z.string().max(64) })), async (c) => {
		const userId = c.get("userId");
		const { slug } = c.req.valid("query");
		const problem = slugProblem(slug);
		if (problem) return c.json({ status: problem, suggestions: [] });
		const [owner] = await db.select({ userId: site.userId }).from(site).where(eq(site.slug, slug)).limit(1);
		if (!owner) return c.json({ status: "free", suggestions: [] });
		if (owner.userId === userId) return c.json({ status: "yours", suggestions: [] });
		const candidates = slugCandidates(slug);
		const taken = new Set((await db.select({ slug: site.slug }).from(site).where(inArray(site.slug, candidates))).map((r) => r.slug));
		return c.json({ status: "taken", suggestions: candidates.filter((s) => !taken.has(s)).slice(0, 3) });
	})

	.post("/", validate("json", created), async (c) => {
		const userId = c.get("userId");
		const { slug, title } = c.req.valid("json");
		const {plan} = c.get("access");
		if (plan.limits.sites === 0) return c.json({ code: "PLAN" }, 403);
		const reservation = await reserve(c.env.DB,userId,"sites",1,plan.limits.sites);
		if (!reservation) return c.json({code:"SITES_LIMIT",limit:plan.limits.sites},403);
		const id = uuidv7();
		try {
			await settle(c.env.DB,reservation,[c.env.DB.prepare("INSERT INTO site(id,user_id,slug,title) VALUES (?,?,?,?)").bind(id,userId,slug,title)]);
		} catch (e) {
			if (String(e).includes("UNIQUE")) return c.json({code:"SLUG_TAKEN"},409);
			throw e;
		} finally {await release(c.env.DB,reservation);}
		return c.json({ id, slug, title, url: siteURL(c.env, slug) }, 201);
	})

	// Taking a site down: its address stops answering and its files go.
	.delete("/:id", validate("param",idParam), async (c) => {
		const s = await ownSite(c.get("userId"), c.req.valid("param").id);
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		const lock=await siteLock(c.env.DB,s.id),domainLock=await siteLock(c.env.DB,`domain:${s.id}`);
		if(!lock||!domainLock){if(lock)await unlockSite(c.env.DB,s.id,lock);if(domainLock)await unlockSite(c.env.DB,`domain:${s.id}`,domainLock);return c.json({code:"DEPLOY_BUSY"},409);}
		try{
			const domain=await c.env.DB.prepare("SELECT hostname,provider_id AS providerId,verified_at AS verifiedAt FROM site_domain WHERE site_id=?").bind(s.id).first<{hostname:string;providerId:string|null;verifiedAt:number|null}>();
			if(domain)await removeDomainProvider(c.env,domain);
			await c.env.DB.batch([
				queueSitePurge(c.env.DB,s.id),
				c.env.DB.prepare("DELETE FROM site WHERE id=?").bind(s.id),
				c.env.DB.prepare("DELETE FROM site_blob WHERE site_id=?").bind(s.id),
				c.env.DB.prepare("DELETE FROM site_domain WHERE site_id=?").bind(s.id),
				c.env.DB.prepare("DELETE FROM retired_deploy WHERE site_id=?").bind(s.id),
				c.env.DB.prepare("INSERT INTO retired_deploy(site_id,deploy,delete_after) VALUES (?,'gc:',?)").bind(s.id,Date.now()+600_000),
			]);
			c.executionCtx.waitUntil(deletePrefix(c.env.SITES,`s/${s.id}/`));
		}catch(e){if(e instanceof DomainError)return c.json({code:e.code},e.code==="DOMAIN_UNAVAILABLE"?503:e.code==="DOMAIN_TAKEN"?409:502);throw e;}
		finally{await unlockSite(c.env.DB,s.id,lock);await unlockSite(c.env.DB,`domain:${s.id}`,domainLock);}

		return c.json({ ok: true });
	})

	// Which of these files the site doesn't have yet (at most 1000 per call).
	.post("/:id/missing", validate("param",idParam), validate("json", z.strictObject({ hashes: z.array(z.string().regex(HASH)).max(1000) })), async (c) => {
		const s = await ownSite(c.get("userId"), c.req.valid("param").id);
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		const hashes = [...new Set(c.req.valid("json").hashes)];
		const found = await storedHashes(c.env.DB, s.id, hashes);
		return c.json({ missing: hashes.filter((h) => !found.has(h)) });
	})

	// One file, as stored; its name is the SHA-256 of these bytes.
	.put("/:id/files/:hash", validate("param",idParam.extend({hash:hashId})), async (c) => {
		const userId = c.get("userId");
		const { id, hash } = c.req.valid("param");
		const { plan, state } = c.get("access");
		const rawLength=c.req.header("content-length"),length=rawLength===undefined?null:safeInt.safeParse(rawLength);
		// The refusals that need no lock, in the order they always had: not
		// found, plan, already stored, invalid or too large.
		const early=plan.limits.sites===0?{code:"PLAN",status:403} as const:length&&!length.success?{code:"INVALID",status:400} as const:length?.success&&length.data>plan.limits.maxFileMB*MB?{code:"TOO_LARGE",status:413} as const:null;
		if(early){
			const [owned,have]=await c.env.DB.batch([ownedSite(c.env.DB,userId,id),blobRead(c.env.DB,id,hash)]);
			if(!owned?.results.length)return c.json({code:"NOT_FOUND"},404);
			if(early.code!=="PLAN"&&have?.results.length)return c.json({ok:true});
			return early.code==="TOO_LARGE"?c.json({code:early.code,limitMB:plan.limits.maxFileMB},413):c.json({code:early.code},early.status);
		}
		// Reserve before receiving a large object. Unknown lengths are bounded
		// to a single sync-sized chunk and reserve that upper bound.
		const upper=length?.success?length.data:Math.min(4*MB,plan.limits.maxFileMB*MB);
		// One trip to D1: the blob's lock (only for the owner's site, and only
		// while the file is not stored yet), the reservation (only under that
		// lock) and the clean-up mark for a write that never settles. Each
		// statement reads what the ones before it left, in one transaction.
		const lockKey=`blob:${id}:${hash}`,lock=crypto.randomUUID(),now=Date.now();
		const held={sql:"EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=?)",values:[lockKey,lock]};
		const hold=reservationStatements(c.env.DB,userId,"storage",upper,storageLimitOf(plan,state),undefined,undefined,held);
		const claimed=await c.env.DB.batch([
			ownedSite(c.env.DB,userId,id),
			blobRead(c.env.DB,id,hash),
			c.env.DB.prepare("INSERT INTO site_lease(site_id,token,expires_at) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM site WHERE id=? AND user_id=?) AND NOT EXISTS(SELECT 1 FROM site_blob WHERE site_id=? AND hash=?) ON CONFLICT(site_id) DO UPDATE SET token=excluded.token,expires_at=excluded.expires_at WHERE expires_at<=? RETURNING token").bind(lockKey,lock,now+180_000,id,userId,id,hash,now),
			...hold.statements,
			c.env.DB.prepare(`INSERT OR IGNORE INTO retired_deploy(site_id,deploy,delete_after) SELECT ?,'gc:',? WHERE ${held.sql}`).bind(id,now+600_000,...held.values),
		]);
		if(!claimed[0]?.results.length)return c.json({code:"NOT_FOUND"},404);
		if(claimed[1]?.results.length)return c.json({ok:true});
		if(!claimed[2]?.results.length)return c.json({code:"DEPLOY_BUSY"},409);
		const reservation=hold.reservation;
		if(!claimed[4]?.results.length){await unlockStatement(c.env.DB,lockKey,lock).run();return c.json({code:"QUOTA",limitMB:plan.limits.storageMB},413);}
		let settled=false;
		try {
			// Straight to its final name: R2 checks the SHA-256 and refuses
			// other bytes, so nothing but this file is ever stored there.
			const stored=await receiveFile(c.req.raw,c.env.SITES,fileKey(id,hash),hash,plan.limits.maxFileMB*MB);
			if(stored.hash!==hash) return c.json({code:"HASH_MISMATCH"},400);
			// The blob, the reservation's release and the unlock in one batch.
			await settle(c.env.DB,reservation,[
				c.env.DB.prepare("INSERT INTO site_blob(site_id,hash,size) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM site WHERE id=? AND user_id=?) ON CONFLICT DO NOTHING").bind(id,hash,stored.size,id,userId),
				c.env.DB.prepare("UPDATE site SET bytes=bytes+? WHERE id=? AND changes()>0").bind(stored.size,id),
			],[unlockStatement(c.env.DB,lockKey,lock)]);
			settled=true;
			return c.json({ok:true},201);
		}finally{if(!settled)await c.env.DB.batch([...releaseStatements(c.env.DB,reservation),unlockStatement(c.env.DB,lockKey,lock)]);}
	})

	// A deploy: every path and its file. Live as soon as it is saved.
	.post("/:id/deploys", validate("param",idParam), validate("json", deploySchema), async (c) => {
		const userId = c.get("userId");
		const s = await ownSite(userId, c.req.valid("param").id);
		if (!s) return c.json({ code: "NOT_FOUND" }, 404);
		if (c.get("access").plan.limits.sites === 0) return c.json({ code: "PLAN" }, 403);
		const lock=await siteLock(c.env.DB,s.id);
		if(!lock) return c.json({code:"DEPLOY_BUSY"},409);
		try {
		const { files } = c.req.valid("json");
		const hashes = [...new Set(files.map((f) => f.hash))];
		const have = await storedHashes(c.env.DB, s.id, hashes);
		const missing = hashes.filter((h) => !have.has(h));
		if (missing.length) return c.json({ code: "MISSING_FILES", missing: missing.slice(0, 1000) }, 409);

		const deploy = uuidv7();
		await c.env.SITES.put(manifestKey(s.id,deploy),JSON.stringify(toManifest(files)),{httpMetadata:{contentType:"application/json"}});
		const now=Date.now();
		const committed=await c.env.DB.batch([
			c.env.DB.prepare("INSERT OR IGNORE INTO retired_deploy(site_id,deploy,delete_after) SELECT id,deploy,? FROM site WHERE id=? AND deploy<>'' AND EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?)").bind(now+600_000,s.id,s.id,lock,now),
			c.env.DB.prepare("UPDATE site SET deploy=?,files=?,updated_at=? WHERE id=? AND EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?) RETURNING id").bind(deploy,files.length,now,s.id,s.id,lock,now),
			c.env.DB.prepare("DELETE FROM retired_deploy WHERE site_id=? AND deploy LIKE 'gc:%' AND EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?)").bind(s.id,s.id,lock,now),
			c.env.DB.prepare("INSERT INTO retired_deploy(site_id,deploy,delete_after) SELECT ?,'gc:',? WHERE EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?)").bind(s.id,now+600_000,s.id,lock,now),
		]);
		if(!committed[1]?.results.length){await c.env.SITES.delete(manifestKey(s.id,deploy));return c.json({code:"DEPLOY_BUSY"},409);}
		return c.json({deploy,url:siteURL(c.env,s.slug)},201);
		} finally {await unlockSite(c.env.DB,s.id,lock);}
	});
