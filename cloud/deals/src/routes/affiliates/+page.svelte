<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import {t} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import {stores,storeName} from "../../lib/catalog";
 import StoreLogo from "../../lib/StoreLogo.svelte";
 // The stores as integrations: a card each with its mark, what it takes
 // and whether it is connected; Add (or Manage, once connected) opens the
 // store's own page.
 let {data}:PageProps=$props();
 const connected=(id:string)=>!!data.stores.find(s=>s.store===id)?.configured;
</script>

{#snippet action(on:boolean,href:string)}
 {#if on}<a {href} class="block rounded-lg border border-neutral-300 px-4 py-2 text-center text-sm font-semibold transition-colors hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800">{t("dealsManage")}</a>
 {:else}<a {href} class="block rounded-lg bg-blup px-4 py-2 text-center text-sm font-semibold text-white transition-opacity hover:opacity-90">{t("dealsAdd")}</a>{/if}
{/snippet}

<PageHeader title={t("dealsAffiliates")} lead={t("dealsAffiliatesLead")}/>

<div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-2">
 {#each stores as info(info.id)}
  {@const on=connected(info.id)}
  {@const href=resolve("/affiliates/[store]",{store:info.id})}
  <article class="flex flex-col rounded-xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
   <div class="flex items-start gap-4">
    <StoreLogo store={info.id}/>
    <div class="min-w-0 flex-1">
     <h2 class="truncate text-lg font-semibold">{storeName[info.id]}</h2>
     <p class="mt-0.5 flex flex-wrap items-center gap-1.5 text-sm text-neutral-500">{#if on}<CircleCheck class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400"/><span class="font-medium text-emerald-700 dark:text-emerald-400">{t("dealsConfigured")}</span><span aria-hidden="true">·</span>{/if}<span>{info.keys?t("dealsNeedsKeys"):t(info.needs??"dealsNeedsTag")}</span><span aria-hidden="true">·</span><span>{t("dealsFree")}</span></p>
    </div>
    <span class="hidden shrink-0 sm:block">{@render action(on,href)}</span>
   </div>
   <p class="mt-4 border-t border-neutral-200 pt-4 text-sm text-neutral-600 dark:border-neutral-800 dark:text-neutral-300">{t(info.about)}</p>
   <!-- On a phone the button goes under the card, so the name keeps its line. -->
   <div class="mt-4 grid sm:hidden">{@render action(on,href)}</div>
  </article>
 {/each}
</div>

<p class="mt-8 text-sm text-neutral-500">{t("dealsKeysNote")}</p>
