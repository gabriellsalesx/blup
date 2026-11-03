<script lang="ts">
 import {goto} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import FieldHelp from "#lib/ui/FieldHelp.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Select from "#lib/ui/Select.svelte";
 import {pulseApi,parseResponse,type Project} from "../../../lib/api";
 import {panel} from "../../../lib/panel.svelte";
 // A new monitor: its name, the address to check, the project whose
 // destinations get its alerts and how often it is checked.
 let {data}=$props();
 const projects=$derived<Project[]>(data.projects),minInterval=$derived(panel.usage?.limits?.intervalMinutes??5);
 let name=$state(""),url=$state("https://"),projectId=$derived(data.projects[0]?.id??""),intervalMinutes=$derived(Math.max(5,minInterval)),busy=$state(false);
 const intervals=$derived([1,2,5,10,15,30,60].filter(m=>m>=minInterval).map(m=>({value:m,label:t("pulseEvery",{minutes:m})})));
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 async function create(e:SubmitEvent){e.preventDefault();busy=true;try{await parseResponse(pulseApi.monitors.$post({json:{name:name.trim(),url:url.trim(),projectId,intervalMinutes}}));toast(()=>t("pulseToastMonitor"));await goto("/monitores");}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

<div class="mb-4"><BackLink href="/monitores" label={t("pulseMonitors")}/></div>
<PageHeader title={t("pulseNewMonitorTitle")} lead={t("pulseMonitorHow")}/>

<form class="mt-8" onsubmit={create}><Card><div class="grid gap-6">
 <FloatField id="monitor-name" surface="bg-white dark:bg-neutral-900" label={t("pulseMonitorName")} maxlength={80} required bind:value={name}/>
 <div><FloatField id="monitor-url" surface="bg-white dark:bg-neutral-900" label={t("pulseURL")} type="url" maxlength={2048} required bind:value={url}/><p class="mt-2 px-3 text-sm text-neutral-500">{t("pulseURLHelp")}</p></div>
 <FieldHelp help={t("pulseProjectHelp")}>{#if projectOptions.length}<Select surface="bg-white dark:bg-neutral-900" label={t("pulseAllProjects")} options={projectOptions} bind:value={projectId}/>{:else}<p class="text-sm text-neutral-500">{t("pulseEmpty")}</p>{/if}</FieldHelp>
 <FieldHelp help={t("pulseIntervalHelp")}><Select surface="bg-white dark:bg-neutral-900" label={t("pulseInterval")} options={intervals} bind:value={intervalMinutes}/></FieldHelp>
 <div class="flex justify-end gap-2"><Button type="button" variant="secondary" onclick={()=>goto("/monitores")}>{t("cancel")}</Button><Button {busy} disabled={!projectId||!name.trim()}><Plus class="size-4"/>{t("pulseCreateMonitor")}</Button></div>
</div></Card></form>
