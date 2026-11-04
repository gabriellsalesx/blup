<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto,invalidateAll} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import Send from "@lucide/svelte/icons/send";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import {dealsApi,parseResponse,type Channel} from "../../lib/api";
 import {dealsError,failureText} from "../../lib/errors";
 import {refreshUsage} from "../../lib/panel.svelte";
 import {hasPlan} from "../../lib/plan";
 import Usage from "../../lib/Usage.svelte";
 // The Telegram and Discord channels the deals go to: whether the last
 // post went through, a test message and removing one.
 let {data}:PageProps=$props();
 const kindName={telegram:"Telegram",discord:"Discord"} as const;
 async function run(fn:()=>Promise<void>){try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}}
 // The test is sent right away: success only when it arrived; otherwise
 // why, and the channel's state again (a refusal switches it off).
 const test=(channel:Channel)=>run(async()=>{const result=await parseResponse(dealsApi.channels[":channel"].test.$post({param:{channel:channel.id},json:{}}));if(result.ok){toast(()=>t("dealsTestSent"));}else{toast(failureText(result.code,"dealsTestFailed"),{kind:"error"});}await invalidateAll();});
 async function remove(channel:Channel){if(!await confirm({title:t("dealsRemove"),message:t("dealsRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(dealsApi.channels[":channel"].$delete({param:{channel:channel.id}}));await invalidateAll();void refreshUsage();toast(()=>t("dealsRemoved"));});}
 const menuFor=(channel:Channel)=>[...(hasPlan()&&channel.active?[{label:t("dealsTest"),Icon:Send,onselect:()=>void test(channel)}]:[]),{label:t("dealsRemove"),Icon:Trash2,danger:true,onselect:()=>void remove(channel)}];
</script>

<PageHeader title={t("dealsChannels")} lead={t("dealsChannelsLead")}>
 {#snippet aside()}{#if hasPlan()}<Button onclick={()=>goto(resolve("/channels/new"))}><Plus class="size-4"/>{t("dealsNewChannel")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-6"><Usage kind="channels"/></div>

<div class="mt-6"><List empty={t("dealsChannelsEmpty")}>
 {#each data.channels as channel(channel.id)}
  {@const failing=!channel.active||!!channel.lastError}
  <ListItem label={channel.label}>
   <div class="flex items-center gap-3"><span class="grid size-10 shrink-0 place-items-center rounded-full border border-neutral-200 dark:border-neutral-700"><ChannelIcon type={channel.type} class="size-5"/></span><div class="min-w-0"><p class="truncate font-semibold">{channel.label}</p><p class="text-sm {failing?'text-amber-700 dark:text-amber-300':'text-neutral-500'}">{kindName[channel.type]} · {failing?t("dealsChannelFailing"):t("dealsChannelActive")}</p></div></div>
   {#snippet menu()}<ActionMenu label={t("dealsActions")} items={menuFor(channel)}/>{/snippet}
  </ListItem>
 {/each}
</List></div>

<p class="mt-10 text-xs text-neutral-500">{t("dealsChannelsPrivacy")}</p>
