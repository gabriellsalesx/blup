<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import ClipboardPaste from "@lucide/svelte/icons/clipboard-paste";
 import Clapperboard from "@lucide/svelte/icons/clapperboard";
 import Rocket from "@lucide/svelte/icons/rocket";
 import Tag from "@lucide/svelte/icons/tag";
 import ExternalLink from "@lucide/svelte/icons/external-link";
 import Settings2 from "@lucide/svelte/icons/settings-2";
 import Clock from "@lucide/svelte/icons/clock";
 import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
 import {t,formatDate,intlLocale} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import ListItem from "#lib/ui/ListItem.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import Pager from "#lib/ui/Pager.svelte";
 import {createPager} from "#lib/pager.svelte.ts";
 import {postsPage} from "../lib/posts";
 import {paceEvery} from "../lib/pace";
 import type {PostState} from "../lib/api";
 import {hasPlan} from "../lib/plan";
 import {storeLabel} from "../lib/catalog";
 import {failureText} from "../lib/errors";
 import Usage from "../lib/Usage.svelte";
 // The deals of the account: those waiting in the queue, those sent (with
 // their clicks) and those that failed (with why). Pasting a link adds one,
 // and so does a new TikTok video; without a channel yet, the first steps
 // come first. The queue says when each goes, in the account's pace.
 let {data}:PageProps=$props();
 const states=$derived([{value:"queued" as const,label:t("dealsQueued")},{value:"sent" as const,label:t("dealsSent")},{value:"failed" as const,label:t("dealsFailed")}]);
 const empty=$derived({queued:t("dealsQueueEmpty"),sent:t("dealsOffersEmpty"),failed:t("dealsFailedEmpty")}[data.state]);
 const pager=$derived(createPager(cursor=>postsPage(data.state,cursor),data.first));
 const channelName=(id:string)=>data.channels.find(c=>c.id===id)?.label??"";
 const clicks=(n:number)=>n===1?t("dealsOneClick"):t("dealsClickCount",{count:n.toLocaleString(intlLocale())});
 function show(state:PostState){void goto(state==="queued"?resolve("/"):`${resolve("/")}?${new URLSearchParams({state})}`,{reset:false});}
</script>

<PageHeader title={t("dealsOffers")} lead={t("dealsOffersLead")}>
 <!-- On a phone, one row: Paste link first, New video beside it, Settings as its icon at the end (the narrowest phones drop the other icons). -->
 {#snippet aside()}{#if hasPlan()}<div class="flex flex-wrap gap-2">{#if !data.channels.length}<Button variant="secondary" class="max-sm:order-first max-sm:basis-full" onclick={()=>goto(resolve("/start"))}><Rocket class="size-4"/>{t("dealsStart")}</Button>{/if}<Button variant="secondary" class="max-sm:order-last max-sm:px-3" title={t("dealsSettings")} onclick={()=>goto(resolve("/settings"))}><Settings2 class="size-4"/><span class="max-sm:sr-only">{t("dealsSettings")}</span></Button><Button variant="secondary" class="whitespace-nowrap max-sm:flex-1 max-sm:px-3" onclick={()=>goto(resolve("/offers/video"))}><Clapperboard class="size-4 max-[359px]:hidden"/>{t("dealsNewVideo")}</Button><Button class="whitespace-nowrap max-sm:order-first max-sm:flex-1 max-sm:px-3" onclick={()=>goto(resolve("/offers/new"))}><ClipboardPaste class="size-4 max-[359px]:hidden"/>{t("dealsPasteLink")}</Button></div>{/if}{/snippet}
</PageHeader>

<div class="mt-6"><Usage kind="posts"/></div>

{#if data.settings?.behind}<p class="mt-6 flex items-start gap-2 rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200"><TriangleAlert class="mt-0.5 size-4 shrink-0"/><span>{t("dealsBehind")} <a class="font-semibold underline" href={resolve("/settings")}>{t("dealsSettings")}</a></span></p>{/if}

<div class="mt-6 flex flex-wrap items-center justify-between gap-3"><Segmented label={t("dealsOfferState")} options={states} value={data.state} onchange={show}/>{#if data.settings}<a class="text-sm text-neutral-500 hover:text-neutral-900 dark:hover:text-white" href={resolve("/settings")}>{t("dealsPaceLine",{pace:paceEvery(data.settings.pace)})}</a>{/if}</div>

<div class="mt-4"><List {empty}>
 {#each pager.items as post(post.id)}
  <ListItem label={post.title}>
   <div class="flex items-center gap-3">
    {#if post.image}<img src={post.image} alt="" class="size-12 shrink-0 rounded-lg bg-neutral-100 object-cover dark:bg-neutral-800" referrerpolicy="no-referrer" loading="lazy"/>
    {:else}<span class="grid size-12 shrink-0 place-items-center rounded-lg bg-neutral-100 text-neutral-400 dark:bg-neutral-800" aria-hidden="true"><Tag class="size-5"/></span>{/if}
    <div class="min-w-0 flex-1">
     <p class="truncate font-semibold">{post.title}</p>
     <p class="truncate text-sm text-neutral-500">{storeLabel(post.store)}{#if channelName(post.channelId)}{" · "}{channelName(post.channelId)}{/if}{#if post.sentAt}{" · "}{formatDate(post.sentAt)}{/if}</p>
     {#if post.sendAt}<p class="flex items-center gap-1 text-sm text-neutral-500"><Clock class="size-3.5"/>{t("dealsSendsAt",{time:formatDate(post.sendAt)})}</p>{/if}
     {#if post.state==="failed"}<p class="text-sm text-red-700 dark:text-red-400">{failureText(post.lastError)}</p>{/if}
    </div>
    {#if post.state==="sent"}<span class="shrink-0 text-sm font-semibold tabular-nums">{clicks(post.clicks)}</span>{/if}
   </div>
   {#snippet actions()}<a href={post.link} target="_blank" rel="noopener noreferrer nofollow" class="relative z-10 grid size-9 place-items-center rounded-lg text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white" aria-label={post.link}><ExternalLink class="size-4"/></a>{/snippet}
  </ListItem>
 {/each}
</List></div>
<Pager {pager}/>
