<script lang="ts">
 import type {PageProps} from "./$types";
 import type {Component} from "svelte";
 import {resolve} from "$app/paths";
 import {goto,invalidateAll} from "$app/navigation";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import Hash from "@lucide/svelte/icons/hash";
 import Send from "@lucide/svelte/icons/send";
 import Handshake from "@lucide/svelte/icons/handshake";
 import Eye from "@lucide/svelte/icons/eye";
 import Power from "@lucide/svelte/icons/power";
 import Gamepad2 from "@lucide/svelte/icons/gamepad-2";
 import ShoppingBag from "@lucide/svelte/icons/shopping-bag";
 import Plus from "@lucide/svelte/icons/plus";
 import {t} from "#lib/i18n.svelte.ts";
 import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import {dealsApi,parseResponse,type HuntKind} from "../../lib/api";
 import {dealsError} from "../../lib/errors";
 import {refreshUsage} from "../../lib/panel.svelte";
 import {hasPlan} from "../../lib/plan";
 import ChannelPicker from "../../lib/ChannelPicker.svelte";
 import PostPreview from "../../lib/PostPreview.svelte";
 import StoreForm from "../../lib/StoreForm.svelte";
 import StoreLogo from "../../lib/StoreLogo.svelte";
 import {storeName} from "../../lib/catalog";
 // The first-run assistant: the channel's topic, where to post, the
 // affiliate IDs (can be skipped), a preview and turning it on. With games
 // as the topic, turning on creates the game hunts; another topic goes on
 // to pasting the first link. Linking a channel leaves for its page and
 // comes back to the same step.
 let {data}:PageProps=$props();
 type Step={id:"topic"|"channel"|"ids"|"preview"|"on";title:PlainMessageKey;Icon:Component<{class?:string}>};
 const steps:Step[]=[
  {id:"topic",title:"dealsStepTopic",Icon:Hash},
  {id:"channel",title:"dealsStepChannel",Icon:Send},
  {id:"ids",title:"dealsStepIds",Icon:Handshake},
  {id:"preview",title:"dealsStepPreview",Icon:Eye},
  {id:"on",title:"dealsStepOn",Icon:Power},
 ];
 const texts:Record<Step["id"],PlainMessageKey>={topic:"dealsStepTopicText",channel:"dealsStepChannelText",ids:"dealsStepIdsText",preview:"dealsStepPreviewText",on:"dealsStepOnText"};
 // Where it starts: the step and topic in the address (back from linking).
 const first=(()=>({step:data.step,topic:data.topic}))();
 let at=$state(first.step),topic=$state<"games"|"other">(first.topic),busy=$state(false);
 let kinds=$state<Record<HuntKind,boolean>>({games_free:true,games_deals:true});
 // Starts on the active channels; a tick changes it.
 let channelIds=$derived(data.channels.filter(c=>c.active).map(c=>c.id));
 const step=$derived(steps[at]);
 const canNext=$derived(step.id!=="channel"||channelIds.length>0);
 const linkHref=$derived(`${resolve("/channels/new")}?${new URLSearchParams({next:`/start?${new URLSearchParams({step:"1",topic})}`})}`);
 let heading:HTMLHeadingElement|undefined=$state();
 // Moving between steps puts the focus on the new step's title.
 function go(to:number){at=Math.max(0,Math.min(steps.length-1,to));queueMicrotask(()=>heading?.focus());}
 async function turnOn(){busy=true;try{
  for(const kind of (["games_free","games_deals"] as const).filter(k=>kinds[k]))await parseResponse(dealsApi.hunts.$post({json:{kind,channelIds,filters:kind==="games_deals"?{minDiscount:50}:{}}}));
  void refreshUsage();toast(()=>t("dealsStartDone"));await goto(resolve("/hunts"));
 }catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 const topics=[{id:"games" as const,title:"dealsTopicGames" as const,help:"dealsTopicGamesHelp" as const,Icon:Gamepad2},{id:"other" as const,title:"dealsTopicOther" as const,help:"dealsTopicOtherHelp" as const,Icon:ShoppingBag}];
</script>

<div class="mb-4"><BackLink href={resolve("/")} label={t("dealsOffers")}/></div>
<PageHeader title={t("dealsStartTitle")} lead={t("dealsStartLead")}/>

<ol class="mt-8 flex flex-wrap gap-2" aria-label={t("dealsStartTitle")}>
 {#each steps as s,i(s.id)}
  <li><button type="button" aria-current={i===at?"step":undefined} disabled={i>at&&!canNext} onclick={()=>go(i)} class="flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-semibold disabled:opacity-50 {i===at?'border-blup bg-blup text-white':i<at?'border-neutral-300 text-blup dark:border-neutral-700':'border-neutral-300 text-neutral-500 dark:border-neutral-700'}">
   {#if i<at}<CircleCheck class="size-4"/>{:else}<span class="tabular-nums">{i+1}</span>{/if}{t(s.title)}
  </button></li>
 {/each}
</ol>

<div class="mt-6"><Card>
 <div class="flex items-start gap-4">
  <span class="hidden size-12 shrink-0 place-items-center rounded-2xl border border-neutral-200 text-blup sm:grid dark:border-neutral-700" aria-hidden="true"><step.Icon class="size-6"/></span>
  <div class="min-w-0 flex-1">
   <p class="text-sm font-semibold text-neutral-500">{t("dealsStepOf",{step:String(at+1),total:String(steps.length)})}</p>
   <h2 bind:this={heading} tabindex="-1" class="mt-1 text-xl font-bold outline-none">{t(step.title)}</h2>
   <p class="mt-2 text-neutral-500">{t(texts[step.id])}</p>

   <div class="mt-5">
    {#if step.id==="topic"}
     <div class="grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label={t("dealsStepTopic")}>
      {#each topics as option(option.id)}
       <button type="button" role="radio" aria-checked={topic===option.id} onclick={()=>topic=option.id} class="flex items-start gap-3 rounded-xl border p-4 text-left transition-colors {topic===option.id?'border-blup bg-orange-50/60 dark:bg-orange-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}">
        <option.Icon class="mt-0.5 size-6 shrink-0 text-blup"/><span class="min-w-0"><span class="block font-semibold">{t(option.title)}</span><span class="block text-sm text-neutral-500">{t(option.help)}</span></span>
       </button>
      {/each}
     </div>
    {:else if step.id==="channel"}
     <ChannelPicker channels={data.channels} bind:selected={channelIds}/>
     {#if !data.channels.length}<p class="mt-2 text-sm text-neutral-500">{t("dealsStartNoChannel")}</p>{/if}
     {#if hasPlan()}<a class="mt-4 inline-flex items-center gap-2 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800" href={linkHref}><Plus class="size-4"/>{t("dealsNewChannel")}</a>{/if}
    {:else if step.id==="ids"}
     <div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-2">
      {#each ["amazon","mercadolivre"] as const as store(store)}
       {@const saved=data.stores.find(s=>s.store===store)}
       <section class="rounded-xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
        <div class="mb-4 flex items-center gap-3"><StoreLogo {store} class="size-10"/><h3 class="font-semibold">{storeName[store]}</h3>{#if saved?.configured}<span class="ml-auto inline-flex items-center gap-1 text-sm font-medium text-emerald-700 dark:text-emerald-400"><CircleCheck class="size-4"/>{t("dealsConfigured")}</span>{/if}</div>
        <StoreForm {store} {saved} writable={hasPlan()} onchange={()=>void invalidateAll()}/>
       </section>
      {/each}
     </div>
    {:else if step.id==="preview"}
     {#if topic==="games"}
      <p class="text-sm font-semibold">{t("dealsStartPickKinds")}</p>
      <div class="mt-3 grid gap-3">
       <Switch label={t("dealsGamesFree")} tone="coral" bind:checked={kinds.games_free}/>
       <Switch label={t("dealsGamesDeals")} tone="coral" bind:checked={kinds.games_deals}/>
      </div>
      <div class="mt-5"><PostPreview example title={t("dealsSampleTitle")} store="GamerPower" affiliate={false}/></div>
     {:else}<p class="text-sm">{t("dealsStartOtherPreview")}</p>{/if}
    {:else}
     {#if topic==="games"}<Button {busy} disabled={!hasPlan()||!channelIds.length||(!kinds.games_free&&!kinds.games_deals)} onclick={turnOn}><Power class="size-4"/>{t("dealsTurnOn")}</Button>
     {:else}<Button disabled={!hasPlan()} onclick={()=>goto(resolve("/offers/new"))}>{t("dealsStartPasteFirst")}</Button>{/if}
    {/if}
   </div>
  </div>
 </div>
 <div class="mt-6 flex flex-wrap justify-end gap-2 border-t border-neutral-200 pt-5 dark:border-neutral-800">
  {#if at>0}<Button variant="secondary" onclick={()=>go(at-1)}>{t("back")}</Button>{/if}
  {#if step.id==="ids"}<Button variant="secondary" onclick={()=>go(at+1)}>{t("dealsSkip")}</Button>{/if}
  {#if at<steps.length-1}<Button disabled={!canNext} onclick={()=>go(at+1)}>{t("next")}</Button>{/if}
 </div>
</Card></div>
