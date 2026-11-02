<script lang="ts">
 import {onMount} from "svelte";
 import {type InferResponseType} from "hono/client";
 import Plus from "@lucide/svelte/icons/plus";
 import Activity from "@lucide/svelte/icons/activity";
 import Pause from "@lucide/svelte/icons/pause";
 import Play from "@lucide/svelte/icons/play";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import StatusDot from "#lib/ui/StatusDot.svelte";
 import UptimeBars from "#lib/ui/UptimeBars.svelte";
 import {pulseApi,parseResponse,type Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
 const checksGET=pulseApi.monitors[":monitor"].checks.$get;
 type History=InferResponseType<typeof checksGET,200>;
 let monitors=$state<Monitor[]>([]),projects=$state<Project[]>([]),name=$state(""),url=$state(""),projectId=$state(""),intervalMinutes=$state(5),busy=$state(false),error=$state<unknown>(null);
 let history=$state<History|null>(null),openMonitor=$state<string|null>(null);
 const writable=$derived(panel.usage?.readOnly===false);
 const minInterval=$derived(panel.usage?.limits?.intervalMinutes??5);
 const intervals=$derived([1,2,5,10,15,30,60].filter(m=>m>=minInterval).map(m=>({value:m,label:t("pulseEvery",{minutes:m})})));
 const projectOptions=$derived(projects.map(p=>({value:p.id,label:p.name})));
 const stateName=(s:string)=>t(s==="up"?"pulseOperational":s==="down"?"pulseDown":"pulseUnknown");
 const text=(m:Monitor)=>!m.active?"text-neutral-500":m.state==="up"?"text-emerald-600 dark:text-emerald-400":m.state==="down"?"text-red-600 dark:text-red-400":"text-neutral-500";
 const day=new Intl.DateTimeFormat(intlLocale(),{dateStyle:"medium",timeZone:"UTC"});
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function refresh(){monitors=(await parseResponse(pulseApi.monitors.$get())).monitors;}
 async function create(e:SubmitEvent){e.preventDefault();await run(async()=>{await parseResponse(pulseApi.monitors.$post({json:{name,url,projectId,intervalMinutes}}));name="";url="";await refresh();});}
 async function toggle(m:Monitor){await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$patch({param:{monitor:m.id},json:{active:!m.active}}));await refresh();});}
 async function remove(m:Monitor){if(!await confirm({title:t("delete"),message:t("pulseRemoveMonitor"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.monitors[":monitor"].$delete({param:{monitor:m.id}}));history=null;openMonitor=null;await refresh();});}
 async function showHistory(m:Monitor){if(openMonitor===m.id){openMonitor=null;history=null;return;}await run(async()=>{history=await parseResponse(checksGET({param:{monitor:m.id}}));openMonitor=m.id;});}
 // The last 30 days, oldest first (days without checks stay empty).
 function bars(h:History){const byDay=new Map(h.days.map(d=>[d.day,d])),today=Math.floor(Date.now()/86_400_000);return Array.from({length:30},(_,i)=>{const key=today-29+i,d=byDay.get(key);return {key,label:day.format(new Date(key*86_400_000)),up:d?.up??0,down:d?.down??0};});}
 const dot=(m:Monitor)=>!m.active?"unknown":m.state==="up"?"up":m.state==="down"?"down":"unknown";
 onMount(()=>{void run(async()=>{await refresh();projects=(await parseResponse(pulseApi.projects.$get())).projects.filter(p=>p.writable);projectId=projects[0]?.id??"";intervalMinutes=Math.max(5,minInterval);});});
</script>

<PageHeader title={t("pulseMonitors")} lead={t("pulseCheckLead")}/>
{#if error}<p class="mt-6 text-red-600" role="alert">{errorMessage(error)}</p>{/if}

{#if writable&&projects.length}<div class="mt-8"><Card title={t("pulseNewMonitor")}><form class="mt-5 grid gap-4 sm:grid-cols-2" onsubmit={create}>
 <FloatField id="monitor-name" surface="bg-white dark:bg-neutral-900" label={t("pulseMonitorName")} maxlength={80} required bind:value={name}/>
 <FloatField id="monitor-url" surface="bg-white dark:bg-neutral-900" label={t("pulseURL")} type="url" maxlength={2048} required bind:value={url}/>
 <Select surface="bg-white dark:bg-neutral-900" label={t("pulseProjects")} options={projectOptions} bind:value={projectId}/>
 <Select surface="bg-white dark:bg-neutral-900" label={t("pulseInterval")} options={intervals} bind:value={intervalMinutes}/>
 <div class="sm:col-span-2 sm:justify-self-end"><Button {busy} disabled={!projectId}><Plus class="size-4"/>{t("pulseCreateMonitor")}</Button></div>
</form></Card></div>{/if}

<div class="mt-6"><List empty={t("pulseNoMonitors")}>
 {#each monitors as monitor(monitor.id)}
  <ListItem>
   <div class="flex items-start gap-3"><span class="mt-1.5"><StatusDot state={dot(monitor)}/></span><div class="min-w-0"><p class="flex flex-wrap items-baseline gap-x-2 font-semibold"><span class="truncate">{monitor.name}</span><span class="text-sm font-medium {text(monitor)}">{monitor.active?stateName(monitor.state):t("pulsePaused")}</span></p><p class="truncate text-sm text-neutral-500">{monitor.url}</p><p class="text-xs text-neutral-500">{t("pulseEvery",{minutes:monitor.intervalMinutes})} · {monitor.lastAt?t("pulseLastCheck",{date:formatDate(monitor.lastAt)}):t("pulseNeverChecked")}</p></div></div>
   {#snippet actions()}<Button small variant="secondary" aria-expanded={openMonitor===monitor.id} onclick={()=>showHistory(monitor)}><Activity class="size-4"/>{t("pulseHistory")}</Button>{#if monitor.writable}<Button small variant="secondary" onclick={()=>toggle(monitor)}>{#if monitor.active}<Pause class="size-4"/>{t("pulsePause")}{:else}<Play class="size-4"/>{t("pulseResume")}{/if}</Button>{/if}<Button small variant="secondary" onclick={()=>remove(monitor)}><Trash2 class="size-4"/>{t("delete")}</Button>{/snippet}
   {#snippet extra()}{#if openMonitor===monitor.id&&history}<div class="mt-5">
    <UptimeBars days={bars(history)} label={t("pulseHistory")}/>
    <p class="mt-2 text-xs text-neutral-500">{t("pulseDetailDays",{days:panel.usage?.limits?.retentionDays??30})}</p>
    {#if history.checks.length}<div class="mt-4 overflow-auto"><table class="w-full text-left text-sm"><thead class="text-neutral-500"><tr><th class="py-2 pr-4 font-medium">{t("pulseCheckedAt")}</th><th class="py-2 pr-4 font-medium">{t("pulseMonitors")}</th><th class="py-2 pr-4 font-medium">HTTP</th><th class="py-2 font-medium">{t("pulseDuration")}</th></tr></thead><tbody>{#each history.checks.slice(0,20) as check(check.id)}<tr class="border-t border-neutral-200 dark:border-neutral-800"><td class="py-2 pr-4">{formatDate(new Date(check.checkedAt))}</td><td class="py-2 pr-4">{stateName(check.outcome)}</td><td class="py-2 pr-4">{check.status??"-"}</td><td class="py-2">{check.durationMs.toLocaleString(intlLocale())} ms</td></tr>{/each}</tbody></table></div>{/if}
   </div>{/if}{/snippet}
  </ListItem>
 {/each}
</List></div>
