import {attempt} from "../../../lib/load";
import {logoPreview} from "../../../lib/logo-image";
import {getPages,getMonitors,getIncidents,getMaintenance,type StatusPage,type Monitor,type Incident,type Maintenance} from "../../../lib/status-pages";
// The page, the monitors it can show, its incidents, its maintenance and
// its logos, together.
// The tab to open, from the address (?tab=incidents after posting one).
const tabs=["settings","incidents","maintenance","domain"] as const;
export async function load({parent,params,url}){
 const asked=url.searchParams.get("tab"),tab=tabs.find(t=>t===asked)??"settings";
 const empty={page:null as StatusPage|null,monitors:[] as Monitor[],incidents:[] as Incident[],maintenance:[] as Maintenance[],logoPreviews:{light:null as string|null,dark:null as string|null},tab};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{
  const [pages,monitors,incidents,maintenance]=await Promise.all([getPages(),getMonitors(),getIncidents(params.id).catch(()=>[] as Incident[]),getMaintenance(params.id).catch(()=>[] as Maintenance[])]);
  const page=pages.find(p=>p.id===params.id)??null;
  // The logos as the panel can show them (the page may not be public).
  const [light,dark]=page?await Promise.all([logoPreview(page.id,"light",page.logos.light).catch(()=>null),logoPreview(page.id,"dark",page.logos.dark).catch(()=>null)]):[null,null];
  return {page,monitors,incidents,maintenance,logoPreviews:{light,dark},tab};
 },empty);
}
