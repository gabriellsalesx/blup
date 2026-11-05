<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {onDestroy,untrack} from "svelte";
 import {goto} from "$app/navigation";
 import {type InferResponseType} from "hono/client";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import BrandIcon from "#lib/ui/BrandIcon.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import {dealsApi,parseResponse} from "../../../lib/api";
 import {dealsError} from "../../../lib/errors";
 import {refreshUsage} from "../../../lib/panel.svelte";
 import {hasPlan} from "../../../lib/plan";
 import WhatsappConnect from "../../../lib/WhatsappConnect.svelte";
 // A new destination, as Pulse's alerts: paste a Discord channel's
 // webhook, link a Telegram channel or group through the Deals bot (the
 // bot is added there as an admin, then the channel is confirmed here), or
 // connect a WhatsApp by QR and pick its Canais (?kind=whatsapp opens it).
 let {data}:PageProps=$props();
 const linkGET=dealsApi.telegram.link[":link"].$get;
 type LinkStatus=InferResponseType<typeof linkGET,200>;
 type Link=InferResponseType<typeof dealsApi.telegram.link.$post,201>;
 type Kind="telegram"|"discord"|"whatsapp";
 let kind=$state<Kind>(untrack(()=>data.kind)),webhook=$state(""),busy=$state(false),link=$state<Link|null>(null),linkStatus=$state<LinkStatus|null>(null);
 const kinds=$derived([{value:"telegram" as const,label:"Telegram",help:t("dealsTelegramHow")},{value:"discord" as const,label:"Discord",help:t("dealsDiscordHow")},{value:"whatsapp" as const,label:"WhatsApp",help:t("dealsWhatsappHow")}]);
 const back=$derived(data.next??resolve("/channels"));
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function done(){void refreshUsage();toast(()=>t("dealsLinked"));await goto(back);}
 async function discord(e:SubmitEvent){e.preventDefault();await run(async()=>{await parseResponse(dealsApi.channels.$post({json:{type:"discord",webhook:webhook.trim()}}));webhook="";await done();});}
 async function telegram(){await run(async()=>{link=await parseResponse(dealsApi.telegram.link.$post({json:{}}));linkStatus=null;startPolling(link);});}
 async function confirmChat(){const current=link;if(!current)return;await run(async()=>{await parseResponse(dealsApi.telegram.link[":link"].confirm.$post({param:{link:current.id},json:{}}));stopPolling();link=null;linkStatus=null;await done();});}
 // While the bot waits to be added to the channel, its state is asked
 // every few seconds (for ten minutes at most).
 let pollTimer:ReturnType<typeof setInterval>|undefined,pollEpoch=0;
 function stopPolling(){pollEpoch++;if(pollTimer!==undefined){clearInterval(pollTimer);pollTimer=undefined;}}
 function startPolling(current:{id:string}){
  stopPolling();const epoch=pollEpoch;let inFlight=false,attempts=0;
  const poll=async()=>{if(inFlight||epoch!==pollEpoch)return;if(++attempts>120){stopPolling();return;}inFlight=true;
   try{const value=await parseResponse(linkGET({param:{link:current.id}},{init:{signal:AbortSignal.timeout(15000)}}));if(epoch===pollEpoch&&link?.id===current.id){linkStatus=value;if(value.state!=="pending")stopPolling();}}
   catch(e){if(epoch===pollEpoch)toast(dealsError(e),{kind:"error"});}finally{inFlight=false;}
  };
  void poll();pollTimer=setInterval(()=>{void poll();},5000);
 }
 onDestroy(stopPolling);
 function choose(next:Kind){if(next===kind)return;kind=next;webhook="";stopPolling();link=null;linkStatus=null;}
</script>

<div class="mb-4"><BackLink href={back} label={data.next?t("dealsStart"):t("dealsChannels")}/></div>
<PageHeader title={t("dealsNewChannelTitle")} lead={t("dealsChannelsPrivacy")}/>

<div class="mt-8"><Card title={t("dealsChooseKind")}>
 <div class="mt-5 grid gap-3 sm:grid-cols-3" role="radiogroup" aria-label={t("dealsChooseKind")}>
  {#each kinds as option(option.value)}
   <button type="button" role="radio" aria-checked={kind===option.value} class="flex items-center gap-4 rounded-xl border p-4 text-left transition-colors {kind===option.value?'border-blup bg-orange-50/60 dark:bg-orange-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}" onclick={()=>choose(option.value)}>
    <ChannelIcon type={option.value} class="size-8"/><span class="min-w-0"><span class="block font-semibold">{option.label}</span><span class="block text-sm text-neutral-500">{option.help}</span></span>
   </button>
  {/each}
 </div>

 <div class="mt-6 border-t border-neutral-200 pt-6 dark:border-neutral-800">
  {#if kind==="whatsapp"}
   <WhatsappConnect onadded={done}/>
  {:else if kind==="discord"}
   <form class="grid grid-cols-[minmax(0,1fr)] gap-4" onsubmit={discord}>
    <p class="text-sm text-neutral-500">{t("dealsDiscordHelp")}</p>
    <FloatField id="channel-webhook" surface="bg-white dark:bg-neutral-900" label={t("dealsDiscordWebhook")} type="password" autocomplete="off" maxlength={2048} required bind:value={webhook}/>
    <div class="flex justify-end"><Button {busy} disabled={!hasPlan()}>{t("dealsAddDiscord")}</Button></div>
   </form>
  {:else}
   <ol class="grid list-decimal gap-1.5 pl-5 text-sm text-neutral-600 dark:text-neutral-300">
    <li>{t("dealsTelegramStep1")}</li><li>{t("dealsTelegramStep2")}</li><li>{t("dealsTelegramStep3")}</li>
   </ol>
   {#if link}
    <div class="mt-5" aria-live="polite">
     <a class="inline-flex items-center gap-2 font-semibold text-blup underline" href={link.url} target="_blank" rel="noopener noreferrer"><BrandIcon name="telegram" class="size-4"/>{t("dealsOpenBot")}</a>
     <p class="mt-2 text-xs text-neutral-500">{t("dealsLinkDeadline",{date:formatDate(link.expiresAt)})}</p>
     {#if linkStatus?.candidate}<p class="my-3 text-sm break-all"><span class="font-semibold">{linkStatus.candidate.label}</span> · ID: {linkStatus.candidate.id}</p><Button onclick={confirmChat} {busy}>{t("dealsConfirmChat")}</Button>
     {:else if linkStatus?.state==="expired"}<p class="mt-3 text-sm">{t("dealsLinkExpired")}</p><Button class="mt-3" variant="secondary" onclick={telegram} {busy}>{t("dealsNewLink")}</Button>
     {:else}<p class="mt-3 text-sm text-neutral-500">{t("dealsWaitingChat")}</p>{/if}
    </div>
   {:else}<div class="mt-5 flex justify-end"><Button type="button" onclick={telegram} {busy} disabled={!hasPlan()}>{t("dealsLinkTelegram")}</Button></div>{/if}
  {/if}
 </div>
</Card></div>
