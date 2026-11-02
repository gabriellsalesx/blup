<script lang="ts">
 import {onMount,onDestroy} from "svelte";
 import {type InferResponseType} from "hono/client";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Send from "@lucide/svelte/icons/send";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import MessageSquare from "@lucide/svelte/icons/message-square";
 import {pulseApi,parseResponse,type Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 type Channel=InferResponseType<typeof pulseApi.channels.$get,200>["channels"][number];
 const linkGET=pulseApi.telegram.link[":link"].$get;
 type LinkStatus=InferResponseType<typeof linkGET,200>;
 let channels=$state<Channel[]>([]),projects=$state<Project[]>([]),webhook=$state(""),busy=$state(false),error=$state<unknown>(null),link=$state<{id:string;url:string;expiresAt:string}|null>(null),linkStatus=$state<LinkStatus|null>(null),bindings=$state<Record<string,{discord:string;telegram:string}>>({});
 const writable=$derived(panel.usage?.readOnly===false);
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function refresh(){const result=await parseResponse(pulseApi.channels.$get());channels=result.channels;const next:typeof bindings={};for(const p of projects)next[p.id]={discord:"",telegram:""};for(const b of result.bindings){const channel=channels.find(c=>c.id===b.channelId),project=next[b.projectId];if(channel&&project)project[channel.type]=channel.id;}bindings=next;}
 async function discord(e:SubmitEvent){e.preventDefault();await run(async()=>{await parseResponse(pulseApi.channels.$post({json:{type:"discord",webhook}}));webhook="";await refresh();});}
 async function telegram(){await run(async()=>{link=await parseResponse(pulseApi.telegram.link.$post({json:{}}));linkStatus=null;if(link)startPolling(link);});}
 async function confirmChat(){const current=link;if(!current)return;await run(async()=>{await parseResponse(pulseApi.telegram.link[":link"].confirm.$post({param:{link:current.id},json:{}}));stopPolling();link=null;linkStatus=null;await refresh();});}
 async function remove(channel:Channel){if(!await confirm({title:t("delete"),message:t("pulseRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.channels[":channel"].$delete({param:{channel:channel.id}}));await refresh();});}
 async function test(channel:Channel){await run(async()=>{await parseResponse(pulseApi.channels[":channel"].test.$post({param:{channel:channel.id},json:{}}));toast(()=>t("pulseTestQueued"));});}
 async function saveBindings(project:Project){await run(async()=>{const b=bindings[project.id];await parseResponse(pulseApi.projects[":project"].channels.$put({param:{project:project.id},json:{channelIds:[b?.discord,b?.telegram].filter((v):v is string=>!!v)}}));await refresh();});}
 onMount(()=>{void run(async()=>{projects=(await parseResponse(pulseApi.projects.$get())).projects;await refresh();});});
 let pollTimer:ReturnType<typeof setInterval>|undefined,pollEpoch=0;
 function stopPolling(){pollEpoch++;if(pollTimer!==undefined){clearInterval(pollTimer);pollTimer=undefined;}}
 function startPolling(current:{id:string}){
  stopPolling();const epoch=pollEpoch;let inFlight=false,attempts=0;
  const poll=async()=>{if(inFlight||epoch!==pollEpoch)return;if(++attempts>120){stopPolling();return;}inFlight=true;
   try{const value=await parseResponse(linkGET({param:{link:current.id}},{init:{signal:AbortSignal.timeout(15000)}}));if(epoch===pollEpoch&&link?.id===current.id){linkStatus=value;if(value.state!=="pending")stopPolling();}}
   catch(e){if(epoch===pollEpoch)error=e;}finally{inFlight=false;}
  };
  void poll();pollTimer=setInterval(()=>{void poll();},5000);
 }
 onDestroy(stopPolling);
 const options=(type:"discord"|"telegram")=>[{value:"",label:t("pulseNoChannel")},...channels.filter(c=>c.type===type&&c.active).map(c=>({value:c.id,label:c.label}))];
</script>

<h1 class="text-3xl font-extrabold tracking-tight">{t("pulseAlerts")}</h1>
<p class="mt-2 text-neutral-500">{t("pulseAlertsLead")}</p>
{#if error}<p class="mt-6 text-red-600" role="alert">{errorMessage(error)}</p>{/if}

{#if writable}<div class="mt-8 grid gap-4 lg:grid-cols-2">
 <Card title="Discord"><form class="mt-5 grid gap-3" onsubmit={discord}><FloatField id="discord-webhook" surface="bg-white dark:bg-neutral-900" label={t("pulseDiscordWebhook")} type="password" autocomplete="off" maxlength={2048} required bind:value={webhook}/><div class="flex justify-end"><Button {busy}>{t("pulseAddDiscord")}</Button></div></form></Card>
 <Card title="Telegram" help={t("pulseLinkHelp")}><div class="mt-5">{#if link}<a class="font-semibold text-blup underline" href={link.url} target="_blank" rel="noopener noreferrer">{t("pulseOpenBot")}</a><p class="mt-2 text-xs text-neutral-500">{t("pulseLinkDeadline",{date:formatDate(link.expiresAt)})}</p>{#if linkStatus?.candidate}<p class="my-3 break-all text-sm">{linkStatus.candidate.label} · ID: {linkStatus.candidate.id}</p><Button onclick={confirmChat} {busy}>{t("pulseConfirmChat")}</Button>{:else if linkStatus?.state==="expired"}<p class="mt-3 text-sm">{t("pulseLinkExpired")}</p>{:else}<p class="mt-3 text-sm text-neutral-500">{t("pulseWaitingChat")}</p>{/if}{:else}<div class="flex justify-end"><Button type="button" variant="secondary" onclick={telegram} {busy}>{t("pulseLinkTelegram")}</Button></div>{/if}</div></Card>
</div>{/if}

<h2 class="mt-10 mb-4 text-xl font-bold">{t("pulseDestinations")}</h2>
<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
 {#each channels as channel(channel.id)}
  <div class="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200 px-5 py-4 last:border-b-0 dark:border-neutral-800">
   <div class="flex min-w-0 flex-1 basis-full items-center gap-3 sm:basis-0"><MessageSquare class="size-5 shrink-0 text-neutral-400"/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{channel.label}</span><span class="rounded-full bg-neutral-100 px-2 py-0.5 text-xs font-semibold capitalize text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">{channel.type}</span></p><p class="text-sm {channel.lastError?'text-amber-700 dark:text-amber-300':'text-neutral-500'}">{channel.lastError?t("pulseDeliveryFailed"):channel.active?t("pulseActive"):t("pulsePaused")}</p></div></div>
   <div class="flex gap-2">{#if writable&&channel.active}<Button small variant="secondary" onclick={()=>test(channel)} {busy}><Send class="size-4"/>{t("pulseTestChannel")}</Button>{/if}<Button small variant="secondary" onclick={()=>remove(channel)}><Trash2 class="size-4"/>{t("delete")}</Button></div>
  </div>
 {:else}<p class="px-5 py-10 text-center text-neutral-500">{t("pulseNoChannels")}</p>{/each}
</div>

{#if writable&&channels.length}<h2 class="mt-10 mb-4 text-xl font-bold">{t("pulseChannelBindings")}</h2><div class="grid gap-3">{#each projects as project(project.id)}{#if bindings[project.id]}<Card title={project.name}><div class="mt-5 grid gap-4 sm:grid-cols-[1fr_1fr_auto] sm:items-center"><Select surface="bg-white dark:bg-neutral-900" label="Discord" options={options("discord")} bind:value={bindings[project.id].discord}/><Select surface="bg-white dark:bg-neutral-900" label="Telegram" options={options("telegram")} bind:value={bindings[project.id].telegram}/><Button class="h-14" onclick={()=>saveBindings(project)} {busy}>{t("save")}</Button></div></Card>{/if}{/each}</div>{/if}

<p class="mt-10 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
