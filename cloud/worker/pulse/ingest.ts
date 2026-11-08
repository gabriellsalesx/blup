import {authKey,ingestBytes,IngestError,normalizeEventId,parseEnvelope} from "./protocol";
import {safeEvent} from "./event";
import {sha256,pulseAccess} from "./limits";

// Which project a key sends to, kept 30 seconds in this isolate (the
// project's Durable Object checks the key and the plan again, with its own
// 30 seconds). Only accepted keys are kept, at most 1000 of them.
type Known={id:string;userId:string;objectName:string};
const known=new Map<string,{project:Known;until:number}>();
async function knownProject(env:Cloudflare.Env,id:string,keyHash:string):Promise<Known>{
 const key=`${id}\n${keyHash}`,now=Date.now(),hit=known.get(key);
 if(hit&&hit.until>now)return hit.project;
 known.delete(key);
 const project=await env.DB.prepare("SELECT p.id,p.user_id AS userId,p.object_name AS objectName FROM pulse_project p JOIN pulse_key k ON k.project_id=p.id WHERE p.id=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL AND k.hash=? AND k.revoked_at IS NULL").bind(id,keyHash).first<Known>();
 if(!project||!(await pulseAccess(env,project.userId,now)).writable)throw new IngestError(403,"KEY");
 if(known.size>=1000)known.delete(known.keys().next().value??"");
 known.set(key,{project,until:now+30_000});
 return project;
}
// Workers in another Cloudflare zone all reach us from one address
// (developers.cloudflare.com/fundamentals/reference/http-headers, CF-Connecting-IP
// in Worker subrequests): every customer's Worker, and Blup's own Sites
// Worker, would share one bucket. Those count by the sending zone
// (CF-Worker, set by Cloudflare) and the project instead.
export const WORKERS_IP="2a06:98c0:3600::103";
export function ingestLimitKey(req:Request,project:string){
 const ip=req.headers.get("cf-connecting-ip")??"local";
 if(ip!==WORKERS_IP)return `ingest:${ip}`;
 const zone=(req.headers.get("cf-worker")??"").toLowerCase().slice(0,253);
 return `ingest:worker:${/^[a-z0-9.-]+$/.test(zone)?zone:"?"}:${project}`;
}
const cors={"Access-Control-Allow-Origin":"*","Access-Control-Allow-Methods":"POST, OPTIONS","Access-Control-Allow-Headers":"Content-Type, Content-Encoding, X-Sentry-Auth","Access-Control-Expose-Headers":"Retry-After, X-Sentry-Rate-Limits, X-Sentry-Error","Access-Control-Max-Age":"600","Cache-Control":"no-store","X-Content-Type-Options":"nosniff"};
export async function pulseIngest(req:Request,env:Cloudflare.Env){
 const url=new URL(req.url),m=/^\/api\/([1-9][0-9]{0,15})\/(envelope|store)\/$/.exec(url.pathname);
 const respond=(body:unknown,status=200,extra:Record<string,string>={})=>Response.json(body,{status,headers:{...cors,...extra}});
 const projectId=m?.[1],route=m?.[2];
 if(!projectId||!route)return respond({detail:"NOT_FOUND"},404);
 if(req.method==="OPTIONS")return new Response(null,{status:204,headers:cors});
 if(req.method!=="POST")return respond({detail:"METHOD"},405,{Allow:"POST, OPTIONS"});
 try{
  if(!(await env.PULSE_INGEST_LIMIT.limit({key:ingestLimitKey(req,projectId)})).success)throw new IngestError(429,"IP",10);
  const content=req.headers.get("content-type")?.split(";")[0]?.trim().toLowerCase();
  if(content&&!["application/x-sentry-envelope","application/json","text/plain"].includes(content))throw new IngestError(415,"CONTENT_TYPE");
  const parsed=parseEnvelope(await ingestBytes(req),route==="store"),key=authKey(req,projectId,parsed.dsn),keyHash=await sha256(key);
  const project=await knownProject(env,projectId,keyHash);
  if(!parsed.event)return respond({});
  // Errors only ("event" is sentry-ruby's name for one); performance data and the rest are skipped.
  if(parsed.event.type!==undefined&&parsed.event.type!=="default"&&parsed.event.type!=="error"&&parsed.event.type!=="event")return respond({});
  const event=await safeEvent(parsed.event,normalizeEventId(parsed.event,parsed.id),parsed.minidump);
  const accepted=await env.PULSE_PROJECT.getByName(project.objectName).ingest(project.id,keyHash,event);
  if(!accepted.ok)throw new IngestError(accepted.status,accepted.code,accepted.wait);
  return respond({id:accepted.id});
 }catch(e){
  if(e instanceof IngestError){const extra:Record<string,string>={"X-Sentry-Error":e.code};if(e.status===429){extra["Retry-After"]=String(e.wait||60);extra["X-Sentry-Rate-Limits"]=`${e.wait||60}:error;default:project:blup_${e.code.toLowerCase()}`;}return respond({detail:e.code},e.status,extra);}
  console.error("[blup] pulse_ingest_failed");return respond({detail:"UNAVAILABLE"},503);
 }
}
