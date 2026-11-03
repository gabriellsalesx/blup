<script lang="ts">
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {refreshSetup} from "../../../lib/setup";
 import Copy from "@lucide/svelte/icons/copy";
 import Plus from "@lucide/svelte/icons/plus";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import type {PlatformId} from "../../../../../shared/pulse-platforms";
 import PlatformPicker from "../../../lib/PlatformPicker.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 // A new project: its platform and name, then the DSN to paste in the SDK
 // (shown once, so it waits here to be copied).
 let platform=$state<PlatformId>("javascript"),name=$state(""),busy=$state(false),dsn=$state("");
 async function create(e:SubmitEvent){e.preventDefault();busy=true;try{dsn=(await parseResponse(pulseApi.projects.$post({json:{name:name.trim(),platform}}))).key.dsn;refreshSetup();toast(()=>t("pulseProjectReady"));}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
 async function copy(){try{await navigator.clipboard.writeText(dsn);toast(()=>t("copied"));}catch{toast(()=>t("copyFailed"),{kind:"error"});}}
</script>

<div class="mb-4"><BackLink href={resolve("/")} label={t("pulseProjects")}/></div>
<PageHeader title={t("pulseCreateProjectTitle")}/>

{#if dsn}
 <div class="mt-8 rounded-xl border border-blup bg-white p-6 dark:bg-neutral-900">
  <h2 class="text-lg font-bold">{t("pulseProjectReady")}: {name}</h2>
  <p class="mt-2 text-neutral-500">{t("pulseSDK")}</p>
  <div class="mt-4 flex items-center gap-2 rounded-lg bg-neutral-100 p-3 dark:bg-neutral-800"><code class="min-w-0 flex-1 break-all text-sm">{dsn}</code><Button small variant="secondary" onclick={copy}><Copy class="size-4"/>{t("copy")}</Button></div>
  <p class="mt-3 text-sm text-neutral-500">{t("pulseDSNOnce")}</p>
  <div class="mt-5 flex justify-end"><Button onclick={()=>goto(resolve("/"))}>{t("pulseBackToProjects")}</Button></div>
 </div>
{:else}
 <form class="mt-8" onsubmit={create}>
  <Card><PlatformPicker bind:value={platform}/>
   <div class="mt-6 grid gap-4 sm:grid-cols-[1fr_auto] sm:items-center"><FloatField id="project-name" surface="bg-white dark:bg-neutral-900" label={t("pulseProjectName")} required maxlength={80} bind:value={name}/><Button {busy} class="h-14" disabled={!name.trim()}><Plus class="size-4"/>{t("pulseCreateProject")}</Button></div>
  </Card>
 </form>
{/if}
