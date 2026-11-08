<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto,refreshAll} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import Pause from "@lucide/svelte/icons/pause";
 import Play from "@lucide/svelte/icons/play";
 import Pencil from "@lucide/svelte/icons/pencil";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {cardLink} from "#lib/card.ts";
 import {pulseApi,parseResponse} from "../../lib/api";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import UptimeBars from "#lib/ui/UptimeBars.svelte";
 import {panel} from "../../lib/panel.svelte";
 import UsageMeter from "#lib/ui/UsageMeter.svelte";
 import {stateOf,stateName,stateColor,since,bars,type Monitor} from "../../lib/monitor";
 // Each monitor is one row: its state as the color of its edge and address,
 // its name,
 // the last days as bars and how long it has been like this. A click opens
 // its own page with the details; the three dots pause, edit or delete it.
 let {data}:PageProps=$props();
 const monitors=$derived(data.monitors),histories=$derived(data.histories);
 const writable=$derived(panel.usage?.readOnly===false);
 async function act(fn:()=>Promise<void>){try{await fn();await refreshAll();}catch(e){toast(errorMessage(e),{kind:"error"});}}
 const toggle=(m:Monitor)=>act(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$patch({param:{monitor:m.id},json:{active:!m.active}}));toast(()=>m.active?t("pulseToastPaused"):t("pulseToastResumed"));});
 async function remove(m:Monitor){if(!await confirm({title:t("delete"),message:t("pulseRemoveMonitor"),danger:true}))return;await act(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$delete({param:{monitor:m.id}}));toast(()=>t("pulseToastDeleted"));});}
 const menu=(m:Monitor)=>[...(m.writable?[{label:m.active?t("pulsePause"):t("pulseResume"),Icon:m.active?Pause:Play,onselect:()=>void toggle(m)},{label:t("pulseEdit"),Icon:Pencil,onselect:()=>void goto(resolve("/monitors/[id]/edit",{id:m.id}))}]:[]),{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void remove(m)}];
 // Both themes named, so the card's own border color never covers the edge.
 const edge={up:"border-l-emerald-500 dark:border-l-emerald-500",down:"border-l-red-500 dark:border-l-red-500",unknown:"border-l-neutral-300 dark:border-l-neutral-700",paused:"border-l-neutral-300 dark:border-l-neutral-700"};
</script>

<PageHeader title={t("pulseMonitors")} lead={t("pulseCheckLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto(resolve("/monitors/new"))}><Plus class="size-4"/>{t("pulseAddMonitor")}</Button>{/if}{/snippet}
</PageHeader>

{#if panel.usage?.limits}<div class="mt-6 max-w-sm"><UsageMeter text={t("pulseMonitorsUsage",{used:monitors.length.toLocaleString(intlLocale()),limit:panel.usage.limits.monitors.toLocaleString(intlLocale())})} used={monitors.length} limit={panel.usage.limits.monitors}/></div>{/if}

<div class="mt-8 grid gap-3">
 {#each monitors as monitor(monitor.id)}
  {@const s=stateOf(monitor)}
  {@const changed=since(monitor,histories[monitor.id])}
  {@const href=resolve("/monitors/[id]",{id:monitor.id})}
  <div class="relative grid cursor-pointer items-center gap-x-6 gap-y-3 rounded-xl border border-l-4 border-neutral-200 bg-white px-5 py-4 transition-colors hover:bg-neutral-50 sm:grid-cols-[minmax(0,1fr)_minmax(0,22rem)_12rem] dark:border-neutral-800 dark:bg-neutral-900 dark:hover:bg-neutral-800/60 {edge[s]}">
   <a class={cardLink} {href} aria-label={monitor.name}></a>
   <span class="min-w-0"><span class="block truncate font-semibold">{monitor.name}</span><span class="block truncate text-sm underline {stateColor[s]}">{monitor.url}</span></span>
   <span class="relative z-10"><UptimeBars days={bars(histories[monitor.id],30)} label={t("pulseHistory")}/></span>
   <span class="flex items-center justify-between gap-2 sm:justify-end sm:text-right"><span><span class="block text-sm font-semibold {stateColor[s]}">{stateName(s)}</span>{#if changed}<span class="block text-xs text-neutral-500">{changed}</span>{/if}</span><ActionMenu label={t("pulseActions")} items={menu(monitor)}/></span>
  </div>
 {:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseNoMonitors")}</div>{/each}
</div>
