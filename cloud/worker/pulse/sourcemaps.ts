import {Hono} from "hono";
import {z} from "zod";
import {validate,boundedBody,BodyError} from "../validation";
import {pulseWrite,ownedProject,type PulseApp} from "./access";
import {pulseAccess,projectId,publicKey,sha256} from "./limits";
import {sourceMapSchema,SourceMapLookup,type OriginalFrame,type SourceMap} from "./sourcemap";
import type {SafeEvent} from "./event";
// Source maps of a project's browser code. The build sends each map with
// the project's upload token (Authorization: Bearer), for one release and
// one minified file; reading an occurrence then shows the original file,
// line, function and code of its frames. Maps live in R2 (one object per
// release and file), listed in D1; they leave with the project and 90
// days after being sent. Each project keeps up to 300 maps and 512 MiB in
// all; a token (and an address) sends at most 120 maps a minute.
export const SOURCEMAP_MAX_BYTES=20*1024*1024,SOURCEMAPS_PER_PROJECT=300,SOURCEMAP_BYTES_PER_PROJECT=512*1024*1024;
const releaseName=z.string().min(1).max(128).regex(/^[^\s/\\][^\s]*$/);
const fileName=z.string().min(1).max(160).regex(/^[\w.@~-]+\.(?:m?js|cjs)$/);
const upload=z.object({release:releaseName,file:fileName}).strict();
const objectKey=async(project:string,release:string,file:string)=>`${project}/${await sha256(`${release}\n${file}`)}.map`;

// The build's side: no session, only the upload token of one project.
export const sourcemapUploadApi=new Hono<{Bindings:Cloudflare.Env}>()
 .post("/sourcemaps",validate("query",upload),async c=>{
  const token=/^Bearer ([a-f0-9]{48})$/.exec(c.req.header("authorization")??"")?.[1];if(!token)return c.json({code:"UNAUTHORIZED"},401);
  const tokenHash=await sha256(token),limited=await Promise.all([`sourcemap-ip:${c.req.header("cf-connecting-ip")??"local"}`,`sourcemap:${tokenHash}`].map(key=>c.env.ABUSE_LIMIT.limit({key})));
  if(limited.some(r=>!r.success)){c.header("Retry-After","60");return c.json({code:"TOO_MANY_REQUESTS"},429);}
  const project=await c.env.DB.prepare("SELECT p.id,p.user_id AS userId FROM pulse_upload_token t JOIN pulse_project p ON p.id=t.project_id WHERE t.hash=? AND t.revoked_at IS NULL AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL").bind(tokenHash).first<{id:string;userId:string}>();
  if(!project||!(await pulseAccess(c.env,project.userId)).writable)return c.json({code:"UNAUTHORIZED"},401);
  let text:string;try{text=new TextDecoder("utf-8",{fatal:true,ignoreBOM:false}).decode(await boundedBody(c.req.raw,SOURCEMAP_MAX_BYTES));}catch(e){return e instanceof BodyError&&e.status===413?c.json({code:"TOO_LARGE"},413):c.json({code:"INVALID"},400);}
  let map:SourceMap;try{const parsed=sourceMapSchema.safeParse(JSON.parse(text));if(!parsed.success)return c.json({code:"INVALID"},400);map=parsed.data;}catch{return c.json({code:"INVALID"},400);}
  const {release,file}=c.req.valid("query"),key=await objectKey(project.id,release,file),now=Date.now();
  const stored=new TextEncoder().encode(JSON.stringify(map)),size=stored.byteLength;
  // The quota (count and bytes, the map it replaces aside) is checked in
  // the same statement that records the map, so two uploads at once cannot
  // both pass it.
  const others="FROM pulse_sourcemap WHERE project_id=?1 AND NOT(release=?2 AND file=?3)";
  const saved=await c.env.DB.prepare(`INSERT INTO pulse_sourcemap(project_id,release,file,object_key,size,uploaded_at) SELECT ?1,?2,?3,?4,?5,?6 WHERE (SELECT count(*) ${others})<?7 AND (SELECT coalesce(sum(size),0) ${others})+?5<=?8 ON CONFLICT(project_id,release,file) DO UPDATE SET object_key=excluded.object_key,size=excluded.size,uploaded_at=excluded.uploaded_at RETURNING project_id`).bind(project.id,release,file,key,size,now,SOURCEMAPS_PER_PROJECT,SOURCEMAP_BYTES_PER_PROJECT).first();
  if(!saved)return c.json({code:"QUOTA"},409);
  await c.env.PULSE_SOURCEMAPS.put(key,stored,{httpMetadata:{contentType:"application/json"}});
  return c.json({release,file,size},201);
 });

