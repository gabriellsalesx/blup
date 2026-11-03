import {pulseApi,parseResponse,type Project} from "../../lib/api";
import {attempt} from "../../lib/load";
export async function load({parent}){
 const empty={projects:[] as Project[],channels:null as Awaited<ReturnType<typeof getChannels>>|null};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{const [projects,channels]=await Promise.all([parseResponse(pulseApi.projects.$get()),getChannels()]);return {projects:projects.projects,channels};},empty);
}
const getChannels=()=>parseResponse(pulseApi.channels.$get());
