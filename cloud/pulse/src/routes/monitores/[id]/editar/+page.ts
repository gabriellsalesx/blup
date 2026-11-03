import {pulseApi,parseResponse,type Project} from "../../../../lib/api";
import {attempt} from "../../../../lib/load";
import type {Monitor} from "../../../../lib/monitor";
// The monitor and the projects, to show its project's name.
export async function load({parent,params}){
 const empty={monitor:null as Monitor|null,projects:[] as Project[]};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{const [monitors,projects]=await Promise.all([parseResponse(pulseApi.monitors.$get()),parseResponse(pulseApi.projects.$get())]);return {monitor:monitors.monitors.find(m=>m.id===params.id)??null,projects:projects.projects};},empty);
}
