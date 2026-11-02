import {Hono} from "hono";
import {pulseSession,pulseWrite,ownedProject,type PulseApp} from "./access";
import {channelsApi,projectChannelsApi} from "./channels";
import {telegramApi,telegramWebhookApi} from "./telegram";
import {pagesApi} from "./pages";
import {publicPage} from "./public";
import {pageSlug} from "./page-rules";
import {monitorsApi} from "./monitors";
import {z} from "zod";
import {validate,hashId,opaqueId} from "../validation";
import {pulseIdParam,projectId,issueState,numericId,publicKey,sha256,monthOf} from "./limits";

const projectBody=z.object({name:z.string().trim().min(1).max(80),platform:z.enum(["javascript","node","go","other"])}).strict();
const filters=z.object({state:issueState.optional(),environment:z.string().max(80).optional(),release:z.string().max(128).optional(),q:z.string().max(128).optional(),before:z.coerce.number().int().min(0).max(Number.MAX_SAFE_INTEGER).optional(),limit:z.coerce.number().int().min(1).max(50).default(30)});
const issuesParam=z.object({project:projectId,issue:hashId});
async function newKey(env:Cloudflare.Env,project:string){const key=publicKey(),id=crypto.randomUUID();await env.DB.prepare("INSERT INTO pulse_key(id,project_id,hash,prefix,created_at) VALUES(?,?,?,?,?)").bind(id,project,await sha256(key),key.slice(0,8),Date.now()).run();return {id,prefix:key.slice(0,8),dsn:`https://${key}@ingest.getblup.io/${project}`};}
export const pulseApi=new Hono<PulseApp>().route("/telegram",telegramWebhookApi).get("/public/:slug",validate("param",z.object({slug:pageSlug})),async c=>{if(!(await c.env.PULSE_PUBLIC_LIMIT.limit({key:`public:${c.req.header("cf-connecting-ip")??"local"}`})).success){c.header("Retry-After","60");return c.json({code:"TOO_MANY_REQUESTS"},429);}const page=await publicPage(c.env,c.req.valid("param").slug);return page?c.json(page):c.json({code:"NOT_FOUND"},404);}).use("*",pulseSession)
 .get("/usage",async c=>{
  const owner=c.get("session").user.id,access=c.get("access"),month=monthOf();
  const projects=await c.env.DB.prepare("SELECT id,object_name AS objectName FROM pulse_project WHERE user_id=? AND deleted_at IS NULL ORDER BY id LIMIT 10").bind(owner).all<{id:string;objectName:string}>();
  const stats=access.readable?await Promise.all(projects.results.map(p=>c.env.PULSE_PROJECT.getByName(p.objectName).usage(month))):[];
  const reserved=await c.env.DB.prepare("SELECT reserved FROM pulse_month WHERE user_id=? AND month=?").bind(owner,month).first<{reserved:number}>();
  return c.json({plan:access.plan,limits:access.limits,readOnly:!access.writable,readOnlyUntil:access.readOnlyUntil===null?null:new Date(access.readOnlyUntil).toISOString(),month,accepted:stats.reduce((n,s)=>n+s.accepted,0),reserved:reserved?.reserved??0,bytes:stats.reduce((n,s)=>n+s.bytes,0)});
 })
 .get("/projects",async c=>{
  const rows=await c.env.DB.prepare("SELECT id,name,platform,active,created_at AS createdAt FROM pulse_project WHERE user_id=? AND deleted_at IS NULL ORDER BY created_at,id LIMIT 10").bind(c.get("session").user.id).all<{id:string;name:string;platform:string;active:number;createdAt:number}>();
  return c.json({projects:c.get("access").readable?rows.results.map(p=>({...p,active:!!p.active,createdAt:new Date(p.createdAt).toISOString()})):[]});
 })
 .post("/projects",pulseWrite,validate("json",projectBody),async c=>{
  const {name,platform}=c.req.valid("json"),owner=c.get("session").user.id,limit=c.get("access").limits?.projects??0,id=numericId();
  const made=await c.env.DB.prepare("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) SELECT ?,?,?,?,?,? WHERE (SELECT count(*) FROM pulse_project WHERE user_id=? AND deleted_at IS NULL)<? RETURNING id").bind(id,owner,name,platform,crypto.randomUUID(),Date.now(),owner,limit).first();if(!made)return c.json({code:"QUOTA"},409);
  const key=await newKey(c.env,id);return c.json({id,name,platform,key},201);
 })
 .patch("/projects/:project",pulseWrite,validate("param",pulseIdParam),validate("json",z.object({name:z.string().trim().min(1).max(80).optional(),active:z.boolean().optional()}).strict().refine(v=>v.name!==undefined||v.active!==undefined)),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project);if(!p)return c.json({code:"NOT_FOUND"},404);const body=c.req.valid("json");await c.env.DB.prepare("UPDATE pulse_project SET name=?,active=? WHERE id=? AND deleted_at IS NULL").bind(body.name??p.name,body.active===undefined?p.active:Number(body.active),p.id).run();return c.json({ok:true});
 })
 .delete("/projects/:project",validate("param",pulseIdParam),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project);if(!p)return c.json({code:"NOT_FOUND"},404);
  await c.env.DB.prepare("UPDATE pulse_project SET active=0,deleted_at=? WHERE id=?").bind(Date.now(),p.id).run();await c.env.PULSE_PROJECT.getByName(p.objectName).purge();return c.json({ok:true});
 })
 .get("/projects/:project/keys",validate("param",pulseIdParam),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project);if(!p||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const keys=await c.env.DB.prepare("SELECT id,prefix,revoked_at AS revokedAt FROM pulse_key WHERE project_id=? ORDER BY created_at DESC LIMIT 20").bind(p.id).all<{id:string;prefix:string;revokedAt:number|null}>();return c.json({keys:keys.results});
 })
 .post("/projects/:project/keys",pulseWrite,validate("param",pulseIdParam),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project);if(!p)return c.json({code:"NOT_FOUND"},404);
  // Rotation revokes previous keys first; only the new public DSN is returned.
  await c.env.DB.prepare("UPDATE pulse_key SET revoked_at=? WHERE project_id=? AND revoked_at IS NULL").bind(Date.now(),p.id).run();return c.json(await newKey(c.env,p.id),201);
 })
 .delete("/projects/:project/keys/:key",validate("param",z.object({project:projectId,key:opaqueId})),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project);if(!p)return c.json({code:"NOT_FOUND"},404);await c.env.DB.prepare("UPDATE pulse_key SET revoked_at=? WHERE id=? AND project_id=?").bind(Date.now(),c.req.valid("param").key,p.id).run();return c.json({ok:true});
 })
 .get("/projects/:project/issues",validate("param",pulseIdParam),validate("query",filters),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project),access=c.get("access");if(!p||!access.readable)return c.json({code:"NOT_FOUND"},404);
  const issues=await c.env.PULSE_PROJECT.getByName(p.objectName).issues(Date.now()-(access.limits?.retentionDays??30)*86_400_000,c.req.valid("query"));return c.json({issues});
 })
 .get("/projects/:project/issues/:issue/events",validate("param",issuesParam),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project),access=c.get("access");if(!p||!access.readable)return c.json({code:"NOT_FOUND"},404);
  return c.json({events:await c.env.PULSE_PROJECT.getByName(p.objectName).samples(c.req.valid("param").issue,Date.now()-(access.limits?.retentionDays??30)*86_400_000)});
 })
 .patch("/projects/:project/issues/:issue",pulseWrite,validate("param",issuesParam),validate("json",z.object({state:issueState}).strict()),async c=>{
  const p=await ownedProject(c.env,c.get("session").user.id,c.req.valid("param").project);if(!p)return c.json({code:"NOT_FOUND"},404);
  const changed=await c.env.PULSE_PROJECT.getByName(p.objectName).setState(c.req.valid("param").issue,c.req.valid("json").state,Date.now()-(c.get("access").limits?.retentionDays??30)*86_400_000);return changed?c.json({ok:true},200):c.json({code:"NOT_FOUND"},404);
 })
 .route("/monitors",monitorsApi)
 .route("/pages",pagesApi)
 .route("/channels",channelsApi)
 .route("/telegram",telegramApi)
 .route("/projects",projectChannelsApi);
