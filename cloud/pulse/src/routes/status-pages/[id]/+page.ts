import {attempt} from "../../../lib/load";
import {getPages,getMonitors,getIncidents,type StatusPage,type Monitor,type Incident} from "../../../lib/status-pages";
// The page, the monitors it can show and its incidents, together.
export async function load({parent,params}){
 const empty={page:null as StatusPage|null,monitors:[] as Monitor[],incidents:[] as Incident[]};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{
  const [pages,monitors,incidents]=await Promise.all([getPages(),getMonitors(),getIncidents(params.id).catch(()=>[] as Incident[])]);
  return {page:pages.find(p=>p.id===params.id)??null,monitors,incidents};
 },empty);
}
