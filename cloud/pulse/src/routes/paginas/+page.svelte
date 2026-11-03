<script lang="ts">
 import {goto} from "$app/navigation";
 import Plus from "@lucide/svelte/icons/plus";
 import Pencil from "@lucide/svelte/icons/pencil";
 import Eye from "@lucide/svelte/icons/eye";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import ExternalLink from "@lucide/svelte/icons/external-link";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import {pulseApi,parseResponse} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 import {getPages,type StatusPage} from "../../lib/status-pages";
 // The status pages: a button on top makes a new one; each row opens,
 // goes live or offline, or is deleted.
 let {data}=$props();
 let pages=$derived<StatusPage[]>(data.pages),busy=$state(false);
 const writable=$derived(panel.usage?.readOnly===false);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function publish(page:StatusPage){await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:page.id},json:{published:!page.published}}));pages=await getPages();toast(()=>page.published?t("pulseToastOffline"):t("pulseToastLive"));});}
 async function remove(page:StatusPage){if(!await confirm({title:t("delete"),message:t("pulseDeletePage"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$delete({param:{page:page.id}}));pages=await getPages();toast(()=>t("pulseToastDeleted"));});}
</script>

<PageHeader title={t("pulsePages")} lead={t("pulsePublicPrivacy")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto("/paginas/nova")}><Plus class="size-4"/>{t("pulseNewPage")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8"><List empty={t("pulseNoPages")}>
 {#each pages as page(page.id)}
  <ListItem>
   <p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{page.title}</span><Badge tone={page.published?"green":"neutral"}>{page.published?t("pulsePublished"):t("pulseDraft")}</Badge></p><a class="mt-0.5 inline-flex items-center gap-1 text-sm text-blup hover:underline" href={`https://pulse.getblup.site/${page.slug}/`} target="_blank" rel="noopener noreferrer">pulse.getblup.site/{page.slug}/<ExternalLink class="size-3.5"/></a>
   {#snippet actions()}<Button small variant="secondary" onclick={()=>goto(`/paginas/${encodeURIComponent(page.id)}`)}><Pencil class="size-4"/>{t("pulseEdit")}</Button>{#if page.writable}<Button small variant="secondary" {busy} onclick={()=>publish(page)}>{#if page.published}<EyeOff class="size-4"/>{t("pulseUnpublish")}{:else}<Eye class="size-4"/>{t("pulsePublish")}{/if}</Button>{/if}<Button small variant="secondary" {busy} onclick={()=>remove(page)}><Trash2 class="size-4"/>{t("delete")}</Button>{/snippet}
  </ListItem>
 {/each}
</List></div>
