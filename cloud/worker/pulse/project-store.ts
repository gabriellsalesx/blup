import type {SafeEvent} from "./event";
import type {PulsePlan} from "./limits";
import {monthOf,pulseLimits} from "./limits";
import {IngestError} from "./protocol";

export type Issue={id:string;state:"open"|"resolved"|"ignored";title:string;level:string;firstAt:number;lastAt:number;count:number;environment:string;release:string};
export type Occurrence={id:string;receivedAt:number;event:SafeEvent};
export type ProjectSQL=Pick<SqlStorage,"exec"|"databaseSize">;

// Pure SQL engine is also exercised against actual SQLite in tests.
export class ProjectStore{
 constructor(readonly sql:ProjectSQL,readonly transaction:<T>(fn:()=>T)=>T){
  sql.exec(`CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS grant_slot(id TEXT PRIMARY KEY,month TEXT NOT NULL,amount INTEGER NOT NULL,used INTEGER NOT NULL DEFAULT 0);
CREATE INDEX IF NOT EXISTS grant_month_idx ON grant_slot(month);
CREATE TABLE IF NOT EXISTS issue(id TEXT PRIMARY KEY,state TEXT NOT NULL,title TEXT NOT NULL,level TEXT NOT NULL,first_at INTEGER NOT NULL,last_at INTEGER NOT NULL,environment TEXT NOT NULL,release TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS issue_last_idx ON issue(last_at,id);
CREATE TABLE IF NOT EXISTS seen(id TEXT PRIMARY KEY,issue_id TEXT NOT NULL,received_at INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS seen_expiry_idx ON seen(received_at);
CREATE TABLE IF NOT EXISTS sample(id TEXT PRIMARY KEY,issue_id TEXT NOT NULL,received_at INTEGER NOT NULL,event TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS sample_issue_idx ON sample(issue_id,received_at,id);
CREATE INDEX IF NOT EXISTS sample_expiry_idx ON sample(received_at);
CREATE TABLE IF NOT EXISTS issue_day(issue_id TEXT NOT NULL,day INTEGER NOT NULL,count INTEGER NOT NULL,PRIMARY KEY(issue_id,day));
CREATE INDEX IF NOT EXISTS issue_day_expiry_idx ON issue_day(day);
CREATE TABLE IF NOT EXISTS burst(minute INTEGER PRIMARY KEY,count INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS pending_alert(id TEXT PRIMARY KEY,issue_id TEXT NOT NULL,kind TEXT NOT NULL,created_at INTEGER NOT NULL);
INSERT OR IGNORE INTO meta VALUES('schema','1');`);
 }
 seen(id:string,cutoff:number){return this.sql.exec<{id:string}>("SELECT id FROM seen WHERE id=? AND received_at>?",id,cutoff).toArray().length>0;}
 slots(month:string){return this.sql.exec<{slots:number}>("SELECT coalesce(sum(amount-used),0) AS slots FROM grant_slot WHERE month=?",month).toArray()[0].slots;}
 sequence(month:string){return this.sql.exec<{n:number}>("SELECT count(*) AS n FROM grant_slot WHERE month=?",month).toArray()[0].n;}
 importGrant(grant:{id:string;amount:number},month:string){this.sql.exec("INSERT OR IGNORE INTO grant_slot(id,month,amount) VALUES(?,?,?)",grant.id,month,grant.amount);}
 accept(event:SafeEvent,plan:PulsePlan,now=Date.now()){
  const limits=pulseLimits[plan],cutoff=now-limits.retentionDays*86_400_000,month=monthOf(now),minute=Math.floor(now/60_000);
  return this.transaction(()=>{
   if(this.seen(event.id,cutoff))return {duplicate:true,id:event.id};
   // Keep one MiB of physical headroom for the bounded event and indexes.
   // A failed transaction cannot free space; never pretend it evicted rows.
   if(this.sql.databaseSize>=limits.projectBytes-1024*1024)throw new IngestError(429,"CAPACITY",60);
   const burst=this.sql.exec<{count:number}>("SELECT count FROM burst WHERE minute=?",minute).toArray()[0]?.count??0;if(burst>=limits.burst)throw new IngestError(429,"BURST",60-Math.floor(now/1000)%60);
   const grant=this.sql.exec<{id:string}>("SELECT id FROM grant_slot WHERE month=? AND used<amount ORDER BY id LIMIT 1",month).toArray()[0];if(!grant)throw new IngestError(429,"MONTHLY",60);
   const previous=this.sql.exec<{state:string}>("SELECT state FROM issue WHERE id=? AND last_at>?",event.group,cutoff).toArray()[0];
   this.sql.exec("INSERT INTO issue VALUES(?,'open',?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=CASE WHEN issue.last_at<=? THEN 'open' WHEN issue.state='ignored' THEN 'ignored' ELSE 'open' END,first_at=CASE WHEN issue.last_at<=? THEN excluded.first_at ELSE issue.first_at END,title=excluded.title,level=excluded.level,last_at=excluded.last_at,environment=excluded.environment,release=excluded.release",event.group,event.title,event.level,now,now,event.environment,event.release,cutoff,cutoff);
   this.sql.exec("INSERT INTO seen VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET issue_id=excluded.issue_id,received_at=excluded.received_at",event.id,event.group,now);
   this.sql.exec("INSERT INTO sample VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET issue_id=excluded.issue_id,received_at=excluded.received_at,event=excluded.event",event.id,event.group,now,JSON.stringify(event));
   this.sql.exec("DELETE FROM sample WHERE issue_id=? AND id NOT IN (SELECT id FROM sample WHERE issue_id=? ORDER BY received_at DESC,id DESC LIMIT 100)",event.group,event.group);
   this.sql.exec("INSERT INTO issue_day VALUES(?,?,1) ON CONFLICT(issue_id,day) DO UPDATE SET count=count+1",event.group,Math.floor(now/86_400_000));
   this.sql.exec("UPDATE grant_slot SET used=used+1 WHERE id=?",grant.id);
   this.sql.exec("INSERT INTO burst VALUES(?,1) ON CONFLICT(minute) DO UPDATE SET count=count+1",minute);
   if(!previous||previous.state==="resolved")this.sql.exec("INSERT OR IGNORE INTO pending_alert VALUES(?,?,?,?)",event.id,event.group,previous?"regression":"issue",now);
   return {duplicate:false,id:event.id};
  });
 }
 issues(cutoff:number,filter:{state?:string;environment?:string;release?:string;q?:string;before?:number;beforeId?:string;limit:number}){
  return this.sql.exec<Issue>(`SELECT i.id,i.state,i.title,i.level,i.first_at AS firstAt,i.last_at AS lastAt,(SELECT coalesce(sum(count),0) FROM issue_day WHERE issue_id=i.id AND day>=?) AS count,i.environment,i.release FROM issue i WHERE last_at>? AND (? IS NULL OR state=?) AND (? IS NULL OR environment=?) AND (? IS NULL OR release=?) AND (? IS NULL OR instr(lower(title),lower(?))>0) AND (? IS NULL OR last_at<? OR (last_at=? AND ? IS NOT NULL AND i.id<?)) ORDER BY last_at DESC,id DESC LIMIT ?`,Math.floor(cutoff/86_400_000),cutoff,filter.state??null,filter.state??null,filter.environment??null,filter.environment??null,filter.release??null,filter.release??null,filter.q??null,filter.q??null,filter.before??null,filter.before??null,filter.before??null,filter.beforeId??null,filter.beforeId??null,filter.limit).toArray();
 }
 samples(id:string,cutoff:number,limit=50){return this.sql.exec<{id:string;receivedAt:number;event:string}>("SELECT id,received_at AS receivedAt,event FROM sample WHERE issue_id=? AND received_at>? ORDER BY received_at DESC,id DESC LIMIT ?",id,cutoff,limit).toArray().map(row=>({...row,event:JSON.parse(row.event) as SafeEvent}));}
 setState(id:string,state:Issue["state"],cutoff:number){return this.sql.exec("UPDATE issue SET state=? WHERE id=? AND last_at>? RETURNING id",state,id,cutoff).toArray().length>0;}
 usage(month:string){return {accepted:this.sql.exec<{n:number}>("SELECT coalesce(sum(used),0) AS n FROM grant_slot WHERE month=?",month).toArray()[0].n,bytes:this.sql.databaseSize};}
 cleanup(cutoff:number,now=Date.now()){
  return this.transaction(()=>{
   for(const table of ["sample","seen"]){this.sql.exec(`DELETE FROM ${table} WHERE id IN (SELECT id FROM ${table} WHERE received_at<=? LIMIT 500)`,cutoff);}
   this.sql.exec("DELETE FROM issue_day WHERE (issue_id,day) IN(SELECT issue_id,day FROM issue_day WHERE day<? LIMIT 500)",Math.floor(cutoff/86_400_000));
   this.sql.exec("DELETE FROM issue WHERE id IN (SELECT id FROM issue WHERE last_at<=? LIMIT 500)",cutoff);
   this.sql.exec("DELETE FROM burst WHERE minute<?",Math.floor(now/60_000)-1);
   this.sql.exec("DELETE FROM grant_slot WHERE month<?",monthOf(cutoff));
   this.sql.exec("DELETE FROM pending_alert WHERE id IN(SELECT id FROM pending_alert WHERE created_at<=? LIMIT 500)",now-86_400_000);
   return this.sql.exec<{n:number}>("SELECT EXISTS(SELECT 1 FROM seen WHERE received_at<=?) OR EXISTS(SELECT 1 FROM sample WHERE received_at<=?) OR EXISTS(SELECT 1 FROM issue WHERE last_at<=?) OR EXISTS(SELECT 1 FROM issue_day WHERE day<?) OR EXISTS(SELECT 1 FROM pending_alert WHERE created_at<=?) AS n",cutoff,cutoff,cutoff,Math.floor(cutoff/86_400_000),now-86_400_000).toArray()[0].n>0;
  });
 }
}
