import {error} from "@sveltejs/kit";
import {pulseApi,parseResponse,type Project} from "../../../../../lib/api";
import {attempt} from "../../../../../lib/load";
import {loadChannels,projectChannels,type Channel} from "../../../../../lib/alerts";
// One destination of a project, to configure: only one the project sends to.
export async function load({parent,params}){
 if(!/^[1-9][0-9]{0,15}$/.test(params.project))error(404);
 const empty={project:null as Project|null,channel:null as Channel|null};
 if(!(await parent()).signedIn)return empty;
 const found=await attempt(async()=>{const [projects,channels]=await Promise.all([parseResponse(pulseApi.projects.$get()),loadChannels()]);const project=projects.projects.find(p=>p.id===params.project)??null;return {project,channel:project?projectChannels(channels,project.id).find(c=>c.id===params.channel)??null:null,loaded:true};},{...empty,loaded:false});
 if(found.loaded&&!found.channel)error(404);
 return {project:found.project,channel:found.channel};
}
