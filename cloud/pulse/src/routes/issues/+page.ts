import {pulseApi,parseResponse,type Project,type Issue} from "../../lib/api";
import {early} from "../../lib/load";
import {issueFacets,noFacets,issueCounts,noCounts} from "../../lib/issues";
const openIssues=async(project:string)=>(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project},query:{state:"open"}}))).issues;
// A project's open issues, its environments and releases, and its counters.
const issuesOf=(project:string)=>Promise.all([openIssues(project),issueFacets(project),issueCounts(project)]);

// The project in the address loads together with the list of projects
// (its issues, filters and counters too), and all of it with the layout.
export async function load({parent,url}){
 const empty={projects:[] as Project[],projectId:"",issues:[] as Issue[],facets:noFacets,counts:noCounts};
 // Only a well-formed id is fetched early; anything else falls back.
 const raw=url.searchParams.get("project"),wanted=raw&&/^[1-9][0-9]{0,15}$/.test(raw)?raw:null;
 return early(async()=>{
  const [projects,first]=await Promise.all([parseResponse(pulseApi.projects.$get()).then(r=>r.projects),wanted?issuesOf(wanted).catch(()=>null):null]);
  const projectId=projects.find(p=>p.id===wanted)?.id??projects[0]?.id??"";
  if(!projectId)return {...empty,projects};
  const [issues,facets,counts]=projectId===wanted&&first?first:await issuesOf(projectId);
  return {projects,projectId,issues,facets,counts};
 },parent,empty);
}
