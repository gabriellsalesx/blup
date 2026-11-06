import {authKey,ingestBytes,IngestError,normalizeEventId,parseEnvelope} from "./protocol";
import {safeEvent} from "./event";
import {sha256,pulseAccess} from "./limits";

const cors={"Access-Control-Allow-Origin":"*","Access-Control-Allow-Methods":"POST, OPTIONS","Access-Control-Allow-Headers":"Content-Type, Content-Encoding, X-Sentry-Auth","Access-Control-Expose-Headers":"Retry-After, X-Sentry-Rate-Limits, X-Sentry-Error","Access-Control-Max-Age":"600","Cache-Control":"no-store","X-Content-Type-Options":"nosniff"};
export async function pulseIngest(req:Request,env:Cloudflare.Env){
 const url=new URL(req.url),m=/^\/api\/([1-9][0-9]{0,15})\/(envelope|store)\/$/.exec(url.pathname);
 const respond=(body:unknown,status=200,extra:Record<string,string>={})=>Response.json(body,{status,headers:{...cors,...extra}});
 if(!m)return respond({detail:"NOT_FOUND"},404);
 if(req.method==="OPTIONS")return new Response(null,{status:204,headers:cors});
 if(req.method!=="POST")return respond({detail:"METHOD"},405,{Allow:"POST, OPTIONS"});
 try{
  const ip=req.headers.get("cf-connecting-ip")??"local";
  if(!(await env.PULSE_INGEST_LIMIT.limit({key:`ingest:${ip}`})).success)throw new IngestError(429,"IP",10);
  const content=req.headers.get("content-type")?.split(";")[0].trim().toLowerCase();
  if(content&&!["application/x-sentry-envelope","application/json","text/plain"].includes(content))throw new IngestError(415,"CONTENT_TYPE");
  const parsed=parseEnvelope(await ingestBytes(req),m[2]==="store"),key=authKey(req,m[1],parsed.dsn),keyHash=await sha256(key);
  const project=await env.DB.prepare("SELECT p.id,p.user_id AS userId,p.object_name AS objectName FROM pulse_project p JOIN pulse_key k ON k.project_id=p.id WHERE p.id=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL AND k.hash=? AND k.revoked_at IS NULL").bind(m[1],keyHash).first<{id:string;userId:string;objectName:string}>();
  if(!project||!(await pulseAccess(env,project.userId)).writable)throw new IngestError(403,"KEY");
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
