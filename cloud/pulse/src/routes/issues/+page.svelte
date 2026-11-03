<script lang="ts">
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import {pulseApi,parseResponse,type Project,type Issue} from "../../lib/api";
 // The issues of one project at a time (chosen at the top, kept in the
 // address), open, resolved or ignored; a click shows the latest events.
 let {data}=$props();
 let projects=$derived<Project[]>(data.projects),projectId=$derived(data.projectId),issues=$derived<Issue[]>(data.issues),busy=$state(false),filter=$state<"open"|"resolved"|"ignored">("open");
 let events=$state<Awaited<ReturnType<typeof getEvents>>>([]),openIssue=$state<string|null>(null);
 const project=$derived(projects.find(p=>p.id===projectId)??null);
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function load(){events=[];openIssue=null;issues=projectId?(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:projectId},query:{state:filter}}))).issues:[];}
 function choose(id:string){projectId=id;void goto(`${resolve("/issues")}?${new URLSearchParams({project:id})}`,{replace:true,shallow:true});void run(load);}
 async function more(){const last=issues.at(-1);if(!projectId||!last)return;await run(async()=>{const next=(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:projectId},query:{state:filter,before:String(last.lastAt),beforeId:last.id}}))).issues;issues=[...issues,...next];});}
 async function change(issue:Issue,state:Issue["state"]){await run(async()=>{await parseResponse(pulseApi.projects[":project"].issues[":issue"].$patch({param:{project:projectId,issue:issue.id},json:{state}}));await load();toast(()=>t("pulseToastIssue"));});}
 async function toggleEvents(issue:Issue){if(openIssue===issue.id){openIssue=null;events=[];return;}await run(async()=>{events=await getEvents(projectId,issue.id);openIssue=issue.id;});}
 async function getEvents(projectKey:string,issue:string){return (await parseResponse(pulseApi.projects[":project"].issues[":issue"].events.$get({param:{project:projectKey,issue}}))).events;}
</script>

<PageHeader title={t("pulseIssues")}/>

<div class="mt-8 flex flex-wrap items-center justify-between gap-4">
 {#if projects.length}<div class="w-full sm:w-72"><Select label={t("pulseAllProjects")} options={projectOptions} value={projectId} onchange={choose}/></div>{/if}
 <Segmented label={t("pulseIssues")} options={[{value:"open",label:t("pulseOpen")},{value:"resolved",label:t("pulseResolved")},{value:"ignored",label:t("pulseIgnored")}]} bind:value={filter} onchange={()=>void run(load)}/>
</div>

<div class="mt-6"><List empty={projects.length?t("pulseNoIssues"):t("pulseEmpty")}>
 {#each issues as issue(issue.id)}
  <ListItem>
   <button class="w-full text-left" onclick={()=>toggleEvents(issue)} aria-expanded={openIssue===issue.id}><span class="line-clamp-2 font-semibold break-words">{issue.title}</span><span class="mt-1 block text-sm text-neutral-500">{t("pulseCount",{count:issue.count.toLocaleString(intlLocale())})} · {formatDate(new Date(issue.lastAt))}</span></button>
   {#snippet actions()}{#if project?.writable}{#if issue.state==="open"}<Button small variant="secondary" onclick={()=>change(issue,"resolved")}>{t("pulseResolve")}</Button><Button small variant="secondary" onclick={()=>change(issue,"ignored")}>{t("pulseIgnore")}</Button>{:else}<Button small variant="secondary" onclick={()=>change(issue,"open")}>{t("pulseReopen")}</Button>{/if}{/if}{/snippet}
   {#snippet extra()}{#if openIssue===issue.id}<div class="mt-4 grid gap-2"><p class="text-sm font-semibold">{t("pulseEvents")}</p>{#each events as event(event.id)}<details class="rounded-lg border border-neutral-200 p-3 dark:border-neutral-800"><summary class="cursor-pointer text-sm">{formatDate(new Date(event.receivedAt))}</summary><pre class="mt-3 overflow-auto whitespace-pre-wrap break-words text-xs">{JSON.stringify(event.event,null,2)}</pre></details>{/each}</div>{/if}{/snippet}
  </ListItem>
 {/each}
</List></div>
<p class="mt-10 text-xs text-neutral-500">{t("pulsePrivacy")}</p>
{#if issues.length>=30}<div class="mt-4"><Button variant="secondary" {busy} onclick={more}>{t("next")}</Button></div>{/if}
