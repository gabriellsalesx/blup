import {Hono} from "hono";
import {HTTPException} from "hono/http-exception";
import {z} from "zod";
import {validate,boundedBody,BodyError} from "../validation";
import {DomainError,hostnameSchema} from "../domain-provider";
import {logoVariant,logoType,LOGO_MAX_BYTES,acceptableLogo,linksOf,type LogoVariant} from "../pulse/page-look";
import {dealsWrite,type DealsApp} from "./access";
import {addChannel} from "./store";
import {accountPlan} from "../lifecycle";
import {reserve,release} from "../quota";
import {siteCreate,siteUpdate,siteSlug,siteLogoKey,siteReserved} from "./site-rules";
import {readSiteDomain,refreshSiteDomain,removeSiteDomain,newProof} from "./site-domain";

// /api/deals/site: the account's public deals page. Making it makes its
// channel (type "site"), so hunts and pasted links send to it like to any
// other; deleting the channel deletes the page (channels.ts).
type SiteRow={id:string;channelId:string;slug:string;title:string;description:string;language:string;logoLight:string|null;logoDark:string|null;showTitle:number;links:string};
const columns="id,channel_id AS channelId,slug,title,description,language,logo_light AS logoLight,logo_dark AS logoDark,show_title AS showTitle,links";
// The page's address; on this computer (deals.localhost) with the local
// server's scheme and port.
export function siteAddress(env:Cloudflare.Env,slug:string){
 const host=env.DEALS_PUBLIC_HOST||"deals.getblup.site";
 if(host.endsWith(".localhost")){try{const app=new URL(env.APP_URL);return `${app.protocol}//${host}${app.port?`:${app.port}`:""}/${slug}/`;}catch{/* below */}}
 return `https://${host}/${slug}/`;
}
export const ownSite=(env:Cloudflare.Env,owner:string)=>env.DB.prepare(`SELECT ${columns} FROM deals_site WHERE user_id=?`).bind(owner).first<SiteRow>();
const view=(env:Cloudflare.Env,s:SiteRow)=>({id:s.id,channelId:s.channelId,slug:s.slug,title:s.title,description:s.description,language:z.enum(["pt","en","es"]).catch("pt").parse(s.language),logos:{light:s.logoLight,dark:s.logoDark},showTitle:s.showTitle===1,links:linksOf(s.links),url:siteAddress(env,s.slug)});
const logoColumn={light:"logo_light",dark:"logo_dark"} as const satisfies Record<LogoVariant,string>;
const logoParam=z.object({variant:logoVariant});

// Everything a page leaves behind: its domain, its logos and the row.
export async function dropSite(env:Cloudflare.Env,site:{id:string}){
 await removeSiteDomain(env,site.id);
 await env.DEALS_ASSETS.delete([siteLogoKey(site.id,"light"),siteLogoKey(site.id,"dark")]);
 await env.DB.prepare("DELETE FROM deals_site WHERE id=?").bind(site.id).run();
}

