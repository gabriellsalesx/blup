<script lang="ts">
 import {onDestroy} from "svelte";
 import {type InferResponseType} from "hono/client";
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Select from "#lib/ui/Select.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import Send from "@lucide/svelte/icons/send";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import MessageSquare from "@lucide/svelte/icons/message-square";
 import {pulseApi,parseResponse,type Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 type Channel=InferResponseType<typeof pulseApi.channels.$get,200>["channels"][number];
 const linkGET=pulseApi.telegram.link[":link"].$get;
 type LinkStatus=InferResponseType<typeof linkGET,200>;
 type Channels=InferResponseType<typeof pulseApi.channels.$get,200>;
 let {data}=$props();
 const projects=$derived<Project[]>(data.projects);
 let channels=$derived<Channel[]>(data.channels?.channels??[]),bindings=$derived(toBindings(data.projects,data.channels)),webhook=$state(""),busy=$state(false),link=$state<{id:string;url:string;expiresAt:string}|null>(null),linkStatus=$state<LinkStatus|null>(null);
 const writable=$derived(panel.usage?.readOnly===false);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 // Which Discord and Telegram destination each project sends to.
 function toBindings(list:Project[],result:Channels|null){const next:Record<string,{discord:string;telegram:string}>={};for(const p of list)next[p.id]={discord:"",telegram:""};for(const b of result?.bindings??[]){const channel=result?.channels.find(c=>c.id===b.channelId),project=next[b.projectId];if(channel&&project)project[channel.type]=channel.id;}return next;}
 async function refresh(){const result=await parseResponse(pulseApi.channels.$get());channels=result.channels;bindings=toBindings(projects,result);}
 async function discord(e:SubmitEvent){e.preventDefault();await run(async()=>{await parseResponse(pulseApi.channels.$post({json:{type:"discord",webhook}}));webhook="";await refresh();toast(()=>t("pulseToastLinked"));});}
 async function telegram(){await run(async()=>{link=await parseResponse(pulseApi.telegram.link.$post({json:{}}));linkStatus=null;if(link)startPolling(link);});}
 async function confirmChat(){const current=link;if(!current)return;await run(async()=>{await parseResponse(pulseApi.telegram.link[":link"].confirm.$post({param:{link:current.id},json:{}}));stopPolling();link=null;linkStatus=null;await refresh();toast(()=>t("pulseToastLinked"));});}
 async function remove(channel:Channel){if(!await confirm({title:t("delete"),message:t("pulseRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.channels[":channel"].$delete({param:{channel:channel.id}}));await refresh();toast(()=>t("pulseToastDeleted"));});}
 async function test(channel:Channel){await run(async()=>{await parseResponse(pulseApi.channels[":channel"].test.$post({param:{channel:channel.id},json:{}}));toast(()=>t("pulseTestQueued"));});}
 async function saveBindings(project:Project){await run(async()=>{const b=bindings[project.id];await parseResponse(pulseApi.projects[":project"].channels.$put({param:{project:project.id},json:{channelIds:[b?.discord,b?.telegram].filter((v):v is string=>!!v)}}));await refresh();toast(()=>t("pulseToastSaved"));});}
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
 const options=(type:"discord"|"telegram")=>[{value:"",label:t("pulseNoChannel")},...channels.filter(c=>c.type===type&&c.active).map(c=>({value:c.id,label:c.label}))];
</script>

<PageHeader title={t("pulseAlerts")} lead={t("pulseAlertsLead")}/>

{#if writable}<div class="mt-8 grid gap-4 lg:grid-cols-2">
 <Card title="Discord"><form class="mt-5 grid gap-3" onsubmit={discord}><FloatField id="discord-webhook" surface="bg-white dark:bg-neutral-900" label={t("pulseDiscordWebhook")} type="password" autocomplete="off" maxlength={2048} required bind:value={webhook}/><div class="flex justify-end"><Button {busy}>{t("pulseAddDiscord")}</Button></div></form></Card>
 <Card title="Telegram" help={t("pulseLinkHelp")}><div class="mt-5">{#if link}<a class="font-semibold text-blup underline" href={link.url} target="_blank" rel="noopener noreferrer">{t("pulseOpenBot")}</a><p class="mt-2 text-xs text-neutral-500">{t("pulseLinkDeadline",{date:formatDate(link.expiresAt)})}</p>{#if linkStatus?.candidate}<p class="my-3 break-all text-sm">{linkStatus.candidate.label} · ID: {linkStatus.candidate.id}</p><Button onclick={confirmChat} {busy}>{t("pulseConfirmChat")}</Button>{:else if linkStatus?.state==="expired"}<p class="mt-3 text-sm">{t("pulseLinkExpired")}</p>{:else}<p class="mt-3 text-sm text-neutral-500">{t("pulseWaitingChat")}</p>{/if}{:else}<div class="flex justify-end"><Button type="button" variant="secondary" onclick={telegram} {busy}>{t("pulseLinkTelegram")}</Button></div>{/if}</div></Card>
</div>{/if}

<h2 class="mt-10 mb-4 text-xl font-bold">{t("pulseDestinations")}</h2>
<List empty={t("pulseNoChannels")}>
 {#each channels as channel(channel.id)}
  <ListItem>
   <div class="flex items-center gap-3"><MessageSquare class="size-5 shrink-0 text-neutral-400"/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{channel.label}</span><Badge><span class="capitalize">{channel.type}</span></Badge></p><p class="text-sm {channel.lastError?'text-amber-700 dark:text-amber-300':'text-neutral-500'}">{channel.lastError?t("pulseDeliveryFailed"):channel.active?t("pulseActive"):t("pulsePaused")}</p></div></div>
   {#snippet actions()}{#if writable&&channel.active}<Button small variant="secondary" onclick={()=>test(channel)} {busy}><Send class="size-4"/>{t("pulseTestChannel")}</Button>{/if}<Button small variant="secondary" onclick={()=>remove(channel)}><Trash2 class="size-4"/>{t("delete")}</Button>{/snippet}
  </ListItem>
 {/each}
</List>

{#if writable&&channels.length}<h2 class="mt-10 mb-4 text-xl font-bold">{t("pulseChannelBindings")}</h2><div class="grid gap-3">{#each projects as project(project.id)}{#if bindings[project.id]}<Card title={project.name}><div class="mt-5 grid gap-4 sm:grid-cols-[1fr_1fr_auto] sm:items-center"><Select surface="bg-white dark:bg-neutral-900" label="Discord" options={options("discord")} bind:value={bindings[project.id].discord}/><Select surface="bg-white dark:bg-neutral-900" label="Telegram" options={options("telegram")} bind:value={bindings[project.id].telegram}/><Button class="h-14" onclick={()=>saveBindings(project)} {busy}>{t("save")}</Button></div></Card>{/if}{/each}</div>{/if}

<p class="mt-10 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
