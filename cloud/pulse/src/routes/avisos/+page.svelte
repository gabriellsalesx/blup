<script lang="ts">
 import {goto} from "$app/navigation";
 import {type InferResponseType} from "hono/client";
 import Plus from "@lucide/svelte/icons/plus";
 import Send from "@lucide/svelte/icons/send";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Select from "#lib/ui/Select.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import BrandIcon from "#lib/ui/BrandIcon.svelte";
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {pulseApi,parseResponse,type Project} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 // Where alerts go: the Discord and Telegram destinations (a button on
 // top adds one), then which of them each project sends to.
 type Channels=InferResponseType<typeof pulseApi.channels.$get,200>;
 type Channel=Channels["channels"][number];
 let {data}=$props();
 const projects=$derived<Project[]>(data.projects);
 let channels=$derived<Channel[]>(data.channels?.channels??[]),busy=$state(false);
 // Which Discord and Telegram destination each project sends to, editable
 // in place (deep state), rebuilt whenever the page loads again.
 let bindings=$derived.by(()=>{const next=$state(toBindings(data.projects,data.channels));return next;});
 const writable=$derived(panel.usage?.readOnly===false);
 function toBindings(list:Project[],result:Channels|null){const next:Record<string,{discord:string;telegram:string}>={};for(const p of list)next[p.id]={discord:"",telegram:""};for(const b of result?.bindings??[]){const channel=result?.channels.find(c=>c.id===b.channelId),project=next[b.projectId];if(channel&&project)project[channel.type]=channel.id;}return next;}
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function refresh(){const result=await parseResponse(pulseApi.channels.$get());channels=result.channels;const next=$state(toBindings(projects,result));bindings=next;}
 async function remove(channel:Channel){if(!await confirm({title:t("delete"),message:t("pulseRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.channels[":channel"].$delete({param:{channel:channel.id}}));await refresh();toast(()=>t("pulseToastDeleted"));});}
 async function test(channel:Channel){await run(async()=>{await parseResponse(pulseApi.channels[":channel"].test.$post({param:{channel:channel.id},json:{}}));toast(()=>t("pulseTestQueued"));});}
 async function saveBindings(project:Project){await run(async()=>{const b=bindings[project.id];await parseResponse(pulseApi.projects[":project"].channels.$put({param:{project:project.id},json:{channelIds:[b?.discord,b?.telegram].filter((v):v is string=>!!v)}}));await refresh();toast(()=>t("pulseToastSaved"));});}
 const options=(type:"discord"|"telegram")=>[{value:"",label:t("pulseNoChannel")},...channels.filter(c=>c.type===type&&c.active).map(c=>({value:c.id,label:c.label}))];
 const actionsFor=(channel:Channel)=>[...(writable&&channel.active?[{label:t("pulseTestChannel"),Icon:Send,onselect:()=>void test(channel)}]:[]),{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void remove(channel)}];
</script>

<PageHeader title={t("pulseAlerts")} lead={t("pulseAlertsLead")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto("/avisos/novo")}><Plus class="size-4"/>{t("pulseAddChannel")}</Button>{/if}{/snippet}
</PageHeader>

<h2 class="mt-8 mb-4 text-xl font-bold">{t("pulseDestinations")}</h2>
<List empty={t("pulseNoChannels")}>
 {#each channels as channel(channel.id)}
  <ListItem>
   <div class="flex items-center gap-3"><BrandIcon name={channel.type} class="size-6"/><div class="min-w-0"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{channel.label}</span><Badge><span class="capitalize">{channel.type}</span></Badge></p><p class="text-sm {channel.lastError?'text-amber-700 dark:text-amber-300':'text-neutral-500'}">{channel.lastError?t("pulseDeliveryFailed"):channel.active?t("pulseActive"):t("pulsePaused")}</p></div></div>
   {#snippet menu()}<ActionMenu label={t("pulseActions")} items={actionsFor(channel)}/>{/snippet}
  </ListItem>
 {/each}
</List>

{#if writable&&channels.length}<h2 class="mt-10 mb-4 text-xl font-bold">{t("pulseChannelBindings")}</h2><div class="grid gap-3">{#each projects as project(project.id)}{#if bindings[project.id]}<Card title={project.name}><div class="mt-5 grid gap-4 sm:grid-cols-[1fr_1fr_auto] sm:items-center"><Select surface="bg-white dark:bg-neutral-900" label="Discord" options={options("discord")} bind:value={bindings[project.id].discord}/><Select surface="bg-white dark:bg-neutral-900" label="Telegram" options={options("telegram")} bind:value={bindings[project.id].telegram}/><Button class="h-14" onclick={()=>saveBindings(project)} {busy}>{t("save")}</Button></div></Card>{/if}{/each}</div>{/if}

<p class="mt-10 text-xs text-neutral-500">{t("pulseChannelPrivacy")}</p>
