import {loadMaintenance} from "../../../../../lib/incident-pages";
// The page and the maintenance to change.
export async function load({parent,params}){const data=await loadMaintenance((await parent()).signedIn,params.id);return {...data,current:data.maintenance.find(m=>m.id===params.maintenance)??null};}
