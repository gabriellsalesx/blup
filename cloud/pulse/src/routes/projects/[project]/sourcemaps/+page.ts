import {type InferResponseType} from "hono/client";
import {pulseApi,parseResponse,type Project} from "../../../../lib/api";
import {attempt} from "../../../../lib/load";
const sourcemapsGET=pulseApi.projects[":project"].sourcemaps.$get;
export type Sourcemaps=InferResponseType<typeof sourcemapsGET,200>;
// The project, its upload token (only its start) and the maps sent.
export async function load({parent,params}){
 const empty={project:null as Project|null,sourcemaps:null as Sourcemaps|null};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{const [projects,sourcemaps]=await Promise.all([parseResponse(pulseApi.projects.$get()).then(r=>r.projects),parseResponse(sourcemapsGET({param:{project:params.project}})).catch(()=>null)]);return {project:projects.find(p=>p.id===params.project)??null,sourcemaps};},empty);
}
