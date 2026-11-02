<script lang="ts">
 import {onMount} from "svelte";
 import {type InferResponseType} from "hono/client";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import {pulseApi,parseResponse,type Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
 const checksGET=pulseApi.monitors[":monitor"].checks.$get;
 type History=InferResponseType<typeof checksGET,200>;
 let monitors=$state<Monitor[]>([]),projects=$state<Project[]>([]),name=$state(""),url=$state(""),projectId=$state(""),intervalMinutes=$state(5),busy=$state(false),error=$state<unknown>(null),history=$state<History|null>(null);
 const writable=$derived(panel.usage?.readOnly===false);
 const stateName=(s:string)=>t(s==="up"?"pulseOperational":s==="down"?"pulseDown":"pulseUnknown");
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function refresh(){monitors=(await parseResponse(pulseApi.monitors.$get())).monitors;}
 async function create(e:SubmitEvent){e.preventDefault();await run(async()=>{await parseResponse(pulseApi.monitors.$post({json:{name,url,projectId,intervalMinutes}}));name="";url="";await refresh();});}
 async function toggle(m:Monitor){await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$patch({param:{monitor:m.id},json:{active:!m.active}}));await refresh();});}
 async function remove(m:Monitor){if(!await confirm({title:t("delete"),message:t("pulseRemoveMonitor"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$delete({param:{monitor:m.id}}));history=null;await refresh();});}
 onMount(()=>{void run(async()=>{await refresh();projects=(await parseResponse(pulseApi.projects.$get())).projects.filter(p=>p.writable);projectId=projects[0]?.id??"";intervalMinutes=panel.usage?.limits?.intervalMinutes??5;});});
</script>
<h1 class="text-3xl font-bold">{t("pulseMonitors")}</h1><p class="mt-3 text-sm text-neutral-500">{t("pulseCheckLead")}</p>
{#if writable&&projects.length}<form class="my-6 grid gap-4" onsubmit={create}><div class="grid gap-4 sm:grid-cols-2"><FloatField id="monitor-name" label={t("pulseMonitorName")} maxlength={80} required bind:value={name}/><FloatField id="monitor-url" label={t("pulseURL")} type="url" maxlength={2048} required bind:value={url}/><label class="grid gap-2 text-sm">{t("pulseProjects")}<select bind:value={projectId} class="rounded-lg border p-3 dark:bg-neutral-900">{#each projects as p(p.id)}<option value={p.id}>{p.name}</option>{/each}</select></label><FloatField id="monitor-interval" label={t("pulseInterval")} type="number" min={panel.usage?.limits?.intervalMinutes??5} max={1440} required bind:value={intervalMinutes}/></div><Button {busy} disabled={!projectId}>{t("pulseCreateMonitor")}</Button></form>{/if}
{#if error}<p class="my-4 text-red-600" role="alert">{errorMessage(error)}</p>{/if}
<div class="mt-6 grid gap-3">{#each monitors as monitor(monitor.id)}<Card><h2 class="font-bold">{monitor.name}</h2><p class="my-2 break-all text-xs text-neutral-500">{monitor.url}</p><p>{monitor.active?stateName(monitor.state):t("pulsePaused")}</p>{#if monitor.lastAt}<p class="my-2 text-xs text-neutral-500">{t("pulseCheckedAt")}: {formatDate(monitor.lastAt)}</p>{/if}<div class="mt-3 flex flex-wrap gap-2"><Button variant="secondary" onclick={()=>run(async()=>{history=await parseResponse(checksGET({param:{monitor:monitor.id}}));})}>{t("pulseHistory")}</Button>{#if monitor.writable}<Button variant="secondary" onclick={()=>toggle(monitor)}>{monitor.active?t("pulsePause"):t("pulseResume")}</Button>{/if}<Button variant="secondary" onclick={()=>remove(monitor)}>{t("delete")}</Button></div></Card>{:else}<p class="text-neutral-500">{t("pulseNoMonitors")}</p>{/each}</div>
{#if history}<section class="mt-8"><h2 class="mb-3 text-xl font-bold">{t("pulseHistory")}</h2><p class="text-xs text-neutral-500">{t("pulseDetailDays",{days:panel.usage?.limits?.retentionDays??30})}</p><div class="my-4 overflow-auto"><table class="w-full text-left text-sm"><thead><tr><th class="p-2">{t("pulseCheckedAt")}</th><th class="p-2">{t("pulseMonitors")}</th><th class="p-2">HTTP</th><th class="p-2">{t("pulseDuration")}</th></tr></thead><tbody>{#each history.checks as check(check.id)}<tr class="border-t border-neutral-200 dark:border-neutral-800"><td class="p-2">{formatDate(new Date(check.checkedAt))}</td><td class="p-2">{stateName(check.outcome)}</td><td class="p-2">{check.status??t("pulseUnknown")}</td><td class="p-2">{check.durationMs.toLocaleString(intlLocale())} ms</td></tr>{/each}</tbody></table></div><div class="overflow-auto"><table class="w-full text-left text-sm"><thead><tr><th class="p-2">UTC</th><th class="p-2">{t("pulseOperational")}</th><th class="p-2">{t("pulseDown")}</th><th class="p-2">{t("pulseUnknown")}</th></tr></thead><tbody>{#each history.days as day(day.day)}<tr class="border-t border-neutral-200 dark:border-neutral-800"><td class="p-2">{new Intl.DateTimeFormat(intlLocale(),{dateStyle:"medium",timeZone:"UTC"}).format(new Date(day.day*86400000))}</td><td class="p-2">{day.up.toLocaleString(intlLocale())}</td><td class="p-2">{day.down.toLocaleString(intlLocale())}</td><td class="p-2">{day.unknown.toLocaleString(intlLocale())}</td></tr>{/each}</tbody></table></div></section>{/if}
