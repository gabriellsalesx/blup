import {siteLock,unlockSite} from "../quota";
import {pulseAccess,sha256} from "./limits";
import {openTarget} from "./channel-crypto";
import {deliverDiscord,deliverTelegram} from "./transports";
import {languageOf} from "../../shared/language";
import {translate,type PlainMessageKey} from "../../src/lib/i18n/catalog";
import {cleanText} from "./event";
import {panelLink,type AlertMessage} from "./alert-message";
export type AlertSignal={id:string;refId:string;kind:"issue"|"regression"|"monitor_up"|"monitor_down";createdAt:number};
export async function enqueueSignals(env:Cloudflare.Env,project:string,signals:AlertSignal[],now=Date.now()){
 if(!signals.length)return;
 const projectRow=await env.DB.prepare("SELECT user_id AS userId FROM pulse_project WHERE id=? AND active=1 AND deleted_at IS NULL").bind(project).first<{userId:string}>();if(!projectRow||!(await pulseAccess(env,projectRow.userId)).writable)return;
 const channels=await env.DB.prepare("SELECT c.id FROM pulse_project_channel pc JOIN pulse_channel c ON c.id=pc.channel_id WHERE pc.project_id=? AND c.user_id=? AND c.active=1 LIMIT 2").bind(project,projectRow.userId).all<{id:string}>();
 for(const signal of signals){if(signal.createdAt<now-86_400_000)continue;for(const channel of channels.results){
  const id=await sha256(JSON.stringify([project,signal.id,channel.id])),cooldown=`${channel.id}:${signal.kind}:${signal.refId}`;
  await env.DB.batch([
   env.DB.prepare("INSERT OR IGNORE INTO pulse_outbox(id,user_id,project_id,channel_id,ref_id,kind,created_at,expires_at,next_at) SELECT ?,?,?,?,?,?,?,?,? WHERE NOT EXISTS(SELECT 1 FROM pulse_cooldown WHERE key=? AND until>?) AND EXISTS(SELECT 1 FROM pulse_channel WHERE id=? AND active=1)").bind(id,projectRow.userId,project,channel.id,signal.refId,signal.kind,now,signal.createdAt+86_400_000,now,cooldown,now,channel.id),
   env.DB.prepare("INSERT INTO pulse_cooldown(key,until) SELECT ?,? WHERE EXISTS(SELECT 1 FROM pulse_outbox WHERE id=? AND created_at=?) ON CONFLICT(key) DO UPDATE SET until=max(until,excluded.until)").bind(cooldown,now+15*60_000,id,now),
  ]);
 }}
}
export async function reconcileAlerts(env:Cloudflare.Env,now=Date.now()){
 const rows=await env.DB.prepare("SELECT p.id,p.object_name AS objectName FROM pulse_project p WHERE p.alert_checked_at<? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL AND EXISTS(SELECT 1 FROM pulse_project_channel pc WHERE pc.project_id=p.id) ORDER BY p.alert_checked_at LIMIT 50").bind(now-60_000).all<{id:string;objectName:string}>();
 for(const p of rows.results){await env.PULSE_PROJECT.getByName(p.objectName).flushAlerts(p.id);await env.DB.prepare("UPDATE pulse_project SET alert_checked_at=? WHERE id=?").bind(now,p.id).run();}
}
async function budget(db:D1Database,kind:string,now:number,spacing:number){
 await db.prepare("INSERT OR IGNORE INTO pulse_delivery_budget VALUES(?,0)").bind(kind).run();
 if(await db.prepare("UPDATE pulse_delivery_budget SET next_at=? WHERE key=? AND next_at<=? RETURNING key").bind(now+spacing,kind,now).first())return {ready:true,nextAt:now};
 const row=await db.prepare("SELECT next_at AS nextAt FROM pulse_delivery_budget WHERE key=?").bind(kind).first<{nextAt:number}>();return {ready:false,nextAt:row?.nextAt??now+1000};
}
async function deliverySlot(db:D1Database,type:string,target:string){
 const scopes:BudgetScope[]=[{key:type,spacing:type==="telegram"?40:100}];
 if(type==="telegram")scopes.unshift({key:`telegram-chat:${await sha256(target)}`,spacing:1100});
 for(const scope of scopes){
  let slot=await budget(db,scope.key,Date.now(),scope.spacing);
  if(!slot.ready&&slot.nextAt-Date.now()<=1200){await new Promise(resolve=>setTimeout(resolve,Math.max(1,slot.nextAt-Date.now())));slot=await budget(db,scope.key,Date.now(),scope.spacing);}
  if(!slot.ready)return slot;
 }
 return {ready:true,nextAt:Date.now()};
}
type BudgetScope={key:string;spacing:number};
const labels={issue:"pulseAlertIssue",regression:"pulseAlertRegression",monitor_up:"pulseAlertUp",monitor_down:"pulseAlertDown",test:"pulseAlertTest"} as const;
type Kind=keyof typeof labels;
function isKind(v:string):v is Kind{return Object.hasOwn(labels,v);}
export async function dispatchAlerts(env:Cloudflare.Env,now=Date.now()){
 const lock=await siteLock(env.DB,"pulse-dispatch",55_000);if(!lock)return;
 try{
  const due=await env.DB.prepare("SELECT o.id,o.user_id AS userId,o.project_id AS projectId,o.channel_id AS channelId,o.ref_id AS refId,o.kind,o.attempts,o.expires_at AS expiresAt,c.type,c.target,c.next_at AS channelNext,c.active,u.language FROM pulse_outbox o JOIN pulse_channel c ON c.id=o.channel_id JOIN user u ON u.id=o.user_id WHERE o.state='pending' AND o.next_at<=? AND o.claim_until<=? ORDER BY o.next_at LIMIT 50").bind(now,now).all<{id:string;userId:string;projectId:string|null;channelId:string;refId:string;kind:string;attempts:number;expiresAt:number;type:string;target:string;channelNext:number;active:number;language:string}>();
  for(const item of due.results){
   if(Date.now()-now>45_000)break;
   const access=await pulseAccess(env,item.userId),discard=async()=>{await env.DB.prepare("UPDATE pulse_outbox SET state='discarded',claim=NULL,claim_until=0 WHERE id=?").bind(item.id).run();};
   if(item.expiresAt<=Date.now()||item.attempts>=5||!item.active||!access.writable||!isKind(item.kind)){await discard();continue;}
   // What the alert says, in the owner's language, from what is true now
   // (an issue still open, a monitor still in that state).
   const lang=languageOf(item.language)??"en",say=(key:PlainMessageKey)=>translate(lang,key);
   const project=item.projectId?await env.DB.prepare("SELECT name,object_name AS objectName FROM pulse_project p WHERE p.id=? AND p.user_id=? AND p.active=1 AND p.plan_disabled=0 AND p.deleted_at IS NULL AND EXISTS(SELECT 1 FROM pulse_project_channel pc WHERE pc.project_id=p.id AND pc.channel_id=?)").bind(item.projectId,item.userId,item.channelId).first<{name:string;objectName:string}>():null;
   if(item.projectId&&!project){await discard();continue;}
   const projectField=project?[{name:say("pulseFieldProject"),value:cleanText(project.name,80)}]:[];
   let message:AlertMessage;
   if(item.kind==="test"&&!item.projectId)message={label:say("pulseAlertTestLabel"),title:say("pulseAlertTest"),url:panelLink("/alerts"),tone:"brand",fields:[]};
   else if(item.kind==="test"||item.kind==="monitor_up"||item.kind==="monitor_down"){
    const monitor=await env.DB.prepare("SELECT name,state,url FROM pulse_monitor WHERE id=? AND project_id=? AND user_id=?"+(item.kind==="test"?"":" AND active=1")).bind(item.refId,item.projectId,item.userId).first<{name:string;state:string;url:string}>();
    if(!monitor||item.kind!=="test"&&`monitor_${monitor.state}`!==item.kind){await discard();continue;}
    const fields=[...projectField,{name:say("pulseFieldAddress"),value:cleanText(monitor.url,300)}];
    message=item.kind==="test"?{label:say("pulseAlertTestLabel"),title:cleanText(monitor.name,160),url:panelLink(`/monitors/${item.refId}`),tone:"brand",description:say("pulseAlertMonitorTest"),fields}:{label:say(labels[item.kind]),title:cleanText(monitor.name,160),url:panelLink(`/monitors/${item.refId}`),tone:item.kind==="monitor_down"?"red":"green",fields};
   }else{
    if(!project){await discard();continue;}
    const issue=await env.PULSE_PROJECT.getByName(project.objectName).notification(item.refId,Date.now()-(access.limits?.retentionDays??30)*86_400_000);if(!issue||issue.state!=="open"){await discard();continue;}
    message={label:say(labels[item.kind]),title:issue.title,url:panelLink(`/issues?${new URLSearchParams({project:item.projectId??""})}`),tone:item.kind==="regression"?"amber":"red",fields:[...projectField,{name:say("pulseFieldEnvironment"),value:cleanText(issue.environment,80)},{name:say("pulseFieldRelease"),value:cleanText(issue.release,80)}]};
   }
   if(!["discord","telegram"].includes(item.type)){await discard();continue;}
   if(item.channelNext>Date.now()){
    await env.DB.prepare("UPDATE pulse_outbox SET next_at=? WHERE id=?").bind(Math.max(Date.now()+1100,item.channelNext),item.id).run();continue;
   }
   const claim=crypto.randomUUID(),claimed=await env.DB.prepare("UPDATE pulse_outbox SET claim=?,claim_until=? WHERE id=? AND state='pending' AND claim_until<=? RETURNING id").bind(claim,Date.now()+20_000,item.id,Date.now()).first();if(!claimed)continue;
   let target:string;
   try{target=await openTarget(env,item.userId,item.channelId,item.type,item.target);}catch{await env.DB.batch([env.DB.prepare("UPDATE pulse_channel SET last_error='CHANNEL_UNAVAILABLE' WHERE id=?").bind(item.channelId),env.DB.prepare("UPDATE pulse_outbox SET next_at=?,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(Date.now()+300_000,item.id,claim)]);continue;}
   const slot=await deliverySlot(env.DB,item.type,target);if(!slot.ready){await env.DB.prepare("UPDATE pulse_outbox SET next_at=?,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(slot.nextAt,item.id,claim).run();continue;}
   const result=item.type==="discord"?await deliverDiscord(target,message):await deliverTelegram(env,target,message),at=Date.now(),retry=Math.max(result.waitMs,Math.min(300_000,15_000*2**item.attempts)+Math.floor(Math.random()*1000));
   await env.DB.batch([
    env.DB.prepare("UPDATE pulse_outbox SET state=?,attempts=attempts+1,next_at=?,claim=NULL,claim_until=0 WHERE id=? AND claim=?").bind(result.sent?"sent":result.permanent||item.attempts>=4?"discarded":"pending",at+retry,item.id,claim),
    env.DB.prepare("UPDATE pulse_channel SET last_error=?,active=CASE WHEN ? THEN 0 ELSE active END,next_at=max(next_at,?) WHERE id=?").bind(result.code,result.permanent?1:0,at+result.waitMs,item.channelId),
    ...(result.global?[env.DB.prepare("UPDATE pulse_delivery_budget SET next_at=max(next_at,?) WHERE key=?").bind(at+result.waitMs,item.type)]:[]),
   ]);
  }
 }finally{await unlockSite(env.DB,"pulse-dispatch",lock);}
}
