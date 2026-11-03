import {attempt} from "../../../lib/load";
import {getMonitors,type Monitor} from "../../../lib/status-pages";
export async function load({parent}){
 if(!(await parent()).signedIn)return {monitors:[] as Monitor[]};
 return {monitors:await attempt(getMonitors,[] as Monitor[])};
}
