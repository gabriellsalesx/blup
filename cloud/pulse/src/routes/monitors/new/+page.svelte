<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import MonitorForm from "../../../lib/MonitorForm.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 // A new monitor; once made, back to the list.
 let {data}:PageProps=$props();
 let busy=$state(false);
 async function save(values:{name:string;url:string;projectId:string;intervalMinutes:number}){busy=true;try{await parseResponse(pulseApi.monitors.$post({json:values}));toast(()=>t("pulseToastMonitor"));await goto(resolve("/monitors"));}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

<div class="mb-4"><BackLink href={resolve("/monitors")} label={t("pulseMonitors")}/></div>
<PageHeader title={t("pulseNewMonitorTitle")} lead={t("pulseMonitorHow")}/>
<div class="mt-8">{#key data.projects}<MonitorForm projects={data.projects} {busy} onsave={save} oncancel={()=>goto(resolve("/monitors"))}/>{/key}</div>
