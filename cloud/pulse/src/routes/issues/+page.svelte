<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
 import FileText from "@lucide/svelte/icons/file-text";
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {openCard,cardLink} from "#lib/card.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import {pulseApi,parseResponse,type Project,type Issue} from "../../lib/api";
 import {ago} from "../../lib/monitor";
 import {issuePage,stateEdge,levelTone,levelName} from "../../lib/issues";
 // The issues of one project at a time (chosen at the top, kept in the
 // address), open, resolved or ignored. Each is a card with its state on
 // the side, like a monitor's; a click opens its own page.
 let {data}:PageProps=$props();
 let projects=$derived<Project[]>(data.projects),projectId=$derived(data.projectId),issues=$derived<Issue[]>(data.issues),busy=$state(false),filter=$state<"open"|"resolved"|"ignored">("open");
 const project=$derived(projects.find(p=>p.id===projectId)??null);
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function load(){issues=projectId?(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:projectId},query:{state:filter}}))).issues:[];}
 function choose(id:string){projectId=id;void goto(resolve(`/issues?${new URLSearchParams({project:id})}`),{replace:true,shallow:true});void run(load);}
 async function more(){const last=issues.at(-1);if(!projectId||!last)return;await run(async()=>{const next=(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:projectId},query:{state:filter,before:String(last.lastAt),beforeId:last.id}}))).issues;issues=[...issues,...next];});}
 async function change(issue:Issue,state:Issue["state"]){await run(async()=>{await parseResponse(pulseApi.projects[":project"].issues[":issue"].$patch({param:{project:projectId,issue:issue.id},json:{state}}));await load();toast(()=>t("pulseToastIssue"));});}
 const menu=(issue:Issue,href:string)=>[{label:t("pulseSeeDetails"),Icon:FileText,onselect:()=>void goto(href)},...(project?.writable?issue.state==="open"?[{label:t("pulseResolve"),Icon:CircleCheck,onselect:()=>void change(issue,"resolved")},{label:t("pulseIgnore"),Icon:EyeOff,onselect:()=>void change(issue,"ignored")}]:[{label:t("pulseReopen"),Icon:RotateCcw,onselect:()=>void change(issue,"open")}]:[])];
</script>

<PageHeader title={t("pulseIssues")}/>

<div class="mt-8 flex flex-wrap items-center justify-between gap-4">
 {#if projects.length}<div class="w-full sm:w-72"><Select label={t("pulseAllProjects")} options={projectOptions} value={projectId} onchange={choose}/></div>{/if}
 <Segmented label={t("pulseIssues")} options={[{value:"open",label:t("pulseOpen")},{value:"resolved",label:t("pulseResolved")},{value:"ignored",label:t("pulseIgnored")}]} bind:value={filter} onchange={()=>void run(load)}/>
</div>

<div class="mt-6 grid gap-3">
 {#each issues as issue(issue.id)}
  {@const href=issuePage(projectId,issue.id)}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div onclick={(event)=>openCard(event,href)} onauxclick={(event)=>openCard(event,href)} class="relative flex cursor-pointer items-center gap-4 rounded-xl border border-l-4 border-neutral-200 bg-white px-5 py-4 transition-colors hover:bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:bg-neutral-800/60 {stateEdge[issue.state]}">
   <a class={cardLink} {href} aria-label={issue.title}></a>
   <div class="min-w-0 flex-1">
    <p class="flex items-center gap-2"><span class="line-clamp-2 font-semibold break-words">{issue.title}</span></p>
    <p class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-neutral-500"><Badge tone={levelTone(issue.level)}>{levelName(issue.level)}</Badge>{#if issue.environment}<span>{issue.environment}</span>{/if}{#if issue.release}<span>{issue.release}</span>{/if}<span>{t("pulseSeenAgo",{time:ago(issue.lastAt)})}</span></p>
   </div>
   <div class="hidden shrink-0 text-right sm:block"><p class="text-lg font-bold tabular-nums">{issue.count.toLocaleString(intlLocale())}</p><p class="text-xs text-neutral-500">{t("pulseOccurrences")}</p></div>
   <ActionMenu label={t("pulseActions")} items={menu(issue,href)}/>
  </div>
 {:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{projects.length?t("pulseNoIssues"):t("pulseEmpty")}</div>{/each}
</div>
<p class="mt-10 text-xs text-neutral-500">{t("pulsePrivacy")}</p>
{#if issues.length>=30}<div class="mt-4"><Button variant="secondary" {busy} onclick={more}>{t("next")}</Button></div>{/if}
