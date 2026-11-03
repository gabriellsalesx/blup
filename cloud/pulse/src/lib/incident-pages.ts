import {resolve} from "$app/paths";
import {attempt} from "./load";
import {getPages,getIncidents,getMaintenance,type StatusPage,type Incident,type Maintenance} from "./status-pages";
// The incident and maintenance screens of a status page: the page with its
// incidents or its maintenance, the way back to their tab and the page of
// one incident.
export const incidentsTab=(page:string)=>`${resolve("/status-pages/[id]",{id:page})}?${new URLSearchParams({tab:"incidents"})}`;
export const maintenanceTab=(page:string)=>`${resolve("/status-pages/[id]",{id:page})}?${new URLSearchParams({tab:"maintenance"})}`;
export const incidentPage=(page:string,incident:string)=>resolve("/status-pages/[id]/incidents/[incident]",{id:page,incident});
export async function loadIncidents(signedIn:boolean,page:string){
 const empty={page:null as StatusPage|null,incidents:[] as Incident[]};
 if(!signedIn)return empty;
 return attempt(async()=>{const [pages,incidents]=await Promise.all([getPages(),getIncidents(page).catch(()=>[] as Incident[])]);return {page:pages.find(p=>p.id===page)??null,incidents};},empty);
}
export async function loadMaintenance(signedIn:boolean,page:string){
 const empty={page:null as StatusPage|null,maintenance:[] as Maintenance[]};
 if(!signedIn)return empty;
 return attempt(async()=>{const [pages,maintenance]=await Promise.all([getPages(),getMaintenance(page).catch(()=>[] as Maintenance[])]);return {page:pages.find(p=>p.id===page)??null,maintenance};},empty);
}
