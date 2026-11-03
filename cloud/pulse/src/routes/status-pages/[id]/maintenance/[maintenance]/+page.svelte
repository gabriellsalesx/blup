<script lang="ts">
 import type {PageProps} from "./$types";
 import {goto} from "$app/navigation";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import MaintenanceForm,{type MaintenanceValues} from "../../../../../lib/MaintenanceForm.svelte";
 import {pulseApi,parseResponse} from "../../../../../lib/api";
 import {panel} from "../../../../../lib/panel.svelte";
 import {maintenanceTab} from "../../../../../lib/incident-pages";
 // One scheduled maintenance: its text, window and services, and removing
 // it (also from the public page). Once saved or removed, back to the list.
 let {data}:PageProps=$props();
 let busy=$state(false);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
 async function save(values:MaintenanceValues){const page=data.page,current=data.current;if(!page||!current)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].maintenance[":maintenance"].$put({param:{page:page.id,maintenance:current.id},json:{title:values.title,body:values.body,startsAt:new Date(values.startsAt).toISOString(),endsAt:new Date(values.endsAt).toISOString(),services:values.services}}));toast(()=>t("pulseToastSaved"));await goto(maintenanceTab(page.id));});}
 async function remove(){const page=data.page,current=data.current;if(!page||!current||!await confirm({title:t("pulseRemoveMaintenanceTitle"),message:t("pulseRemoveMaintenance"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].maintenance[":maintenance"].$delete({param:{page:page.id,maintenance:current.id}}));toast(()=>t("pulseToastDeleted"));await goto(maintenanceTab(page.id));});}
</script>

{#if !data.page||!data.current}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseMaintenanceNotFound")}</p>
{:else}
 <div class="mb-4"><BackLink href={maintenanceTab(data.page.id)} label={data.page.title}/></div>
 <PageHeader title={data.current.title} lead={t("pulseEditMaintenanceLead")}/>
 <div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-10">
  {#key data.current.id}<MaintenanceForm components={data.page.components} maintenance={data.current} keepDays={panel.usage?.limits?.retentionDays??30} {busy} saveLabel={t("pulseSaveChanges")} onsave={save}/>{/key}
  <FormSection title={t("pulseRemoveMaintenanceTitle")} help={t("pulseDestructive")}>
   <button type="button" class="inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2.5 font-semibold text-white transition-colors hover:bg-red-700 disabled:opacity-50" disabled={busy} onclick={remove}><Trash2 class="size-4"/>{t("pulseRemoveThisMaintenance")}</button>
  </FormSection>
 </div>
{/if}
