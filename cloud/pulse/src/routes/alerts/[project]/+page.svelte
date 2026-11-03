<script lang="ts">
 import type {PageProps} from "./$types";
 import type {Component} from "svelte";
 import {onMount} from "svelte";
 import {resolve} from "$app/paths";
 import {goto,invalidateAll} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import Send from "@lucide/svelte/icons/send";
 import Bug from "@lucide/svelte/icons/bug";
 import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
 import ArrowDown from "@lucide/svelte/icons/arrow-down";
 import ArrowUp from "@lucide/svelte/icons/arrow-up";
 import Check from "@lucide/svelte/icons/check";
 import Clock from "@lucide/svelte/icons/clock";
 import Settings from "@lucide/svelte/icons/settings";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 import {panel} from "../../../lib/panel.svelte";
 import {projectChannels,kindName,roundAfter,clock,type Channel,type Channels,type TestKind} from "../../../lib/alerts";
 // One project's alert destinations. Picking one shows the alerts it can
 // be tested with (an example of each kind, or a plain test), how long
 // they take, and once sent a countdown to the round that sends them and
 // to when it can be tested again. Nothing is picked to start with, so
 // the send button waits for a choice. The three dots configure the
 // destination (its address) or delete it.
 let {data}:PageProps=$props();
 const project=$derived(data.project);
 const writable=$derived(panel.usage?.readOnly===false&&!!project?.writable);
 // Switched off ones too (a provider refused them): configuring a new
 // address switches them on again.
 const channels=$derived(project?projectChannels(data.channels,project.id):[]);
 // The destination being tested: the first one until another is picked.
 let selectedId=$derived(channels[0]?.id??"");
 const selected=$derived(channels.find(c=>c.id===selectedId)??null);
 let kinds=$state<TestKind[]>([]),busy=$state(false);
 // The last test of each destination: when it was asked for, the round
 // that sends it and when another can be asked for; from the server after
 // a reload.
 type Sent={at:number;sendsAt:number;againAt:number};
 let tests=$derived.by(()=>{const next=$state(sentFrom(data.channels));return next;});
 function sentFrom(all:Channels){const out:Record<string,Sent>={};for(const c of all.channels)if(c.testAgainAt>0){const at=c.testAgainAt-60_000;out[c.id]={at,sendsAt:roundAfter(at),againAt:c.testAgainAt};}return out;}
 let now=$state(Date.now());
 onMount(()=>{const timer=setInterval(()=>now=Date.now(),500);return()=>clearInterval(timer);});
 const sent=$derived(selected?tests[selected.id]??null:null);
 const waiting=$derived(!!sent&&now<sent.againAt);
 const progress=$derived(sent?Math.min(1,Math.max(0,(now-sent.at)/Math.max(1,sent.sendsAt-sent.at))):0);

 const testOptions:{value:TestKind;label:()=>string;help:()=>string;Icon:Component<{class?:string}>;bg:string}[]=[
  {value:"test",label:()=>t("pulseTestKindTest"),help:()=>t("pulseTestKindTestHelp"),Icon:Send,bg:"bg-blup"},
  {value:"issue",label:()=>t("pulseAlertIssue"),help:()=>t("pulseTestKindIssueHelp"),Icon:Bug,bg:"bg-red-500"},
  {value:"regression",label:()=>t("pulseAlertRegression"),help:()=>t("pulseTestKindRegressionHelp"),Icon:RotateCcw,bg:"bg-amber-500"},
  {value:"monitor_down",label:()=>t("pulseAlertDown"),help:()=>t("pulseTestKindDownHelp"),Icon:ArrowDown,bg:"bg-red-500"},
  {value:"monitor_up",label:()=>t("pulseAlertUp"),help:()=>t("pulseTestKindUpHelp"),Icon:ArrowUp,bg:"bg-emerald-500"},
 ];
 function toggle(kind:TestKind){kinds=kinds.includes(kind)?kinds.filter(k=>k!==kind):[...kinds,kind];}

 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function send(){const channel=selected,current=project;if(!channel||!current||!kinds.length)return;await run(async()=>{
  const at=Date.now(),result=await parseResponse(pulseApi.channels[":channel"].test.$post({param:{channel:channel.id},json:{projectId:current.id,kinds:testOptions.map(o=>o.value).filter(k=>kinds.includes(k))}}));
  tests[channel.id]={at,sendsAt:result.sendsAt,againAt:result.testAgainAt};toast(()=>t("pulseTestQueued"));
 });}
 async function remove(channel:Channel){if(!await confirm({title:t("delete"),message:t("pulseRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.channels[":channel"].$delete({param:{channel:channel.id}}));await invalidateAll();toast(()=>t("pulseToastDeleted"));});}
 const settingsHref=(channel:Channel)=>resolve("/alerts/[project]/[channel]/settings",{project:project?.id??"",channel:channel.id});
 const menu=(channel:Channel)=>[...(writable?[{label:t("pulseConfigure"),Icon:Settings,onselect:()=>void goto(settingsHref(channel))}]:[]),{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void remove(channel)}];
 const addHref=$derived(project?`${resolve("/alerts/new")}?${new URLSearchParams({project:project.id})}`:resolve("/alerts/new"));
</script>

<div class="mb-4"><BackLink href={resolve("/alerts")} label={t("pulseAlerts")}/></div>
<PageHeader title={project?.name??t("pulseAlerts")} lead={t("pulseProjectAlertsLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto(addHref)}><Plus class="size-4"/>{t("pulseAddChannel")}</Button>{/if}{/snippet}
</PageHeader>

<h2 class="mt-8 mb-4 text-xl font-bold">{t("pulseProjectChannels")}</h2>
{#if channels.length}
 <div class="grid grid-cols-[minmax(0,1fr)] gap-3 sm:grid-cols-2" role="radiogroup" aria-label={t("pulseProjectChannels")}>
  {#each channels as channel(channel.id)}
   <div class="relative flex items-center gap-4 rounded-xl border p-4 transition-colors {channel.id===selectedId?"border-blup bg-emerald-50/60 dark:bg-emerald-950/30":"border-neutral-200 bg-white hover:border-neutral-300 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-700"}">
    <button type="button" role="radio" aria-checked={channel.id===selectedId} aria-label={channel.label} class="absolute inset-0 rounded-xl" onclick={()=>selectedId=channel.id}></button>
    <ChannelIcon type={channel.type} class="size-8"/>
    <span class="min-w-0 flex-1"><span class="flex flex-wrap items-center gap-2"><span class="truncate font-semibold">{channel.label}</span><Badge>{kindName[channel.type]}</Badge></span><span class="block text-sm {channel.lastError?"text-amber-700 dark:text-amber-300":"text-neutral-500"}">{!channel.active?t("pulseChannelOff"):channel.lastError?t("pulseDeliveryFailed"):t("pulseActive")}</span></span>
    <span class="relative"><ActionMenu label={t("pulseActions")} items={menu(channel)}/></span>
   </div>
  {/each}
 </div>
{:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseNoProjectChannels")}</div>{/if}

{#if selected&&writable}
 <div class="mt-6"><Card title={t("pulseChooseTestKinds")} help={t("pulseChooseTestKindsHelp")}>
  <div class="mt-5 grid grid-cols-[minmax(0,1fr)] gap-3 sm:grid-cols-2 lg:grid-cols-3">
   {#each testOptions as option(option.value)}
    {@const on=kinds.includes(option.value)}
    <button type="button" role="checkbox" aria-checked={on} class="flex items-center gap-3 rounded-xl border p-4 text-left transition-colors {on?"border-blup bg-emerald-50/60 dark:bg-emerald-950/30":"border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700"}" disabled={waiting} onclick={()=>toggle(option.value)}>
     <span class="grid size-9 shrink-0 place-items-center rounded-full text-white {option.bg}"><option.Icon class="size-4"/></span>
     <span class="min-w-0 flex-1"><span class="block font-semibold">{option.label()}</span><span class="block text-sm text-neutral-500">{option.help()}</span></span>
     <span class="grid size-5 shrink-0 place-items-center rounded border {on?"border-blup bg-blup text-white":"border-neutral-300 dark:border-neutral-600"}">{#if on}<Check class="size-3.5"/>{/if}</span>
    </button>
   {/each}
  </div>
  <p class="mt-5 flex items-start gap-2 text-sm text-neutral-500"><Clock class="mt-0.5 size-4 shrink-0"/>{t("pulseTestTiming")}</p>
  {#if !selected.active}<p class="mt-3 text-sm text-amber-700 dark:text-amber-300">{t("pulseChannelOffHelp")}</p>{/if}

  {#if sent&&now<sent.againAt}
   <div class="mt-5 rounded-xl border border-neutral-200 p-4 dark:border-neutral-800" aria-live="polite">
    {#if now<sent.sendsAt}
     <div class="flex items-center justify-between gap-4"><span class="text-sm font-semibold">{t("pulseTestLeaving")}</span><span class="font-mono text-2xl font-bold tabular-nums">{clock(sent.sendsAt-now)}</span></div>
     <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full bg-blup transition-[width] duration-500" style:width="{progress*100}%"></div></div>
    {:else}
     <p class="flex items-start gap-3 text-sm"><span class="grid size-6 shrink-0 place-items-center rounded-full bg-emerald-500 text-white"><Check class="size-3.5"/></span>{t("pulseTestLeft")}</p>
    {/if}
   </div>
  {/if}

  <div class="mt-5 flex justify-end"><Button onclick={send} {busy} disabled={!kinds.length||waiting||!selected.active}>{#if waiting&&sent}<Clock class="size-4"/>{t("pulseTestAgainIn",{time:clock(sent.againAt-now)})}{:else}<Send class="size-4"/>{t("pulseSendTests")}{/if}</Button></div>
 </Card></div>
{/if}


<p class="mt-10 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
