import {pulseApi,parseResponse,type Project} from "../../../../lib/api";
import {attempt} from "../../../../lib/load";
import {loadIssue,type IssueDetail,type IssueEvent} from "../../../../lib/issues";
// One issue: its project, the issue with its days and its latest
// occurrences, together.
export async function load({parent,params}){
 const empty={project:null as Project|null,issue:null as IssueDetail["issue"]|null,days:[] as IssueDetail["days"],events:[] as IssueEvent[]};
 if(!(await parent()).signedIn)return empty;
 return attempt(async()=>{
  const [projects,found]=await Promise.all([parseResponse(pulseApi.projects.$get()).then(r=>r.projects),loadIssue(params.project,params.issue).catch(()=>null)]);
  return {project:projects.find(p=>p.id===params.project)??null,issue:found?.issue??null,days:found?.days??[],events:found?.events??[]};
 },empty);
}
