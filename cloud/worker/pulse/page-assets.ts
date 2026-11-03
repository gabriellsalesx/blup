import {Hono} from "hono";
import {z} from "zod";
import {validate,opaqueId,boundedBody,BodyError} from "../validation";
import {pulseWrite,type PulseApp} from "./access";
import {logoVariant,logoKey,logoType,LOGO_MAX_BYTES,acceptableLogo,type LogoVariant} from "./page-look";
// A status page's logos in R2 (one object per page and variant), with
// their version in D1. The panel sends a WebP (or PNG) the browser already
// made small; the page's public address serves it.
const column={light:"logo_light",dark:"logo_dark"} as const satisfies Record<LogoVariant,string>;
const logoParam=z.object({page:opaqueId,variant:logoVariant});
const owned=(env:Cloudflare.Env,owner:string,page:string)=>env.DB.prepare("SELECT id,plan_disabled AS planDisabled FROM pulse_page WHERE id=? AND user_id=? AND (retire_at IS NULL OR retire_at>?)").bind(page,owner,Date.now()).first<{id:string;planDisabled:number}>();

export const pageAssetsApi=new Hono<PulseApp>()
 .put("/:page/logo/:variant",pulseWrite,validate("param",logoParam),async c=>{
  const {page,variant}=c.req.valid("param"),found=await owned(c.env,c.get("session").user.id,page);if(!found)return c.json({code:"NOT_FOUND"},404);if(found.planDisabled)return c.json({code:"PLAN"},403);
  const type=logoType.safeParse(c.req.header("content-type"));if(!type.success)return c.json({code:"INVALID"},400);
  let bytes:Uint8Array;try{bytes=await boundedBody(c.req.raw,LOGO_MAX_BYTES);}catch(e){return e instanceof BodyError&&e.status===413?c.json({code:"TOO_LARGE"},413):c.json({code:"INVALID"},400);}
  if(!acceptableLogo(bytes,type.data))return c.json({code:"INVALID"},400);
  const digest=new Uint8Array(await crypto.subtle.digest("SHA-256",bytes)),version=Array.from(digest.subarray(0,8),b=>b.toString(16).padStart(2,"0")).join("");
  await c.env.PULSE_ASSETS.put(logoKey(page,variant),bytes,{httpMetadata:{contentType:type.data},customMetadata:{version}});
  await c.env.DB.prepare(`UPDATE pulse_page SET ${column[variant]}=? WHERE id=?`).bind(version,page).run();
  return c.json({version},201);
 })
 // The panel's preview (the page may not be published yet).
 .get("/:page/logo/:variant",validate("param",logoParam),async c=>{
  const {page,variant}=c.req.valid("param");if(!await owned(c.env,c.get("session").user.id,page)||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const object=await c.env.PULSE_ASSETS.get(logoKey(page,variant));if(!object)return c.json({code:"NOT_FOUND"},404);
  return new Response(object.body,{headers:{"content-type":object.httpMetadata?.contentType??"image/webp","cache-control":"private, no-store","x-content-type-options":"nosniff"}});
 })
 .delete("/:page/logo/:variant",validate("param",logoParam),async c=>{
  const {page,variant}=c.req.valid("param");if(!await owned(c.env,c.get("session").user.id,page))return c.json({code:"NOT_FOUND"},404);
  await c.env.PULSE_ASSETS.delete(logoKey(page,variant));await c.env.DB.prepare(`UPDATE pulse_page SET ${column[variant]}=NULL WHERE id=?`).bind(page).run();
  return c.json({ok:true});
 });
