<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {onDestroy} from "svelte";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
 import FileText from "@lucide/svelte/icons/file-text";
 import CircleAlert from "@lucide/svelte/icons/circle-alert";
 import Layers from "@lucide/svelte/icons/layers";
 import Activity from "@lucide/svelte/icons/activity";
 import X from "@lucide/svelte/icons/x";
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {openCard,cardLink} from "#lib/card.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import Select from "#lib/ui/Select.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import {pulseApi,parseResponse,type Project,type Issue} from "../../lib/api";
 import {ago} from "../../lib/monitor";
 import {issuePage,stateEdge,levelTone,levelName,issueFacets,issueCounts} from "../../lib/issues";
 // The issues of one project at a time (chosen at the top, kept in the
 // address): counters of how many are open, resolved and ignored (those
 // cards choose which the list shows), the total and the occurrences, then
 // the list narrowed by words of the title, environment and release. Each
 // is a card with its state on the side, like a monitor's; a click opens
 // its own page.
 let {data}:PageProps=$props();
 let projects=$derived<Project[]>(data.projects),projectId=$derived(data.projectId),issues=$derived<Issue[]>(data.issues),busy=$state(false),filter=$state<"open"|"resolved"|"ignored">("open");
 const project=$derived(projects.find(p=>p.id===projectId)??null);
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 let search=$state(""),environment=$state(""),release=$state("");
 let facets=$derived(data.facets),counts=$derived(data.counts);
 // The counters follow the project, environment and release (not the
 // words searched); the state cards are also the list's tabs.
 async function recount(){const id=projectId,next=await issueCounts(id,{...(environment?{environment}:{}),...(release?{release}:{})});if(projectId===id)counts=next;}
 function filtersChanged(){void run(load);void recount();}
 const stateCards=$derived([{value:"open" as const,label:t("pulseCountOpen"),Icon:CircleAlert,count:counts.open,on:"border-red-500 bg-red-50/60 dark:border-red-500 dark:bg-red-950/30",mark:"text-red-600 dark:text-red-400"},{value:"resolved" as const,label:t("pulseCountResolved"),Icon:CircleCheck,count:counts.resolved,on:"border-emerald-500 bg-emerald-50/60 dark:border-emerald-500 dark:bg-emerald-950/30",mark:"text-emerald-600 dark:text-emerald-400"},{value:"ignored" as const,label:t("pulseCountIgnored"),Icon:EyeOff,count:counts.ignored,on:"border-neutral-500 bg-neutral-100 dark:border-neutral-500 dark:bg-neutral-800/60",mark:"text-neutral-500"}]);
 function pick(state:typeof filter){if(state===filter)return;filter=state;void run(load);}
 const environmentOptions=$derived([{value:"",label:t("pulseAllEnvironments")},...facets.environments.map(v=>({value:v,label:v}))]);
 const releaseOptions=$derived([{value:"",label:t("pulseAllReleases")},...facets.releases.map(v=>({value:v,label:v}))]);
 const filtered=$derived(!!(search.trim()||environment||release));
 // The list's query: state and the filters that are set.
 const query=()=>({state:filter,...(search.trim()?{q:search.trim()}:{}),...(environment?{environment}:{}),...(release?{release}:{})});
 async function load(){issues=projectId?(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:projectId},query:query()}))).issues:[];}
 // Typing waits a moment before asking again.
 let typing:ReturnType<typeof setTimeout>|undefined;
 function typed(){clearTimeout(typing);typing=setTimeout(()=>void run(load),300);}
 onDestroy(()=>clearTimeout(typing));
 function clearFilters(){search="";environment="";release="";filtersChanged();}

 function choose(id:string){projectId=id;environment="";release="";void issueFacets(id).then(next=>{if(projectId===id)facets=next;});void recount();void goto(resolve(`/issues?${new URLSearchParams({project:id})}`),{replace:true,shallow:true});void run(load);}
 async function more(){const last=issues.at(-1);if(!projectId||!last)return;await run(async()=>{const next=(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:projectId},query:{...query(),before:String(last.lastAt),beforeId:last.id}}))).issues;issues=[...issues,...next];});}
 async function change(issue:Issue,state:Issue["state"]){await run(async()=>{await parseResponse(pulseApi.projects[":project"].issues[":issue"].$patch({param:{project:projectId,issue:issue.id},json:{state}}));await load();void recount();toast(()=>t("pulseToastIssue"));});}
 const menu=(issue:Issue,href:string)=>[{label:t("pulseSeeDetails"),Icon:FileText,onselect:()=>void goto(href)},...(project?.writable?issue.state==="open"?[{label:t("pulseResolve"),Icon:CircleCheck,onselect:()=>void change(issue,"resolved")},{label:t("pulseIgnore"),Icon:EyeOff,onselect:()=>void change(issue,"ignored")}]:[{label:t("pulseReopen"),Icon:RotateCcw,onselect:()=>void change(issue,"open")}]:[])];
