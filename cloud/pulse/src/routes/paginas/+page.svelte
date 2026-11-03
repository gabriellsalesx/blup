<script lang="ts">
 import Domain from "../../lib/Domain.svelte";
 import {type InferResponseType} from "hono/client";
 import Plus from "@lucide/svelte/icons/plus";
 import Pencil from "@lucide/svelte/icons/pencil";
 import Eye from "@lucide/svelte/icons/eye";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import ExternalLink from "@lucide/svelte/icons/external-link";
 import {t,language,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import {pulseApi,parseResponse} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 type Page=InferResponseType<typeof pulseApi.pages.$get,200>["pages"][number];
 type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
 const incidentsGET=pulseApi.pages[":page"].incidents.$get;
 type Incident=InferResponseType<typeof incidentsGET,200>["incidents"][number];
 type IncidentState="investigating"|"identified"|"monitoring"|"resolved";
 type Impact="operational"|"degraded"|"partial_outage"|"major_outage";
 let {data}=$props();
 let pages=$derived<Page[]>(data.pages);
 const monitors=$derived<Monitor[]>(data.monitors);
 let incidents=$state<Incident[]>([]),editing=$state<Page|null>(null),chosen=$state<Record<string,string>>({}),title=$state(""),slug=$state(""),busy=$state(false);
 let incidentTitle=$state(""),text=$state(""),incidentId=$state<string|null>(null),incidentStateValue=$state<IncidentState>("investigating"),impact=$state<Impact>("degraded");
 const writable=$derived(panel.usage?.readOnly===false);
 const validComponents=$derived(Object.values(chosen).every(v=>v.trim().length>0));
 const states=$derived([{value:"investigating",label:t("pulseInvestigating")},{value:"identified",label:t("pulseIdentified")},{value:"monitoring",label:t("pulseMonitoring")},{value:"resolved",label:t("pulseResolved")}] as const);
 const impacts=$derived([{value:"operational",label:t("pulseOperational")},{value:"degraded",label:t("pulseDegraded")},{value:"partial_outage",label:t("pulsePartial")},{value:"major_outage",label:t("pulseMajor")}] as const);
 const stateLabel=(s:string)=>states.find(o=>o.value===s)?.label??s;
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function refresh(){pages=(await parseResponse(pulseApi.pages.$get())).pages;}
 async function edit(page:Page){editing=page;title=page.title;slug=page.slug;chosen=Object.fromEntries(page.components.map(c=>[c.monitorId,c.name]));incidents=(await parseResponse(incidentsGET({param:{page:page.id}}))).incidents;}
 function clear(){editing=null;title="";slug="";chosen={};incidents=[];incidentId=null;text="";}
 async function save(e:SubmitEvent){e.preventDefault();await run(async()=>{const components=Object.entries(chosen).map(([monitorId,name])=>({monitorId,name:name.trim()}));if(editing)await parseResponse(pulseApi.pages[":page"].$patch({param:{page:editing.id},json:{title,components,language:language()}}));else await parseResponse(pulseApi.pages.$post({json:{title,slug,components,language:language()}}));const wasEditing=!!editing;await refresh();clear();toast(()=>wasEditing?t("pulseToastSaved"):t("pulseToastPage"));});}
 async function publish(page:Page){await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:page.id},json:{published:!page.published}}));await refresh();toast(()=>page.published?t("pulseToastOffline"):t("pulseToastLive"));});}
 async function remove(page:Page){if(!await confirm({title:t("delete"),message:t("pulseDeletePage"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$delete({param:{page:page.id}}));await refresh();clear();toast(()=>t("pulseToastDeleted"));});}
 async function incident(e:SubmitEvent){e.preventDefault();const page=editing;if(!page)return;await run(async()=>{if(incidentId)await parseResponse(pulseApi.pages[":page"].incidents[":incident"].updates.$post({param:{page:page.id,incident:incidentId},json:{body:text,state:incidentStateValue}}));else await parseResponse(pulseApi.pages[":page"].incidents.$post({param:{page:page.id},json:{title:incidentTitle,body:text,state:incidentStateValue,impact}}));incidentId=null;incidentTitle="";text="";incidents=(await parseResponse(incidentsGET({param:{page:page.id}}))).incidents;toast(()=>t("pulseToastIncident"));});}
 function toggleMonitor(id:string,checked:boolean){if(checked)chosen[id]="";else{const copy={...chosen};delete copy[id];chosen=copy;}}
</script>

<PageHeader title={t("pulsePages")} lead={t("pulsePublicPrivacy")}/>

{#if writable&&(!editing||editing.writable)}<div class="mt-8"><Card title={editing?`${t("pulseEdit")}: ${editing.title}`:t("pulseNewPage")}><form class="mt-5 grid gap-5" onsubmit={save}>
 <div class="grid gap-4 sm:grid-cols-2"><FloatField id="page-title" surface="bg-white dark:bg-neutral-900" label={t("pulsePageTitle")} required maxlength={100} bind:value={title}/><FloatField id="page-slug" surface="bg-white dark:bg-neutral-900" label={t("pulseSlug")} required disabled={!!editing} maxlength={40} pattern="[a-z0-9]+(-[a-z0-9]+)*" bind:value={slug}/></div>
 <fieldset><legend class="mb-3 text-sm font-semibold">{t("pulseComponents")}</legend><div class="overflow-hidden rounded-lg border border-neutral-200 dark:border-neutral-800">{#each monitors as monitor(monitor.id)}<div class="grid gap-3 border-b border-neutral-200 px-4 py-3 last:border-b-0 sm:grid-cols-[1fr_minmax(0,18rem)] sm:items-center dark:border-neutral-800"><Switch tone="mint" label={monitor.name} checked={Object.hasOwn(chosen,monitor.id)} onchange={on=>toggleMonitor(monitor.id,on)}/>{#if Object.hasOwn(chosen,monitor.id)}<FloatField id={`public-${monitor.id}`} surface="bg-white dark:bg-neutral-900" label={t("pulsePublicName")} maxlength={80} required bind:value={chosen[monitor.id]}/>{/if}</div>{:else}<p class="px-4 py-6 text-center text-sm text-neutral-500">{t("pulseNoMonitors")}</p>{/each}</div></fieldset>
 <div class="flex flex-wrap justify-end gap-2">{#if editing}<Button type="button" variant="secondary" onclick={clear}>{t("cancel")}</Button>{/if}<Button {busy} disabled={!validComponents}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("pulseCreatePage")}</Button></div>
</form></Card></div>{/if}

<div class="mt-6"><List empty={t("pulseNoPages")}>
 {#each pages as page(page.id)}
  <ListItem selected={editing?.id===page.id}>
   <p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{page.title}</span><Badge tone={page.published?"green":"neutral"}>{page.published?t("pulsePublished"):t("pulseDraft")}</Badge></p><a class="mt-0.5 inline-flex items-center gap-1 text-sm text-blup hover:underline" href={`https://pulse.getblup.site/${page.slug}/`} target="_blank" rel="noopener noreferrer">pulse.getblup.site/{page.slug}/<ExternalLink class="size-3.5"/></a>
   {#snippet actions()}<Button small variant="secondary" onclick={()=>run(()=>edit(page))}><Pencil class="size-4"/>{t("pulseEdit")}</Button>{#if page.writable}<Button small variant="secondary" onclick={()=>publish(page)}>{#if page.published}<EyeOff class="size-4"/>{t("pulseUnpublish")}{:else}<Eye class="size-4"/>{t("pulsePublish")}{/if}</Button>{/if}<Button small variant="secondary" onclick={()=>remove(page)}><Trash2 class="size-4"/>{t("delete")}</Button>{/snippet}
  </ListItem>
 {/each}
</List></div>

{#if editing}
 <Domain page={editing.id} eligible={editing.writable&&!!panel.usage?.limits?.customDomain}/>
 <section class="mt-6"><Card title={t("pulseIncidents")}>
  {#if editing.writable}<form class="mt-5 grid gap-4" onsubmit={incident}>
   {#if !incidentId}<FloatField id="incident-title" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentTitle")} required maxlength={120} bind:value={incidentTitle}/>{/if}
   <TextArea id="incident-text" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentText")} required maxlength={2000} bind:value={text}/>
   <div class="grid gap-4 sm:grid-cols-2"><Select surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentState")} options={states} bind:value={incidentStateValue}/>{#if !incidentId}<Select surface="bg-white dark:bg-neutral-900" label={t("pulseImpact")} options={impacts} bind:value={impact}/>{/if}</div>
   <div class="flex flex-wrap justify-end gap-2">{#if incidentId}<Button type="button" variant="secondary" onclick={()=>{incidentId=null;text="";}}>{t("cancel")}</Button>{/if}<Button {busy}>{incidentId?t("pulseUpdateIncident"):t("pulseNewIncident")}</Button></div>
  </form>{/if}
  <div class="mt-5 grid">{#each incidents as item(item.id)}<div class="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-200 py-3 dark:border-neutral-800"><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span>{item.title}</span><Badge tone={item.state==="resolved"?"green":"amber"}>{stateLabel(item.state)}</Badge></p><p class="text-xs text-neutral-500">{formatDate(new Date(item.updatedAt))}</p></div>{#if editing.writable&&item.state!=="resolved"}<Button small variant="secondary" onclick={()=>{incidentId=item.id;incidentTitle=item.title;text="";}}>{t("pulseUpdateIncident")}</Button>{/if}</div>{/each}</div>
 </Card></section>
{/if}