// The panel's side: the project's upload token (a new one replaces the
// old and is shown once) and the maps sent.
export const sourcemapPanelApi=new Hono<PulseApp>()
 .get("/projects/:project/sourcemaps",validate("param",z.object({project:projectId})),async c=>{
  const owner=c.get("session").user.id,project=c.req.valid("param").project;if(!await ownedProject(c.env,owner,project)||!c.get("access").readable)return c.json({code:"NOT_FOUND"},404);
  const [token,maps]=await c.env.DB.batch([
   c.env.DB.prepare("SELECT prefix,created_at AS createdAt FROM pulse_upload_token WHERE project_id=? AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1").bind(project),
   c.env.DB.prepare("SELECT release,file,size,uploaded_at AS uploadedAt FROM pulse_sourcemap WHERE project_id=? ORDER BY uploaded_at DESC LIMIT 50").bind(project),
  ]) as [D1Result<{prefix:string;createdAt:number}>,D1Result<{release:string;file:string;size:number;uploadedAt:number}>];
  return c.json({token:token.results[0]??null,maps:maps.results,limit:SOURCEMAPS_PER_PROJECT,bytesLimit:SOURCEMAP_BYTES_PER_PROJECT});
 })
 .post("/projects/:project/upload-token",pulseWrite,validate("param",z.object({project:projectId})),validate("json",z.object({}).strict()),async c=>{
  const owner=c.get("session").user.id,project=c.req.valid("param").project,found=await ownedProject(c.env,owner,project);if(!found||!found.active||found.planDisabled)return c.json({code:"NOT_FOUND"},404);
  const token=publicKey(),now=Date.now();
  await c.env.DB.batch([c.env.DB.prepare("UPDATE pulse_upload_token SET revoked_at=? WHERE project_id=? AND revoked_at IS NULL").bind(now,project),c.env.DB.prepare("INSERT INTO pulse_upload_token(id,project_id,hash,prefix,created_at) VALUES(?,?,?,?,?)").bind(crypto.randomUUID(),project,await sha256(token),token.slice(0,8),now)]);
  return c.json({token,prefix:token.slice(0,8)},201);
 });

// The original place of each frame of occurrences that came with a
// release, from the maps sent for it (at most 5 maps per read). A frame the
// map does not cover stays as it came.
export async function withOriginals<T extends {event:SafeEvent}>(env:Cloudflare.Env,project:string,items:T[]):Promise<(T&{originals:Record<string,OriginalFrame>})[]>{
 const wanted=new Map<string,{release:string;file:string}>();
 for(const item of items)if(item.event.release)for(const e of item.event.exceptions)for(const f of e.frames)if(f.filename&&f.line!==null&&/\.(?:m?js|cjs)$/.test(f.filename))wanted.set(`${item.event.release}\n${f.filename}`,{release:item.event.release,file:f.filename});
 const lookups=new Map<string,SourceMapLookup>();
 for(const {release,file} of [...wanted.values()].slice(0,20)){
  if(lookups.size>=5)break;
  const row=await env.DB.prepare("SELECT object_key AS objectKey FROM pulse_sourcemap WHERE project_id=? AND release=? AND file=?").bind(project,release,file).first<{objectKey:string}>();if(!row)continue;
  const object=await env.PULSE_SOURCEMAPS.get(row.objectKey);if(!object)continue;
  try{const parsed=sourceMapSchema.safeParse(await object.json());if(parsed.success)lookups.set(`${release}\n${file}`,new SourceMapLookup(parsed.data));}catch{/* a broken map is skipped */}
 }
 return items.map(item=>{
  const originals:Record<string,OriginalFrame>={};
  item.event.exceptions.forEach((e,ei)=>e.frames.forEach((f,fi)=>{const lookup=lookups.get(`${item.event.release}\n${f.filename}`);if(!lookup||f.line===null)return;try{const original=lookup.originalOf(f.line,f.column??1);if(original)originals[`${ei}:${fi}`]=original;}catch{/* outside the map */}}));
  return {...item,originals};
 });
}
