import {early} from "../../lib/load";
import {getPages,type StatusPage} from "../../lib/status-pages";
export async function load({parent}){
 return {pages:await early(getPages,parent,[] as StatusPage[])};
}
