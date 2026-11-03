<script lang="ts">
 import {onDestroy} from "svelte";
 import {goto} from "$app/navigation";
 import {type InferResponseType} from "hono/client";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import BrandIcon from "#lib/ui/BrandIcon.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 // A new alert destination: pick Discord or Telegram, then paste the
 // Discord webhook or link a Telegram chat through the bot. Once added,
 // back to the list.
 const linkGET=pulseApi.telegram.link[":link"].$get;
 type LinkStatus=InferResponseType<typeof linkGET,200>;
 type Kind="discord"|"telegram";
 let kind=$state<Kind>("discord"),webhook=$state(""),busy=$state(false),link=$state<{id:string;url:string;expiresAt:string}|null>(null),linkStatus=$state<LinkStatus|null>(null);
 const kinds=$derived([{value:"discord" as const,label:"Discord",help:t("pulseDiscordHow")},{value:"telegram" as const,label:"Telegram",help:t("pulseTelegramHow")}]);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function done(){toast(()=>t("pulseToastLinked"));await goto("/avisos");}
 async function discord(e:SubmitEvent){e.preventDefault();await run(async()=>{await parseResponse(pulseApi.channels.$post({json:{type:"discord",webhook}}));webhook="";await done();});}
 async function telegram(){await run(async()=>{link=await parseResponse(pulseApi.telegram.link.$post({json:{}}));linkStatus=null;if(link)startPolling(link);});}
 async function confirmChat(){const current=link;if(!current)return;await run(async()=>{await parseResponse(pulseApi.telegram.link[":link"].confirm.$post({param:{link:current.id},json:{}}));stopPolling();link=null;linkStatus=null;await done();});}
 // While the bot waits for the chat, its state is asked every few seconds.
 let pollTimer:ReturnType<typeof setInterval>|undefined,pollEpoch=0;
 function stopPolling(){pollEpoch++;if(pollTimer!==undefined){clearInterval(pollTimer);pollTimer=undefined;}}
 function startPolling(current:{id:string}){
  stopPolling();const epoch=pollEpoch;let inFlight=false,attempts=0;
  const poll=async()=>{if(inFlight||epoch!==pollEpoch)return;if(++attempts>120){stopPolling();return;}inFlight=true;
   try{const value=await parseResponse(linkGET({param:{link:current.id}},{init:{signal:AbortSignal.timeout(15000)}}));if(epoch===pollEpoch&&link?.id===current.id){linkStatus=value;if(value.state!=="pending")stopPolling();}}
   catch(e){if(epoch===pollEpoch)toast(errorMessage(e),{kind:"error"});}finally{inFlight=false;}
  };
  void poll();pollTimer=setInterval(()=>{void poll();},5000);
 }
 onDestroy(stopPolling);
 function choose(next:Kind){if(next===kind)return;kind=next;stopPolling();link=null;linkStatus=null;}
</script>

<div class="mb-4"><BackLink href="/avisos" label={t("pulseAlerts")}/></div>
<PageHeader title={t("pulseNewChannelTitle")} lead={t("pulseChannelPrivacy")}/>

<div class="mt-8"><Card title={t("pulseChooseChannel")}>
 <div class="mt-5 grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label={t("pulseChooseChannel")}>
  {#each kinds as option(option.value)}
   <button type="button" role="radio" aria-checked={kind===option.value} class="flex items-center gap-4 rounded-xl border p-4 text-left transition-colors {kind===option.value?'border-blup bg-emerald-50/60 dark:bg-emerald-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}" onclick={()=>choose(option.value)}>
    <BrandIcon name={option.value} class="size-8"/><span class="min-w-0"><span class="block font-semibold">{option.label}</span><span class="block text-sm text-neutral-500">{option.help}</span></span>
   </button>
  {/each}
 </div>

 <div class="mt-6 border-t border-neutral-200 pt-6 dark:border-neutral-800">
  {#if kind==="discord"}
   <form class="grid gap-4" onsubmit={discord}>
    <p class="text-sm text-neutral-500">{t("pulseDiscordHelp")}</p>
    <FloatField id="discord-webhook" surface="bg-white dark:bg-neutral-900" label={t("pulseDiscordWebhook")} type="password" autocomplete="off" maxlength={2048} required bind:value={webhook}/>
    <div class="flex justify-end gap-2"><Button type="button" variant="secondary" onclick={()=>goto("/avisos")}>{t("cancel")}</Button><Button {busy}><BrandIcon name="discord" class="size-4"/>{t("pulseAddDiscord")}</Button></div>
   </form>
  {:else}
   <p class="text-sm text-neutral-500">{t("pulseLinkHelp")}</p>
   {#if link}
    <div class="mt-4"><a class="inline-flex items-center gap-2 font-semibold text-blup underline" href={link.url} target="_blank" rel="noopener noreferrer"><BrandIcon name="telegram" class="size-4"/>{t("pulseOpenBot")}</a><p class="mt-2 text-xs text-neutral-500">{t("pulseLinkDeadline",{date:formatDate(link.expiresAt)})}</p>
     {#if linkStatus?.candidate}<p class="my-3 break-all text-sm">{linkStatus.candidate.label} · ID: {linkStatus.candidate.id}</p><Button onclick={confirmChat} {busy}>{t("pulseConfirmChat")}</Button>{:else if linkStatus?.state==="expired"}<p class="mt-3 text-sm">{t("pulseLinkExpired")}</p>{:else}<p class="mt-3 text-sm text-neutral-500">{t("pulseWaitingChat")}</p>{/if}
    </div>
   {:else}<div class="mt-4 flex justify-end gap-2"><Button type="button" variant="secondary" onclick={()=>goto("/avisos")}>{t("cancel")}</Button><Button type="button" onclick={telegram} {busy}><BrandIcon name="telegram" class="size-4"/>{t("pulseLinkTelegram")}</Button></div>{/if}
  {/if}
 </div>
</Card></div>
