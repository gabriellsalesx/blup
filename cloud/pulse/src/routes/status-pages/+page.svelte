<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
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
 import ActionMenu from "#lib/ui/ActionMenu.svelte";
 import {openCard,cardLink} from "#lib/card.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import {pulseApi,parseResponse} from "../../lib/api";
 import {panel} from "../../lib/panel.svelte";
 import {getPages,type StatusPage} from "../../lib/status-pages";
 // The status pages: a button on top makes a new one; each card opens,
 // goes live or offline, or is deleted.
 let {data}:PageProps=$props();
 let pages=$derived<StatusPage[]>(data.pages),busy=$state(false);
 const writable=$derived(panel.usage?.readOnly===false);
 // Live pages are green on the edge and address, the others gray; both
 // themes named, so the card's own border color never covers the edge.
 const edge={live:"border-l-emerald-500 dark:border-l-emerald-500",off:"border-l-neutral-300 dark:border-l-neutral-700"};
 const address={live:"text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 dark:hover:text-emerald-300",off:"text-neutral-500 hover:text-neutral-900 dark:hover:text-white"};
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 async function publish(page:StatusPage){await run(async()=>{await parseResponse(pulseApi.pages[":page"].$patch({param:{page:page.id},json:{published:!page.published}}));pages=await getPages();toast(()=>page.published?t("pulseToastOffline"):t("pulseToastLive"));});}
 async function remove(page:StatusPage){if(!await confirm({title:t("delete"),message:t("pulseDeletePage"),danger:true}))return;await run(async()=>{await parseResponse(pulseApi.pages[":page"].$delete({param:{page:page.id}}));pages=await getPages();toast(()=>t("pulseToastDeleted"));});}
</script>

<PageHeader title={t("pulsePages")} lead={t("pulsePublicPrivacy")}>
 {#snippet aside()}{#if writable}<Button onclick={()=>goto(resolve("/status-pages/new"))}><Plus class="size-4"/>{t("pulseNewPage")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8 grid gap-3">
 {#each pages as page(page.id)}
  {@const href=resolve("/status-pages/[id]",{id:page.id})}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div onclick={(event)=>openCard(event,href)} onauxclick={(event)=>openCard(event,href)} class="relative flex cursor-pointer items-center gap-3 rounded-xl border border-l-4 border-neutral-200 bg-white px-5 py-4 transition-colors hover:bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:bg-neutral-800/60 {page.published?edge.live:edge.off}">
   <a class={cardLink} {href} aria-label={page.title}></a>
   <div class="min-w-0 flex-1"><p class="flex flex-wrap items-center gap-2 font-semibold"><span class="truncate">{page.title}</span><Badge tone={page.published?"green":"neutral"}>{page.published?t("pulsePublished"):t("pulseDraft")}</Badge></p><a class="relative z-10 mt-0.5 inline-flex max-w-full items-center gap-1 text-sm underline {page.published?address.live:address.off}" href={`https://pulse.getblup.site/${page.slug}/`} target="_blank" rel="noopener noreferrer"><span class="truncate">pulse.getblup.site/{page.slug}/</span><ExternalLink class="size-3.5 shrink-0"/></a></div>
   <ActionMenu label={t("pulseActions")} items={[{label:t("pulseEdit"),Icon:Pencil,onselect:()=>void goto(href)},...(page.writable?[{label:page.published?t("pulseUnpublish"):t("pulsePublish"),Icon:page.published?EyeOff:Eye,onselect:()=>void publish(page)}]:[]),{label:t("delete"),Icon:Trash2,danger:true,onselect:()=>void remove(page)}]}/>
  </div>
 {:else}<div class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseNoPages")}</div>{/each}
</div>
