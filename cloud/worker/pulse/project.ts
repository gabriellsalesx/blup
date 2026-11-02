import {DurableObject} from "cloudflare:workers";
import {ProjectStore,type Issue} from "./project-store";
import {type SafeEvent} from "./event";
import {pulseAccess,monthOf} from "./limits";
import {reserveGrant} from "./quota";
import {IngestError} from "./protocol";
import {MonitorStore,type MonitorObservation} from "./monitor-store";

export class PulseProject extends DurableObject<Cloudflare.Env>{
 private store:ProjectStore;
 private tail:Promise<unknown>=Promise.resolve();
 constructor(ctx:DurableObjectState,env:Cloudflare.Env){super(ctx,env);this.store=new ProjectStore(ctx.storage.sql,fn=>ctx.storage.transactionSync(fn));}
 private serial<T>(fn:()=>Promise<T>):Promise<T>{const result=this.tail.then(fn);this.tail=result.catch(()=>{});return result;}
 async ingest(project:string,keyHash:string,event:SafeEvent){return this.serial(async()=>{
  try{
   const now=Date.now();
   const row=await this.env.DB.prepare("SELECT p.user_id AS userId FROM pulse_project p JOIN pulse_key k ON k.project_id=p.id WHERE p.id=? AND p.active=1 AND p.deleted_at IS NULL AND k.hash=? AND k.revoked_at IS NULL").bind(project,keyHash).first<{userId:string}>();
   if(!row)throw new IngestError(403,"KEY");
   const access=await pulseAccess(this.env,row.userId,now);if(!access.writable||!access.plan||!access.limits)throw new IngestError(403,"PLAN");
   const cutoff=now-access.limits.retentionDays*86_400_000;
   if(this.store.seen(event.id,cutoff))return {ok:true as const,id:event.id,duplicate:true};
   if(this.store.slots(monthOf(now))===0){const grant=await reserveGrant(this.env.DB,row.userId,project,this.store.sequence(monthOf(now)),access.limits.events,now);this.store.importGrant(grant,monthOf(now));}
   const stillActive=await this.env.DB.prepare("SELECT p.id FROM pulse_project p JOIN pulse_key k ON k.project_id=p.id WHERE p.id=? AND p.active=1 AND p.deleted_at IS NULL AND k.hash=? AND k.revoked_at IS NULL").bind(project,keyHash).first();
   if(!stillActive||!(await pulseAccess(this.env,row.userId)).writable)throw new IngestError(403,"KEY");
   const result=this.store.accept(event,access.plan,now);
   this.ctx.storage.sql.exec("INSERT INTO meta VALUES('project',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",project);
   const nextAlarm=now+60*60*1000,scheduled=await this.ctx.storage.getAlarm();
   if(scheduled===null||scheduled>nextAlarm)await this.ctx.storage.setAlarm(nextAlarm);
   return {ok:true as const,...result};
  }catch(e){if(e instanceof IngestError)return {ok:false as const,status:e.status,code:e.code,wait:e.wait};throw e;}
 });}
 async issues(cutoff:number,filter:Parameters<ProjectStore["issues"]>[1]){return this.store.issues(cutoff,filter);}
 async samples(id:string,cutoff:number){return this.store.samples(id,cutoff);}
 async setState(id:string,state:Issue["state"],cutoff:number){return this.serial(async()=>this.store.setState(id,state,cutoff));}
 async usage(month:string){return this.store.usage(month);}
 async recordChecks(observations:MonitorObservation[]){return this.serial(async()=>{
  const monitorStore=new MonitorStore(this.ctx.storage.sql,fn=>this.ctx.storage.transactionSync(fn));
  const results=[];
  for(const observation of observations){
   const current=await this.env.DB.prepare("SELECT id,project_id AS project FROM pulse_monitor WHERE id=? AND active=1 AND revision=?").bind(observation.monitor,observation.revision).first<{id:string;project:string}>();
   if(current)this.ctx.storage.sql.exec("INSERT INTO meta VALUES('project',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",current.project);
   results.push(current?monitorStore.record(observation):{state:"unknown" as const,duplicate:true});
  }
  const alarm=await this.ctx.storage.getAlarm();if(alarm===null)await this.ctx.storage.setAlarm(Date.now()+60*60_000);
  return results;
 });}
 async monitorHistory(monitor:string,retentionDays:number){return new MonitorStore(this.ctx.storage.sql,fn=>this.ctx.storage.transactionSync(fn)).history(monitor,retentionDays);}
 private async reset(){await this.ctx.storage.deleteAll();}
 async purge(){return this.serial(()=>this.reset());}
 async cleanup(project:string){return this.serial(async()=>{
  const row=await this.env.DB.prepare("SELECT user_id AS userId FROM pulse_project WHERE id=? AND deleted_at IS NULL").bind(project).first<{userId:string}>();
  if(!row){await this.reset();return;}
  const access=await pulseAccess(this.env,row.userId);
  const more=this.store.cleanup(Date.now()-(access.limits?.retentionDays??30)*86_400_000);
  new MonitorStore(this.ctx.storage.sql,fn=>this.ctx.storage.transactionSync(fn)).cleanup(access.limits?.retentionDays??30);
  await this.ctx.storage.setAlarm(Date.now()+(more?60_000:60*60*1000));
 });}
 async alarm(){const project=this.ctx.storage.sql.exec<{value:string}>("SELECT value FROM meta WHERE key='project'").toArray()[0]?.value;if(project)await this.cleanup(project);}
}
