<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Eye from "@lucide/svelte/icons/eye";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import Plus from "@lucide/svelte/icons/plus";
 import Settings from "@lucide/svelte/icons/settings";
 import FileText from "@lucide/svelte/icons/file-text";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import Pencil from "@lucide/svelte/icons/pencil";
 import Wrench from "@lucide/svelte/icons/wrench";
 import {t,language,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import DetailHeader from "#lib/ui/DetailHeader.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import StateMark from "#lib/ui/StateMark.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import Domain from "../../../lib/Domain.svelte";
 import PageForm from "../../../lib/PageForm.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 import {panel} from "../../../lib/panel.svelte";
 import {getPages,getIncidents,getMaintenance,type Incident,type Maintenance} from "../../../lib/status-pages";
 import {ago} from "../../../lib/monitor";
 // One status page: its title and monitors, its own domain and the
 // incidents told to visitors (each opens its own page, with its updates;
 // a button on top reports a new one) and its scheduled maintenance.
 let {data}:PageProps=$props();
 let page=$derived(data.page),incidents=$derived<Incident[]>(data.incidents),maintenance=$derived<Maintenance[]>(data.maintenance),busy=$state(false);
 const writable=$derived(panel.usage?.readOnly===false&&!!page?.writable);
 const states=$derived([{value:"investigating",label:t("pulseInvestigating")},{value:"identified",label:t("pulseIdentified")},{value:"monitoring",label:t("pulseMonitoring")},{value:"resolved",label:t("pulseResolved")}] as const);
 const stateLabel=(s:string)=>states.find(o=>o.value===s)?.label??s;
 // Settings, incidents, maintenance and domain, one at a time; another page
 // starts on settings, or on the tab in the address (?tab=incidents).
 const tabs=$derived([{value:"settings",label:t("pulseSettings")},{value:"incidents",label:t("pulseIncidents")},{value:"maintenance",label:t("pulseMaintenance")},{value:"domain",label:t("pulseDomainTab")}] as const);
 let tab=$derived<"settings"|"incidents"|"maintenance"|"domain">(data.tab);
 const maintenanceLabel=(s:Maintenance["state"])=>s==="scheduled"?t("pulseMaintenanceScheduled"):s==="active"?t("pulseMaintenanceActive"):t("pulseMaintenanceFinished");
 // The mark of an incident: green once resolved, else by how bad it is.
 const toneOf=(i:Incident)=>i.state==="resolved"?"up" as const:i.impact==="major_outage"||i.impact==="partial_outage"?"down" as const:"partial" as const;
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function save(values:{title:string;components:{monitorId:string;name:string}[]}){const current=page;if(!current)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:current.id},json:{title:values.title,components:values.components,language:language()}}));page=(await getPages()).find(p=>p.id===current.id)??null;toast(()=>t("pulseToastSaved"));});}
 async function removeIncident(item:Incident){const current=page;if(!current||!await confirm({title:t("delete"),message:t("pulseRemoveIncident"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].incidents[":incident"].$delete({param:{page:current.id,incident:item.id}}));incidents=await getIncidents(current.id);toast(()=>t("pulseToastDeleted"));});}
 async function removeMaintenance(item:Maintenance){const current=page;if(!current||!await confirm({title:t("pulseRemoveMaintenanceTitle"),message:t("pulseRemoveMaintenance"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].maintenance[":maintenance"].$delete({param:{page:current.id,maintenance:item.id}}));maintenance=await getMaintenance(current.id);toast(()=>t("pulseToastDeleted"));});}
 async function publish(){const current=page;if(!current)return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:current.id},json:{published:!current.published}}));page=(await getPages()).find(p=>p.id===current.id)??null;toast(()=>current.published?t("pulseToastOffline"):t("pulseToastLive"));});}
</script>

<BackLink href={resolve("/status-pages")} label={t("pulsePages")}/>