</script>

<PageHeader title={t("pulseIssues")}/>

<div class="mt-8 flex flex-wrap items-center justify-between gap-4">
 {#if projects.length}<div class="w-full sm:w-72"><Select label={t("pulseAllProjects")} options={projectOptions} value={projectId} onchange={choose}/></div>{/if}
</div>

{#if projectId}
 <div class="mt-6 grid grid-cols-6 gap-3 sm:grid-cols-5">
  <div class="contents" role="radiogroup" aria-label={t("pulseIssues")}>
   {#each stateCards as card(card.value)}
    <button type="button" role="radio" aria-checked={filter===card.value} class="col-span-2 flex flex-col items-start justify-start rounded-xl border px-3 py-3 text-left sm:px-4 transition-colors sm:col-span-1 {filter===card.value?card.on:"border-neutral-200 bg-white hover:border-neutral-300 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-700"}" onclick={()=>pick(card.value)}>
     <span class="flex min-w-0 items-center gap-1.5 text-xs break-words text-neutral-500 sm:text-sm"><card.Icon class="size-4 shrink-0 {card.mark}"/>{card.label}</span>
     <span class="mt-1 block text-2xl font-bold tabular-nums">{card.count.toLocaleString(intlLocale())}</span>
    </button>
   {/each}
  </div>
  <div class="col-span-3 rounded-xl border border-neutral-200 bg-white px-3 py-3 sm:px-4 sm:col-span-1 dark:border-neutral-800 dark:bg-neutral-900"><span class="flex min-w-0 items-center gap-1.5 text-xs break-words text-neutral-500 sm:text-sm"><Layers class="size-4 shrink-0"/>{t("pulseIssuesTotal")}</span><span class="mt-1 block text-2xl font-bold tabular-nums">{counts.total.toLocaleString(intlLocale())}</span></div>
  <div class="col-span-3 rounded-xl border border-neutral-200 bg-white px-3 py-3 sm:px-4 sm:col-span-1 dark:border-neutral-800 dark:bg-neutral-900"><span class="flex min-w-0 items-center gap-1.5 text-xs break-words text-neutral-500 sm:text-sm"><Activity class="size-4 shrink-0"/>{t("pulseOccurrences")}</span><span class="mt-1 block text-2xl font-bold tabular-nums">{counts.occurrences.toLocaleString(intlLocale())}</span><span class="block text-xs text-neutral-500">{t("pulseLastDays",{days:counts.days})}</span></div>
 </div>
 <div class="mt-4 grid grid-cols-[minmax(0,1fr)] gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,12rem)_minmax(0,12rem)_auto] sm:items-center">
  <FloatField id="issue-search" label={t("pulseSearchIssues")} maxlength={128} bind:value={search} oninput={typed}/>
  <Select label={t("pulseFieldEnvironment")} options={environmentOptions} bind:value={environment} onchange={filtersChanged}/>
  <Select label={t("pulseFieldRelease")} options={releaseOptions} bind:value={release} onchange={filtersChanged}/>
  {#if filtered}<Button small variant="secondary" onclick={clearFilters}><X class="size-4"/>{t("pulseClearFilters")}</Button>{/if}
 </div>
{/if}

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
