<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {onDestroy} from "svelte";
 import {goto} from "$app/navigation";
 import {type InferResponseType} from "hono/client";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import BrandIcon from "#lib/ui/BrandIcon.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import CodeBlock from "#lib/ui/CodeBlock.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 // A new alert destination: pick the project it is for and the kind, then
 // paste the Discord or Slack webhook, give a webhook address (its signing
 // secret is shown once) or link a Telegram chat through the bot. Once
 // added, to the project's alerts.
 let {data}:PageProps=$props();
 const projectOptions=$derived(data.projects.map(p=>({value:p.id,label:p.name})));
 // Starts on the project asked for in the address, else the first one;
 // another visit starts again.
 let projectId=$derived(data.projects.find(p=>p.id===data.wanted)?.id??data.projects[0]?.id??"");
 const linkGET=pulseApi.telegram.link[":link"].$get;
 type LinkStatus=InferResponseType<typeof linkGET,200>;
 type Link=InferResponseType<typeof pulseApi.telegram.link.$post,201>;
 type Kind="discord"|"telegram"|"slack"|"webhook";
 // The signing secret of a webhook just made: shown once, here.
 let kind=$state<Kind>("discord"),webhook=$state(""),secret=$state<string|null>(null),busy=$state(false),link=$state<Link|null>(null),linkStatus=$state<LinkStatus|null>(null);
 const kinds=$derived([{value:"discord" as const,label:"Discord",help:t("pulseDiscordHow")},{value:"telegram" as const,label:"Telegram",help:t("pulseTelegramHow")},{value:"slack" as const,label:"Slack",help:t("pulseSlackHow")},{value:"webhook" as const,label:"Webhook",help:t("pulseWebhookHow")}]);
 // How the person's system checks a request came from Pulse (Node).
 const verifySample=`// X-Blup-Signature: t=<seconds>,v1=<hex>
const [t, v1] = header.split(",").map((p) => p.split("=")[1]);
const mac = crypto.createHmac("sha256", process.env.PULSE_WEBHOOK_SECRET).update(\`\${t}.\${rawBody}\`).digest("hex");
const ok = crypto.timingSafeEqual(Buffer.from(mac), Buffer.from(v1)) && Date.now() / 1000 - Number(t) < 300;`;
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 // Back to where the destination is: the project's alerts (or, before one
 // is made, where the page was opened from).
 const back=$derived(data.wanted?resolve("/alerts/[project]",{project:data.wanted}):resolve("/alerts"));
 async function done(){toast(()=>t("pulseToastLinked"));await goto(resolve("/alerts/[project]",{project:projectId}));}
 // A destination by a pasted address; a webhook stays on this page to show
 // its secret once.
 async function pasted(e:SubmitEvent){e.preventDefault();await run(async()=>{
  if(kind==="webhook"){const made=await parseResponse(pulseApi.channels.$post({json:{projectId,type:"webhook",url:webhook}}));webhook="";secret=made.secret;toast(()=>t("pulseToastLinked"));return;}
  if(kind!=="discord"&&kind!=="slack")return;
  await parseResponse(pulseApi.channels.$post({json:{projectId,type:kind,webhook}}));webhook="";await done();
 });}
 async function telegram(){await run(async()=>{link=await parseResponse(pulseApi.telegram.link.$post({json:{}}));linkStatus=null;if(link)startPolling(link);});}
 async function confirmChat(){const current=link;if(!current)return;await run(async()=>{await parseResponse(pulseApi.telegram.link[":link"].confirm.$post({param:{link:current.id},json:{projectId}}));stopPolling();link=null;linkStatus=null;await done();});}
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
 function choose(next:Kind){if(next===kind)return;kind=next;webhook="";stopPolling();link=null;linkStatus=null;}
</script>

<div class="mb-4"><BackLink href={back} label={t("pulseAlerts")}/></div>
<PageHeader title={t("pulseNewChannelTitle")} lead={t("pulseChannelPrivacy")}/>

