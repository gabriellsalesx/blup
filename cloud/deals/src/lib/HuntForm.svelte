<script lang="ts">
 import Check from "@lucide/svelte/icons/check";
 import {untrack} from "svelte";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import ChannelPicker from "./ChannelPicker.svelte";
 import {huntKinds,platformsFor,huntCurrency,type HuntInput,type Platform} from "./catalog";
 import type {Channel,Hunt,HuntKind} from "./api";
 // A hunt's form (new and edit): what to hunt (games now; stores and
 // mirroring are coming), its filters and the channels it posts to. The
 // kind of a saved hunt can't change. Prices are typed in the kind's
 // currency (dollars for CheapShark, reais for Nuuvem) and sent in cents.
 // Nuuvem's sales have no platforms; they can leave out game credits.
 let {channels,hunt=null,busy=false,submitLabel,onsubmit}:{channels:Channel[];hunt?:Hunt|null;busy?:boolean;submitLabel:string;onsubmit:(input:HuntInput)=>void}=$props();
 // Starts from the saved hunt, if any; editing stays on this page.
 const start=untrack(()=>hunt);
 let kind=$state<HuntKind>(start?.kind??"games_free");
 let picked=$state<Platform[]>([...(start?.filters.platforms??[])]);
 let minDiscount=$state(start?.filters.minDiscount!==undefined?String(start.filters.minDiscount):"50");
 let maxPrice=$state(start?.filters.maxPrice!==undefined?(start.filters.maxPrice/100).toFixed(2):"");
 let gamesOnly=$state(start?.filters.gamesOnly!==false);
 let channelIds=$state<string[]>(untrack(()=>[...(start?.channelIds??channels.filter(c=>c.active).slice(0,1).map(c=>c.id))]));
 const options=$derived(platformsFor(kind));
 // Platforms the kind does not take are dropped when it changes.
 function choose(next:HuntKind){kind=next;const allowed=new Set<Platform>(platformsFor(next).map(p=>p.id));picked=picked.filter(p=>allowed.has(p));}
 function togglePlatform(id:Platform){picked=picked.includes(id)?picked.filter(p=>p!==id):[...picked,id];}
 const discount=$derived(Math.round(Number(minDiscount)));
 const cents=$derived(maxPrice.trim()?Math.round(Number(maxPrice.replace(",","."))*100):null);
 const priced=$derived(huntCurrency[kind]!==undefined);
 const valid=$derived(channelIds.length>0&&(!priced||(Number.isInteger(discount)&&discount>=0&&discount<=100&&(cents===null||(Number.isFinite(cents)&&cents>0)))));
 function submit(e:SubmitEvent){e.preventDefault();if(!valid)return;
  const filters:HuntInput["filters"]={...(picked.length?{platforms:picked}:{}),...(priced?{minDiscount:discount,...(cents!==null?{maxPrice:cents}:{})}:{}),...(kind==="nuuvem_deals"&&!gamesOnly?{gamesOnly:false}:{})};
  onsubmit({kind,channelIds,filters});
 }
</script>

<form class="mt-8 grid gap-6" onsubmit={submit}>
 <Card title={t("dealsHuntKind")}>
  <div class="mt-5 grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label={t("dealsHuntKind")}>
   {#each huntKinds as option(option.id)}
    {@const on=option.id===kind}
    <button type="button" role="radio" aria-checked={on} disabled={option.soon||(!!start&&option.id!==start.kind)} onclick={()=>{if(!option.soon&&option.id!=="stores"&&option.id!=="mirror")choose(option.id);}} class="flex items-center gap-4 rounded-xl border p-4 text-left transition-colors disabled:opacity-60 {on?'border-blup bg-orange-50/60 dark:bg-orange-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}">
     <img src={option.art} alt="" width="56" height="56" class="size-12 shrink-0 sm:size-14 {option.soon?'grayscale-[.5]':''}"/>
     <span class="min-w-0"><span class="flex flex-wrap items-center gap-2 font-semibold">{t(option.title)}{#if option.soon}<Badge>{t("dealsComingSoon")}</Badge>{/if}</span><span class="mt-0.5 block text-sm text-neutral-500">{t(option.help)}</span></span>
    </button>
   {/each}
  </div>
 </Card>

 <Card title={t("dealsFilters")}>
  {#if options.length}<fieldset class="mt-5"><legend class="font-semibold">{t("dealsPlatforms")}</legend><p class="text-sm text-neutral-500">{t("dealsPlatformsHelp")}</p>
   <div class="mt-3 flex flex-wrap gap-2">
    {#each options as platform(platform.id)}
     {@const on=picked.includes(platform.id)}
     <button type="button" role="checkbox" aria-checked={on} onclick={()=>togglePlatform(platform.id)} class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-semibold transition-colors {on?'border-blup bg-blup text-white':'border-neutral-300 hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800'}">{#if on}<Check class="size-3.5"/>{/if}{platform.label}</button>
    {/each}
   </div>
  </fieldset>{/if}
  {#if kind==="nuuvem_deals"}
   <div class="mt-5"><Switch label={t("dealsGamesOnly")} tone="coral" bind:checked={gamesOnly}/><p class="mt-1 text-sm text-neutral-500">{t("dealsGamesOnlyHelp")}</p></div>
  {/if}
  {#if priced}
   <div class="mt-6 grid gap-4 sm:grid-cols-2">
    <FloatField id="hunt-discount" surface="bg-white dark:bg-neutral-900" label={t("dealsMinDiscount")} type="number" inputmode="numeric" min={0} max={100} step={1} required bind:value={minDiscount}/>
    <FloatField id="hunt-price" surface="bg-white dark:bg-neutral-900" label={kind==="nuuvem_deals"?t("dealsMaxPriceBrl"):t("dealsMaxPrice")} type="text" inputmode="decimal" pattern={"[0-9]+([.,][0-9]{1,2})?"} bind:value={maxPrice}/>
   </div>
  {/if}
  {#if kind==="nuuvem_deals"}<p class="mt-4 text-sm text-neutral-500">{t("dealsNuuvemDaily")}</p>{/if}
 </Card>

 <Card title={t("dealsPickChannels")} help={t("dealsPickChannelsHelp")}>
  <div class="mt-5"><ChannelPicker {channels} bind:selected={channelIds}/></div>
 </Card>

 <div class="flex justify-end"><Button {busy} disabled={!valid}>{submitLabel}</Button></div>
</form>
