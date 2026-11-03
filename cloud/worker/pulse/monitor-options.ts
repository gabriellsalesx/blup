import {z} from "zod";
import {monitorOptions,monitorHeader,defaultMonitorOptions,type MonitorOptions,type MonitorHeader} from "./monitor-http";
import {sealTarget,openTarget} from "./channel-crypto";
// A monitor's options and request headers as stored in D1 (the headers
// sealed like a destination), for the routes and the scheduler.
const HEADERS="monitor-headers";
export type OptionRow={method:string;expectedStatus:string;keyword:string|null;keywordMode:string|null;timeoutSeconds:number};
// The stored options, checked on the way out like any stored value.
export function optionsOf(row:OptionRow):MonitorOptions{let status:unknown=[];try{status=JSON.parse(row.expectedStatus||"[]");}catch{/* checked below */}const parsed=monitorOptions.safeParse({method:row.method,expectedStatus:status,keyword:row.keyword&&row.keywordMode?{text:row.keyword,mode:row.keywordMode}:null,timeoutSeconds:row.timeoutSeconds});return parsed.success?parsed.data:defaultMonitorOptions;}
export const optionColumns=(o:MonitorOptions)=>[o.method,JSON.stringify(o.expectedStatus),o.keyword?.text??null,o.keyword?.mode??null,o.timeoutSeconds] as const;
export async function openHeaders(env:Cloudflare.Env,owner:string,monitor:string,sealed:string|null):Promise<MonitorHeader[]>{if(!sealed)return [];const parsed=z.array(monitorHeader).max(5).safeParse(JSON.parse(await openTarget(env,owner,monitor,HEADERS,sealed)));return parsed.success?parsed.data:[];}
export const sealHeaders=async(env:Cloudflare.Env,owner:string,monitor:string,headers:MonitorHeader[])=>headers.length?sealTarget(env,owner,monitor,HEADERS,JSON.stringify(headers)):null;
export const OPTION_SELECT="method,expected_status AS expectedStatus,keyword,keyword_mode AS keywordMode,timeout_seconds AS timeoutSeconds";
