import {pulseApi,parseResponse} from "../../../../../lib/api";
import {loadIncidents} from "../../../../../lib/incident-pages";
import type {IncidentUpdate} from "../../../../../lib/status-pages";
// One incident: the page, the incident and all its updates.
export async function load({parent,params}){
 const [data,updates]=await Promise.all([loadIncidents((await parent()).signedIn,params.id),parseResponse(pulseApi.pages[":page"].incidents[":incident"].updates.$get({param:{page:params.id,incident:params.incident}})).then(r=>r.updates,()=>[] as IncidentUpdate[])]);
 return {...data,incident:data.incidents.find(i=>i.id===params.incident)??null,updates};
}
