import { z } from "zod";
import { opaqueId } from "./validation";
import { siteTags } from "./site-cache-purge";
import { retainedSiteIds } from "./site-entitlement";
import { observePlan } from "./lifecycle";
import { planNamed,ownerMatches } from "./plans";
import { planOrder } from "../shared/pulse-plans";
import { manifestSchema } from "./site-rules";
import { acceptsGzip, cacheControl, fileKey, manifestKey, resolvePath, type Manifest } from "./site-rules";

// Authorization always runs in the uncached gateway. Only immutable
// public deploys reach the cached entrypoint, isolated by site + deploy.
const manifests = new Map<string, Manifest>();
const MAX_MANIFESTS = 32;
const BROWSER_CC = "x-blup-browser-cache";
export const cachedSiteProps = z.object({siteId:opaqueId,deploy:opaqueId});
export type CachedSiteProps = z.infer<typeof cachedSiteProps>;

type SiteHead={id:string;deploy:string;owner:string;email:string;verified:number;grace:number|null;chosen:string|null;lastPlan:string|null;plan:string|null;sites:number;hostname:string|null;domainStatus:string|null;ssl:string|null;proof:number|null;disableAt:number|null};
async function siteHead(env:Cloudflare.Env,slug:string){
 return env.DB.prepare(`SELECT s.id,s.deploy,s.user_id AS owner,u.email,u.email_verified AS verified,q.grace_until AS grace,q.selected_site_id AS chosen,q.last_plan AS lastPlan,(SELECT count(*) FROM site WHERE user_id=u.id) AS sites,(SELECT plan FROM subscription WHERE reference_id=u.id AND status IN ('active','trialing','past_due') ORDER BY ${planOrder()} LIMIT 1) AS plan,d.hostname,d.status AS domainStatus,d.ssl,d.verified_at AS proof,d.disable_at AS disableAt FROM site s JOIN user u ON u.id=s.user_id LEFT JOIN resource_quota q ON q.user_id=u.id LEFT JOIN site_domain d ON d.site_id=s.id WHERE s.slug=? AND u.banned=0 AND q.deleted_at IS NULL LIMIT 1`).bind(slug).first<SiteHead>();
}
async function liveDeploy(env:Cloudflare.Env,slug:string,customHostname:string|undefined){
 let row=await siteHead(env,slug);if(!row)return null;
 const plan=planNamed(ownerMatches(env.STAFF_OWNER_EMAIL,row.email,!!row.verified)?"ultra":row.plan);
 if(row.lastPlan!==plan.name || row.grace===null&&row.sites>plan.limits.sites || customHostname&&!plan.limits.customDomain&&row.disableAt===null){
  await observePlan(env.DB,row.owner,plan.name);
  row=await siteHead(env,slug);if(!row)return null;
 }
 if(plan.limits.sites>=0&&row.sites>plan.limits.sites&&(!row.grace||row.grace<=Date.now())){
  if(!(await retainedSiteIds(env.DB,row.owner,plan.limits.sites,row.chosen)).includes(row.id))return null;
 }
 if(customHostname){
  if(row.hostname!==customHostname||row.domainStatus!=="active"||row.ssl!=="active"||row.proof===null)return null;
  if(row.disableAt!==null&&row.disableAt<=Date.now())return null;
  if(!plan.limits.customDomain&&row.disableAt===null)return null;
 }
 return row;
}
// Who may be served, kept 10 seconds per site and address in this isolate:
// a visited site reads D1 a few times a minute, not on every request. A
// new deploy, a ban or a lost plan shows within those 10 seconds.
type LiveDeploy=Awaited<ReturnType<typeof liveDeploy>>;
const heads=new Map<string,{row:LiveDeploy;until:number}>();
const HEAD_MS=10_000,MAX_HEADS=1024;
export function forgetSiteHeads(){heads.clear();}
async function currentDeploy(env:Cloudflare.Env,slug:string,customHostname:string|undefined):Promise<LiveDeploy>{
 const key=`${slug}\n${customHostname??""}`,now=Date.now(),hit=heads.get(key);
 if(hit&&hit.until>now)return hit.row;
 heads.delete(key);
 const row=await liveDeploy(env,slug,customHostname);
 if(heads.size>=MAX_HEADS){const oldest=heads.keys().next().value;if(oldest!==undefined)heads.delete(oldest);}
 heads.set(key,{row,until:now+HEAD_MS});
 return row;
}

