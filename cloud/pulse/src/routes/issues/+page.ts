import {pulseApi,parseResponse,type Project,type Issue} from "../../lib/api";
import {attempt} from "../../lib/load";
const openIssues=async(project:string)=>(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project},query:{state:"open"}}))).issues;
// The project in the address loads together with the list of projects.
export async function load({parent,url}){
 const empty={projects:[] as Project[],projectId:"",issues:[] as Issue[]};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{
  // Only a well-formed id is fetched early; anything else falls back.
  const raw=url.searchParams.get("project"),wanted=raw&&/^[1-9][0-9]{0,15}$/.test(raw)?raw:null;
  const [projects,early]=await Promise.all([parseResponse(pulseApi.projects.$get()).then(r=>r.projects),wanted?openIssues(wanted).catch(()=>null):null]);
  const projectId=projects.find(p=>p.id===wanted)?.id??projects[0]?.id??"";
  const issues=!projectId?[]:projectId===wanted&&early?early:await openIssues(projectId);
  return {projects,projectId,issues};
 },empty);
}
