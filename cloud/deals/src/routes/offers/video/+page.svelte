<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import Search from "@lucide/svelte/icons/search";
 import Send from "@lucide/svelte/icons/send";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import Info from "@lucide/svelte/icons/info";
 import {t} from "#lib/i18n.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import DateTimeField from "#lib/ui/DateTimeField.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import {dealsApi,parseResponse,type VideoPreview} from "../../../lib/api";
 import {dealsError} from "../../../lib/errors";
 import {refreshUsage} from "../../../lib/panel.svelte";
 import {hasPlan} from "../../../lib/plan";
 import PostPreview from "../../../lib/PostPreview.svelte";
 import ChannelPicker from "../../../lib/ChannelPicker.svelte";
 // A new TikTok video about a product: the link is checked first (a public
 // video), then the person names the product and, if they want, its
 // price; the preview shows the news as the channels will see it. The
 // click goes to the video, where TikTok Shop sells. Changing the link
 // asks for a new check.
 let {data}:PageProps=$props();
 let url=$state(""),video=$state<VideoPreview|null>(null),videoOf=$state(""),busy=$state(false);
 let title=$state(""),price=$state("");
 let scheduled=$state(false),sendAt=$state(Math.ceil((Date.now()+3_600_000)/900_000)*900_000);
 let channelIds=$derived(data.channels.filter(c=>c.active).map(c=>c.id));
 const stale=$derived(!!video&&videoOf!==url.trim());
 // "89,90", "89.90" or "89" in cents; empty is no price.
 const cents=$derived.by(()=>{const v=price.trim();if(!v)return undefined;const m=/^(\d{1,7})(?:[.,](\d{1,2}))?$/.exec(v);return m?Number(m[1])*100+Number((m[2]??"0").padEnd(2,"0")):null;});
 const minSend=$derived(Date.now()+60_000),maxSend=$derived(Date.now()+30*86_400_000);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function check(e:SubmitEvent){e.preventDefault();const target=url.trim();if(!target)return;await run(async()=>{video=await parseResponse(dealsApi.offers.video.preview.$post({json:{url:target}}));videoOf=target;});}
 async function announce(){if(!video||stale||!title.trim()||cents===null||!channelIds.length)return;await run(async()=>{
  await parseResponse(dealsApi.offers.video.$post({json:{url:videoOf,title:title.trim(),...(cents!==undefined?{price:cents}:{}),channelIds,...(scheduled?{sendAt:new Date(sendAt).toISOString()}:{})}}));
  void refreshUsage();toast(()=>t("dealsVideoQueuedToast"));await goto(resolve("/"));
 });}
</script>

<div class="mb-4"><BackLink href={resolve("/")} label={t("dealsOffers")}/></div>
<PageHeader title={t("dealsVideoTitle")} lead={t("dealsVideoLead")}/>

<form class="mt-8" onsubmit={check}><Card>
 <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
  <FloatField id="video-url" surface="bg-white dark:bg-neutral-900" label={t("dealsVideoField")} type="url" inputmode="url" autocomplete="off" maxlength={2048} required bind:value={url}/>
  <Button {busy} class="h-14" disabled={!url.trim()||!hasPlan()}><Search class="size-4"/>{t("dealsCheckVideo")}</Button>
 </div>
 {#if video&&!stale}<p class="mt-3 flex items-center gap-2 text-sm font-semibold text-emerald-700 dark:text-emerald-400"><CircleCheck class="size-4 shrink-0"/>{t("dealsVideoFound",{author:video.author})}</p>{/if}
</Card></form>

{#if video&&!stale}
 <div class="mt-6"><Card title={t("dealsVideoProduct")} help={t("dealsVideoProductHelp")}>
  <div class="mt-5 grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,24rem)] md:items-start">
   <div class="grid gap-4">
    <FloatField id="video-product" surface="bg-white dark:bg-neutral-900" label={t("dealsProductName")} maxlength={200} required bind:value={title}/>
    <div>
     <FloatField id="video-price" surface="bg-white dark:bg-neutral-900" label={t("dealsProductPrice")} inputmode="decimal" autocomplete="off" maxlength={10} bind:value={price}/>
     {#if cents===null}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsPriceInvalid")}</p>{/if}
    </div>
    <p class="flex items-start gap-2 text-sm text-neutral-500"><Info class="mt-0.5 size-4 shrink-0"/>{t("dealsVideoTelegram")}</p>
   </div>
   <PostPreview title={title.trim()||t("dealsProductName")} price={cents??null} currency={cents===undefined||cents===null?null:"BRL"} store="TikTok" affiliate author={video.author}/>
  </div>
 </Card></div>

 <div class="mt-6"><Card title={t("dealsPickChannels")} help={t("dealsVideoPickHelp")}>
  <div class="mt-5"><ChannelPicker channels={data.channels} bind:selected={channelIds}/></div>
  {#if !data.channels.length}<a class="mt-3 inline-block text-sm font-semibold text-blup underline" href={resolve("/channels/new")}>{t("dealsNewChannel")}</a>{/if}
  <div class="mt-6 border-t border-neutral-200 pt-6 dark:border-neutral-800">
   <Switch label={t("dealsSchedule")} tone="coral" bind:checked={scheduled}/>
   <p class="mt-1 text-sm text-neutral-500">{t("dealsScheduleHelp")}</p>
   {#if scheduled}<div class="mt-4 max-w-sm"><DateTimeField label={t("dealsSendAt")} surface="bg-white dark:bg-neutral-900" min={minSend} max={maxSend} bind:value={sendAt}/></div>{/if}
  </div>
  <div class="mt-6 flex justify-end"><Button {busy} disabled={!title.trim()||cents===null||!channelIds.length||(scheduled&&(sendAt<minSend||sendAt>maxSend))||!hasPlan()} onclick={announce}><Send class="size-4"/>{t("dealsAnnounce")}</Button></div>
 </Card></div>
{/if}
