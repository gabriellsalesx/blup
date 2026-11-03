<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto,refreshAll} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import Settings from "@lucide/svelte/icons/settings";
 import ArrowRight from "@lucide/svelte/icons/arrow-right";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import BackLink from "#lib/ui/BackLink.svelte";
 import DetailHeader from "#lib/ui/DetailHeader.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import Button from "#lib/ui/Button.svelte";
 import Modal from "#lib/ui/Modal.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {pulseApi,parseResponse} from "../../../../../lib/api";
 import {panel} from "../../../../../lib/panel.svelte";
 import {incidentsTab} from "../../../../../lib/incident-pages";
 import {serviceIcon,serviceName} from "../../../../../lib/service-status";
 import type {IncidentState} from "../../../../../lib/status-pages";
 // One incident: how it leaves each service of the page and every update
 // posted, newest first. While it is open, a button on top posts another,
 // and another resolves it: a last update, "resolved", that marks every
 // affected service resolved.
 let {data}:PageProps=$props();
 const writable=$derived(panel.usage?.readOnly===false&&!!data.page?.writable);
 const stateLabel=(s:IncidentState)=>s==="investigating"?t("pulseInvestigating"):s==="identified"?t("pulseIdentified"):s==="monitoring"?t("pulseMonitoring"):t("pulseResolved");
 // Every service of the page, with how this incident leaves it.
 const services=$derived((data.page?.components??[]).map(c=>({...c,status:data.incident?.services.find(s=>s.monitorId===c.monitorId)?.status??"none" as const})));
 const affected=$derived(services.filter(s=>s.status==="downtime"||s.status==="degraded"));
 let resolving=$state(false),message=$state(""),busy=$state(false);
 async function resolveIncident(e:SubmitEvent){e.preventDefault();const page=data.page,incident=data.incident;if(!page||!incident)return;busy=true;try{await parseResponse(pulseApi.pages[":page"].incidents[":incident"].updates.$post({param:{page:page.id,incident:incident.id},json:{body:message.trim(),state:"resolved",services:services.map(s=>({monitorId:s.monitorId,status:s.status==="downtime"||s.status==="degraded"?"resolved":s.status}))}}));resolving=false;message="";toast(()=>t("pulseToastResolved"));await refreshAll();}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

{#if !data.page||!data.incident}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseIncidentNotFound")}</p>
{:else}
 {@const incident=data.incident}
 {@const pageId=data.page.id}
 <BackLink href={incidentsTab(pageId)} label={data.page.title}/>
 <DetailHeader title={incident.title}>
  {#snippet badge()}<Badge tone={incident.state==="resolved"?"green":"amber"}>{stateLabel(incident.state)}</Badge>{/snippet}
  {#snippet actions()}{#if writable&&incident.state!=="resolved"}<Button small variant="secondary" onclick={()=>resolving=true}>{t("pulseResolveIncident")}</Button><Button small onclick={()=>goto(resolve("/status-pages/[id]/incidents/[incident]/update",{id:pageId,incident:incident.id}))}><Plus class="size-4"/>{t("pulseCreateUpdate")}</Button>{/if}{/snippet}
 </DetailHeader>
 <p class="mt-1 text-sm text-neutral-500">{t("pulseIncidentCreatedAt",{date:formatDate(new Date(incident.createdAt))})}</p>
 {#if writable}<a href={resolve("/status-pages/[id]/incidents/[incident]/settings",{id:pageId,incident:incident.id})} class="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-neutral-600 transition-colors hover:text-blup dark:text-neutral-300"><Settings class="size-4"/>{t("pulseConfigure")}</a>{/if}

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
 <Modal bind:open={resolving} title={t("pulseResolveIncident")}>
  <form class="grid gap-5" onsubmit={resolveIncident}>
   {#if affected.length}<div><p class="text-sm font-semibold">{t("pulseAffectedServices")}</p><div class="mt-2 grid rounded-lg border border-neutral-200 dark:border-neutral-800">{#each affected as service(service.monitorId)}{@const from=serviceIcon[service.status]}{@const to=serviceIcon.resolved}<div class="flex flex-wrap items-center justify-between gap-2 border-t border-neutral-200 px-3 py-2.5 first:border-t-0 dark:border-neutral-800"><span class="min-w-0 truncate font-medium">{service.name}</span><span class="flex items-center gap-1.5 text-sm"><from.Icon class="size-4 {from.class}"/><span class="text-neutral-500">{serviceName(service.status)}</span><ArrowRight class="size-3.5 text-neutral-400"/><to.Icon class="size-4 {to.class}"/>{serviceName("resolved")}</span></div>{/each}</div></div>{/if}
   <TextArea id="resolve-message" surface="bg-white dark:bg-neutral-900" label={t("pulseResolveMessage")} placeholder={t("pulseResolveExample")} required maxlength={2000} bind:value={message}/>
   <div class="flex flex-wrap items-center justify-between gap-3"><a href={resolve("/status-pages/[id]/incidents/[incident]/update",{id:pageId,incident:incident.id})} class="text-sm text-neutral-500 underline hover:text-blup">{t("pulseMoreOptions")}</a><div class="flex gap-2"><Button type="button" variant="secondary" onclick={()=>resolving=false}>{t("cancel")}</Button><Button {busy} disabled={!message.trim()}>{t("pulseResolveIncident")}</Button></div></div>
  </form>
 </Modal>
{/if}
