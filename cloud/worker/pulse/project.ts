import {DurableObject} from "cloudflare:workers";
import {ProjectStore,type Issue} from "./project-store";
import {type SafeEvent} from "./event";
import {pulseAccess,monthOf,monthWait} from "./limits";
import {reserveGrant} from "./quota";
import {IngestError} from "./protocol";
import {enqueueSignals,type AlertSignal} from "./outbox";
import {MonitorStore,type MonitorObservation} from "./monitor-store";

export class PulseProject extends DurableObject<Cloudflare.Env>{
 private readonly store:ProjectStore;
 private readonly monitorStore:MonitorStore;
 private tail:Promise<unknown>=Promise.resolve();
 constructor(ctx:DurableObjectState,env:Cloudflare.Env){super(ctx,env);this.store=new ProjectStore(ctx.storage.sql,fn=>ctx.storage.transactionSync(fn));this.monitorStore=new MonitorStore(ctx.storage.sql,fn=>ctx.storage.transactionSync(fn));}
 private serial<T>(fn:()=>Promise<T>):Promise<T>{const result=this.tail.then(fn);this.tail=result.catch(()=>{});return result;}
 async ingest(project:string,keyHash:string,event:SafeEvent){return this.serial(async()=>{
  try{
   const now=Date.now();
   const row=await this.env.DB.prepare("SELECT p.user_id AS userId FROM pulse_project p JOIN pulse_key k ON k.project_id=p.id WHERE p.id=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL AND k.hash=? AND k.revoked_at IS NULL").bind(project,keyHash).first<{userId:string}>();
   if(!row)throw new IngestError(403,"KEY");
   const access=await pulseAccess(this.env,row.userId,now);if(!access.writable||!access.plan||!access.limits)throw new IngestError(403,"PLAN");
   const cutoff=now-access.limits.retentionDays*86_400_000;
   if(this.store.seen(event.id,cutoff))return {ok:true as const,id:event.id,duplicate:true};
   if(this.store.slots(monthOf(now))===0){const grant=await reserveGrant(this.env.DB,row.userId,project,this.store.sequence(monthOf(now)),access.limits.events,now);this.store.importGrant(grant,monthOf(now));}
   const stillActive=await this.env.DB.prepare("SELECT p.id FROM pulse_project p JOIN pulse_key k ON k.project_id=p.id WHERE p.id=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL AND k.hash=? AND k.revoked_at IS NULL").bind(project,keyHash).first();
   const current=await pulseAccess(this.env,row.userId);if(!stillActive||!current.writable||!current.plan||!current.limits)throw new IngestError(403,"KEY");
   const monthly=await this.env.DB.prepare("SELECT reserved FROM pulse_month WHERE user_id=? AND month=?").bind(row.userId,monthOf(now)).first<{reserved:number}>();
   if((monthly?.reserved??0)>current.limits.events)throw new IngestError(429,"MONTHLY",monthWait(now));
   const result=this.store.accept(event,current.plan,now);
   this.ctx.waitUntil(this.flushAlerts(project).catch(()=>console.error("[blup] pulse_alert_enqueue_retry")));
   this.ctx.storage.sql.exec("INSERT INTO meta VALUES('project',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",project);
   const nextAlarm=now+60*60*1000,scheduled=await this.ctx.storage.getAlarm();
   if(scheduled===null||scheduled>nextAlarm)await this.ctx.storage.setAlarm(nextAlarm);
   return {ok:true as const,...result};
  }catch(e){if(e instanceof IngestError)return {ok:false as const,status:e.status,code:e.code,wait:e.wait};throw e;}
 });}
 async issues(cutoff:number,filter:Parameters<ProjectStore["issues"]>[1]){return this.store.issues(cutoff,filter);}
 async issue(id:string,cutoff:number){return this.store.issue(id,cutoff);}
 async facets(cutoff:number){return this.store.facets(cutoff);}
 async counts(cutoff:number,filter:Parameters<ProjectStore["counts"]>[1]){return this.store.counts(cutoff,filter);}
 async samples(id:string,cutoff:number){return this.store.samples(id,cutoff);}
 async setState(id:string,state:Issue["state"],cutoff:number){return this.serial(async()=>this.store.setState(id,state,cutoff));}
 async flushAlerts(project:string){
  const pending=this.ctx.storage.sql.exec<{id:string;refId:string;kind:AlertSignal["kind"];createdAt:number}>("SELECT id,issue_id AS refId,kind,created_at AS createdAt FROM pending_alert ORDER BY created_at LIMIT 100").toArray();
  if(!pending.length)return;
  await enqueueSignals(this.env,project,pending);
  for(const signal of pending)this.ctx.storage.sql.exec("DELETE FROM pending_alert WHERE id=?",signal.id);
 }
 async notification(id:string,cutoff:number){return this.ctx.storage.sql.exec<{title:string;state:string;environment:string;release:string}>("SELECT title,state,environment,release FROM issue WHERE id=? AND last_at>?",id,cutoff).toArray()[0]??null;}
 async usage(month:string){return this.store.usage(month);}
 // Whether any error reached this project (still kept).
 async hasIssues(){return this.ctx.storage.sql.exec("SELECT 1 FROM issue LIMIT 1").toArray().length>0;}
 async recordChecks(observations:MonitorObservation[]){return this.serial(async()=>{
  const monitorStore=this.monitorStore;
  const results=[];
  for(const observation of observations){
   const current=await this.env.DB.prepare("SELECT m.id,m.project_id AS project,p.user_id AS userId,EXISTS(SELECT 1 FROM pulse_maintenance_component mc JOIN pulse_maintenance w ON w.id=mc.maintenance_id WHERE mc.monitor_id=m.id AND w.starts_at<=? AND w.ends_at>?) AS maintenance FROM pulse_monitor m JOIN pulse_project p ON p.id=m.project_id WHERE m.id=? AND m.active=1 AND m.plan_disabled=0 AND m.revision=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL").bind(observation.checkedAt,observation.checkedAt,observation.monitor,observation.revision).first<{id:string;project:string;userId:string;maintenance:number}>();
   const access=current?await pulseAccess(this.env,current.userId):null,canRecord=!!(current&&access?.writable&&access.limits&&this.ctx.storage.sql.databaseSize<access.limits.projectBytes-1024*1024);
   if(current&&canRecord)this.ctx.storage.sql.exec("INSERT INTO meta VALUES('project',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",current.project);
   results.push(canRecord&&current?monitorStore.record(observation,current.maintenance===1):{state:"unknown" as const,duplicate:true});
  }
  const project=this.ctx.storage.sql.exec<{value:string}>("SELECT value FROM meta WHERE key='project'").toArray()[0]?.value;
  if(project)this.ctx.waitUntil(this.flushAlerts(project).catch(()=>console.error("[blup] pulse_alert_enqueue_retry")));
  const alarm=await this.ctx.storage.getAlarm();if(alarm===null)await this.ctx.storage.setAlarm(Date.now()+60*60_000);
  return results;
 });}
 async monitorHistory(monitor:string,retentionDays:number){return this.monitorStore.history(monitor,retentionDays);}
 private async reset(){await this.ctx.storage.deleteAll();}
 async purge(){return this.serial(()=>this.reset());}
 async cleanup(project:string){return this.serial(async()=>{
  const row=await this.env.DB.prepare("SELECT user_id AS userId FROM pulse_project WHERE id=? AND deleted_at IS NULL").bind(project).first<{userId:string}>();
  if(!row){await this.reset();return;}
  const access=await pulseAccess(this.env,row.userId);
  const more=this.store.cleanup(Date.now()-(access.limits?.retentionDays??30)*86_400_000);
  const moreChecks=this.monitorStore.cleanup(access.limits?.retentionDays??30);
  await this.ctx.storage.setAlarm(Date.now()+(more||moreChecks?60_000:60*60*1000));
 });}
 async alarm(){const project=this.ctx.storage.sql.exec<{value:string}>("SELECT value FROM meta WHERE key='project'").toArray()[0]?.value;if(project)await this.cleanup(project);}
}
