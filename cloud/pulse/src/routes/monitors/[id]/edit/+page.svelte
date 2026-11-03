<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import MonitorForm,{type MonitorValues} from "../../../../lib/MonitorForm.svelte";
 import {pulseApi,parseResponse} from "../../../../lib/api";
 // Changing a monitor's name, address, interval or how it is checked; it
 // is checked again from scratch after saving.
 let {data}:PageProps=$props();
 let busy=$state(false);
 const back=$derived(data.monitor?resolve("/monitors/[id]",{id:data.monitor.id}):resolve("/monitors"));
 async function save(values:MonitorValues){const monitor=data.monitor;if(!monitor)return;busy=true;try{await parseResponse(pulseApi.monitors[":monitor"].$patch({param:{monitor:monitor.id},json:{name:values.name,url:values.url,intervalMinutes:values.intervalMinutes,options:values.options,headers:values.headers}}));toast(()=>t("pulseToastSaved"));await goto(back);}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

<div class="mb-4"><BackLink href={back} label={data.monitor?.name??t("pulseMonitors")}/></div>
{#if data.monitor}
 <PageHeader title={t("pulseEditMonitor")} lead={t("pulseMonitorHow")}/>
 <div class="mt-8">{#key data.monitor.id}<MonitorForm projects={data.projects} initial={data.monitor} editing {busy} onsave={save} oncancel={()=>goto(back)}/>{/key}</div>
{:else}<p class="mt-4 rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseMonitorNotFound")}</p>{/if}
