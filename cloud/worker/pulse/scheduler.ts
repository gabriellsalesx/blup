import {checkHTTP} from "./monitor-http";
import {pulseAccess,type PulseAccess} from "./limits";
import {ownerMatches} from "../plans";
import {siteLock,unlockSite} from "../quota";
import type {MonitorObservation} from "./monitor-store";
export async function checkDue(env:Cloudflare.Env,now=Date.now()){
 const lock=await siteLock(env.DB,"pulse-cron",55_000);if(!lock)return;
 try{
  const due=await env.DB.prepare("SELECT m.id,m.user_id AS userId,m.project_id AS projectId,m.url,m.interval_minutes AS intervalMinutes,m.next_at AS nextAt,m.revision,p.object_name AS objectName,u.email,u.email_verified AS verified FROM pulse_monitor m JOIN pulse_project p ON p.id=m.project_id JOIN user u ON u.id=m.user_id WHERE m.active=1 AND m.next_at<=? AND m.claim_until<=? AND p.active=1 AND p.deleted_at IS NULL ORDER BY m.next_at,m.id LIMIT 100").bind(now,now).all<{id:string;userId:string;projectId:string;url:string;intervalMinutes:number;nextAt:number;revision:number;objectName:string;email:string;verified:number}>();
  const accessByOwner=new Map<string,PulseAccess>();
  for(let start=0;start<due.results.length&&Date.now()-now<45_000;start+=5){
   const group=due.results.slice(start,start+5),observations=new Map<string,{monitor:typeof group[number];observation:MonitorObservation;claim:string}[]>();
   await Promise.all(group.map(async monitor=>{
    let access=accessByOwner.get(monitor.userId);if(!access){access=await pulseAccess(env,monitor.userId);accessByOwner.set(monitor.userId,access);}
    if(!access.writable){await env.DB.prepare("UPDATE pulse_monitor SET next_at=? WHERE id=? AND next_at=?").bind(now+86_400_000,monitor.id,monitor.nextAt).run();return;}
    const claim=crypto.randomUUID(),claimed=await env.DB.prepare("UPDATE pulse_monitor SET claim=?,claim_until=? WHERE id=? AND active=1 AND next_at=? AND claim_until<=? AND revision=? RETURNING id").bind(claim,now+55_000,monitor.id,monitor.nextAt,now,monitor.revision).first();if(!claimed)return;
    const intervalMinutes=Math.max(monitor.intervalMinutes,access.limits?.intervalMinutes??5);
    const result=await checkHTTP(monitor.url,ownerMatches(env.STAFF_OWNER_EMAIL,monitor.email,!!monitor.verified));
    const observation={id:`${monitor.id}:${monitor.revision}:${monitor.nextAt}`,monitor:monitor.id,revision:monitor.revision,scheduledAt:monitor.nextAt,checkedAt:Date.now(),intervalMinutes,result};
    const list=observations.get(monitor.objectName)??[];list.push({monitor,observation,claim});observations.set(monitor.objectName,list);
   }));
   for(const [objectName,list] of observations){
    const results=await env.PULSE_PROJECT.getByName(objectName).recordChecks(list.map(item=>item.observation));
    await env.DB.batch(list.map((item,i)=>env.DB.prepare("UPDATE pulse_monitor SET state=?,last_at=?,next_at=?,claim=NULL,claim_until=0 WHERE id=? AND claim=? AND revision=? AND active=1").bind(results[i]?.state??"unknown",item.observation.checkedAt,item.observation.checkedAt+item.observation.intervalMinutes*60_000,item.monitor.id,item.claim,item.monitor.revision)));
   }
  }
 }finally{await unlockSite(env.DB,"pulse-cron",lock);}
}
