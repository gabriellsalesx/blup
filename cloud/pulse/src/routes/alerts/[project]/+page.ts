import {error} from "@sveltejs/kit";
import {pulseApi,parseResponse,type Project} from "../../../lib/api";
import {attempt} from "../../../lib/load";
import {loadChannels,noChannels} from "../../../lib/alerts";
// One project's alert destinations: the project and every destination of
// the account (those it sends to, and those it could).
export async function load({parent,params}){
 if(!/^[1-9][0-9]{0,15}$/.test(params.project))error(404);
 const empty={project:null as Project|null,channels:noChannels};
 if(!(await parent()).signedIn)return empty;
 const found=await attempt(async()=>{const [projects,channels]=await Promise.all([parseResponse(pulseApi.projects.$get()),loadChannels()]);return {project:projects.projects.find(p=>p.id===params.project)??null,channels,loaded:true};},{...empty,loaded:false});
 if(found.loaded&&!found.project)error(404);
 return {project:found.project,channels:found.channels};
}
