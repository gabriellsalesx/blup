<script lang="ts">
 import {goto} from "$app/navigation";
 import ArrowLeft from "@lucide/svelte/icons/arrow-left";
 import ExternalLink from "@lucide/svelte/icons/external-link";
 import {t,language,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import Domain from "../../../lib/Domain.svelte";
 import PageForm from "../../../lib/PageForm.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 import {panel} from "../../../lib/panel.svelte";
 import {getPages,getIncidents,type Incident} from "../../../lib/status-pages";
 // One status page: its title and monitors, its own domain and the
 // incidents told to visitors.
 type IncidentState="investigating"|"identified"|"monitoring"|"resolved";
 type Impact="operational"|"degraded"|"partial_outage"|"major_outage";
 let {data}=$props();
 let page=$derived(data.page),incidents=$derived<Incident[]>(data.incidents),busy=$state(false);
 let incidentTitle=$state(""),text=$state(""),incidentId=$state<string|null>(null),incidentStateValue=$state<IncidentState>("investigating"),impact=$state<Impact>("degraded");
 const writable=$derived(panel.usage?.readOnly===false&&!!page?.writable);
 const states=$derived([{value:"investigating",label:t("pulseInvestigating")},{value:"identified",label:t("pulseIdentified")},{value:"monitoring",label:t("pulseMonitoring")},{value:"resolved",label:t("pulseResolved")}] as const);
 const impacts=$derived([{value:"operational",label:t("pulseOperational")},{value:"degraded",label:t("pulseDegraded")},{value:"partial_outage",label:t("pulsePartial")},{value:"major_outage",label:t("pulseMajor")}] as const);
 const stateLabel=(s:string)=>states.find(o=>o.value===s)?.label??s;
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function save(values:{title:string;components:{monitorId:string;name:string}[]}){const current=page;if(!current)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:current.id},json:{title:values.title,components:values.components,language:language()}}));page=(await getPages()).find(p=>p.id===current.id)??null;toast(()=>t("pulseToastSaved"));});}
 async function incident(e:SubmitEvent){e.preventDefault();const current=page;if(!current)return;await run(async()=>{if(incidentId)await parseResponse(pulseApi.pages[":page"].incidents[":incident"].updates.$post({param:{page:current.id,incident:incidentId},json:{body:text,state:incidentStateValue}}));else await parseResponse(pulseApi.pages[":page"].incidents.$post({param:{page:current.id},json:{title:incidentTitle,body:text,state:incidentStateValue,impact}}));incidentId=null;incidentTitle="";text="";incidents=await getIncidents(current.id);toast(()=>t("pulseToastIncident"));});}
</script>

<a href="/paginas" class="mb-4 inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><ArrowLeft class="size-4"/>{t("pulsePages")}</a>

{#if !page}<p class="mt-4 rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulsePageNotFound")}</p>
{:else}
 <header class="flex flex-wrap items-start justify-between gap-4">
  <div class="min-w-0"><h1 class="flex flex-wrap items-center gap-3 text-2xl font-bold"><span class="truncate">{page.title}</span><Badge tone={page.published?"green":"neutral"}>{page.published?t("pulsePublished"):t("pulseDraft")}</Badge></h1><a class="mt-1 inline-flex items-center gap-1 text-sm text-blup hover:underline" href={`https://pulse.getblup.site/${page.slug}/`} target="_blank" rel="noopener noreferrer">pulse.getblup.site/{page.slug}/<ExternalLink class="size-3.5"/></a></div>
 </header>

 {#if writable}<div class="mt-8">{#key page.id}<PageForm monitors={data.monitors} initial={page} editing {busy} onsave={save} oncancel={()=>goto("/paginas")}/>{/key}</div>{/if}

 <Domain page={page.id} eligible={writable&&!!panel.usage?.limits?.customDomain}/>

 <section class="mt-6"><Card title={t("pulseIncidents")}>
  {#if writable}<form class="mt-5 grid gap-4" onsubmit={incident}>
   {#if !incidentId}<FloatField id="incident-title" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentTitle")} required maxlength={120} bind:value={incidentTitle}/>{/if}
   <TextArea id="incident-text" surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentText")} required maxlength={2000} bind:value={text}/>
   <div class="grid gap-4 sm:grid-cols-2"><Select surface="bg-white dark:bg-neutral-900" label={t("pulseIncidentState")} options={states} bind:value={incidentStateValue}/>{#if !incidentId}<Select surface="bg-white dark:bg-neutral-900" label={t("pulseImpact")} options={impacts} bind:value={impact}/>{/if}</div>
   <div class="flex flex-wrap justify-end gap-2">{#if incidentId}<Button type="button" variant="secondary" onclick={()=>{incidentId=null;text="";}}>{t("cancel")}</Button>{/if}<Button {busy}>{incidentId?t("pulseUpdateIncident"):t("pulseNewIncident")}</Button></div>
  </form>{/if}
  <div class="mt-5 grid">{#each incidents as item(item.id)}<div class="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-200 py-3 dark:border-neutral-800"><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span>{item.title}</span><Badge tone={item.state==="resolved"?"green":"amber"}>{stateLabel(item.state)}</Badge></p><p class="text-xs text-neutral-500">{formatDate(new Date(item.updatedAt))}</p></div>{#if writable&&item.state!=="resolved"}<Button small variant="secondary" onclick={()=>{incidentId=item.id;incidentTitle=item.title;text="";}}>{t("pulseUpdateIncident")}</Button>{/if}</div>{:else}<p class="text-sm text-neutral-500">{t("pulseNoIncidents")}</p>{/each}</div>
 </Card></section>
{/if}
