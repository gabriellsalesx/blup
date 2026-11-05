<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import Card from "#lib/ui/Card.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Tag from "@lucide/svelte/icons/tag";
 import type {Range} from "./+page";
 import ClicksChart from "../../lib/ClicksChart.svelte";
 // The clicks of the period: the totals, a column per day and the five
 // deals clicked the most, with their picture.
 let {data}:PageProps=$props();
 const ranges=$derived([{value:"7" as const,label:t("dealsLast7")},{value:"30" as const,label:t("dealsLast30")},{value:"90" as const,label:t("dealsLast90")}]);
 const n=(v:number)=>v.toLocaleString(intlLocale());
 const clicks=$derived(data.stats.days.reduce((s,d)=>s+d.clicks,0)),posts=$derived(data.stats.days.reduce((s,d)=>s+d.posts,0));
 const show=(days:Range)=>void goto(`${resolve("/stats")}?${new URLSearchParams({days})}`,{reset:false});
</script>

<PageHeader title={t("dealsStats")} lead={t("dealsStatsLead")}/>

<div class="mt-6"><Segmented label={t("dealsPeriod")} options={ranges} value={data.days} onchange={show}/></div>

<div class="mt-6 grid max-w-md grid-cols-2 gap-4">
 <div class="rounded-xl border border-neutral-200 px-4 py-3 dark:border-neutral-800"><p class="text-sm text-neutral-500">{t("dealsTotalClicks")}</p><p class="mt-1 text-2xl font-extrabold tabular-nums">{n(clicks)}</p></div>
 <div class="rounded-xl border border-neutral-200 px-4 py-3 dark:border-neutral-800"><p class="text-sm text-neutral-500">{t("dealsTotalPosts")}</p><p class="mt-1 text-2xl font-extrabold tabular-nums">{n(posts)}</p></div>
</div>

<div class="mt-6"><Card title={t("dealsClicksPerDay")}>
 <div class="mt-5">{#if clicks}<ClicksChart days={data.stats.days}/>{:else}<p class="py-8 text-center text-neutral-500">{t("dealsStatsEmpty")}</p>{/if}</div>
</Card></div>

<h2 class="mt-10 text-lg font-bold">{t("dealsTopOffers")}</h2>
<div class="mt-4"><List empty={t("dealsStatsEmpty")}>
 {#each data.stats.top as offer,i(i)}
  <ListItem label={offer.title}><div class="flex items-center gap-3"><span class="w-6 shrink-0 text-right text-sm font-bold text-neutral-400 tabular-nums">{i+1}</span>{#if offer.image}<img src={offer.image} alt="" class="size-10 shrink-0 rounded-lg bg-neutral-100 object-cover dark:bg-neutral-800" referrerpolicy="no-referrer" loading="lazy"/>{:else}<span class="grid size-10 shrink-0 place-items-center rounded-lg bg-neutral-100 text-neutral-400 dark:bg-neutral-800" aria-hidden="true"><Tag class="size-5"/></span>{/if}<p class="min-w-0 flex-1 truncate font-semibold">{offer.title}</p><span class="shrink-0 text-sm font-semibold tabular-nums">{n(offer.clicks)}</span></div></ListItem>
 {/each}
</List></div>

<p class="mt-10 text-xs text-neutral-500">{t("dealsStatsPrivacy")}</p>
