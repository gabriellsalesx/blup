<script lang="ts">
 import type {PageProps} from "./$types";
 import {goto} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import MaintenanceForm,{type MaintenanceValues} from "../../../../../lib/MaintenanceForm.svelte";
 import {pulseApi,parseResponse} from "../../../../../lib/api";
 import {panel} from "../../../../../lib/panel.svelte";
 import {maintenanceTab} from "../../../../../lib/incident-pages";
 // A new scheduled maintenance of a status page; once saved, back to its
 // maintenance.
 let {data}:PageProps=$props();
 let busy=$state(false);
 async function save(values:MaintenanceValues){const page=data.page;if(!page)return;busy=true;try{await parseResponse(pulseApi.pages[":page"].maintenance.$post({param:{page:page.id},json:{title:values.title,body:values.body,startsAt:new Date(values.startsAt).toISOString(),endsAt:new Date(values.endsAt).toISOString(),services:values.services}}));toast(()=>t("pulseToastMaintenance"));await goto(maintenanceTab(page.id));}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

{#if !data.page}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulsePageNotFound")}</p>
{:else}
 <div class="mb-4"><BackLink href={maintenanceTab(data.page.id)} label={data.page.title}/></div>
 <PageHeader title={t("pulseScheduleMaintenance")} lead={t("pulseScheduleMaintenanceLead")}/>
 <div class="mt-8">{#key data.page.id}<MaintenanceForm components={data.page.components} keepDays={panel.usage?.limits?.retentionDays??30} {busy} saveLabel={t("pulseScheduleMaintenance")} onsave={save}/>{/key}</div>
{/if}
