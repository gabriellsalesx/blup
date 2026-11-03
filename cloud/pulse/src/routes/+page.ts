import {pulseApi,parseResponse,type Project} from "../lib/api";
import {attempt} from "../lib/load";
export async function load({parent}){
 if(!(await parent()).signedIn)return {projects:[] as Project[]};
 return {projects:await attempt(async()=>(await parseResponse(pulseApi.projects.$get())).projects,[] as Project[])};
}
