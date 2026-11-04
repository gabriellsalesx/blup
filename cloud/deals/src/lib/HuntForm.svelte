<script lang="ts">
 import Check from "@lucide/svelte/icons/check";
 import {untrack} from "svelte";
 import {resolve} from "$app/paths";
 import {t,intlLocale,language} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import TagInput from "./TagInput.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import Select from "#lib/ui/Select.svelte";
 import ChannelPicker from "./ChannelPicker.svelte";
 import StoreLogo from "./StoreLogo.svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import {dealsApi,parseResponse} from "./api";
 import {dealsError} from "./errors";
 import {huntKinds,platformsFor,nuuvemCountries,countryName,countryOptions,stores,storeName,shopStores,aliexpressCountries,currencyDigits,currencyName,shopCurrency,type HuntInput,type Platform,type NuuvemCountry,type AliexpressCountry,type ShopeeCountry} from "./catalog";
 import type {Channel,Hunt,HuntKind,Store,StoreState} from "./api";
 // A hunt's form (new and edit): what to hunt, its filters and the
 // channels it posts to. The kind of a saved hunt can't change. A store
 // hunt searches Shopee or AliExpress with the person's own API keys
 // (affiliates: what is saved there) for up to five words, comma
 // separated, in the money of where it sells: Shopee's account country,
 // or the country the AliExpress hunt picks. Its price limit is typed in
 // that money and sent in its minor units. Nuuvem's sales take the country of its
 // store, a minimum discount, a price limit only for Brazil (typed in
 // reais, sent in cents) and can leave out game credits. A mirror takes a
 // public Telegram channel, or several (each checked as it is added, each
 // its own mirror; fixed after) and
 // the stores it keeps.
 let {channels,affiliates=[],hunt=null,busy=false,submitLabel,onsubmit}:{channels:Channel[];affiliates?:StoreState[];hunt?:Hunt|null;busy?:boolean;submitLabel:string;onsubmit:(input:HuntInput)=>void}=$props();
 // Starts from the saved hunt, if any; editing stays on this page.
 const start=untrack(()=>hunt);
 let kind=$state<HuntKind>(start?.kind??"games_free");
 let picked=$state<Platform[]>([...(start?.filters.platforms??[])]);
 let minDiscount=$state(start?.filters.minDiscount!==undefined?String(start.filters.minDiscount):"50");
 const startDigits=currencyDigits(start?.currency??"BRL");
 let maxPrice=$state(start?.filters.maxPrice!==undefined?(start.filters.maxPrice/10**startDigits).toFixed(startDigits):"");
 let gamesOnly=$state(start?.filters.gamesOnly!==false);
 let country=$state<NuuvemCountry>(start?.filters.country??"br");
 const nuuvemOptions=$derived(countryOptions(nuuvemCountries,intlLocale(),"br"));
 let mirrors=$state<string[]>([]),checking=$state(false);
 let keep=$state<Store[]>([...(start?.filters.stores??[])]);
 type ShopStore=(typeof shopStores)[number];
 const configured=$derived(affiliates.filter(a=>a.configured).map(a=>a.store));
 const account=$derived((affiliates.find(a=>a.store==="shopee")?.country??"br") as ShopeeCountry);
 let shop=$state<ShopStore>(start?.filters.store??untrack(()=>shopStores.find(s=>affiliates.some(a=>a.store===s&&a.configured)))??"shopee");
 // A new AliExpress hunt starts in the panel's own region.
 let market=$state<AliexpressCountry>(start?.filters.market??untrack(()=>({pt:"br",es:"mx",en:"us"} as const)[language()]));
 const marketOptions=$derived(countryOptions(aliexpressCountries,intlLocale()));
 let words=$state<string[]>([...(start?.filters.keywords??[])]);
 const shopping=$derived(kind==="store_deals"),shopReady=$derived(configured.includes(shop));
 const wordsOk=$derived(words.length>=1&&words.length<=5&&words.every(w=>w.length>=2&&w.length<=60));
 const mirror=$derived(kind==="mirror"),channelReady=$derived(!!start||mirrors.length>0);
 // A channel is added only once the server finds it public.
 async function checkChannel(raw:string){checking=true;try{const r=await parseResponse(dealsApi.hunts.mirror.check.$post({json:{channel:raw}}));return `@${r.name}`;}catch(e){toast(dealsError(e),{kind:"error"});return null;}finally{checking=false;}}
 const keyword=(w:string)=>w.length>=2&&w.length<=60?w:null;
 function toggleStore(id:Store){keep=keep.includes(id)?keep.filter(s=>s!==id):[...keep,id];}
 let channelIds=$state<string[]>(untrack(()=>[...(start?.channelIds??channels.filter(c=>c.active).slice(0,1).map(c=>c.id))]));
 const options=$derived(platformsFor(kind));
 // Platforms the kind does not take are dropped when it changes.
 function choose(next:HuntKind){if(!start&&(next==="store_deals")!==(kind==="store_deals"))minDiscount=next==="store_deals"?"20":"50";kind=next;const allowed=new Set<Platform>(platformsFor(next).map(p=>p.id));picked=picked.filter(p=>allowed.has(p));}
 function togglePlatform(id:Platform){picked=picked.includes(id)?picked.filter(p=>p!==id):[...picked,id];}
 const discount=$derived(Math.round(Number(minDiscount)));
 const currency=$derived(shopping?shopCurrency(shop,market,account):"BRL"),digits=$derived(currencyDigits(currency));
 const cents=$derived(maxPrice.trim()?Math.round(Number(maxPrice.replace(",","."))*10**digits):null);
 const priced=$derived(kind==="nuuvem_deals"||shopping),reais=$derived(shopping||(kind==="nuuvem_deals"&&country==="br"));
 const valid=$derived(channelIds.length>0&&(!mirror||channelReady)&&(!shopping||(wordsOk&&shopReady))&&(!priced||(Number.isInteger(discount)&&discount>=0&&discount<=100&&(!reais||cents===null||(Number.isFinite(cents)&&cents>0)))));
 function submit(e:SubmitEvent){e.preventDefault();if(!valid)return;
  if(mirror){onsubmit({kind,channelIds,filters:keep.length?{stores:keep}:{},...(!start?{mirrors:mirrors.map(m=>m.replace(/^@/,""))}:{})});return;}
  if(shopping){onsubmit({kind,channelIds,filters:{store:shop,...(shop==="aliexpress"?{market}:{}),keywords:words,minDiscount:discount,...(cents!==null?{maxPrice:cents}:{})}});return;}
  const filters:HuntInput["filters"]={...(picked.length?{platforms:picked}:{}),...(priced?{country,minDiscount:discount,...(reais&&cents!==null?{maxPrice:cents}:{}),...(!gamesOnly?{gamesOnly:false}:{})}:{})};
  onsubmit({kind,channelIds,filters});
 }
