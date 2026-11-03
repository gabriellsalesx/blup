<script lang="ts">
 import {onMount} from "svelte";
 import {goto} from "$app/navigation";
 import {page} from "$app/state";
 import ArrowLeft from "@lucide/svelte/icons/arrow-left";
 import ChevronLeft from "@lucide/svelte/icons/chevron-left";
 import ChevronRight from "@lucide/svelte/icons/chevron-right";
 import Pause from "@lucide/svelte/icons/pause";
 import Play from "@lucide/svelte/icons/play";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import Bell from "@lucide/svelte/icons/bell";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import StatusDot from "#lib/ui/StatusDot.svelte";
 import UptimeBars from "#lib/ui/UptimeBars.svelte";
 import ResponseChart from "../../../lib/ResponseChart.svelte";
 import {pulseApi,parseResponse,type Project} from "../../../lib/api";
 import {panel} from "../../../lib/panel.svelte";
 import {checksGET,stateOf,stateName,stateColor,since,bars,reasonName,type Monitor,type History} from "../../../lib/monitor";
 // One monitor in full: its days as bars, the response time of the latest
 // checks, where its alerts go and every check, a page at a time.
 const PAGE=20;
 let monitor=$state<Monitor|null>(null),history=$state<History|undefined>(),project=$state<Project|null>(null),destinations=$state(0),missing=$state(false),busy=$state(false);
 let filter=$state<"all"|"down">("all"),pageIndex=$state(0);
 const id=$derived(page.params.id??"");
 const s=$derived(monitor?stateOf(monitor):"unknown");
 const days=$derived(Math.min(90,panel.usage?.limits?.retentionDays??30));
 const listed=$derived((history?.checks??[]).filter(c=>filter==="all"||c.outcome==="down"));
 const pages=$derived(Math.max(1,Math.ceil(listed.length/PAGE)));
 const shown=$derived(listed.slice(pageIndex*PAGE,pageIndex*PAGE+PAGE));
 const alerts=$derived(project?destinations===0?t("pulseAlertsNone",{project:project.name}):destinations===1?t("pulseAlertsOne",{project:project.name}):t("pulseAlertsMany",{project:project.name,count:String(destinations)}):"");
 const filters=$derived([{value:"all",label:t("pulseAllChecks")},{value:"down",label:t("pulseIncidents")}] as const);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function load(){
  const found=(await parseResponse(pulseApi.monitors.$get())).monitors.find(m=>m.id===id)??null;monitor=found;missing=!found;if(!found)return;
  const [checks,projects,channels]=await Promise.all([parseResponse(checksGET({param:{monitor:found.id}})),parseResponse(pulseApi.projects.$get()),parseResponse(pulseApi.channels.$get())]);
  history=checks;project=projects.projects.find(p=>p.id===found.projectId)??null;destinations=channels.bindings.filter(b=>b.projectId===found.projectId).length;
 }
 async function toggle(m:Monitor){await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$patch({param:{monitor:m.id},json:{active:!m.active}}));await load();toast(()=>m.active?t("pulseToastPaused"):t("pulseToastResumed"));});}
 async function remove(m:Monitor){if(!await confirm({title:t("delete"),message:t("pulseRemoveMonitor"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$delete({param:{monitor:m.id}}));toast(()=>t("pulseToastDeleted"));await goto("/monitores");});}
 onMount(()=>{void run(load);});
</script>

<a class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white" href="/monitores"><ArrowLeft class="size-4"/>{t("pulseBackToMonitors")}</a>

{#if missing}<p class="mt-8 rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseMonitorNotFound")}</p>
{:else if monitor}
 {@const m=monitor}
 {@const changed=since(m,history)}
 <header class="mt-4 flex flex-wrap items-start justify-between gap-4">
  <div class="min-w-0"><h1 class="truncate text-2xl font-bold">{m.name}</h1><p class="mt-1 truncate text-sm text-neutral-500">{m.url}</p></div>
  <div class="flex flex-wrap gap-2">{#if m.writable}<Button small variant="secondary" {busy} onclick={()=>toggle(m)}>{#if m.active}<Pause class="size-4"/>{t("pulsePause")}{:else}<Play class="size-4"/>{t("pulseResume")}{/if}</Button>{/if}<Button small variant="secondary" {busy} onclick={()=>remove(m)}><Trash2 class="size-4"/>{t("delete")}</Button></div>
 </header>

 <div class="mt-6 grid grid-cols-[minmax(0,1fr)] gap-6">
  <Card>
   <div class="flex flex-wrap items-baseline justify-between gap-2"><p class="text-lg font-bold {stateColor[s]}">{stateName(s)}{#if changed}<span class="ml-2 text-sm font-normal text-neutral-500">{changed}</span>{/if}</p><p class="text-sm text-neutral-500"><span class="hidden sm:inline">{t("pulseLastDays",{days:String(days)})}</span><span class="sm:hidden">{t("pulseLastDays",{days:String(Math.min(30,days))})}</span></p></div>
   <div class="mt-4 hidden sm:block"><UptimeBars days={bars(history,days)} label={t("pulseHistory")}/></div>
   <div class="mt-4 sm:hidden"><UptimeBars days={bars(history,Math.min(30,days))} label={t("pulseHistory")}/></div>
   <p class="mt-4 text-sm text-neutral-500">{t("pulseEvery",{minutes:m.intervalMinutes})} · {m.lastAt?t("pulseLastCheck",{date:formatDate(m.lastAt)}):t("pulseNeverChecked")}</p>
  </Card>

  <Card title={t("pulseDuration")}><ResponseChart checks={history?.checks??[]}/></Card>

  <Card title={t("pulseAlertsTitle")}>
   {#snippet action()}<Button small variant="secondary" onclick={()=>goto("/avisos")}><Bell class="size-4"/>{t("pulseEditAlerts")}</Button>{/snippet}
   <p class="mt-3 text-sm">{alerts}</p>
  </Card>

  <section class="rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
   <div class="flex flex-wrap items-center justify-between gap-3 p-4">
    <Segmented label={t("pulseHistory")} options={filters} bind:value={filter} onchange={()=>pageIndex=0}/>
    {#if pages>1}<div class="flex items-center gap-2 text-sm text-neutral-500"><span>{t("pulsePageOf",{page:String(pageIndex+1),pages:String(pages)})}</span><Button small variant="secondary" disabled={pageIndex===0} aria-label={t("pulsePrevious")} onclick={()=>pageIndex--}><ChevronLeft class="size-4"/></Button><Button small variant="secondary" disabled={pageIndex>=pages-1} aria-label={t("pulseNext")} onclick={()=>pageIndex++}><ChevronRight class="size-4"/></Button></div>{/if}
   </div>
   <div class="overflow-auto"><table class="w-full text-left text-sm">
    <thead class="whitespace-nowrap border-y border-neutral-200 text-neutral-500 dark:border-neutral-800"><tr><th class="px-4 py-2.5 font-medium">{t("pulseState")}</th><th class="px-4 py-2.5 font-medium">{t("pulseReason")}</th><th class="px-4 py-2.5 font-medium">{t("pulseDuration")}</th><th class="px-4 py-2.5 font-medium">{t("pulseTime")}</th></tr></thead>
    <tbody>
     {#each shown as check(check.id)}<tr class="border-b border-neutral-200 last:border-b-0 dark:border-neutral-800"><td class="px-4 py-2.5 whitespace-nowrap"><span class="inline-flex items-center gap-2"><StatusDot state={check.outcome==="up"?"up":check.outcome==="down"?"down":"unknown"}/>{stateName(check.outcome)}</span></td><td class="px-4 py-2.5 whitespace-nowrap">{reasonName(check)}</td><td class="px-4 py-2.5 tabular-nums">{check.durationMs.toLocaleString(intlLocale())} ms</td><td class="px-4 py-2.5 whitespace-nowrap">{formatDate(new Date(check.checkedAt))}</td></tr>
     {:else}<tr><td colspan="4" class="px-4 py-10 text-center text-neutral-500">{t("pulseNoChecks")}</td></tr>{/each}
    </tbody>
   </table></div>
  </section>
  <p class="text-xs text-neutral-500">{t("pulseDetailDays",{days:panel.usage?.limits?.retentionDays??30})}</p>
 </div>
{/if}
