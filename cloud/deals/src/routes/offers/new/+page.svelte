<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Eye from "@lucide/svelte/icons/eye";
 import Send from "@lucide/svelte/icons/send";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
 import {t} from "#lib/i18n.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import {dealsApi,parseResponse,type Preview} from "../../../lib/api";
 import {dealsError} from "../../../lib/errors";
 import {refreshUsage} from "../../../lib/panel.svelte";
 import {hasPlan} from "../../../lib/plan";
 import {storeLabel} from "../../../lib/catalog";
 import ChannelPicker from "../../../lib/ChannelPicker.svelte";
 import SendWhen,{type SendMode} from "../../../lib/SendWhen.svelte";
 import PostExtras from "../../../lib/PostExtras.svelte";
 import {hasAddress} from "../../../../../shared/deals-post";
 import PostPreview from "../../../lib/PostPreview.svelte";
 // Paste a product link: the server reads the page and swaps in the
 // affiliate ID; the preview shows the post as it will look, then the
 // channels and, if wanted, when to send. Changing the link asks for a
 // new preview.
 let {data}:PageProps=$props();
 let url=$state(""),preview=$state<Preview|null>(null),previewOf=$state(""),busy=$state(false);
 let description=$state(""),coupon=$state("");
 const extrasBad=$derived(hasAddress(description)||hasAddress(coupon));
 let mode=$state<SendMode>("now"),sendAt=$state(Math.ceil((Date.now()+3_600_000)/900_000)*900_000);
 // Starts on the active channels; a tick changes it.
 let channelIds=$derived(data.channels.filter(c=>c.active).map(c=>c.id));
 const stale=$derived(!!preview&&previewOf!==url.trim());
 const reason=$derived(preview?.reason==="not_configured"?t("dealsReasonNotConfigured"):preview?.reason==="unsupported_store"?t("dealsReasonUnsupported"):preview?.reason==="conversion_failed"?t("dealsReasonFailed"):null);
 // The server takes a time from now up to 30 days ahead.
 const minSend=$derived(Date.now()+60_000),maxSend=$derived(Date.now()+30*86_400_000);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function look(e:SubmitEvent){e.preventDefault();const target=url.trim();if(!target)return;await run(async()=>{preview=await parseResponse(dealsApi.offers.preview.$post({json:{url:target}}));previewOf=target;});}
 async function queue(){if(!preview||stale||!channelIds.length)return;await run(async()=>{
  await parseResponse(dealsApi.offers.$post({json:{url:previewOf,channelIds,...(description.trim()?{description:description.trim()}:{}),...(coupon.trim()?{coupon:coupon.trim()}:{}),...(mode==="schedule"?{sendAt:new Date(sendAt).toISOString()}:mode==="queue"?{queue:true}:{})}}));
  void refreshUsage();toast(()=>t("dealsQueuedToast"));await goto(resolve("/"));
 });}
</script>

<div class="mb-4"><BackLink href={resolve("/")} label={t("dealsOffers")}/></div>
<PageHeader title={t("dealsPasteTitle")} lead={t("dealsPasteLead")}/>

<form class="mt-8" onsubmit={look}><Card>
 <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
  <FloatField id="offer-url" surface="bg-white dark:bg-neutral-900" label={t("dealsLinkField")} type="url" inputmode="url" autocomplete="off" maxlength={2048} required bind:value={url}/>
  <Button {busy} class="h-14" disabled={!url.trim()||!hasPlan()}><Eye class="size-4"/>{t("dealsSeePreview")}</Button>
 </div>
</Card></form>

{#if preview&&!stale}
 <div class="mt-6"><Card title={t("dealsPreviewTitle")}>
  <div class="mt-5 grid gap-6 md:grid-cols-[minmax(0,24rem)_1fr] md:items-start">
   <PostPreview title={preview.title} image={preview.image} price={preview.price} currency={preview.currency} store={storeLabel(preview.store)} affiliate={preview.affiliate} description={description.trim()||null} coupon={coupon.trim()||null} style={data.style}/>
   <div class="grid gap-3 text-sm">
    {#if preview.affiliate}<p class="flex items-start gap-2 font-semibold text-emerald-700 dark:text-emerald-400"><CircleCheck class="mt-0.5 size-4 shrink-0"/>{t("dealsWithAffiliate")}</p>
    {:else if reason}<p class="flex items-start gap-2 text-amber-800 dark:text-amber-300"><TriangleAlert class="mt-0.5 size-4 shrink-0"/><span>{reason}{#if preview.reason==="not_configured"} <a class="font-semibold underline" href={resolve("/affiliates")}>{t("dealsAffiliates")}</a>{/if}</span></p>{/if}
    <PostExtras title={preview.title} store={storeLabel(preview.store)} price={preview.price} currency={preview.currency} bind:description bind:coupon/>
   </div>
  </div>
 </Card></div>

 <div class="mt-6"><Card title={t("dealsPickChannels")} help={t("dealsPickChannelsHelp")}>
  <div class="mt-5"><ChannelPicker channels={data.channels} bind:selected={channelIds}/></div>
  {#if !data.channels.length}<a class="mt-3 inline-block text-sm font-semibold text-blup underline" href={resolve("/channels/new")}>{t("dealsNewChannel")}</a>{/if}
  <div class="mt-6 border-t border-neutral-200 pt-6 dark:border-neutral-800">
   <SendWhen bind:mode bind:sendAt min={minSend} max={maxSend} pace={data.pace}/>
  </div>
  <div class="mt-6 flex justify-end"><Button {busy} disabled={extrasBad||!channelIds.length||(mode==="schedule"&&(sendAt<minSend||sendAt>maxSend))||!hasPlan()} onclick={queue}><Send class="size-4"/>{mode==="now"?t("dealsWhenNow"):mode==="queue"?t("dealsWhenQueue"):t("dealsSchedule")}</Button></div>
 </Card></div>
{/if}
