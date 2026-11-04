<script lang="ts">
 import Check from "@lucide/svelte/icons/check";
 import {t} from "#lib/i18n.svelte.ts";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import type {Channel} from "./api";
 // The channels a post or a hunt goes to, as boxes to tick (as Pulse's
 // test kinds). A channel that is switched off can't be picked.
 let {channels,selected=$bindable([]),disabled=false}:{channels:Channel[];selected?:string[];disabled?:boolean}=$props();
 function toggle(id:string){selected=selected.includes(id)?selected.filter(s=>s!==id):[...selected,id];}
</script>
{#if channels.length}
 <div class="grid gap-3 sm:grid-cols-2" role="group" aria-label={t("dealsPickChannels")}>
  {#each channels as channel(channel.id)}
   {@const on=selected.includes(channel.id)}
   <button type="button" role="checkbox" aria-checked={on} disabled={disabled||!channel.active} onclick={()=>toggle(channel.id)} class="flex items-center gap-3 rounded-xl border p-4 text-left transition-colors disabled:opacity-50 {on?'border-blup bg-orange-50/60 dark:bg-orange-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}">
    <ChannelIcon type={channel.type} class="size-6 shrink-0"/>
    <span class="min-w-0 flex-1 truncate font-semibold">{channel.label}</span>
    <span class="grid size-5 shrink-0 place-items-center rounded border {on?'border-blup bg-blup text-white':'border-neutral-300 dark:border-neutral-600'}" aria-hidden="true">{#if on}<Check class="size-3.5"/>{/if}</span>
   </button>
  {/each}
 </div>
{:else}<p class="text-sm text-neutral-500">{t("dealsNoChannels")}</p>{/if}
