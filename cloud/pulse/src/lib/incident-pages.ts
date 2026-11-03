import {resolve} from "$app/paths";
import {attempt} from "./load";
import {getPages,getIncidents,type StatusPage,type Incident} from "./status-pages";
// The incident screens of a status page: the page and its incidents, the
// way back to its Incidents tab and the page of one incident.
export const incidentsTab=(page:string)=>`${resolve("/status-pages/[id]",{id:page})}?${new URLSearchParams({tab:"incidents"})}`;
export const incidentPage=(page:string,incident:string)=>resolve("/status-pages/[id]/incidents/[incident]",{id:page,incident});
export async function loadIncidents(signedIn:boolean,page:string){
 const empty={page:null as StatusPage|null,incidents:[] as Incident[]};
 if(!signedIn)return empty;
 return attempt(async()=>{const [pages,incidents]=await Promise.all([getPages(),getIncidents(page).catch(()=>[] as Incident[])]);return {page:pages.find(p=>p.id===page)??null,incidents};},empty);
}
