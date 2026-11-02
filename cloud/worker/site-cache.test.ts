import { afterEach,beforeEach,describe,expect,it,vi } from "vitest";
import { TestBucket,TestD1,seedUser } from "./security-test-support";
vi.mock("cloudflare:workers",()=>({WorkerEntrypoint:class{
	constructor(readonly ctx:ExecutionContext,readonly env:Cloudflare.Env){}
}}));
import { CachedSite } from "./sites-entry";
import { serveDeploy,serveSite } from "./serve-site";
import { drainSitePurges,expiredSitePurges,queueAccountPurge,queueSitePurge,siteTags } from "./site-cache-purge";
import { softDelete,restoreAccount } from "./lifecycle";
import { providerCall } from "./domain-provider";

let db:TestD1,bucket:TestBucket,env:Cloudflare.Env;
const HASH="8".repeat(64);
beforeEach(()=>{
	db=new TestD1();bucket=new TestBucket();seedUser(db,"cache-owner","max");
	env={DB:db.asD1(),SITES:bucket.asR2(),SITES_DOMAIN:"getblup.site",SYNC_LIVE:{getByName:()=>({revoke:async()=>{}})}} as unknown as Cloudflare.Env;
	vi.stubGlobal("fetch",vi.fn(async()=>{throw new Error("Network forbidden");}));
});
afterEach(()=>{db.close();vi.unstubAllGlobals();});
async function deploy(id:string,version:string,text=id){
	db.sqlite.prepare("INSERT INTO site(id,user_id,slug,title,deploy) VALUES (?,'cache-owner',?,?,?) ON CONFLICT(id) DO UPDATE SET deploy=excluded.deploy").run(id,id,id,version);
	await bucket.put(`s/${id}/d/${version}`,JSON.stringify({"/index.html":[HASH,"text/html",0],"/css/bento.abcd.css":[HASH,"text/css",0],"/fonts/inter-latin.woff2":[HASH,"font/woff2",0]}));
	await bucket.put(`s/${id}/f/${HASH}`,text);
}
describe("Workers Cache gateway and entrypoint",()=>{
	it("isolates manifests by site even if deploy IDs collide",async()=>{
		await deploy("cache-one","same-deploy");await deploy("cache-two","same-deploy","different site");
		await bucket.put("s/cache-two/d/same-deploy",JSON.stringify({"/different.html":[HASH,"text/html",0]}));
		expect((await serveDeploy(new Request("https://internal/"),env,{siteId:"cache-one",deploy:"same-deploy"})).status).toBe(200);
		expect((await serveDeploy(new Request("https://internal/"),env,{siteId:"cache-two",deploy:"same-deploy"})).status).toBe(404);
	});
	it("passes site/deploy in props, strips bypass headers and restores browser policy",async()=>{
		await deploy("cache-one","cached-v1");
		const calls:{siteId:string;deploy:string;url:string;headers:Headers}[]=[];
		const bound=(options:{props:{siteId:string;deploy:string}})=>({fetch:async(r:Request)=>{
			calls.push({...options.props,url:r.url,headers:r.headers});return serveDeploy(r,env,options.props);
		}});
		const loopback={exports:{CachedSite:bound}} as unknown as ExecutionContext;
		const res=await serveSite(new Request("https://cache-one.getblup.site/?unbounded=query",{headers:{cookie:"private",authorization:"secret fixture","accept-encoding":"gzip;q=0, *;q=1"}}),env,loopback,"cache-one");
		expect(calls).toHaveLength(1);expect(calls[0]).toMatchObject({siteId:"cache-one",deploy:"cached-v1",url:"https://public-site.internal/"});
		expect([...calls[0]!.headers.entries()]).toEqual([["accept-encoding","identity"]]);
		expect(res.headers.get("cache-control")).toBe("public, max-age=0, must-revalidate");
		expect(res.headers.has("x-blup-browser-cache")).toBe(false);expect(res.headers.has("cache-tag")).toBe(false);
		expect(res.headers.get("link")).toContain("</css/bento.abcd.css>; rel=preload; as=style");
		expect(res.headers.get("link")).toContain("</fonts/inter-latin.woff2>; rel=preload; as=font");
	});
	it("rechecks D1 before every hit and does not invoke a cached site after revocation",async()=>{
		await deploy("cache-one","cached-revocation");
		const fetchCached=vi.fn(async()=>new Response("cached public body",{headers:{"cache-control":"public, max-age=600"}}));
		const loopback={exports:{CachedSite:()=>({fetch:fetchCached})}} as unknown as ExecutionContext;
		const req=new Request("https://cache-one.getblup.site/");
		expect((await serveSite(req,env,loopback,"cache-one")).status).toBe(200);
		db.sqlite.exec("UPDATE user SET banned=1 WHERE id='cache-owner'");
		const blocked=await serveSite(req,env,loopback,"cache-one");expect(blocked.status).toBe(404);expect(blocked.headers.get("cache-control")).toBe("no-store");expect(fetchCached).toHaveBeenCalledTimes(1);
	});
	it("tags successful deploy responses and never caches a missing file",async()=>{
		await deploy("cache-tags","tags-v1");
		const res=await serveDeploy(new Request("https://internal/"),env,{siteId:"cache-tags",deploy:"tags-v1"});
		expect(res.headers.get("cache-tag")).toBe(siteTags("cache-tags","tags-v1").join(","));
		expect(res.headers.get("cache-control")).toBe("public, max-age=600");
		const missing=await serveDeploy(new Request("https://internal/missing"),env,{siteId:"cache-tags",deploy:"tags-v1"});expect(missing.status).toBe(404);expect(missing.headers.get("cache-control")).toBe("no-store");
	});
	it("named entrypoint rejects fetch without trusted props and performs purge inside its context",async()=>{
		await queueSitePurge(env.DB,"cache-one").run();const purge=vi.fn(async()=>({success:true,errors:[]}));
		const entry=new CachedSite({props:{},cache:{purge}} as unknown as ExecutionContext,env);
		expect((await entry.fetch(new Request("https://public/"))).status).toBe(404);
		await entry.purgePending();expect(purge).toHaveBeenCalledWith({tags:["site:cache-one"]});
	});
});
describe("persistent purge queue and Free-tier limits",()=>{
	it("coalesces tags and admits only one purge per minute across concurrent drainers",async()=>{
		for(let i=0;i<105;i++)await queueSitePurge(env.DB,`site-${i}`).run();
		const purge=vi.fn(async(_options:CachePurgeOptions)=>({success:true,errors:[]})),cache={purge} as unknown as CacheContext,now=Date.now();
		await Promise.all([drainSitePurges(env.DB,cache,now),drainSitePurges(env.DB,cache,now)]);
		expect(purge).toHaveBeenCalledTimes(1);expect(purge.mock.calls[0]?.[0]).toMatchObject({tags:expect.any(Array)});
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(5);
		await drainSitePurges(env.DB,cache,now+60_001);expect(purge).toHaveBeenCalledTimes(2);
	});
	it("retains failed purges and does not acknowledge a concurrent new event",async()=>{
		await queueSitePurge(env.DB,"cache-one").run();const now=Date.now();
		await drainSitePurges(env.DB,{purge:async()=>({success:false,errors:[]})} as unknown as CacheContext,now);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(1);
		await drainSitePurges(env.DB,{purge:async()=>{await queueSitePurge(env.DB,"cache-one").run();return {success:true,errors:[]};}} as unknown as CacheContext,now+60_001);
		expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(1);
	});
	it("queues account deletion, restoration and grace expiry without repeating completed expiry events",async()=>{
		await deploy("cache-one","expire-v1");await deploy("cache-two","expire-v2");
		await softDelete(env,"cache-owner");expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(2);
		db.sqlite.exec("DELETE FROM site_cache_purge");await restoreAccount(env.DB,"cache-owner");expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(2);
		db.sqlite.exec("DELETE FROM site_cache_purge;UPDATE resource_quota SET grace_until=1 WHERE user_id='cache-owner'");
		await expiredSitePurges(env.DB);expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(2);
		db.sqlite.exec("DELETE FROM site_cache_purge");await expiredSitePurges(env.DB);expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(0);
		await queueAccountPurge(env.DB,"absent-owner").run();expect(db.sqlite.prepare("SELECT count(*) AS n FROM site_cache_purge").get()?.n).toBe(0);
	});
	it("requests Early Hints when provisioning a custom hostname",async()=>{
		Object.assign(env,{CF_API_TOKEN:"audit-placeholder",CF_ZONE_ID:"f".repeat(32),SITES_CNAME_TARGET:"sites.audit.invalid"});
		const fetchMock=vi.fn(async()=>Response.json({success:true,result:{id:"test-provider",hostname:"owned.audit.example",status:"pending",ssl:{status:"pending"}}}));vi.stubGlobal("fetch",fetchMock);
		await providerCall(env,"POST",undefined,"owned.audit.example");
		expect(fetchMock).toHaveBeenCalledWith(expect.any(String),expect.objectContaining({body:expect.stringContaining('"early_hints":"on"')}));
	});
});
