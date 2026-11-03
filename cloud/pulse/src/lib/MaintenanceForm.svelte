<script module lang="ts">
 export type MaintenanceValues={title:string;body:string;startsAt:number;endsAt:number;services:string[]};
</script>

<script lang="ts">
 import {untrack} from "svelte";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import DateTimeField from "#lib/ui/DateTimeField.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import type {Maintenance} from "./status-pages";
 // A scheduled maintenance: what it is about, its window (start and end,
 // at most 30 days, starting within a year) and the services of the page
 // in it. A new one starts at the next full hour, for two hours.
 const HOUR=3_600_000,DAY=86_400_000;
 let {components,maintenance=null,keepDays,busy=false,saveLabel,onsave}:{components:{monitorId:string;name:string}[];maintenance?:Maintenance|null;keepDays:number;busy?:boolean;saveLabel:string;onsave:(values:MaintenanceValues)=>void}=$props();
 const start=untrack(()=>maintenance),now=Date.now(),nextHour=Math.ceil(now/HOUR)*HOUR;
 let title=$state(start?.title??""),body=$state(start?.body??""),startsAt=$state(start?.startsAt??nextHour),endsAt=$state(start?.endsAt??nextHour+2*HOUR);
 let chosen=$state<string[]>(start?.services??(untrack(()=>components).length===1?[untrack(()=>components)[0].monitorId]:[]));
 const oldest=now-untrack(()=>keepDays)*DAY,latest=now+365*DAY;
 // The end follows the start when the start moves past it.
 function moved(){if(endsAt<=startsAt)endsAt=startsAt+2*HOUR;}
 const valid=$derived(title.trim().length>0&&chosen.length>0&&endsAt>startsAt&&endsAt-startsAt<=30*DAY);
 function toggle(id:string,on:boolean){chosen=on?[...chosen,id]:chosen.filter(c=>c!==id);}
 function submit(e:SubmitEvent){e.preventDefault();onsave({title:title.trim(),body:body.trim(),startsAt,endsAt,services:components.map(c=>c.monitorId).filter(id=>chosen.includes(id))});}
</script>

<form class="grid grid-cols-[minmax(0,1fr)] gap-10" onsubmit={submit}>
 <FormSection title={t("pulseBasicInfo")} help={t("pulseMaintenanceBasicHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6">
   <FloatField id="maintenance-title" surface="bg-white dark:bg-neutral-900" label={t("pulseMaintenanceTitle")} required maxlength={120} bind:value={title}/>
   <TextArea id="maintenance-body" surface="bg-white dark:bg-neutral-900" label={t("pulseMaintenanceBody")} maxlength={2000} bind:value={body}/>
  </div>
 </FormSection>
 <FormSection title={t("pulseMaintenanceWhen")} help={t("pulseMaintenanceWhenHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6 sm:grid-cols-2">
   <DateTimeField surface="bg-white dark:bg-neutral-900" label={t("pulseMaintenanceStarts")} min={oldest} max={latest} bind:value={()=>startsAt,v=>{startsAt=v;moved();}}/>
   <DateTimeField surface="bg-white dark:bg-neutral-900" label={t("pulseMaintenanceEnds")} min={startsAt+60_000} max={startsAt+30*DAY} bind:value={endsAt}/>
  </div>
 </FormSection>
 <FormSection title={t("pulseMaintenanceServices")} help={t("pulseMaintenanceServicesHelp")}>
  <div class="-my-1 grid grid-cols-[minmax(0,1fr)]">
   {#each components as c(c.monitorId)}<div class="border-b border-neutral-200 py-3 last:border-b-0 dark:border-neutral-800"><Switch tone="mint" label={c.name} checked={chosen.includes(c.monitorId)} onchange={on=>toggle(c.monitorId,on)}/></div>
   {:else}<p class="py-4 text-center text-sm text-neutral-500">{t("pulseNoServices")}</p>{/each}
  </div>
 </FormSection>
 <div class="flex justify-end"><Button {busy} disabled={!valid}>{saveLabel}</Button></div>
</form>
