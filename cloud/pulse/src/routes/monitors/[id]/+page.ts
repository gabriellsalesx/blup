import {type Project} from "../../../lib/api";
import {attempt} from "../../../lib/load";
import {loadMonitor,type Monitor,type History} from "../../../lib/monitor";
export async function load({parent,params}){
 const empty={monitor:null as Monitor|null,history:undefined as History|undefined,project:null as Project|null,destinations:0};
 if(!(await parent()).signedIn)return empty;
 return attempt(()=>loadMonitor(params.id),empty);
}
