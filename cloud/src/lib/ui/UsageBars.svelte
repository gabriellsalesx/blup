<script lang="ts">
 import {t,formatDate,relativeTime} from "#lib/i18n.svelte.ts";
 import {bytes,megabytes,type Usage} from "#lib/usage.ts";
 import type {AIShare,Limits} from "#lib/api.ts";
 let {usage,limits,ai}:{usage:Usage;limits:Limits;ai?:AIShare}=$props();
 const bars=$derived([
  {label:t("storage"),used:usage.storageBytes,max:limits.storageMB*1024*1024,text:t("ofLimit",{used:bytes(usage.storageBytes),limit:megabytes(limits.storageMB)})},
  {label:t("sites"),used:usage.sites,max:limits.sites,text:limits.sites<0?t("unlimited",{count:usage.sites}):limits.sites===0?t("notIncluded"):t("ofLimit",{used:usage.sites,limit:limits.sites})},
  ...(ai?[{label:t("aiUsage"),used:ai.percent,max:ai.included?100:0,text:ai.included?t("aiReset",{percent:ai.percent,date:formatDate(ai.resetsAt,{day:"numeric",month:"long",timeZone:"UTC"})}):t("notIncluded")}]:[]),
 ]);
 const pct=(used:number,max:number)=>max>0?Math.min(100,used/max*100):0;
</script>
<div class="mt-3 grid gap-4">
 {#each bars as bar (bar.label)}
  <div><div class="flex items-baseline justify-between gap-3 text-sm"><span class="font-medium">{bar.label}</span><span class="text-neutral-500">{bar.text}</span></div>
   {#if bar.max>0}<div class="mt-1.5 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full {pct(bar.used,bar.max)>=90?'bg-red-500':'bg-blup'}" style:width="{pct(bar.used,bar.max)}%"></div></div>{/if}
  </div>
 {/each}
 <p class="text-sm text-neutral-500">{(limits.devices<0?usage.devices===1:limits.devices===1)?t("studioComputer",{count:usage.devices}):t("studioComputers",{count:limits.devices>=0?t("ofLimit",{used:usage.devices,limit:limits.devices}):usage.devices})} · {usage.lastSyncAt?t("lastSync",{time:relativeTime(usage.lastSyncAt)}):t("neverSynced")}</p>
</div>