async function manifestOf(env: Cloudflare.Env, siteId: string, deploy: string): Promise<Manifest | null> {
	const key = `${siteId}:${deploy}`;
	const hit = manifests.get(key);
	if (hit) return hit;
	const obj = await env.SITES.get(manifestKey(siteId, deploy));
	if (!obj) return null;
	const parsed=manifestSchema.safeParse(await obj.json().catch(()=>null));
	if(!parsed.success)return null;
	const m=parsed.data;
	const oldest=manifests.keys().next().value;
	if (manifests.size >= MAX_MANIFESTS && oldest!==undefined) manifests.delete(oldest);
	manifests.set(key, m);
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

// Use only known theme assets present in this exact manifest. Encode
// URLs before putting them into headers, including hostile file names.
function preloadLinks(manifest:Manifest):string {
 const paths=Object.keys(manifest);
 const css=paths.find(p=>/^\/css\/(?:bento|mochi|model[0-9]*)(?:\.[a-f0-9]+)?\.css$/.test(p)&&manifest[p]?.[1]==="text/css");
 const font=paths.find(p=>p==="/fonts/inter-latin.woff2"&&manifest[p]?.[1]==="font/woff2");
 return [css?`<${encodeURI(css)}>; rel=preload; as=style`:null,font?`<${encodeURI(font)}>; rel=preload; as=font; type="font/woff2"; crossorigin`:null].filter(Boolean).join(", ");
}

export async function serveDeploy(req:Request,env:Cloudflare.Env,props:CachedSiteProps):Promise<Response> {
 if(req.method!=="GET"&&req.method!=="HEAD")return new Response(null,{status:405,headers:{allow:"GET, HEAD","cache-control":"no-store"}});
 const url=new URL(req.url);
 const manifest=await manifestOf(env,props.siteId,props.deploy);
 if(!manifest)return notFound();
 const found=resolvePath(manifest,url.pathname);
 // The gateway restores the original query after the cache lookup.
 if(found&&"redirect" in found)return new Response(null,{status:301,headers:{location:found.redirect,"cache-control":"no-store"}});
 const path=found?.file??"/404.html",entry=manifest[path];
 if(!entry)return notFound();
 const res=await fileResponse(env,req,props.siteId,path,entry,found?200:404);
 if(res.status===200){
  res.headers.set(BROWSER_CC,res.headers.get("cache-control")??"public, max-age=0, must-revalidate");
  res.headers.set("cache-control","public, max-age=600");
  res.headers.set("cache-tag",siteTags(props.siteId,props.deploy).join(","));
  if(entry[1].startsWith("text/html")){const links=preloadLinks(manifest);if(links)res.headers.set("link",links);}
 }
 return res;
}

export async function serveSite(req: Request, env: Cloudflare.Env, ctx: ExecutionContext, slug: string, customHostname?:string): Promise<Response> {
 if(req.method!=="GET"&&req.method!=="HEAD")return new Response(null,{status:405,headers:{allow:"GET, HEAD","cache-control":"no-store"}});
 const live=await currentDeploy(env,slug,customHostname);
 if(!live?.deploy)return notFound();
 const original=new URL(req.url),url=new URL("https://public-site.internal");url.pathname=original.pathname;
 // Do not forward cookies, authorization, ranges, or arbitrary vary values.
 // The response is public, and does not depend on any of those headers.
 const originalEncoding=req.cf?.clientAcceptEncoding;
 const encoding=acceptsGzip(typeof originalEncoding==="string"?originalEncoding:req.headers.get("accept-encoding"))?"gzip":"identity";
 const forwarded=new Request(url,{headers:{"accept-encoding":encoding}});
 const props={siteId:live.id,deploy:live.deploy};
 const cached=ctx.exports?.CachedSite;
 const viaCache=typeof cached==="function";
 const answer=cached?await cached({props}).fetch(forwarded):await serveDeploy(forwarded,env,props);
 const headers=new Headers(answer.headers);
 if(answer.status===200)headers.set("cache-control",headers.get(BROWSER_CC)??"public, max-age=0, must-revalidate");
 headers.delete(BROWSER_CC);headers.delete("cache-tag");
 if(answer.status===301&&headers.get("location"))headers.set("location",headers.get("location")+original.search);
 const etag=headers.get("etag");
 const notModified=answer.status===200&&etag!==null&&req.headers.get("if-none-match")===etag;
 if(notModified||req.method==="HEAD")await answer.body?.cancel();
 // Through the cached entrypoint (ctx.exports), the runtime hands the body
 // over already unpacked, with Content-Encoding still saying gzip: sent
 // as "manual", browsers got plain bytes labelled gzip and dropped every
 // CSS and JS file. Left to the runtime ("automatic"), the body goes back
 // gzipped as the header says (passed through, not packed twice). Served
 // here directly (no ctx.exports), the body is the stored gzip itself:
 // that one goes as it is.
 return new Response(notModified||req.method==="HEAD"?null:answer.body,{status:notModified?304:answer.status,headers,encodeBody:viaCache?"automatic":"manual"});
}
