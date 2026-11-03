import {attempt} from "../../../lib/load";
import {getPages,getMonitors,getIncidents,type StatusPage,type Monitor,type Incident} from "../../../lib/status-pages";
// The page, the monitors it can show and its incidents, together.
// The tab to open, from the address (?tab=incidents after posting one).
const tabs=["settings","incidents","domain"] as const;
export async function load({parent,params,url}){
 const asked=url.searchParams.get("tab"),tab=tabs.find(t=>t===asked)??"settings";
 const empty={page:null as StatusPage|null,monitors:[] as Monitor[],incidents:[] as Incident[],tab};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{
  const [pages,monitors,incidents]=await Promise.all([getPages(),getMonitors(),getIncidents(params.id).catch(()=>[] as Incident[])]);
  return {page:pages.find(p=>p.id===params.id)??null,monitors,incidents,tab};
 },empty);
}
