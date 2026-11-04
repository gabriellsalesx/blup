<script lang="ts">
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import BarTip from "#lib/ui/BarTip.svelte";
 import type {Stats} from "./api";
 // Clicks per day as columns (one series, the Deals coral; a lighter step
 // in the dark theme so it stays clear of the surface). Columns grow from
 // the baseline, at most 24px wide with rounded tops; each one opens its
 // day (clicks and posts) on hover, focus or tap. The top line marks the
 // highest day. A table with the same numbers sits under it.
 let {days}:{days:Stats["days"]}=$props();
 const max=$derived(Math.max(1,...days.map(d=>d.clicks)));
 const n=(v:number)=>v.toLocaleString(intlLocale());
 const date=(day:string)=>formatDate(`${day}T12:00:00Z`,{day:"numeric",month:"short"});
</script>

<figure>
 <figcaption class="sr-only">{t("dealsClicksPerDay")}</figcaption>
 <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3">
  <div class="flex h-48 flex-col justify-between text-right text-xs text-neutral-500 tabular-nums" aria-hidden="true"><span>{n(max)}</span><span>0</span></div>
  <div class="relative h-48 border-b border-neutral-300 dark:border-neutral-700">
   <div class="pointer-events-none absolute inset-x-0 top-0 border-t border-dashed border-neutral-200 dark:border-neutral-800" aria-hidden="true"></div>
   <div class="relative flex h-full items-end {days.length>31?'gap-px':'gap-[2px]'}" role="group" aria-label={t("dealsClicksPerDay")}>
    {#each days as day,index(day.day)}
     <BarTip class="h-full flex-1 items-end justify-center rounded-none" label={`${date(day.day)}: ${t("dealsDayTip",{clicks:n(day.clicks),posts:n(day.posts)})}`} align={index<5?"start":index>=days.length-5?"end":"center"}>
      <span class="block w-full max-w-6 rounded-t bg-blup dark:bg-orange-600" style:height="{day.clicks?Math.max(2,day.clicks/max*100):0}%"></span>
      {#snippet tip()}<p class="font-semibold">{date(day.day)}</p><p class="mt-1 text-neutral-500">{t("dealsDayTip",{clicks:n(day.clicks),posts:n(day.posts)})}</p>{/snippet}
     </BarTip>
    {/each}
   </div>
  </div>
  <span></span>
  {#if days.length}<div class="mt-2 flex justify-between text-xs text-neutral-500" aria-hidden="true"><span>{date(days[0].day)}</span><span>{date(days[days.length-1].day)}</span></div>{/if}
 </div>
</figure>

<details class="mt-4 text-sm">
 <summary class="cursor-pointer font-semibold text-neutral-600 dark:text-neutral-300">{t("dealsTableView")}</summary>
 <div class="mt-3 max-h-72 overflow-auto rounded-lg border border-neutral-200 dark:border-neutral-800">
  <table class="w-full text-left tabular-nums">
   <thead class="sticky top-0 bg-neutral-50 dark:bg-neutral-900"><tr><th scope="col" class="px-3 py-2">{t("dealsDay")}</th><th scope="col" class="px-3 py-2 text-right">{t("dealsTotalClicks")}</th><th scope="col" class="px-3 py-2 text-right">{t("dealsTotalPosts")}</th></tr></thead>
   <tbody>{#each days as day(day.day)}<tr class="border-t border-neutral-200 dark:border-neutral-800"><th scope="row" class="px-3 py-1.5 font-normal">{date(day.day)}</th><td class="px-3 py-1.5 text-right">{n(day.clicks)}</td><td class="px-3 py-1.5 text-right">{n(day.posts)}</td></tr>{/each}</tbody>
  </table>
 </div>
</details>
