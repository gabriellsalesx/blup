<script lang="ts">
 import type {PageProps} from "./$types";
 import {goto} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import IncidentForm,{type IncidentValues} from "../../../../../lib/IncidentForm.svelte";
 import {pulseApi,parseResponse} from "../../../../../lib/api";
 import {panel} from "../../../../../lib/panel.svelte";
 import {incidentsTab} from "../../../../../lib/incident-pages";
 // An update of an open incident: its new text, state and services; once
 // posted, back to the page's incidents.
 let {data}:PageProps=$props();
 let busy=$state(false);
 const back=$derived(data.page?incidentsTab(data.page.id):"");
 async function save(values:IncidentValues){const page=data.page,incident=data.incident;if(!page||!incident)return;busy=true;try{await parseResponse(pulseApi.pages[":page"].incidents[":incident"].updates.$post({param:{page:page.id,incident:incident.id},json:{body:values.body,state:values.state,services:values.services}}));toast(()=>t("pulseToastIncident"));await goto(incidentsTab(page.id));}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

{#if !data.page||!data.incident}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseIncidentNotFound")}</p>
{:else}
 <div class="mb-4"><BackLink href={back} label={data.page.title}/></div>
 <PageHeader title={t("pulseUpdateIncidentTitle")} lead={t("pulseUpdateIncidentLead")}/>
 <div class="mt-8">{#key data.incident.id}<IncidentForm components={data.page.components} incident={data.incident} keepDays={panel.usage?.limits?.retentionDays??30} {busy} onsave={save}/>{/key}</div>
{/if}
