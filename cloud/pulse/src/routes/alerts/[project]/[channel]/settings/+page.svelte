<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import KeyRound from "@lucide/svelte/icons/key-round";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Button from "#lib/ui/Button.svelte";
 import CodeBlock from "#lib/ui/CodeBlock.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import {pulseApi,parseResponse} from "../../../../../lib/api";
 import {panel} from "../../../../../lib/panel.svelte";
 // A destination's settings: a new address (never showing the current
 // one), a webhook's new signing secret (shown once), and deleting it.
 // Telegram has no address: another chat is linked through the bot.
 let {data}:PageProps=$props();
 const project=$derived(data.project),channel=$derived(data.channel);
 const writable=$derived(panel.usage?.readOnly===false&&!!project?.writable);
 const back=$derived(project?resolve("/alerts/[project]",{project:project.id}):resolve("/alerts"));
 let address=$state(""),secret=$state<string|null>(null),busy=$state(false);
 const addressLabel=$derived(channel?.type==="discord"?t("pulseNewDiscordWebhook"):channel?.type==="slack"?t("pulseNewSlackWebhook"):t("pulseNewWebhookURL"));
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
 async function save(e:SubmitEvent){e.preventDefault();const current=channel;if(!current||!address.trim())return;await run(async()=>{await parseResponse(pulseApi.channels[":channel"].$patch({param:{channel:current.id},json:{address:address.trim()}}));address="";toast(()=>t("pulseToastSaved"));await goto(back,{invalidateAll:true});});}
 async function rotate(){const current=channel;if(!current||!await confirm({title:t("pulseRotateSecret"),message:t("pulseRotateSecretConfirm")}))return;await run(async()=>{secret=(await parseResponse(pulseApi.channels[":channel"].secret.$post({param:{channel:current.id},json:{}}))).secret;toast(()=>t("pulseToastSaved"));});}
 async function remove(){const current=channel;if(!current||!await confirm({title:t("pulseRemoveChannelTitle"),message:t("pulseRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.channels[":channel"].$delete({param:{channel:current.id}}));toast(()=>t("pulseToastDeleted"));await goto(back,{invalidateAll:true});});}
</script>

{#if project&&channel}
 <div class="mb-4"><BackLink href={back} label={project.name}/></div>
 <PageHeader title={t("pulseConfigure")} lead={t("pulseChannelSettingsLead")}/>
 <p class="mt-6 flex items-center gap-3 font-semibold"><ChannelIcon type={channel.type} class="size-6"/><span class="min-w-0 truncate">{channel.label}</span></p>
 <div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-10">
  {#if channel.type==="telegram"}
   <FormSection title={t("pulseChannelAddress")} help={t("pulseTelegramSettings")}>
    {#if writable}<Button variant="secondary" onclick={()=>goto(`${resolve("/alerts/new")}?${new URLSearchParams({project:project.id})}`)}>{t("pulseRelinkTelegram")}</Button>{/if}
   </FormSection>
  {:else}
   <form class="grid grid-cols-[minmax(0,1fr)] gap-6" onsubmit={save}>
    <FormSection title={t("pulseChannelAddress")} help={t("pulseChannelAddressHelp")}>
     <div class="grid grid-cols-[minmax(0,1fr)] gap-2">
      <FloatField id="channel-address" surface="bg-white dark:bg-neutral-900" label={addressLabel} type={channel.type==="webhook"?"url":"password"} autocomplete="off" maxlength={2048} disabled={!writable} bind:value={address}/>
      <p class="px-3 text-sm text-neutral-500">{t("pulseCurrentDestination",{label:channel.label})}</p>
     </div>
    </FormSection>
    {#if writable}<div class="flex justify-end"><Button {busy} disabled={!address.trim()}>{t("pulseSaveChanges")}</Button></div>{/if}
   </form>
  {/if}
  {#if channel.type==="webhook"&&writable}
   <FormSection title={t("pulseSigningSecret")} help={t("pulseSigningSecretHelp")}>
    {#if secret}
     <div class="grid grid-cols-[minmax(0,1fr)] gap-3"><p class="font-semibold">{t("pulseNewSecretReady")}</p><CodeBlock code={secret}/><p class="text-sm text-neutral-500">{t("pulseWebhookSecretOnce")}</p></div>
    {:else}<Button variant="secondary" {busy} onclick={rotate}><KeyRound class="size-4"/>{t("pulseRotateSecret")}</Button>{/if}
   </FormSection>
  {/if}
  <FormSection title={t("pulseRemoveChannelTitle")} help={t("pulseDestructive")}>
   <button type="button" class="inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2.5 font-semibold text-white transition-colors hover:bg-red-700 disabled:opacity-50" disabled={busy} onclick={remove}><Trash2 class="size-4"/>{t("pulseRemoveThisChannel")}</button>
  </FormSection>
 </div>
{/if}
