import {type InferResponseType} from "hono/client";
import {t,intlLocale} from "#lib/i18n.svelte.ts";
import {pulseApi} from "./api";
// What the monitor list and a monitor's own page share: its state, its days
// as bars with their tips and how long it has been like this.
export type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
export const checksGET=pulseApi.monitors[":monitor"].checks.$get;
export type History=InferResponseType<typeof checksGET,200>;
export type Check=History["checks"][number];
export type MonitorState="up"|"down"|"unknown"|"paused";
const DAY=86_400_000;
export const stateOf=(m:Monitor):MonitorState=>!m.active?"paused":m.state==="up"?"up":m.state==="down"?"down":"unknown";
export const stateName=(s:string)=>s==="paused"?t("pulsePaused"):t(s==="up"?"pulseOperational":s==="down"?"pulseDown":"pulseUnknown");
export const stateColor:Record<MonitorState,string>={up:"text-emerald-600 dark:text-emerald-400",down:"text-red-600 dark:text-red-400",unknown:"text-neutral-500",paused:"text-neutral-500"};
const ago=(ms:number)=>{const rtf=new Intl.RelativeTimeFormat(intlLocale(),{numeric:"always"}),m=Math.round((Date.now()-ms)/60000);return m<60?rtf.format(-Math.max(1,m),"minute"):m<1440?rtf.format(-Math.round(m/60),"hour"):rtf.format(-Math.round(m/1440),"day");};
// When the state last changed, from the detailed checks (newest first,
// kept a few days).
export function since(m:Monitor,history?:History){const checks=history?.checks??[],current=m.state;if(!checks.length||!m.active||(current!=="up"&&current!=="down"))return "";let first=checks[0];for(const c of checks){if(c.outcome!==current)return t("pulseChangedAt",{time:ago(first.checkedAt)});first=c;}return t("pulseStableFor",{days:7});}
export function bars(history:History|undefined,count:number){const day=new Intl.DateTimeFormat(intlLocale(),{day:"numeric",month:"short",timeZone:"UTC"}),byDay=new Map((history?.days??[]).map(d=>[d.day,d])),today=Math.floor(Date.now()/DAY);return Array.from({length:count},(_,i)=>{const key=today-count+1+i,d=byDay.get(key);const up=d?.up??0,down=d?.down??0,total=up+down,date=day.format(new Date(key*DAY));const tip=!total?[date,t("pulseNotMonitored")]:up===total?[date,t("pulseOperational")]:[date,up/total>=0.9?t("pulseDegraded"):t("pulseDown"),t("pulseUptime",{value:(up/total*100).toLocaleString(intlLocale(),{maximumFractionDigits:1})})];return {key,tip:tip.join("\n"),up,down};});}
export const reasonName=(c:Check)=>c.reason==="http"?t("pulseReasonHttp",{status:String(c.status??"-")}):c.reason==="timeout"?t("pulseReasonTimeout"):c.reason==="network"?t("pulseReasonNetwork"):c.reason==="blocked"?t("pulseReasonBlocked"):t("pulseReasonGap");