<div class="mt-8"><Card title={t("pulseChannelProject")} help={t("pulseChannelProjectHelp")}>
 <div class="mt-5">{#if projectOptions.length}<Select surface="bg-white dark:bg-neutral-900" label={t("pulseAllProjects")} options={projectOptions} disabled={!!link||!!secret} bind:value={projectId}/>{:else}<p class="text-sm text-neutral-500">{t("pulseEmpty")}</p>{/if}</div>
</Card></div>

<div class="mt-6"><Card title={t("pulseChooseChannel")}>
 <div class="mt-5 grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label={t("pulseChooseChannel")}>
  {#each kinds as option(option.value)}
   <button type="button" role="radio" aria-checked={kind===option.value} class="flex items-center gap-4 rounded-xl border p-4 text-left transition-colors {kind===option.value?'border-blup bg-emerald-50/60 dark:bg-emerald-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}" disabled={!!secret} onclick={()=>choose(option.value)}>
    <ChannelIcon type={option.value} class="size-8"/><span class="min-w-0"><span class="block font-semibold">{option.label}</span><span class="block text-sm text-neutral-500">{option.help}</span></span>
   </button>
  {/each}
 </div>

 <div class="mt-6 border-t border-neutral-200 pt-6 dark:border-neutral-800">
  {#if secret}
   <div class="grid grid-cols-[minmax(0,1fr)] gap-4">
    <p class="font-semibold">{t("pulseWebhookReady")}</p>
    <CodeBlock code={secret}/>
    <p class="text-sm text-neutral-500">{t("pulseWebhookSecretOnce")}</p>
    <p class="text-sm">{t("pulseWebhookVerify")}</p>
    <CodeBlock code={verifySample}/>
    <div class="flex justify-end"><Button onclick={()=>goto(resolve("/alerts/[project]",{project:projectId}))}>{t("pulseDone")}</Button></div>
   </div>
  {:else if kind==="discord"||kind==="slack"}
   <form class="grid grid-cols-[minmax(0,1fr)] gap-4" onsubmit={pasted}>
    <p class="text-sm text-neutral-500">{kind==="discord"?t("pulseDiscordHelp"):t("pulseSlackHelp")}</p>
    <FloatField id="channel-webhook" surface="bg-white dark:bg-neutral-900" label={kind==="discord"?t("pulseDiscordWebhook"):t("pulseSlackWebhook")} type="password" autocomplete="off" maxlength={2048} required bind:value={webhook}/>
    <div class="flex justify-end gap-2"><Button {busy} disabled={!projectId}>{kind==="discord"?t("pulseAddDiscord"):t("pulseAddSlack")}</Button></div>
   </form>
  {:else if kind==="webhook"}
   <form class="grid grid-cols-[minmax(0,1fr)] gap-4" onsubmit={pasted}>
    <p class="text-sm text-neutral-500">{t("pulseWebhookHelp")}</p>
    <FloatField id="channel-webhook" surface="bg-white dark:bg-neutral-900" label={t("pulseWebhookURL")} type="url" autocomplete="off" maxlength={2048} required bind:value={webhook}/>
    <div class="flex justify-end gap-2"><Button {busy} disabled={!projectId}>{t("pulseAddWebhook")}</Button></div>
   </form>
  {:else}
   <p class="text-sm text-neutral-500">{t("pulseLinkHelp")}</p>
   {#if link}
    <div class="mt-4"><a class="inline-flex items-center gap-2 font-semibold text-blup underline" href={link.url} target="_blank" rel="noopener noreferrer"><BrandIcon name="telegram" class="size-4"/>{t("pulseOpenBot")}</a><p class="mt-2 text-xs text-neutral-500">{t("pulseLinkDeadline",{date:formatDate(link.expiresAt)})}</p>
     {#if linkStatus?.candidate}<p class="my-3 break-all text-sm">{linkStatus.candidate.label} · ID: {linkStatus.candidate.id}</p><Button onclick={confirmChat} {busy}>{t("pulseConfirmChat")}</Button>{:else if linkStatus?.state==="expired"}<p class="mt-3 text-sm">{t("pulseLinkExpired")}</p>{:else}<p class="mt-3 text-sm text-neutral-500">{t("pulseWaitingChat")}</p>{/if}
    </div>
   {:else}<div class="mt-4 flex justify-end gap-2"><Button type="button" onclick={telegram} {busy} disabled={!projectId}>{t("pulseLinkTelegram")}</Button></div>{/if}
  {/if}
 </div>
</Card></div>