{#if !page}<p class="mt-4 rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulsePageNotFound")}</p>
{:else}
 {@const live=page.published}
 {@const pageId=page.id}
 <DetailHeader title={page.title} url={`https://pulse.getblup.site/${page.slug}/`}>
  {#snippet badge()}<Badge tone={live?"green":"neutral"}>{live?t("pulsePublished"):t("pulseDraft")}</Badge>{/snippet}
  {#snippet actions()}{#if writable}{#if tab==="maintenance"}<Button small onclick={()=>goto(resolve("/status-pages/[id]/maintenance/new",{id:pageId}))}><Wrench class="size-4"/>{t("pulseScheduleMaintenance")}</Button>{:else}<Button small onclick={()=>goto(resolve("/status-pages/[id]/incidents/new",{id:pageId}))}><Plus class="size-4"/>{t("pulseReportIncident")}</Button>{/if}<Button small variant="secondary" {busy} onclick={publish}>{#if live}<EyeOff class="size-4"/>{t("pulseUnpublish")}{:else}<Eye class="size-4"/>{t("pulsePublish")}{/if}</Button>{/if}{/snippet}
 </DetailHeader>

 <div class="mt-8"><Segmented label={t("pulsePages")} options={tabs} bind:value={tab}/></div>

 {#if tab==="settings"}
  <div class="mt-8">{#key page.id}{#if writable}<PageForm monitors={data.monitors} initial={page} editing {busy} onsave={save} oncancel={()=>goto(resolve("/status-pages"))}/>{:else}<p class="text-neutral-500">{t("pulseReadOnlyPage")}</p>{/if}{/key}</div>
 {:else if tab==="domain"}
  <Domain page={page.id} eligible={writable&&!!panel.usage?.limits?.customDomain}/>
 {:else if tab==="maintenance"}
  <div class="mt-8"><List empty={t("pulseNoMaintenance")}>
   {#each maintenance as item(item.id)}
    {@const href=resolve("/status-pages/[id]/maintenance/[maintenance]",{id:pageId,maintenance:item.id})}
    <ListItem href={writable?href:undefined} label={item.title}>
     <div class="flex items-center gap-4"><StateMark tone={item.state==="finished"?"past":"maint"} label={maintenanceLabel(item.state)}/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{item.title}</span><Badge tone={item.state==="active"?"indigo":"neutral"}>{maintenanceLabel(item.state)}</Badge></p><p class="text-sm text-neutral-500">{formatDate(new Date(item.startsAt))} – {formatDate(new Date(item.endsAt))}</p></div></div>
     {#snippet actions()}<span class="text-sm text-neutral-500">{item.services.length===1?t("pulseOneService"):t("pulseManyServices",{count:String(item.services.length)})}</span>{/snippet}
     {#snippet menu()}{#if writable}<ActionMenu label={t("pulseActions")} items={[{label:t("pulseEdit"),Icon:Pencil,onselect:()=>void goto(href)},{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void removeMaintenance(item)}]}/>{/if}{/snippet}
    </ListItem>
   {/each}
  </List></div>
 {:else}
  <div class="mt-8"><List empty={t("pulseNoIncidents")}>
   {#each incidents as item(item.id)}
    {@const href=resolve("/status-pages/[id]/incidents/[incident]",{id:page.id,incident:item.id})}
    <ListItem {href} label={item.title}>
     <div class="flex items-center gap-4"><StateMark tone={toneOf(item)} label={stateLabel(item.state)}/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{item.title}</span><Badge tone={item.state==="resolved"?"green":"amber"}>{stateLabel(item.state)}</Badge></p><p class="text-sm text-neutral-500" title={formatDate(new Date(item.createdAt))}>{t("pulseIncidentCreated",{time:ago(item.createdAt)})}</p></div></div>
     {#snippet actions()}<span class="text-sm text-neutral-500">{item.updates===1?t("pulseOneUpdate"):t("pulseManyUpdates",{count:String(item.updates)})}</span>{/snippet}
     {#snippet menu()}{#if writable}<ActionMenu label={t("pulseActions")} items={[{label:t("pulseConfigure"),Icon:Settings,onselect:()=>void goto(resolve("/status-pages/[id]/incidents/[incident]/settings",{id:pageId,incident:item.id}))},{label:t("pulseSeeUpdates"),Icon:FileText,onselect:()=>void goto(href)},{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void removeIncident(item)}]}/>{/if}{/snippet}
    </ListItem>
   {/each}
  </List></div>
 {/if}
{/if}
