<script lang="ts">
 import {onMount} from "svelte";
 import Copy from "@lucide/svelte/icons/copy";
 import Plus from "@lucide/svelte/icons/plus";
 import RefreshCw from "@lucide/svelte/icons/refresh-cw";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import ChevronRight from "@lucide/svelte/icons/chevron-right";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {pulseApi,parseResponse,type Project,type Issue} from "../lib/api";
 import {panel} from "../lib/panel.svelte";
 type Platform="javascript"|"node"|"go"|"other";
 let projects=$state<Project[]>([]),selected=$state<Project|null>(null),issues=$state<Issue[]>([]),name=$state(""),platform=$state<Platform>("javascript"),dsn=$state(""),busy=$state(false),error=$state<unknown>(null),filter=$state<"open"|"resolved"|"ignored">("open");
 let events=$state<Awaited<ReturnType<typeof getEvents>>>([]),openIssue=$state<string|null>(null);
 const writable=$derived(panel.usage?.readOnly===false);
 const platforms=$derived([{value:"javascript",label:"JavaScript"},{value:"node",label:"Node.js"},{value:"go",label:"Go"},{value:"other",label:t("pulseOther")}] as const);
 const platformName=(p:string)=>platforms.find(o=>o.value===p)?.label??p;
 const usage=$derived(panel.usage?.limits?{used:panel.usage.accepted,limit:panel.usage.limits.events}:null);
 async function refresh(){projects=(await parseResponse(pulseApi.projects.$get())).projects;if(!selected&&projects[0])await select(projects[0]);}
 async function select(p:Project){selected=p;events=[];openIssue=null;issues=(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:p.id},query:{state:filter}}))).issues;}
 async function more(){const project=selected,last=issues.at(-1);if(!project||!last)return;await run(async()=>{const next=(await parseResponse(pulseApi.projects[":project"].issues.$get({param:{project:project.id},query:{state:filter,before:String(last.lastAt),beforeId:last.id}}))).issues;issues=[...issues,...next];});}
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function create(e:SubmitEvent){e.preventDefault();await run(async()=>{const result=await parseResponse(pulseApi.projects.$post({json:{name:name.trim(),platform}}));dsn=result.key.dsn;name="";await refresh();});}
 async function copy(){try{await navigator.clipboard.writeText(dsn);toast(()=>t("copied"));}catch{toast(()=>t("copyFailed"));}}
 async function rotate(p:Project){if(!await confirm({title:t("pulseRotate"),message:t("pulseRotateAsk")}))return;await run(async()=>{dsn=(await parseResponse(pulseApi.projects[":project"].keys.$post({param:{project:p.id}}))).dsn;});}
 async function remove(p:Project){if(!await confirm({title:t("delete"),message:t("pulseDeleteProject")}))return;await run(async()=>{await parseResponse(pulseApi.projects[":project"].$delete({param:{project:p.id}}));selected=null;dsn="";await refresh();});}
 async function change(issue:Issue,state:Issue["state"]){if(!selected)return;const project=selected;await run(async()=>{await parseResponse(pulseApi.projects[":project"].issues[":issue"].$patch({param:{project:project.id,issue:issue.id},json:{state}}));await select(project);});}
 async function toggleEvents(issue:Issue){const project=selected;if(!project)return;if(openIssue===issue.id){openIssue=null;events=[];return;}await run(async()=>{events=await getEvents(project.id,issue.id);openIssue=issue.id;});}
 async function getEvents(project:string,issue:string){return (await parseResponse(pulseApi.projects[":project"].issues[":issue"].events.$get({param:{project,issue}}))).events;}
 onMount(()=>{void run(refresh);});
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
 <div><h1 class="text-3xl font-extrabold tracking-tight">{t("pulseProjects")}</h1><p class="mt-2 text-neutral-500">{t("pulseScope")}</p></div>
 {#if usage}<div class="w-full max-w-xs"><p class="text-sm text-neutral-500">{t("pulseMonthUsage",{accepted:usage.used.toLocaleString(intlLocale()),limit:usage.limit.toLocaleString(intlLocale())})}</p><div class="mt-2 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full bg-blup" style:width="{Math.min(100,usage.used/Math.max(1,usage.limit)*100)}%"></div></div></div>{/if}
</div>

{#if error}<p role="alert" class="mt-6 text-red-600">{errorMessage(error)}</p>{/if}

{#if writable}<div class="mt-8"><Card title={t("pulseNewProject")}><form class="mt-5 grid gap-4 sm:grid-cols-[1fr_12rem_auto] sm:items-center" onsubmit={create}><FloatField id="project-name" surface="bg-white dark:bg-neutral-900" label={t("pulseProjectName")} required maxlength={80} bind:value={name}/><Select surface="bg-white dark:bg-neutral-900" label={t("pulsePlatform")} options={platforms} bind:value={platform}/><Button {busy} class="h-14"><Plus class="size-4"/>{t("pulseCreateProject")}</Button></form></Card></div>{/if}

{#if dsn}<div class="mt-4 rounded-xl border border-blup bg-white p-5 dark:bg-neutral-900"><p class="font-semibold">{t("pulseSDK")}</p><div class="mt-3 flex items-center gap-2 rounded-lg bg-neutral-100 p-3 dark:bg-neutral-800"><code class="min-w-0 flex-1 break-all text-sm">{dsn}</code><Button small variant="secondary" onclick={copy}><Copy class="size-4"/>{t("copy")}</Button></div><p class="mt-3 text-sm text-neutral-500">{t("pulseDSNOnce")}</p></div>{/if}

<div class="mt-6 overflow-hidden rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
 {#each projects as project(project.id)}
  <div class="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200 px-5 py-4 last:border-b-0 dark:border-neutral-800 {selected?.id===project.id?'bg-neutral-50 dark:bg-neutral-800/50':''}">
   <button class="flex min-w-0 flex-1 basis-full items-center gap-3 text-left sm:basis-0" onclick={()=>run(()=>select(project))}>
    <span class="size-2.5 shrink-0 rounded-full {project.active?'bg-emerald-500':'bg-neutral-400'}"></span>
    <span class="min-w-0"><span class="block truncate font-semibold">{project.name}</span><span class="block text-sm text-neutral-500">{platformName(project.platform)} · {project.active?t("pulseActive"):t("pulsePaused")}</span></span>
    <ChevronRight class="ml-auto size-4 shrink-0 text-neutral-400"/>
   </button>
   <div class="flex gap-2">{#if project.writable}<Button small variant="secondary" onclick={()=>rotate(project)}><RefreshCw class="size-4"/>{t("pulseRotate")}</Button>{/if}<Button small variant="secondary" onclick={()=>remove(project)}><Trash2 class="size-4"/>{t("delete")}</Button></div>
  </div>
 {:else}<p class="px-5 py-10 text-center text-neutral-500">{t("pulseEmpty")}</p>{/each}
</div>

{#if selected}<section class="mt-10">
 <div class="mb-4 flex flex-wrap items-center justify-between gap-3"><h2 class="text-xl font-bold">{t("pulseIssuesOf",{name:selected.name})}</h2><Segmented label={t("pulseIssues")} options={[{value:"open",label:t("pulseOpen")},{value:"resolved",label:t("pulseResolved")},{value:"ignored",label:t("pulseIgnored")}]} bind:value={filter} onchange={()=>{const project=selected;if(project)void run(()=>select(project));}}/></div>
 <div class="overflow-hidden rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
  {#each issues as issue(issue.id)}
   <div class="border-b border-neutral-200 px-5 py-4 last:border-b-0 dark:border-neutral-800">
    <div class="flex flex-wrap items-start justify-between gap-3">
     <button class="min-w-0 flex-1 basis-full text-left sm:basis-0" onclick={()=>toggleEvents(issue)} aria-expanded={openIssue===issue.id}><span class="line-clamp-2 font-semibold break-words">{issue.title}</span><span class="mt-1 block text-sm text-neutral-500">{t("pulseCount",{count:issue.count.toLocaleString(intlLocale())})} · {formatDate(new Date(issue.lastAt))}</span></button>
     {#if selected.writable}<div class="flex gap-2">{#if issue.state==="open"}<Button small variant="secondary" onclick={()=>change(issue,"resolved")}>{t("pulseResolve")}</Button><Button small variant="secondary" onclick={()=>change(issue,"ignored")}>{t("pulseIgnore")}</Button>{:else}<Button small variant="secondary" onclick={()=>change(issue,"open")}>{t("pulseReopen")}</Button>{/if}</div>{/if}
    </div>
    {#if openIssue===issue.id}<div class="mt-4 grid gap-2"><p class="text-sm font-semibold">{t("pulseEvents")}</p>{#each events as event(event.id)}<details class="rounded-lg border border-neutral-200 p-3 dark:border-neutral-800"><summary class="cursor-pointer text-sm">{formatDate(new Date(event.receivedAt))}</summary><pre class="mt-3 overflow-auto whitespace-pre-wrap break-words text-xs">{JSON.stringify(event.event,null,2)}</pre></details>{/each}</div>{/if}
   </div>
  {:else}<p class="px-5 py-10 text-center text-neutral-500">{t("pulseNoIssues")}</p>{/each}
 </div>
 {#if issues.length>=30}<div class="mt-4"><Button variant="secondary" {busy} onclick={more}>{t("next")}</Button></div>{/if}
</section>{/if}

<p class="mt-10 text-xs text-neutral-500">{t("pulsePrivacy")}</p>
