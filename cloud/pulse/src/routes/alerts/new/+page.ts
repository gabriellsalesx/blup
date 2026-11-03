import {pulseApi,parseResponse,type Project} from "../../../lib/api";
import {attempt} from "../../../lib/load";
// The projects a new destination can be made for; `?project=` picks one.
export async function load({parent,url}){
 const raw=url.searchParams.get("project"),wanted=raw&&/^[1-9][0-9]{0,15}$/.test(raw)?raw:null;
 if(!(await parent()).signedIn)return {projects:[] as Project[],wanted};
 return {projects:await attempt(async()=>(await parseResponse(pulseApi.projects.$get())).projects.filter(p=>p.writable),[] as Project[]),wanted};
}
