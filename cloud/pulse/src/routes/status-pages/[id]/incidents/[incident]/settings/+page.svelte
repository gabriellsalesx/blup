<script lang="ts">
 import type {PageProps} from "./$types";
 import {goto} from "$app/navigation";
 import {untrack} from "svelte";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Button from "#lib/ui/Button.svelte";
 import {pulseApi,parseResponse} from "../../../../../../lib/api";
 import {incidentPage,incidentsTab} from "../../../../../../lib/incident-pages";
 // An incident's settings: its title, and removing it for good (with all
 // its updates, also from the public page).
 let {data}:PageProps=$props();
 let busy=$state(false),title=$state(untrack(()=>data.incident?.title??""));
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
 async function save(e:SubmitEvent){e.preventDefault();const page=data.page,incident=data.incident;if(!page||!incident)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].incidents[":incident"].$patch({param:{page:page.id,incident:incident.id},json:{title:title.trim()}}));toast(()=>t("pulseToastSaved"));await goto(incidentPage(page.id,incident.id));});}
 async function remove(){const page=data.page,incident=data.incident;if(!page||!incident||!await confirm({title:t("pulseRemoveIncidentTitle"),message:t("pulseRemoveIncident"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].incidents[":incident"].$delete({param:{page:page.id,incident:incident.id}}));toast(()=>t("pulseToastDeleted"));await goto(incidentsTab(page.id));});}
</script>

{#if !data.page||!data.incident}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseIncidentNotFound")}</p>
{:else}
 <div class="mb-4"><BackLink href={incidentPage(data.page.id,data.incident.id)} label={data.incident.title}/></div>
 <PageHeader title={t("pulseConfigure")} lead={t("pulseIncidentSettingsLead")}/>
 <form class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-10" onsubmit={save}>
  <FormSection title={t("pulseBasicInfo")} help={t("pulseIncidentTitleHelp")}>
   <FloatField id="incident-title" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentTitle")} required maxlength={120} bind:value={title}/>
  </FormSection>
  <FormSection title={t("pulseRemoveIncidentTitle")} help={t("pulseDestructive")}>
   <button type="button" class="inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2.5 font-semibold text-white transition-colors hover:bg-red-700 disabled:opacity-50" disabled={busy} onclick={remove}><Trash2 class="size-4"/>{t("pulseRemoveThisIncident")}</button>
  </FormSection>
  <div class="flex justify-end"><Button {busy} disabled={!title.trim()}>{t("pulseSaveChanges")}</Button></div>
 </form>
{/if}