</script>

<form class="mt-8 grid gap-6" onsubmit={submit}>
 <Card title={t("dealsHuntKind")}>
  <div class="mt-5 grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label={t("dealsHuntKind")}>
   {#each huntKinds as option(option.id)}
    {@const on=option.id===kind}
    <button type="button" role="radio" aria-checked={on} disabled={option.soon||(!!start&&option.id!==start.kind)} onclick={()=>{if(!option.soon)choose(option.id);}} class="flex items-center gap-4 rounded-xl border p-4 text-left transition-colors disabled:opacity-60 {on?'border-blup bg-orange-50/60 dark:bg-orange-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}">
     <img src={option.art} alt="" width="56" height="56" class="size-12 shrink-0 sm:size-14 {option.soon?'grayscale-[.5]':''}"/>
     <span class="min-w-0"><span class="flex flex-wrap items-center gap-2 font-semibold">{t(option.title)}{#if option.soon}<Badge>{t("dealsComingSoon")}</Badge>{/if}</span><span class="mt-0.5 block text-sm text-neutral-500">{t(option.help)}</span></span>
    </button>
   {/each}
  </div>
 </Card>

 {#if mirror}
  <Card title={t("dealsMirrorChannel")} help={t("dealsMirrorChannelHelp")}>
   {#if start?.mirror}<p class="mt-4 font-semibold">{t("dealsMirrorFixed",{channel:start.mirror.channel})}{#if start.mirror.title}{" · "}<span class="font-normal text-neutral-500">{start.mirror.title}</span>{/if}</p>
   {:else}
    <div class="mt-5"><TagInput id="mirror-channel" label={t("dealsMirrorChannel")} bind:tags={mirrors} max={10} maxlength={200} add={checkChannel}/></div>
    <p class="mt-1.5 text-sm text-neutral-500">{checking?t("dealsMirrorChecking"):t("dealsMirrorTagsHelp")}</p>
   {/if}
  </Card>
 {/if}

 {#if shopping}
  <Card title={t("dealsShopStore")} help={t("dealsShopStoreHelp")}>
   <div class="mt-5 flex flex-wrap gap-2" role="radiogroup" aria-label={t("dealsShopStore")}>
    {#each shopStores as id(id)}
     {@const on=shop===id}
     <button type="button" role="radio" aria-checked={on} onclick={()=>shop=id} class="inline-flex items-center gap-2 rounded-full border py-1 pr-3 pl-1 text-sm font-semibold transition-colors {on?'border-blup bg-orange-50 dark:bg-orange-950/40':'border-neutral-300 hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800'}"><StoreLogo store={id} class="size-6 !rounded-full"/>{storeName[id]}{#if on}<Check class="size-3.5 text-blup"/>{/if}</button>
    {/each}
   </div>
   {#if !shopReady}<p class="mt-3 text-sm text-amber-800 dark:text-amber-300">{t("dealsShopNoKeys",{store:storeName[shop]})} <a href={resolve("/affiliates")} class="font-semibold underline">{t("dealsAffiliates")}</a></p>
   {:else if shop==="shopee"}<p class="mt-3 text-sm text-neutral-500">{t("dealsShopeeAccount",{country:countryName(account,intlLocale()),currency:currencyName(currency,intlLocale())})}</p>{/if}
   {#if shop==="aliexpress"}<div class="mt-5 max-w-sm"><Select label={t("dealsAliMarket")} surface="bg-white dark:bg-neutral-900" options={marketOptions} bind:value={market}/></div>
    <p class="mt-1.5 text-sm text-neutral-500">{t("dealsAliMarketHelp",{currency:currencyName(currency,intlLocale())})}</p>{/if}
   <div class="mt-5"><TagInput id="shop-keywords" label={t("dealsShopKeywords")} bind:tags={words} max={5} add={keyword}/></div>
   <p class="mt-1.5 text-sm text-neutral-500">{t("dealsShopKeywordsHelp")}</p>
  </Card>
 {/if}

 <Card title={t("dealsFilters")}>
  {#if mirror}
   <fieldset class="mt-5"><legend class="font-semibold">{t("dealsMirrorStores")}</legend><p class="text-sm text-neutral-500">{t("dealsMirrorStoresHelp")}</p>
    <div class="mt-3 flex flex-wrap gap-2">
     {#each stores as store(store.id)}
      {@const on=keep.includes(store.id)}
      <button type="button" role="checkbox" aria-checked={on} onclick={()=>toggleStore(store.id)} class="inline-flex items-center gap-2 rounded-full border py-1 pr-3 pl-1 text-sm font-semibold transition-colors {on?'border-blup bg-orange-50 dark:bg-orange-950/40':'border-neutral-300 hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800'}"><StoreLogo store={store.id} class="size-6 !rounded-full"/>{storeName[store.id]}{#if on}<Check class="size-3.5 text-blup"/>{/if}</button>
     {/each}
    </div>
   </fieldset>
   <p class="mt-4 text-sm text-neutral-500">{t("dealsMirrorHow")}</p>
  {/if}
  {#if options.length}<fieldset class="mt-5"><legend class="font-semibold">{t("dealsPlatforms")}</legend><p class="text-sm text-neutral-500">{t("dealsPlatformsHelp")}</p>
   <div class="mt-3 flex flex-wrap gap-2">
    {#each options as platform(platform.id)}
     {@const on=picked.includes(platform.id)}
     <button type="button" role="checkbox" aria-checked={on} onclick={()=>togglePlatform(platform.id)} class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-semibold transition-colors {on?'border-blup bg-blup text-white':'border-neutral-300 hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800'}">{#if on}<Check class="size-3.5"/>{/if}{platform.label}</button>
    {/each}
   </div>
  </fieldset>{/if}
  {#if kind==="nuuvem_deals"}
   <div class="mt-5 max-w-sm"><Select label={t("dealsNuuvemCountry")} surface="bg-white dark:bg-neutral-900" options={nuuvemOptions} bind:value={country}/></div>
   <p class="mt-1.5 text-sm text-neutral-500">{reais?t("dealsNuuvemCountryBr"):t("dealsNuuvemCountryOther")}</p>
   <div class="mt-5"><Switch label={t("dealsGamesOnly")} tone="coral" bind:checked={gamesOnly}/><p class="mt-1 text-sm text-neutral-500">{t("dealsGamesOnlyHelp")}</p></div>
  {/if}
  {#if priced}
   <div class="mt-6 grid gap-4 sm:grid-cols-2">
    <FloatField id="hunt-discount" surface="bg-white dark:bg-neutral-900" label={t("dealsMinDiscount")} type="number" inputmode="numeric" min={0} max={100} step={1} required bind:value={minDiscount}/>
    {#if reais}<FloatField id="hunt-price" surface="bg-white dark:bg-neutral-900" label={shopping?t("dealsMaxPriceIn",{currency:currencyName(currency,intlLocale())}):t("dealsMaxPriceBrl")} type="text" inputmode={digits?"decimal":"numeric"} pattern={digits?`[0-9]+([.,][0-9]{1,${digits}})?`:"[0-9]+"} bind:value={maxPrice}/>{/if}
   </div>
  {/if}
  {#if priced}<p class="mt-4 text-sm text-neutral-500">{shopping?t("dealsShopHow"):t("dealsNuuvemDaily")}</p>{/if}
 </Card>

 <Card title={t("dealsPickChannels")} help={t("dealsPickChannelsHelp")}>
  <div class="mt-5"><ChannelPicker {channels} bind:selected={channelIds}/></div>
 </Card>

 <div class="flex justify-end"><Button {busy} disabled={!valid}>{submitLabel}</Button></div>
</form>
