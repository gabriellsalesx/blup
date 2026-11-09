import {pulseApi,parseResponse,type Project} from "../lib/api";
import {early} from "../lib/load";
export async function load({parent}){
 return {projects:await early(async()=>(await parseResponse(pulseApi.projects.$get())).projects,parent,[] as Project[])};
}