export const sitesApi=new Hono<DealsApp>()
 .onError((error,c)=>{
  if(error instanceof HTTPException)return error.getResponse();
  const code=error instanceof DomainError?error.code:"INTERNAL";
  if(code==="INTERNAL")console.error("deals site:",error);
  return c.json({code},code==="DOMAIN_UNAVAILABLE"?503:code==="DOMAIN_TAKEN"?409:code==="INTERNAL"?500:502);
 })
 .get("/",async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);
  return c.json({site:site?view(c.env,site):null});
 })
 // Whether an address can be had, as the person types it.
 .get("/slug",validate("query",z.object({slug:z.string().max(64)})),async c=>{
  const raw=c.req.valid("query").slug,parsed=siteSlug.safeParse(raw);
  if(!parsed.success)return c.json({state:(siteReserved as readonly string[]).includes(raw)?"reserved" as const:"invalid" as const});
  const taken=await c.env.DB.prepare("SELECT user_id AS owner FROM deals_site WHERE slug=?").bind(parsed.data).first<{owner:string}>();
  return c.json({state:!taken?"available" as const:taken.owner===c.get("session").user.id?"yours" as const:"taken" as const});
 })
 .post("/",dealsWrite,validate("json",siteCreate),async c=>{
  const owner=c.get("session").user.id,body=c.req.valid("json"),now=Date.now();
  if(await ownSite(c.env,owner))return c.json({code:"SITE_EXISTS"},409);
  if(await c.env.DB.prepare("SELECT 1 FROM deals_site WHERE slug=?").bind(body.slug).first())return c.json({code:"SLUG_TAKEN"},409);
  // The page is one of the plan's sites (with the ones Blup publishes).
  const limit=(await accountPlan(c.env,owner))?.limits.sites??0;
  const reservation=limit>0?await reserve(c.env.DB,owner,"sites",1,limit):null;
  if(!reservation)return c.json({code:"SITES_LIMIT",limit},403);
  try{
   const id=crypto.randomUUID();
   let channel:Awaited<ReturnType<typeof addChannel>>;try{channel=await addChannel(c.env,owner,"site",id,body.title,now);}catch{return c.json({code:"CHANNEL_UNAVAILABLE"},503);}
   if(!channel)return c.json({code:"QUOTA"},409);
   try{
    await c.env.DB.prepare("INSERT INTO deals_site(id,user_id,channel_id,slug,title,description,language,show_title,links,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)").bind(id,owner,channel.id,body.slug,body.title,body.description,body.language,body.showTitle?1:0,JSON.stringify(body.links),now,now).run();
   }catch(e){
    // Someone took the address (or a second page) a moment before.
    await c.env.DB.prepare("DELETE FROM deals_channel WHERE id=? AND user_id=?").bind(channel.id,owner).run();
    if(String(e).includes("UNIQUE"))return c.json({code:String(e).includes("slug")?"SLUG_TAKEN":"SITE_EXISTS"},409);
    throw e;
   }
   const site=await ownSite(c.env,owner);
   return site?c.json({site:view(c.env,site)},201):c.json({code:"INTERNAL"},500);
  }finally{await release(c.env.DB,reservation);}
 })
 .patch("/",dealsWrite,validate("json",siteUpdate),async c=>{
  const owner=c.get("session").user.id,body=c.req.valid("json");
  const row=await c.env.DB.prepare("UPDATE deals_site SET title=?,description=?,language=?,show_title=?,links=?,updated_at=? WHERE user_id=? RETURNING id,channel_id AS channelId").bind(body.title,body.description,body.language,body.showTitle?1:0,JSON.stringify(body.links),Date.now(),owner).first<{id:string;channelId:string}>();
  if(!row)return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.prepare("UPDATE deals_channel SET name=? WHERE id=? AND user_id=?").bind(body.title,row.channelId,owner).run();
  return c.json({ok:true});
 })
 // The logos: one for light backgrounds and one for dark, made small in
 // the browser (WebP, or PNG), checked here, kept in R2.
 .put("/logo/:variant",dealsWrite,validate("param",logoParam),async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);if(!site)return c.json({code:"NOT_FOUND"},404);
  const variant=c.req.valid("param").variant,type=logoType.safeParse(c.req.header("content-type"));if(!type.success)return c.json({code:"INVALID"},400);
  let bytes:Uint8Array;
  try{bytes=await boundedBody(c.req.raw,LOGO_MAX_BYTES);}catch(e){return e instanceof BodyError&&e.status===413?c.json({code:"TOO_LARGE"},413):c.json({code:"INVALID"},400);}
  if(!acceptableLogo(bytes,type.data))return c.json({code:"INVALID"},400);
  const digest=new Uint8Array(await crypto.subtle.digest("SHA-256",bytes)),version=Array.from(digest.subarray(0,8),b=>b.toString(16).padStart(2,"0")).join("");
  await c.env.DEALS_ASSETS.put(siteLogoKey(site.id,variant),bytes,{httpMetadata:{contentType:type.data},customMetadata:{version}});
  await c.env.DB.prepare(`UPDATE deals_site SET ${logoColumn[variant]}=?,updated_at=? WHERE id=?`).bind(version,Date.now(),site.id).run();
  return c.json({version},201);
 })
 .get("/logo/:variant",validate("param",logoParam),async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);if(!site||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const object=await c.env.DEALS_ASSETS.get(siteLogoKey(site.id,c.req.valid("param").variant));if(!object)return c.json({code:"NOT_FOUND"},404);
  return new Response(object.body,{headers:{"content-type":object.httpMetadata?.contentType??"image/webp","cache-control":"private, no-store","x-content-type-options":"nosniff"}});
 })
 .delete("/logo/:variant",validate("param",logoParam),async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);if(!site)return c.json({code:"NOT_FOUND"},404);
  const variant=c.req.valid("param").variant;
  await c.env.DEALS_ASSETS.delete(siteLogoKey(site.id,variant));
  await c.env.DB.prepare(`UPDATE deals_site SET ${logoColumn[variant]}=NULL,updated_at=? WHERE id=?`).bind(Date.now(),site.id).run();
  return c.json({ok:true});
 })
 // The page's own domain (Ultra includes it).
 .get("/domain",async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);if(!site)return c.json({code:"NOT_FOUND"},404);
  const row=await readSiteDomain(c.env,site.id);
  return c.json(row?await refreshSiteDomain(c.env,row):null);
 })
 .post("/domain",dealsWrite,validate("json",z.object({hostname:hostnameSchema}).strict()),async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);if(!site)return c.json({code:"NOT_FOUND"},404);
  if(!c.env.CF_API_TOKEN||!c.env.CF_ZONE_ID||!c.env.SITES_CNAME_TARGET)return c.json({code:"DOMAIN_UNAVAILABLE"},503);
  const hostname=c.req.valid("json").hostname,current=await readSiteDomain(c.env,site.id);
  if(current&&current.hostname!==hostname)return c.json({code:"DOMAIN_EXISTS"},409);
  await c.env.DB.prepare("INSERT OR IGNORE INTO deals_site_domain(site_id,hostname,proof) VALUES(?,?,?)").bind(site.id,hostname,newProof()).run();
  const row=await readSiteDomain(c.env,site.id);if(!row)throw new DomainError("DOMAIN_PROVIDER");
  return c.json(await refreshSiteDomain(c.env,{...row,checked_at:null}));
 })
 .delete("/domain",async c=>{
  const site=await ownSite(c.env,c.get("session").user.id);if(!site)return c.json({code:"NOT_FOUND"},404);
  await removeSiteDomain(c.env,site.id);return c.json({ok:true});
 });
