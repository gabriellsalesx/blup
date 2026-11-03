<script lang="ts">
 import {onMount} from "svelte";
 import {goto} from "$app/navigation";
 import {type InferResponseType} from "hono/client";
 import Plus from "@lucide/svelte/icons/plus";
 import Pause from "@lucide/svelte/icons/pause";
 import Play from "@lucide/svelte/icons/play";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import ChevronDown from "@lucide/svelte/icons/chevron-down";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import UptimeBars from "#lib/ui/UptimeBars.svelte";
 import {pulseApi,parseResponse} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
 const checksGET=pulseApi.monitors[":monitor"].checks.$get;
 type History=InferResponseType<typeof checksGET,200>;
 // Each monitor is one row: its state as a colored edge, name and address,
 // the last days as bars and how long it has been like this. A click opens
 // its latest checks and actions.
 let monitors=$state<Monitor[]>([]),histories=$state<Record<string,History>>({}),open=$state<string|null>(null),busy=$state(false);
 const writable=$derived(panel.usage?.readOnly===false);
 const DAY=86_400_000;
 const day=new Intl.DateTimeFormat(intlLocale(),{dateStyle:"medium",timeZone:"UTC"});
 const ago=(ms:number)=>{const rtf=new Intl.RelativeTimeFormat(intlLocale(),{numeric:"always"}),m=Math.round((Date.now()-ms)/60000);return m<60?rtf.format(-Math.max(1,m),"minute"):m<1440?rtf.format(-Math.round(m/60),"hour"):rtf.format(-Math.round(m/1440),"day");};
 const stateOf=(m:Monitor)=>!m.active?"paused":m.state==="up"?"up":m.state==="down"?"down":"unknown";
 const edge={up:"border-l-emerald-500",down:"border-l-red-500",unknown:"border-l-neutral-300 dark:border-l-neutral-700",paused:"border-l-neutral-300 dark:border-l-neutral-700"};
 const color={up:"text-emerald-600 dark:text-emerald-400",down:"text-red-600 dark:text-red-400",unknown:"text-neutral-500",paused:"text-neutral-500"};
 const stateName=(s:string)=>s==="paused"?t("pulsePaused"):t(s==="up"?"pulseOperational":s==="down"?"pulseDown":"pulseUnknown");
 // When the state last changed, from the detailed checks (newest first,
 // kept a few days).
 function since(m:Monitor){const checks=histories[m.id]?.checks??[],current=m.state;if(!checks.length||!m.active||(current!=="up"&&current!=="down"))return "";let first=checks[0];for(const c of checks){if(c.outcome!==current)return t("pulseChangedAt",{time:ago(first.checkedAt)});first=c;}return t("pulseStableFor",{days:7});}
 function bars(m:Monitor,count:number){const h=histories[m.id],byDay=new Map((h?.days??[]).map(d=>[d.day,d])),today=Math.floor(Date.now()/DAY);return Array.from({length:count},(_,i)=>{const key=today-count+1+i,d=byDay.get(key);return {key,label:day.format(new Date(key*DAY)),up:d?.up??0,down:d?.down??0};});}
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function refresh(){monitors=(await parseResponse(pulseApi.monitors.$get())).monitors;const loaded=await Promise.all(monitors.map(async m=>[m.id,await parseResponse(checksGET({param:{monitor:m.id}}))] as const));histories=Object.fromEntries(loaded);}
 async function toggle(m:Monitor){await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$patch({param:{monitor:m.id},json:{active:!m.active}}));await refresh();toast(()=>m.active?t("pulseToastPaused"):t("pulseToastResumed"));});}
 async function remove(m:Monitor){if(!await confirm({title:t("delete"),message:t("pulseRemoveMonitor"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$delete({param:{monitor:m.id}}));open=null;await refresh();toast(()=>t("pulseToastDeleted"));});}
 onMount(()=>{void run(refresh);});
</script>

<PageHeader title={t("pulseMonitors")} lead={t("pulseCheckLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto("/monitores/novo")}><Plus class="size-4"/>{t("pulseAddMonitor")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8 grid gap-3">
 {#each monitors as monitor(monitor.id)}
  {@const s=stateOf(monitor)}
  <div class="overflow-hidden rounded-xl border border-l-4 border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900 {edge[s]}">
   <button class="grid w-full items-center gap-x-6 gap-y-3 px-5 py-4 text-left sm:grid-cols-[minmax(0,1fr)_minmax(0,22rem)_10rem]" aria-expanded={open===monitor.id} onclick={()=>open=open===monitor.id?null:monitor.id}>
    <span class="min-w-0"><span class="block truncate font-semibold">{monitor.name}</span><span class="block truncate text-sm text-neutral-500 underline">{monitor.url}</span></span>
    <span class="hidden sm:block"><UptimeBars days={bars(monitor,30)} label={t("pulseHistory")}/></span>
    <span class="sm:hidden"><UptimeBars days={bars(monitor,14)} label={t("pulseHistory")}/></span>
    <span class="flex items-center justify-between gap-2 sm:justify-end sm:text-right"><span><span class="block text-sm font-semibold {color[s]}">{stateName(s)}</span>{#if since(monitor)}<span class="block text-xs text-neutral-500">{since(monitor)}</span>{/if}</span><ChevronDown class="size-4 shrink-0 text-neutral-400 transition-transform {open===monitor.id?'rotate-180':''}"/></span>
   </button>
   {#if open===monitor.id}
    {@const history=histories[monitor.id]}
    <div class="border-t border-neutral-200 px-5 py-4 dark:border-neutral-800">
     <div class="flex flex-wrap items-center justify-between gap-3"><p class="text-sm text-neutral-500">{t("pulseEvery",{minutes:monitor.intervalMinutes})} · {monitor.lastAt?t("pulseLastCheck",{date:formatDate(monitor.lastAt)}):t("pulseNeverChecked")}</p><div class="flex flex-wrap gap-2">{#if monitor.writable}<Button small variant="secondary" {busy} onclick={()=>toggle(monitor)}>{#if monitor.active}<Pause class="size-4"/>{t("pulsePause")}{:else}<Play class="size-4"/>{t("pulseResume")}{/if}</Button>{/if}<Button small variant="secondary" {busy} onclick={()=>remove(monitor)}><Trash2 class="size-4"/>{t("delete")}</Button></div></div>
     {#if history?.checks.length}<div class="mt-4 overflow-auto"><table class="w-full text-left text-sm"><thead class="text-neutral-500"><tr><th class="py-2 pr-4 font-medium">{t("pulseCheckedAt")}</th><th class="py-2 pr-4 font-medium">{t("pulseMonitors")}</th><th class="py-2 pr-4 font-medium">HTTP</th><th class="py-2 font-medium">{t("pulseDuration")}</th></tr></thead><tbody>{#each history.checks.slice(0,15) as check(check.id)}<tr class="border-t border-neutral-200 dark:border-neutral-800"><td class="py-2 pr-4">{formatDate(new Date(check.checkedAt))}</td><td class="py-2 pr-4">{stateName(check.outcome)}</td><td class="py-2 pr-4">{check.status??"-"}</td><td class="py-2">{check.durationMs.toLocaleString(intlLocale())} ms</td></tr>{/each}</tbody></table></div><p class="mt-3 text-xs text-neutral-500">{t("pulseDetailDays",{days:panel.usage?.limits?.retentionDays??30})}</p>{/if}
    </div>
   {/if}
  </div>
 {:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseNoMonitors")}</div>{/each}
</div>
