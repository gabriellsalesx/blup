<script lang="ts">
 import {onMount} from "svelte";
 import {goto} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import ChevronRight from "@lucide/svelte/icons/chevron-right";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import UptimeBars from "#lib/ui/UptimeBars.svelte";
 import {pulseApi,parseResponse} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 import {checksGET,stateOf,stateName,stateColor,since,bars,type Monitor,type History} from "../../lib/monitor";
 // Each monitor is one row: its state as a colored edge, name and address,
 // the last days as bars and how long it has been like this. A click opens
 // its own page with the details.
 let monitors=$state<Monitor[]>([]),histories=$state<Record<string,History>>({});
 const writable=$derived(panel.usage?.readOnly===false);
 const edge={up:"border-l-emerald-500",down:"border-l-red-500",unknown:"border-l-neutral-300 dark:border-l-neutral-700",paused:"border-l-neutral-300 dark:border-l-neutral-700"};
 async function refresh(){monitors=(await parseResponse(pulseApi.monitors.$get())).monitors;const loaded=await Promise.all(monitors.map(async m=>[m.id,await parseResponse(checksGET({param:{monitor:m.id}}))] as const));histories=Object.fromEntries(loaded);}
 onMount(()=>{refresh().catch(e=>toast(errorMessage(e),{kind:"error"}));});
</script>

<PageHeader title={t("pulseMonitors")} lead={t("pulseCheckLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto("/monitores/novo")}><Plus class="size-4"/>{t("pulseAddMonitor")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8 grid gap-3">
 {#each monitors as monitor(monitor.id)}
  {@const s=stateOf(monitor)}
  {@const changed=since(monitor,histories[monitor.id])}
  <a class="grid items-center gap-x-6 gap-y-3 rounded-xl border border-l-4 border-neutral-200 bg-white px-5 py-4 transition-colors hover:bg-neutral-50 sm:grid-cols-[minmax(0,1fr)_minmax(0,22rem)_10rem] dark:border-neutral-800 dark:bg-neutral-900 dark:hover:bg-neutral-800/60 {edge[s]}" href={`/monitores/${encodeURIComponent(monitor.id)}`}>
   <span class="min-w-0"><span class="block truncate font-semibold">{monitor.name}</span><span class="block truncate text-sm text-neutral-500 underline">{monitor.url}</span></span>
   <span class="hidden sm:block"><UptimeBars days={bars(histories[monitor.id],30)} label={t("pulseHistory")}/></span>
   <span class="sm:hidden"><UptimeBars days={bars(histories[monitor.id],14)} label={t("pulseHistory")}/></span>
   <span class="flex items-center justify-between gap-2 sm:justify-end sm:text-right"><span><span class="block text-sm font-semibold {stateColor[s]}">{stateName(s)}</span>{#if changed}<span class="block text-xs text-neutral-500">{changed}</span>{/if}</span><ChevronRight class="size-4 shrink-0 text-neutral-400"/></span>
  </a>
 {:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseNoMonitors")}</div>{/each}
</div>
