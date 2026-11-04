<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto,invalidateAll} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import Pencil from "@lucide/svelte/icons/pencil";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import Card from "#lib/ui/Card.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {money} from "#lib/money.ts";
 import {language} from "#lib/i18n.svelte.ts";
 import {dealsApi,parseResponse,type Hunt} from "../../lib/api";
 import {dealsError} from "../../lib/errors";
 import {refreshUsage} from "../../lib/panel.svelte";
 import {hasPlan} from "../../lib/plan";
 import {huntKinds,huntName,platformLabel} from "../../lib/catalog";
 import Usage from "../../lib/Usage.svelte";
 // The hunts: what each one looks for, its filters in one line, how many
 // channels it posts to, and a switch to pause it. Stores and mirroring
 // come in the next phase and show as such.
 let {data}:PageProps=$props();
 const soon=huntKinds.filter(k=>k.soon);
 const artOf=(kind:Hunt["kind"])=>huntKinds.find(k=>k.id===kind)!.art;
 function summary(hunt:Hunt){
  const f=hunt.filters,parts=[f.platforms?.length?f.platforms.map(platformLabel).join(", "):t("dealsAllPlatforms")];
  if(f.minDiscount!==undefined)parts.push(t("dealsDiscountFrom",{percent:String(f.minDiscount)}));
  if(f.maxPrice!==undefined)parts.push(t("dealsUpTo",{price:money(f.maxPrice,"USD",language())}));
  const n=hunt.channelIds.length;parts.push(n===1?t("dealsOneChannel"):t("dealsChannelCount",{count:n.toLocaleString(intlLocale())}));
  return parts.join(" · ");
 }
 async function run(fn:()=>Promise<void>){try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});await invalidateAll();}}
 const toggle=(hunt:Hunt,active:boolean)=>run(async()=>{await parseResponse(dealsApi.hunts[":hunt"].$patch({param:{hunt:hunt.id},json:{active}}));toast(()=>t("dealsSaved"));await invalidateAll();});
 async function remove(hunt:Hunt){if(!await confirm({title:t("dealsRemove"),message:t("dealsRemoveHunt"),danger:true}))return;await run(async()=>{await parseResponse(dealsApi.hunts[":hunt"].$delete({param:{hunt:hunt.id}}));await invalidateAll();void refreshUsage();toast(()=>t("dealsRemoved"));});}
 const editHref=(hunt:Hunt)=>resolve("/hunts/[id]",{id:hunt.id});
 const menuFor=(hunt:Hunt)=>[...(hasPlan()?[{label:t("dealsEditHunt"),Icon:Pencil,onselect:()=>void goto(editHref(hunt))}]:[]),{label:t("dealsRemove"),Icon:Trash2,danger:true,onselect:()=>void remove(hunt)}];
</script>

<PageHeader title={t("dealsHunts")} lead={t("dealsHuntsLead")}>
 {#snippet aside()}{#if hasPlan()}<Button onclick={()=>goto(resolve("/hunts/new"))}><Plus class="size-4"/>{t("dealsNewHunt")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-6"><Usage kind="hunts"/></div>

<div class="mt-6"><List empty={t("dealsHuntsEmpty")}>
 {#each data.hunts as hunt(hunt.id)}
  <ListItem href={hasPlan()?editHref(hunt):undefined} label={t(huntName(hunt.kind))}>
   <div class="flex items-center gap-3"><img src={artOf(hunt.kind)} alt="" width="40" height="40" class="size-10 shrink-0 transition-opacity {hunt.active?'':'opacity-50 grayscale'}"/><div class="min-w-0"><p class="truncate font-semibold">{t(huntName(hunt.kind))}</p><p class="text-sm text-neutral-500">{summary(hunt)}</p></div></div>
   {#snippet actions()}<div class="relative z-10"><Switch label={hunt.active?t("dealsHuntOn"):t("dealsHuntOff")} tone="coral" checked={hunt.active} disabled={!hasPlan()} onchange={(on)=>void toggle(hunt,on)}/></div>{/snippet}
   {#snippet menu()}<ActionMenu label={t("dealsActions")} items={menuFor(hunt)}/>{/snippet}
  </ListItem>
 {/each}
</List></div>

<div class="mt-8 grid gap-4 sm:grid-cols-2">
 {#each soon as kind(kind.id)}
  <Card><div class="flex items-center gap-4"><img src={kind.art} alt="" width="56" height="56" class="size-12 shrink-0 opacity-60 grayscale-[.5] sm:size-14"/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold">{t(kind.title)}<Badge>{t("dealsComingSoon")}</Badge></p><p class="mt-1 text-sm text-neutral-500">{t(kind.help)}</p></div></div></Card>
 {/each}
</div>
