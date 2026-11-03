import {attempt} from "../../../lib/load";
import {getPages,getMonitors,getIncidents,getMaintenance,type StatusPage,type Monitor,type Incident,type Maintenance} from "../../../lib/status-pages";
// The page, the monitors it can show, its incidents and its maintenance,
// together.
// The tab to open, from the address (?tab=incidents after posting one).
const tabs=["settings","incidents","maintenance","domain"] as const;
export async function load({parent,params,url}){
 const asked=url.searchParams.get("tab"),tab=tabs.find(t=>t===asked)??"settings";
 const empty={page:null as StatusPage|null,monitors:[] as Monitor[],incidents:[] as Incident[],maintenance:[] as Maintenance[],tab};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{
  const [pages,monitors,incidents,maintenance]=await Promise.all([getPages(),getMonitors(),getIncidents(params.id).catch(()=>[] as Incident[]),getMaintenance(params.id).catch(()=>[] as Maintenance[])]);
  return {page:pages.find(p=>p.id===params.id)??null,monitors,incidents,maintenance,tab};
 },empty);
}
