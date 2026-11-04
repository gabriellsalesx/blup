<script lang="ts">
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import UsageMeter from "#lib/ui/UsageMeter.svelte";
 import {panel} from "./panel.svelte";
 // One limit of the plan, used and allowed: today's posts (with when the
 // count starts again), the channels, the hunts or the mirrors.
 let {kind}:{kind:"posts"|"channels"|"hunts"|"mirrors"}=$props();
 const n=(v:number)=>v.toLocaleString(intlLocale());
 const meter=$derived.by(()=>{
  const u=panel.usage;if(!u?.limits)return null;
  if(kind==="posts")return {used:u.used.postsToday,limit:u.limits.postsPerDay,text:t("dealsPostsUsage",{used:n(u.used.postsToday),limit:n(u.limits.postsPerDay)})};
  if(kind==="channels")return {used:u.used.channels,limit:u.limits.channels,text:t("dealsChannelsUsage",{used:n(u.used.channels),limit:n(u.limits.channels)})};
  if(kind==="mirrors")return {used:u.used.mirrors,limit:u.limits.mirrors,text:t("dealsMirrorsUsage",{used:n(u.used.mirrors),limit:n(u.limits.mirrors)})};
  return {used:u.used.hunts,limit:u.limits.hunts,text:t("dealsHuntsUsage",{used:n(u.used.hunts),limit:n(u.limits.hunts)})};
 });
</script>
{#if meter}<div class="max-w-sm"><UsageMeter text={meter.text} used={meter.used} limit={meter.limit}/>{#if kind==="posts"&&panel.usage}<p class="mt-1 text-xs text-neutral-500">{t("dealsResets",{time:formatDate(panel.usage.resetsAt,{timeStyle:"short"})})}</p>{/if}</div>{/if}
