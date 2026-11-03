import {loadIncidents} from "../../../../../lib/incident-pages";
// The page and the incident to update.
export async function load({parent,params}){const data=await loadIncidents((await parent()).signedIn,params.id);return {...data,incident:data.incidents.find(i=>i.id===params.incident)??null};}
