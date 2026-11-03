<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {refreshAll} from "$app/navigation";
 import KeyRound from "@lucide/svelte/icons/key-round";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Button from "#lib/ui/Button.svelte";
 import CopyButton from "#lib/ui/CopyButton.svelte";
 import {cloudURL,pulseApi,parseResponse} from "../../../../lib/api";
 import {panel} from "../../../../lib/panel.svelte";
 // The source maps of a project's browser code: the upload token its build
 // sends them with (a new one replaces the old and is shown only now), the
 // command to send each map and the maps already sent.
 let {data}:PageProps=$props();
 let busy=$state(false),token=$state("");
 const writable=$derived(panel.usage?.readOnly===false&&!!data.project?.writable);
 const endpoint=`${cloudURL}/api/pulse/sourcemaps`;
 // The token goes in from the build's secret, never written in the command.
 const command=`curl -X POST "${endpoint}?release=1.4.2&file=app.min.js" \\\n  -H "Authorization: Bearer $PULSE_UPLOAD_TOKEN" \\\n  --data-binary @dist/assets/app.min.js.map`;
 const size=(bytes:number)=>bytes<1024*1024?`${Math.max(1,Math.round(bytes/1024)).toLocaleString(intlLocale())} KB`:`${(bytes/1024/1024).toLocaleString(intlLocale(),{maximumFractionDigits:1})} MB`;
 async function mint(){const project=data.project;if(!project)return;if(data.sourcemaps?.token&&!await confirm({title:t("pulseNewUploadToken"),message:t("pulseNewUploadTokenAsk")}))return;busy=true;try{token=(await parseResponse(pulseApi.projects[":project"]["upload-token"].$post({param:{project:project.id},json:{}}))).token;toast(()=>t("pulseToastUploadToken"));await refreshAll();}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

{#if !data.project||!data.sourcemaps}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseProjectNotFound")}</p>
{:else}
 {@const maps=data.sourcemaps}
 <div class="mb-4"><BackLink href={resolve("/")} label={t("pulseProjects")}/></div>
 <PageHeader title={t("pulseSourcemapsOf",{project:data.project.name})} lead={t("pulseSourcemapsLead")}/>

 <div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-6">
  <Card title={t("pulseUploadToken")} help={t("pulseUploadTokenHelp")}>
   {#snippet action()}{#if writable}<Button small variant={maps.token?"secondary":"primary"} {busy} onclick={mint}><KeyRound class="size-4"/>{maps.token?t("pulseNewUploadToken"):t("pulseCreateUploadToken")}</Button>{/if}{/snippet}
   {#if token}<div class="mt-4 rounded-lg border border-blup p-3"><div class="flex items-center gap-2"><code class="min-w-0 flex-1 font-mono text-sm break-all">{token}</code><CopyButton text={token}/></div><p class="mt-2 text-xs text-neutral-500">{t("pulseUploadTokenOnce")}</p></div>
   {:else if maps.token}<p class="mt-4 text-sm">{t("pulseUploadTokenActive",{prefix:maps.token.prefix,date:formatDate(new Date(maps.token.createdAt))})}</p>
   {:else}<p class="mt-4 text-sm text-neutral-500">{t("pulseUploadTokenNone")}</p>{/if}
  </Card>

  <Card title={t("pulseSourcemapsHow")}>
   <ol class="mt-4 grid list-decimal gap-3 pl-5 text-sm">
    <li>{t("pulseSourcemapsStep1")}</li>
    <li>{t("pulseSourcemapsStep2")}</li>
    <li>{t("pulseSourcemapsStep3")}</li>
   </ol>
   <div class="relative mt-4"><pre class="overflow-x-auto rounded-lg bg-neutral-100 p-4 pr-24 font-mono text-[13px] leading-6 dark:bg-neutral-950">{command}</pre><div class="absolute top-2 right-2"><CopyButton text={command}/></div></div>
   <p class="mt-3 text-xs text-neutral-500">{t("pulseSourcemapsNote",{days:"90",limit:String(maps.limit)})}</p>
  </Card>

  <section class="overflow-hidden rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
   <div class="flex flex-wrap items-baseline justify-between gap-2 px-5 pt-5"><h2 class="font-bold">{t("pulseSourcemapsSent")}</h2><span class="text-sm text-neutral-500">{maps.maps.length.toLocaleString(intlLocale())} / {maps.limit.toLocaleString(intlLocale())}</span></div>
   <div class="mt-3 overflow-auto"><table class="w-full text-left text-sm">
    <thead class="border-y border-neutral-200 whitespace-nowrap text-neutral-500 dark:border-neutral-800"><tr><th class="px-5 py-2.5 font-medium">{t("pulseFieldRelease")}</th><th class="px-5 py-2.5 font-medium">{t("pulseSourcemapFile")}</th><th class="px-5 py-2.5 font-medium">{t("pulseSourcemapSize")}</th><th class="px-5 py-2.5 font-medium">{t("pulseSourcemapSentAt")}</th></tr></thead>
    <tbody>
     {#each maps.maps as map(`${map.release}\n${map.file}`)}<tr class="border-b border-neutral-200 last:border-b-0 dark:border-neutral-800"><td class="px-5 py-2.5 font-mono whitespace-nowrap">{map.release}</td><td class="px-5 py-2.5 font-mono break-all">{map.file}</td><td class="px-5 py-2.5 whitespace-nowrap">{size(map.size)}</td><td class="px-5 py-2.5 whitespace-nowrap">{formatDate(new Date(map.uploadedAt))}</td></tr>
     {:else}<tr><td colspan="4" class="px-5 py-10 text-center text-neutral-500">{t("pulseNoSourcemaps")}</td></tr>{/each}
    </tbody>
   </table></div>
  </section>
 </div>
{/if}
