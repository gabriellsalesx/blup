import {loadMaintenance} from "../../../../../lib/incident-pages";
export async function load({parent,params}){return loadMaintenance((await parent()).signedIn,params.id);}
