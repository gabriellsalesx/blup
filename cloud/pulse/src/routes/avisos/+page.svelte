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
</script>
<h1 class="text-3xl font-bold">{t("pulseAlerts")}</h1><p class="my-4 text-sm text-neutral-500">{t("pulseAlertsLead")}</p>
{#if error}<p class="my-4 text-red-600" role="alert">{errorMessage(error)}</p>{/if}
{#if writable}<form class="my-6 grid gap-3" onsubmit={discord}><FloatField id="discord-webhook" label={t("pulseDiscordWebhook")} type="password" autocomplete="off" maxlength={2048} required bind:value={webhook}/><div class="flex flex-wrap gap-3"><Button {busy}>{t("pulseAddDiscord")}</Button><Button type="button" variant="secondary" onclick={telegram}>{t("pulseLinkTelegram")}</Button></div></form>{/if}
{#if link}<Card><p class="mb-3 text-sm">{t("pulseLinkHelp")}</p><a class="font-semibold text-blup underline" href={link.url} target="_blank" rel="noopener noreferrer">{t("pulseOpenBot")}</a><p class="my-3 text-xs text-neutral-500">{t("pulseLinkDeadline",{date:formatDate(link.expiresAt)})}</p>{#if linkStatus?.candidate}<p class="my-3 break-all">{linkStatus.candidate.label} · ID: {linkStatus.candidate.id}</p><Button onclick={confirmChat} {busy}>{t("pulseConfirmChat")}</Button>{:else if linkStatus?.state==="expired"}<p>{t("pulseLinkExpired")}</p>{:else}<p>{t("pulseWaitingChat")}</p>{/if}</Card>{/if}
<div class="my-6 grid gap-3">{#each channels as channel(channel.id)}<Card><h2 class="font-semibold">{channel.label}</h2><p class="my-2 text-sm text-neutral-500">{channel.active?t("pulseActive"):t("pulsePaused")}</p>{#if channel.lastError}<p class="my-3 text-sm text-amber-700 dark:text-amber-300">{t("pulseDeliveryFailed")}</p>{/if}<div class="flex gap-3">{#if writable&&channel.active}<Button variant="secondary" onclick={()=>test(channel)} {busy}>{t("pulseTestChannel")}</Button>{/if}<Button variant="secondary" onclick={()=>remove(channel)}>{t("delete")}</Button></div></Card>{:else}<p class="text-neutral-500">{t("pulseNoChannels")}</p>{/each}</div>
{#if writable&&channels.length}<section class="my-8"><h2 class="mb-4 text-xl font-bold">{t("pulseChannelBindings")}</h2><div class="grid gap-3">{#each projects as project(project.id)}{#if bindings[project.id]}<Card><h3 class="mb-3 font-semibold">{project.name}</h3><div class="flex flex-wrap gap-3"><label class="grid gap-2 text-sm">Discord<select class="rounded-lg border p-3 dark:bg-neutral-900" bind:value={bindings[project.id].discord}><option value="">{t("pulseNoChannel")}</option>{#each channels.filter(c=>c.type==="discord"&&c.active) as channel(channel.id)}<option value={channel.id}>{channel.label}</option>{/each}</select></label><label class="grid gap-2 text-sm">Telegram<select class="rounded-lg border p-3 dark:bg-neutral-900" bind:value={bindings[project.id].telegram}><option value="">{t("pulseNoChannel")}</option>{#each channels.filter(c=>c.type==="telegram"&&c.active) as channel(channel.id)}<option value={channel.id}>{channel.label}</option>{/each}</select></label><Button onclick={()=>saveBindings(project)} {busy}>{t("save")}</Button></div></Card>{/if}{/each}</div></section>{/if}
<p class="mt-8 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
