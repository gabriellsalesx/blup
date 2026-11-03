<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Eye from "@lucide/svelte/icons/eye";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import {t,language,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import DetailHeader from "#lib/ui/DetailHeader.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import Domain from "../../../lib/Domain.svelte";
 import PageForm from "../../../lib/PageForm.svelte";
 import IncidentForm from "../../../lib/IncidentForm.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 import {panel} from "../../../lib/panel.svelte";
 import {getPages,getIncidents,type Incident,type IncidentState,type ServiceStatus} from "../../../lib/status-pages";
 // One status page: its title and monitors, its own domain and the
 // incidents told to visitors.
 let {data}:PageProps=$props();
 let page=$derived(data.page),incidents=$derived<Incident[]>(data.incidents),busy=$state(false);
 // The open incident being updated, else a new one; the form starts over
 // after each save.
 let updating=$state<Incident|null>(null),formRun=$state(0);
 const writable=$derived(panel.usage?.readOnly===false&&!!page?.writable);
 const states=$derived([{value:"investigating",label:t("pulseInvestigating")},{value:"identified",label:t("pulseIdentified")},{value:"monitoring",label:t("pulseMonitoring")},{value:"resolved",label:t("pulseResolved")}] as const);
 const stateLabel=(s:string)=>states.find(o=>o.value===s)?.label??s;
 // Settings, incidents and domain, one at a time; another page starts on settings.
 const tabs=$derived([{value:"settings",label:t("pulseSettings")},{value:"incidents",label:t("pulseIncidents")},{value:"domain",label:t("pulseDomainTab")}] as const);
 let tab=$derived.by<"settings"|"incidents"|"domain">(()=>{void data;return "settings";});
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function save(values:{title:string;components:{monitorId:string;name:string}[]}){const current=page;if(!current)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:current.id},json:{title:values.title,components:values.components,language:language()}}));page=(await getPages()).find(p=>p.id===current.id)??null;toast(()=>t("pulseToastSaved"));});}
 async function publish(){const current=page;if(!current)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:current.id},json:{published:!current.published}}));page=(await getPages()).find(p=>p.id===current.id)??null;toast(()=>current.published?t("pulseToastOffline"):t("pulseToastLive"));});}
 async function incident(values:{title:string;body:string;state:IncidentState;publishedAt:number;services:{monitorId:string;status:ServiceStatus}[]}){const current=page,editing=updating;if(!current)return;await run(async()=>{if(editing)await parseResponse(pulseApi.pages[":page"].incidents[":incident"].updates.$post({param:{page:current.id,incident:editing.id},json:{body:values.body,state:values.state,services:values.services}}));else await parseResponse(pulseApi.pages[":page"].incidents.$post({param:{page:current.id},json:{title:values.title,body:values.body,state:values.state,publishedAt:new Date(values.publishedAt).toISOString(),services:values.services}}));updating=null;formRun++;incidents=await getIncidents(current.id);toast(()=>t("pulseToastIncident"));});}
</script>

<BackLink href={resolve("/status-pages")} label={t("pulsePages")}/>

{#if !page}<p class="mt-4 rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulsePageNotFound")}</p>
{:else}
 {@const live=page.published}
 <DetailHeader title={page.title} url={`https://pulse.getblup.site/${page.slug}/`}>
  {#snippet badge()}<Badge tone={live?"green":"neutral"}>{live?t("pulsePublished"):t("pulseDraft")}</Badge>{/snippet}
  {#snippet actions()}{#if writable}<Button small variant="secondary" {busy} onclick={publish}>{#if live}<EyeOff class="size-4"/>{t("pulseUnpublish")}{:else}<Eye class="size-4"/>{t("pulsePublish")}{/if}</Button>{/if}{/snippet}
 </DetailHeader>

 <div class="mt-8"><Segmented label={t("pulsePages")} options={tabs} bind:value={tab}/></div>

 {#if tab==="settings"}
  <div class="mt-8">{#key page.id}{#if writable}<PageForm monitors={data.monitors} initial={page} editing {busy} onsave={save} oncancel={()=>goto(resolve("/status-pages"))}/>{:else}<p class="text-neutral-500">{t("pulseReadOnlyPage")}</p>{/if}{/key}</div>
 {:else if tab==="domain"}
  <Domain page={page.id} eligible={writable&&!!panel.usage?.limits?.customDomain}/>
 {:else}
 {#if writable}<div class="mt-8">{#key `${updating?.id??"new"}-${formRun}`}<IncidentForm components={page.components} incident={updating} keepDays={panel.usage?.limits?.retentionDays??30} {busy} onsave={incident} oncancel={updating?()=>{updating=null;}:undefined}/>{/key}</div>{/if}
 <section class="mt-10"><Card title={t("pulseIncidents")}>
  <div class="mt-5 grid">{#each incidents as item(item.id)}<div class="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-200 py-3 dark:border-neutral-800"><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span>{item.title}</span><Badge tone={item.state==="resolved"?"green":"amber"}>{stateLabel(item.state)}</Badge></p><p class="text-xs text-neutral-500">{formatDate(new Date(item.createdAt))}</p></div>{#if writable&&item.state!=="resolved"}<Button small variant="secondary" onclick={()=>{updating=item;scrollTo({top:0,behavior:"smooth"});}}>{t("pulseUpdateIncident")}</Button>{/if}</div>{:else}<p class="text-sm text-neutral-500">{t("pulseNoIncidents")}</p>{/each}</div>
 </Card></section>
 {/if}
{/if}
