<script module lang="ts">
 import type {IncidentState,ServiceStatus} from "./status-pages";
 export type IncidentValues={title:string;body:string;state:IncidentState;publishedAt:number;services:{monitorId:string;status:ServiceStatus}[]};
</script>

<script lang="ts">
 import {untrack} from "svelte";
 import Circle from "@lucide/svelte/icons/circle";
 import CircleMinus from "@lucide/svelte/icons/circle-minus";
 import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import Select from "#lib/ui/Select.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import DateTimeField from "#lib/ui/DateTimeField.svelte";
 import type {Incident} from "./status-pages";
 // A new incident, or an update of an open one: what is going on, its
 // state, since when (new ones only, up to `keepDays` back) and how it
 // leaves each service of the page. An update starts from how the
 // incident left them.
 let {components,incident=null,keepDays,busy=false,onsave}:{components:{monitorId:string;name:string}[];incident?:Incident|null;keepDays:number;busy?:boolean;onsave:(values:IncidentValues)=>void}=$props();
 const start=untrack(()=>incident);
 let title=$state(start?.title??""),body=$state(""),situation=$state<IncidentState>(start?.state??"investigating"),publishedAt=$state(Date.now());
 let statuses=$state<Record<string,ServiceStatus>>(Object.fromEntries(untrack(()=>components).map(c=>[c.monitorId,start?.services.find(s=>s.monitorId===c.monitorId)?.status??"none"])));
 const now=Date.now(),oldest=now-untrack(()=>keepDays)*86_400_000;
 const states=$derived([{value:"investigating",label:t("pulseInvestigating")},{value:"identified",label:t("pulseIdentified")},{value:"monitoring",label:t("pulseMonitoring")},{value:"resolved",label:t("pulseResolved")}] as const);
 const serviceOptions=$derived([
  {value:"none",label:t("pulseNotAffected"),icon:{Icon:Circle,class:"text-neutral-400"}},
  {value:"downtime",label:t("pulseDown"),icon:{Icon:CircleMinus,class:"text-red-500"}},
  {value:"degraded",label:t("pulseDegraded"),icon:{Icon:TriangleAlert,class:"text-amber-500"}},
  {value:"resolved",label:t("pulseResolved"),icon:{Icon:CircleCheck,class:"text-emerald-500"}},
 ] as const);
 function submit(e:SubmitEvent){e.preventDefault();onsave({title:title.trim(),body:body.trim(),state:situation,publishedAt,services:components.map(c=>({monitorId:c.monitorId,status:statuses[c.monitorId]??"none"}))});}
</script>

<form class="grid grid-cols-[minmax(0,1fr)] gap-10" onsubmit={submit}>
 <FormSection title={t("pulseBasicInfo")} help={t("pulseIncidentBasicHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6">
   {#if incident}<p class="font-semibold">{incident.title}</p>{:else}<FloatField id="incident-title" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentTitle")} required maxlength={120} bind:value={title}/>{/if}
   <TextArea id="incident-text" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentText")} required maxlength={2000} bind:value={body}/>
   <div class="grid grid-cols-[minmax(0,1fr)] gap-6 sm:grid-cols-2">
    <Select surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentState")} options={states} bind:value={situation}/>
    {#if !incident}<DateTimeField surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentPublishedAt")} min={oldest} max={now} bind:value={publishedAt}/>{/if}
   </div>
  </div>
 </FormSection>
 <FormSection title={t("pulseAffectedServices")} help={t("pulseAffectedHelp")}>
  <div class="-my-1 grid grid-cols-[minmax(0,1fr)]">
   {#each components as c(c.monitorId)}<div class="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200 py-3 last:border-b-0 dark:border-neutral-800"><span class="min-w-0 truncate font-medium">{c.name}</span><Select compact class="w-48" surface="bg-white dark:bg-neutral-900" label={c.name} options={serviceOptions} bind:value={statuses[c.monitorId]}/></div>
   {:else}<p class="py-4 text-center text-sm text-neutral-500">{t("pulseNoServices")}</p>{/each}
  </div>
 </FormSection>
 <div class="flex justify-end"><Button {busy}>{incident?t("pulseUpdateIncident"):t("pulseCreateUpdate")}</Button></div>
</form>
