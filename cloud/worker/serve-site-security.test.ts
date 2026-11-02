import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TestBucket, TestD1, seedUser } from "./security-test-support";
// @ts-expect-error test-only Node builtin, no additional dependency
import { gzipSync } from "node:zlib";
import { fileKey, manifestKey, pathAllowed, resolvePath } from "./site-rules";
import { serveSite } from "./serve-site";
import entry from "./sites-entry";

vi.mock("cloudflare:workers",()=>({WorkerEntrypoint:class{}}));

let d1:TestD1;let bucket:TestBucket;let env:Cloudflare.Env;let tasks:Promise<unknown>[];let ctx:ExecutionContext;
const HA="a".repeat(64),HB="b".repeat(64);
beforeEach(()=>{
	d1=new TestD1();bucket=new TestBucket();tasks=[];ctx={waitUntil:(p:Promise<unknown>)=>tasks.push(p)} as unknown as ExecutionContext;
	seedUser(d1,"serve-owner");
	env={DB:d1.asD1(),SITES:bucket.asR2(),SITES_DOMAIN:"getblup.site"} as Cloudflare.Env;
	vi.stubGlobal("caches",{default:{match:async()=>undefined,put:async()=>{}}});
});
afterEach(async()=>{await Promise.allSettled(tasks);d1.close();vi.unstubAllGlobals();});

async function fixture(slug:string,gz=false){
	d1.sqlite.prepare("INSERT INTO site(id,user_id,slug,title,deploy) VALUES (?,'serve-owner',?,'fixture',?)").run(slug,slug,`deploy-${slug}`);
	await bucket.put(manifestKey(slug,`deploy-${slug}`),JSON.stringify({"/index.html":[HA,"text/html",gz?1:0],"/folder/index.html":[HB,"text/plain",0]}));
	await bucket.put(fileKey(slug,HA),gz?new Uint8Array(gzipSync("public fixture")):"public fixture");
	await bucket.put(fileKey(slug,HB),"folder fixture");
}
describe("published site attacks and response semantics",()=>{
	it("does not allow a protocol-relative redirect path",()=>{
		expect(pathAllowed("//evil.invalid/path")).toBe(false);
		expect(resolvePath({"//evil.invalid/index.html":[HA,"text/html",0]},"//evil.invalid")).toBeNull();
	});
	it("honors an explicit gzip;q=0 refusal",async()=>{
		await fixture("gzip-refusal",true);
		const res=await serveSite(new Request("https://gzip-refusal.getblup.site/",{headers:{"accept-encoding":"gzip;q=0, identity;q=1"}}),env,ctx,"gzip-refusal");
		expect(res.headers.get("content-encoding")).toBeNull();expect(await res.text()).toBe("public fixture");
	});
	it("isolates R2 prefixes of different sites, supports HEAD and conditional GET",async()=>{
		await fixture("scope-one");await fixture("scope-two");await bucket.put(fileKey("scope-two",HA),"second site");
		const one=await serveSite(new Request("https://scope-one.getblup.site/"),env,ctx,"scope-one");
		expect(await one.text()).toBe("public fixture");
		const head=await serveSite(new Request("https://scope-two.getblup.site/",{method:"HEAD"}),env,ctx,"scope-two");expect(await head.text()).toBe("");
		const cached=await serveSite(new Request("https://scope-two.getblup.site/",{headers:{"if-none-match":`"${HA}"`}}),env,ctx,"scope-two");expect(cached.status).toBe(304);
	});
	it.each(["/%2e%2e/private","/a/%2F../private","/a%5c../private","/%00bad","/%zz"])("rejects traversal %s",async path=>{
		await fixture("traversal-fixture");
		const res=await serveSite(new Request("https://traversal-fixture.getblup.site"+path),env,ctx,"traversal-fixture");expect(res.status).toBe(404);
	});
	it("canonicalizes HTTPS and www without trusting an unrelated host",async()=>{
		const r=await entry.fetch(new Request("http://www.slug-site.getblup.site/path?q=x"),env,ctx);expect(r.status).toBe(301);expect(r.headers.get("location")).toBe("https://slug-site.getblup.site/path?q=x");
		expect((await entry.fetch(new Request("https://slug-site.getblup.site.evil.invalid/"),env,ctx)).status).toBe(404);
	});
	it("bounds memory used by random nonexistent host names",async()=>{
		await serveSite(new Request("https://absent-first.getblup.site/"),env,ctx,"absent-first");
		for(let i=0;i<1025;i++)await serveSite(new Request(`https://absent-${i}.getblup.site/`),env,ctx,`absent-${i}`);
		const before=d1.statements.length;
		await serveSite(new Request("https://absent-first.getblup.site/"),env,ctx,"absent-first");
		expect(d1.statements.length).toBeGreaterThan(before);
	});
});

describe("owner decisions: revocation and retention",()=>{
	it("a downgraded Free account must stop serving a paid site",async()=>{
		await fixture("downgrade-fixture");
		d1.sqlite.prepare("INSERT INTO resource_quota(user_id,last_plan,grace_until) VALUES ('serve-owner','free',?)").run(Date.now()-1);
		expect((await serveSite(new Request("https://downgrade-fixture.getblup.site/"),env,ctx,"downgrade-fixture")).status).toBe(404);
	});
	it("removing the owner must not leave the site online forever",async()=>{
		await fixture("deleted-fixture");d1.sqlite.exec("DELETE FROM user WHERE id='serve-owner'");
		expect((await serveSite(new Request("https://deleted-fixture.getblup.site/"),env,ctx,"deleted-fixture")).status).toBe(404);
	});
});
