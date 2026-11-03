import {pulseApi,parseResponse,type Project} from "../../lib/api";
import {attempt} from "../../lib/load";
import {loadChannels,noChannels} from "../../lib/alerts";
export async function load({parent}){
 const empty={projects:[] as Project[],channels:noChannels};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{const [projects,channels]=await Promise.all([parseResponse(pulseApi.projects.$get()),loadChannels()]);return {projects:projects.projects,channels};},empty);
}
