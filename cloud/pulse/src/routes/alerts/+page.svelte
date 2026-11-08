<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import FileText from "@lucide/svelte/icons/file-text";
 import Folder from "@lucide/svelte/icons/folder";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import type {Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 import {projectChannels,kindName} from "../../lib/alerts";
 // Each project is a row, as on Projects: a folder for the project, its
 // name and how many destinations it has (or that one is failing), and the
 // marks of those destinations side by side. A click opens the project's
 // destinations and their tests.
 let {data}:PageProps=$props();
 const writable=$derived(panel.usage?.readOnly===false);
 const cards=$derived(data.projects.map(project=>({project,channels:projectChannels(data.channels,project.id)})));
 const count=(n:number)=>n===0?t("pulseNoDestination"):n===1?t("pulseOneDestination"):t("pulseDestinationCount",{count:n});
 const menuItems=(project:Project,href:string)=>[{label:t("pulseSeeDetails"),Icon:FileText,onselect:()=>void goto(href)},...(writable&&project.writable?[{label:t("pulseAddChannel"),Icon:Plus,onselect:()=>void goto(`${resolve("/alerts/new")}?${new URLSearchParams({project:project.id})}`)}]:[])];
</script>

<PageHeader title={t("pulseAlerts")} lead={t("pulseAlertsLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto(resolve("/alerts/new"))}><Plus class="size-4"/>{t("pulseAddChannel")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8"><List empty={t("pulseEmpty")}>
 {#each cards as {project,channels}(project.id)}
  {@const href=resolve("/alerts/[project]",{project:project.id})}
  {@const live=channels.filter(c=>c.active)}
  {@const failing=live.some(c=>c.lastError)}
  <ListItem {href} label={project.name}>
   <!-- The marks of its destinations stay on the project's line, on a phone too. -->
   <div class="flex items-center gap-3"><Folder class="size-5 shrink-0 text-neutral-500"/><div class="min-w-0 flex-1"><p class="truncate font-semibold">{project.name}</p><p class="text-sm {failing?"text-amber-700 dark:text-amber-300":"text-neutral-500"}">{failing?t("pulseDeliveryFailed"):count(live.length)}</p></div>{#if live.length}<span class="relative z-10 flex shrink-0 items-center" aria-label={live.map(c=>kindName[c.type]).join(", ")}>{#each live as channel,i(channel.id)}<span class="grid size-8 place-items-center sm:size-10 rounded-full border border-neutral-200 bg-white ring-4 ring-white dark:border-neutral-700 dark:bg-neutral-900 dark:ring-neutral-900 {i?"-ml-2":""}" title={kindName[channel.type]}><ChannelIcon type={channel.type} class="size-4 sm:size-[18px]"/></span>{/each}</span>{/if}</div>
   {#snippet menu()}<ActionMenu label={t("pulseActions")} items={menuItems(project,href)}/>{/snippet}
  </ListItem>
 {/each}
</List></div>

<p class="mt-10 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
