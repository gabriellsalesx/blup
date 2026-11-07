// What uptime monitors (Blup Pulse's own) check, on every host of the
// Cloud's Workers. Out of /api, which Pulse's monitors may not call, and
// under /_blup/, which no page of a published site can use:
// - /_blup/health: the Worker answers (nothing else is touched, so a slow
//   database never makes it look down);
// - /_blup/ready: it can work: the database answers a trivial query in
//   time. 503 when not, since monitors read the status code.
// Never cached, never with details: only "ok" or "unavailable".
const HEALTH="/_blup/health",READY="/_blup/ready";
const headers={"content-type":"application/json; charset=utf-8","cache-control":"no-store","x-content-type-options":"nosniff"};
const answer=(req:Request,status:number)=>new Response(req.method==="HEAD"?null:JSON.stringify({status:status===200?"ok":"unavailable"}),{status,headers});

export const isHealthPath=(path:string)=>path===HEALTH||path===READY;

export async function serveHealth(req:Request,env:Pick<Cloudflare.Env,"DB">):Promise<Response|null>{
 const path=new URL(req.url).pathname;
 if(!isHealthPath(path))return null;
 if(req.method!=="GET"&&req.method!=="HEAD")return new Response(null,{status:405,headers:{allow:"GET, HEAD","cache-control":"no-store"}});
 if(path===HEALTH)return answer(req,200);
 try{
  const timeout=new Promise<never>((_,fail)=>setTimeout(()=>fail(new Error("TIMEOUT")),3000));
  const row=await Promise.race([env.DB.prepare("SELECT 1 AS ok").first<{ok:number}>(),timeout]);
  return answer(req,row?.ok===1?200:503);
 }catch{return answer(req,503);}
}
