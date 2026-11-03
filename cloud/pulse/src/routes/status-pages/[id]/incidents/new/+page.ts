import {loadIncidents} from "../../../../../lib/incident-pages";
export async function load({parent,params}){return loadIncidents((await parent()).signedIn,params.id);}
