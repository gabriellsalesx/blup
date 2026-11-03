import {type InferResponseType} from "hono/client";
import {resolve} from "$app/paths";
import {t} from "#lib/i18n.svelte.ts";
import {pulseApi,parseResponse,type Issue} from "./api";
// What the issue list and an issue's own page share: the issue with its
// days, its occurrences, the colors of its state and level, and where its
// page is.
export const issueGET=pulseApi.projects[":project"].issues[":issue"].$get;
export type IssueDetail=InferResponseType<typeof issueGET,200>;
export const eventsGET=pulseApi.projects[":project"].issues[":issue"].events.$get;
export type IssueEvent=InferResponseType<typeof eventsGET,200>["events"][number];
export type IssueState=Issue["state"];
export const issuePage=(project:string,issue:string)=>resolve("/issues/[project]/[issue]",{project,issue});
export const issuesOf=(project:string)=>resolve(`/issues?${new URLSearchParams({project})}`);
export async function loadIssue(project:string,issue:string){
 const [detail,events]=await Promise.all([parseResponse(issueGET({param:{project,issue}})),parseResponse(eventsGET({param:{project,issue}})).then(r=>r.events,()=>[] as IssueEvent[])]);
 return {...detail,events};
}
// The side of a card and the badge of a state: red while open, green once
// resolved, gray when ignored.
// (Each needs its dark variant too: the card's dark border would cover it.)
export const stateEdge:Record<IssueState,string>={open:"border-l-red-500 dark:border-l-red-500",resolved:"border-l-emerald-500 dark:border-l-emerald-500",ignored:"border-l-neutral-400 dark:border-l-neutral-600"};
export const stateTone:Record<IssueState,"red"|"green"|"neutral">={open:"red",resolved:"green",ignored:"neutral"};
export const stateName=(s:IssueState)=>s==="open"?t("pulseOpen"):s==="resolved"?t("pulseResolved"):t("pulseIgnored");
// The severity the SDK sent (fatal, error, warning, info, debug).
export const levelTone=(level:string):"red"|"amber"|"blue"|"neutral"=>level==="fatal"||level==="error"?"red":level==="warning"?"amber":level==="info"?"blue":"neutral";
export const levelName=(level:string)=>level==="fatal"?t("pulseLevelFatal"):level==="error"?t("pulseLevelError"):level==="warning"?t("pulseLevelWarning"):level==="info"?t("pulseLevelInfo"):t("pulseLevelDebug");
// The environments and releases of a project's issues, for the filters.
export const noFacets={environments:[] as string[],releases:[] as string[]};
// The counters of a project's issues: by state, the total and how many
// times they happened in the last `days` (for an environment and a release
// when chosen).
const countsGET=pulseApi.projects[":project"]["issue-counts"].$get;
export type IssueCounts=InferResponseType<typeof countsGET,200>;
export const noCounts:IssueCounts={open:0,resolved:0,ignored:0,total:0,occurrences:0,days:30};
export const issueCounts=async(project:string,filter:{environment?:string;release?:string}={})=>project?parseResponse(countsGET({param:{project},query:filter})).catch(()=>noCounts):noCounts;
export const issueFacets=async(project:string)=>project?parseResponse(pulseApi.projects[":project"]["issue-facets"].$get({param:{project}})).catch(()=>noFacets):noFacets;
