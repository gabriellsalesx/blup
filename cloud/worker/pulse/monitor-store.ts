import type {ProjectSQL} from "./project-store";
import type {CheckResult} from "./monitor-http";
export type MonitorObservation={id:string;monitor:string;revision:number;scheduledAt:number;checkedAt:number;intervalMinutes:number;result:CheckResult};
export class MonitorStore{
 constructor(readonly sql:ProjectSQL,readonly transaction:<T>(fn:()=>T)=>T){sql.exec(`CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS monitor_state(id TEXT PRIMARY KEY,state TEXT NOT NULL,failures INTEGER NOT NULL,successes INTEGER NOT NULL,last_at INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS monitor_state_expiry ON monitor_state(last_at);
CREATE TABLE IF NOT EXISTS monitor_check(id TEXT PRIMARY KEY,monitor_id TEXT NOT NULL,scheduled_at INTEGER NOT NULL,checked_at INTEGER NOT NULL,outcome TEXT NOT NULL,state TEXT NOT NULL,status INTEGER,duration_ms INTEGER NOT NULL,reason TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS monitor_check_lookup ON monitor_check(monitor_id,checked_at,id);
CREATE INDEX IF NOT EXISTS monitor_check_expiry ON monitor_check(checked_at);
CREATE TABLE IF NOT EXISTS monitor_day(monitor_id TEXT NOT NULL,day INTEGER NOT NULL,up INTEGER NOT NULL DEFAULT 0,down INTEGER NOT NULL DEFAULT 0,unknown INTEGER NOT NULL DEFAULT 0,duration_ms INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(monitor_id,day));
CREATE INDEX IF NOT EXISTS monitor_day_expiry ON monitor_day(day);
INSERT INTO meta VALUES('schema','2') ON CONFLICT(key) DO UPDATE SET value=excluded.value WHERE CAST(meta.value AS INTEGER)<2;`);}
 record(observation:MonitorObservation){return this.transaction(()=>{
  const {id,monitor,scheduledAt,checkedAt,intervalMinutes}=observation,stateId=`${monitor}:${observation.revision}`,existing=this.sql.exec<{state:"up"|"down"|"unknown"}>("SELECT state FROM monitor_check WHERE id=?",id).toArray()[0];if(existing)return {state:existing.state,duplicate:true};
  const previous=this.sql.exec<{state:"up"|"down"|"unknown";failures:number;successes:number;last_at:number}>("SELECT * FROM monitor_state WHERE id=?",stateId).toArray()[0];
  if(previous&&checkedAt<previous.last_at)return {state:previous.state,duplicate:true};
  const result=checkedAt-scheduledAt>intervalMinutes*120_000?{...observation.result,outcome:"unknown" as const,reason:"gap" as const}:observation.result;
  const uninterrupted=previous&&scheduledAt-previous.last_at<=intervalMinutes*120_000;
  const failures=result.outcome==="down"?(uninterrupted?previous.failures:0)+1:0,successes=result.outcome==="up"?(uninterrupted?previous.successes:0)+1:0;
  const state=result.outcome==="unknown"?"unknown":failures>=2?"down":successes>=2?"up":previous?.state??"unknown";
  this.sql.exec("INSERT INTO monitor_state VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,failures=excluded.failures,successes=excluded.successes,last_at=excluded.last_at",stateId,state,failures,successes,checkedAt);
  this.sql.exec("INSERT INTO monitor_check VALUES(?,?,?,?,?,?,?,?,?)",id,monitor,scheduledAt,checkedAt,result.outcome,state,result.status,result.durationMs,result.reason);
  this.sql.exec("INSERT INTO monitor_day VALUES(?,?,?,?,?,?) ON CONFLICT(monitor_id,day) DO UPDATE SET up=up+excluded.up,down=down+excluded.down,unknown=unknown+excluded.unknown,duration_ms=duration_ms+excluded.duration_ms",monitor,Math.floor(checkedAt/86_400_000),Number(result.outcome==="up"),Number(result.outcome==="down"),Number(result.outcome==="unknown"),result.durationMs);
  if(previous?.state!==state&&(state==="up"||state==="down"))this.sql.exec("INSERT OR IGNORE INTO pending_alert VALUES(?,?,?,?)",id,monitor,`monitor_${state}`,checkedAt);
  return {state,duplicate:false};
 });}
 history(monitor:string,retentionDays:number,now=Date.now()){
  const cutoff=now-retentionDays*86_400_000;
  const checks=this.sql.exec<{id:string;scheduledAt:number;checkedAt:number;outcome:string;state:string;status:number|null;durationMs:number;reason:string}>("SELECT id,scheduled_at AS scheduledAt,checked_at AS checkedAt,outcome,state,status,duration_ms AS durationMs,reason FROM monitor_check WHERE monitor_id=? AND checked_at>? ORDER BY checked_at DESC LIMIT 100",monitor,Math.max(cutoff,now-7*86_400_000)).toArray();
  const days=this.sql.exec<{day:number;up:number;down:number;unknown:number;durationMs:number}>("SELECT day,up,down,unknown,duration_ms AS durationMs FROM monitor_day WHERE monitor_id=? AND day>=? ORDER BY day DESC LIMIT 90",monitor,Math.ceil(cutoff/86_400_000)).toArray();
  return {checks,days};
 }
 cleanup(retentionDays:number,now=Date.now()){
  const checkCutoff=now-7*86_400_000,dayCutoff=Math.ceil((now-retentionDays*86_400_000)/86_400_000),stateCutoff=now-retentionDays*86_400_000;
  this.sql.exec("DELETE FROM monitor_state WHERE id IN(SELECT id FROM monitor_state WHERE last_at<=? LIMIT 500)",stateCutoff);
  this.sql.exec("DELETE FROM monitor_check WHERE id IN(SELECT id FROM monitor_check WHERE checked_at<=? LIMIT 500)",checkCutoff);
  this.sql.exec("DELETE FROM monitor_day WHERE (monitor_id,day) IN(SELECT monitor_id,day FROM monitor_day WHERE day<? LIMIT 500)",dayCutoff);
  return this.sql.exec<{more:number}>("SELECT EXISTS(SELECT 1 FROM monitor_check WHERE checked_at<=?) OR EXISTS(SELECT 1 FROM monitor_day WHERE day<?) OR EXISTS(SELECT 1 FROM monitor_state WHERE last_at<=?) AS more",checkCutoff,dayCutoff,stateCutoff).toArray()[0].more===1;
 }
}
