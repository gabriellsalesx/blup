<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import BackLink from "#lib/ui/BackLink.svelte";
 import DetailHeader from "#lib/ui/DetailHeader.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import Button from "#lib/ui/Button.svelte";
 import {panel} from "../../../../../lib/panel.svelte";
 import {incidentsTab} from "../../../../../lib/incident-pages";
 import {serviceIcon,serviceName} from "../../../../../lib/service-status";
 import type {IncidentState} from "../../../../../lib/status-pages";
 // One incident: how it leaves each service of the page and every update
 // posted, newest first. While it is open, a button on top posts another.
 let {data}:PageProps=$props();
 const writable=$derived(panel.usage?.readOnly===false&&!!data.page?.writable);
 const stateLabel=(s:IncidentState)=>s==="investigating"?t("pulseInvestigating"):s==="identified"?t("pulseIdentified"):s==="monitoring"?t("pulseMonitoring"):t("pulseResolved");
 // Every service of the page, with how this incident leaves it.
 const services=$derived((data.page?.components??[]).map(c=>({...c,status:data.incident?.services.find(s=>s.monitorId===c.monitorId)?.status??"none" as const})));
</script>

{#if !data.page||!data.incident}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseIncidentNotFound")}</p>
{:else}
 {@const incident=data.incident}
 {@const pageId=data.page.id}
 <BackLink href={incidentsTab(pageId)} label={data.page.title}/>
 <DetailHeader title={incident.title}>
  {#snippet badge()}<Badge tone={incident.state==="resolved"?"green":"amber"}>{stateLabel(incident.state)}</Badge>{/snippet}
  {#snippet actions()}{#if writable&&incident.state!=="resolved"}<Button small onclick={()=>goto(resolve("/status-pages/[id]/incidents/[incident]/update",{id:pageId,incident:incident.id}))}><Plus class="size-4"/>{t("pulseCreateUpdate")}</Button>{/if}{/snippet}
 </DetailHeader>
 <p class="mt-1 text-sm text-neutral-500">{t("pulseIncidentCreatedAt",{date:formatDate(new Date(incident.createdAt))})}</p>

 <div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-6">
  <Card title={t("pulseAffectedServices")}>
   <div class="mt-3 grid grid-cols-[minmax(0,1fr)]">
    {#each services as service(service.monitorId)}
     {@const mark=serviceIcon[service.status]}
     <div class="flex items-center justify-between gap-3 border-t border-neutral-200 py-3 first:border-t-0 dark:border-neutral-800"><span class="min-w-0 truncate font-medium">{service.name}</span><span class="flex shrink-0 items-center gap-2 text-sm"><mark.Icon class="size-[18px] {mark.class}"/>{serviceName(service.status)}</span></div>
    {:else}<p class="py-2 text-sm text-neutral-500">{t("pulseNoServices")}</p>{/each}
   </div>
  </Card>

  <Card title={t("pulseIncidentUpdates")}>
   <ol class="mt-4">
    {#each data.updates as update(update.id)}
     <li class="relative border-l-2 border-neutral-200 pb-6 pl-5 last:pb-0 dark:border-neutral-800">
      <span class="absolute top-1 -left-[7px] size-3 rounded-full border-2 border-white {update.state==="resolved"?"bg-emerald-500":"bg-amber-500"} dark:border-neutral-900" aria-hidden="true"></span>
      <p class="flex flex-wrap items-center gap-2"><Badge tone={update.state==="resolved"?"green":"amber"}>{stateLabel(update.state)}</Badge><span class="text-sm text-neutral-500">{formatDate(new Date(update.createdAt))}</span></p>
      <p class="mt-2 whitespace-pre-line">{update.body}</p>
     </li>
    {:else}<p class="text-sm text-neutral-500">{t("pulseNoUpdates")}</p>{/each}
   </ol>
  </Card>
 </div>
{/if}
