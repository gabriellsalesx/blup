<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto,refreshAll} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import FileText from "@lucide/svelte/icons/file-text";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {openCard,cardLink} from "#lib/card.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {pulseApi,parseResponse,type Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 import {projectChannels,unboundChannels,kindName,type Channel} from "../../lib/alerts";
 // Each project is a card, like a monitor's: the marks of the destinations
 // it sends to side by side, and its state on the edge (green when they
 // deliver, amber when one is failing, gray with none). A click opens the
 // project's destinations and their tests. Destinations no project sends
 // to are listed apart.
 let {data}:PageProps=$props();
 const writable=$derived(panel.usage?.readOnly===false);
 const cards=$derived(data.projects.map(project=>({project,channels:projectChannels(data.channels,project.id)})));
 const unbound=$derived(unboundChannels(data.channels));
 const edge={ok:"border-l-emerald-500 dark:border-l-emerald-500",failing:"border-l-amber-500 dark:border-l-amber-500",none:"border-l-neutral-300 dark:border-l-neutral-700"};
 function situation(channels:Channel[]){const live=channels.filter(c=>c.active);return !live.length?"none":live.some(c=>c.lastError)?"failing":"ok";}
 const count=(n:number)=>n===0?t("pulseNoDestination"):n===1?t("pulseOneDestination"):t("pulseDestinationCount",{count:n});
 const menu=(project:Project,href:string)=>[{label:t("pulseSeeDetails"),Icon:FileText,onselect:()=>void goto(href)},...(writable&&project.writable?[{label:t("pulseAddChannel"),Icon:Plus,onselect:()=>void goto(`${resolve("/alerts/new")}?${new URLSearchParams({project:project.id})}`)}]:[])];
 async function remove(channel:Channel){if(!await confirm({title:t("delete"),message:t("pulseRemoveChannel"),danger:true}))return;try{await parseResponse(pulseApi.channels[":channel"].$delete({param:{channel:channel.id}}));await refreshAll();toast(()=>t("pulseToastDeleted"));}catch(e){toast(errorMessage(e),{kind:"error"});}}
</script>

<PageHeader title={t("pulseAlerts")} lead={t("pulseAlertsLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto(resolve("/alerts/new"))}><Plus class="size-4"/>{t("pulseAddChannel")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8 grid gap-3">
 {#each cards as {project,channels}(project.id)}
  {@const href=resolve("/alerts/[project]",{project:project.id})}
  {@const live=channels.filter(c=>c.active)}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div onclick={(event)=>openCard(event,href)} onauxclick={(event)=>openCard(event,href)} class="relative flex cursor-pointer items-center gap-4 rounded-xl border border-l-4 border-neutral-200 bg-white px-5 py-4 transition-colors hover:bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:bg-neutral-800/60 {edge[situation(channels)]}">
   <a class={cardLink} {href} aria-label={project.name}></a>
   <span class="min-w-0 flex-1"><span class="block truncate font-semibold">{project.name}</span><span class="block text-sm {live.some(c=>c.lastError)?"text-amber-700 dark:text-amber-300":"text-neutral-500"}">{live.some(c=>c.lastError)?t("pulseDeliveryFailed"):count(live.length)}</span></span>
   {#if live.length}
    <span class="flex shrink-0 items-center" aria-label={live.map(c=>kindName[c.type]).join(", ")}>
     {#each live as channel,i(channel.id)}<span class="grid size-10 place-items-center rounded-full border border-neutral-200 bg-white ring-4 ring-white dark:border-neutral-700 dark:bg-neutral-900 dark:ring-neutral-900 {i?"-ml-2":""}" title={kindName[channel.type]}><ChannelIcon type={channel.type} class="size-[18px]"/></span>{/each}
    </span>
   {/if}
   <ActionMenu label={t("pulseActions")} items={menu(project,href)}/>
  </div>
 {:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseEmpty")}</div>{/each}
</div>

{#if unbound.length}
 <h2 class="mt-10 text-xl font-bold">{t("pulseUnboundChannels")}</h2><p class="mt-1 mb-4 text-sm text-neutral-500">{t("pulseUnboundChannelsHelp")}</p>
 <List>
  {#each unbound as channel(channel.id)}
   <ListItem>
    <div class="flex items-center gap-3"><ChannelIcon type={channel.type} class="size-6"/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{channel.label}</span><Badge>{kindName[channel.type]}</Badge></p><p class="text-sm text-neutral-500">{channel.active?t("pulseActive"):t("pulsePaused")}</p></div></div>
    {#snippet menu()}<ActionMenu label={t("pulseActions")} items={[{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void remove(channel)}]}/>{/snippet}
   </ListItem>
  {/each}
 </List>
{/if}

<p class="mt-10 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
