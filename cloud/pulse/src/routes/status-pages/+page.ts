import {attempt} from "../../lib/load";
import {getPages,type StatusPage} from "../../lib/status-pages";
export async function load({parent}){
 if(!(await parent()).signedIn)return {pages:[] as StatusPage[]};
 return {pages:await attempt(getPages,[] as StatusPage[])};
}
