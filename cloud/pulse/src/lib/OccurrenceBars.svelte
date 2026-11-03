<script lang="ts">
 import {t,intlLocale} from "#lib/i18n.svelte.ts";
 import BarTip from "#lib/ui/BarTip.svelte";
 // How many times an issue happened each day, oldest on the left: one red
 // bar per day, as tall as its share of the busiest day; a day without it
 // is a thin gray line. Each bar opens a card with its count and day.
 const DAY=86_400_000;
 let {days,range,label}:{days:{day:number;count:number}[];range:number;label:string}=$props();
 const byDay=$derived(new Map(days.map(d=>[d.day,d.count])));
 const shown=$derived.by(()=>{const today=Math.floor(Date.now()/DAY);return Array.from({length:range},(_,i)=>{const day=today-range+1+i;return {day,count:byDay.get(day)??0};});});
 const peak=$derived(Math.max(1,...shown.map(d=>d.count)));
 const date=$derived(new Intl.DateTimeFormat(intlLocale(),{dateStyle:"medium",timeZone:"UTC"}));
 const count=(n:number)=>n===1?t("pulseOneOccurrence"):t("pulseCount",{count:n.toLocaleString(intlLocale())});
</script>

<div class="flex h-24 items-end gap-[3px]" role="group" aria-label={label}>
 {#each shown as d,index(d.day)}
  {@const when=date.format(new Date(d.day*DAY))}
  <BarTip class="h-full flex-1 items-end rounded-sm" label={`${when}: ${count(d.count)}`} align={index<5?"start":index>=shown.length-5?"end":"center"}>
   <span class="w-full rounded-sm {d.count?"bg-red-500":"bg-neutral-200 dark:bg-neutral-800"}" style:height={d.count?`${Math.max(8,d.count/peak*100)}%`:"3px"}></span>
   {#snippet tip()}<span class="flex items-center gap-2 font-semibold"><span class="size-2.5 shrink-0 rounded-full {d.count?"bg-red-500":"bg-neutral-300 dark:bg-neutral-600"}"></span>{count(d.count)}</span><span class="mt-2.5 block border-t border-neutral-200 pt-2 text-center text-[13px] font-normal text-neutral-500 dark:border-neutral-800">{when}</span>{/snippet}
  </BarTip>
 {/each}
</div>
<div class="mt-2 flex justify-between text-xs text-neutral-500"><span>{t("pulseDaysAgo",{days:range})}</span><span>{t("pulseToday")}</span></div>
