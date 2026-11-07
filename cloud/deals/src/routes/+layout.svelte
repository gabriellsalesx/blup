<script lang="ts">
 import {appURLs} from "#lib/app-urls.ts";
 import type {LayoutProps} from "./$types";
 import {resolve} from "$app/paths";
 import {page} from "$app/state";
 import type {RouteId} from "$app/types";
 import "../app.css";
 import {onDestroy} from "svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Tag from "@lucide/svelte/icons/tag";
 import Crosshair from "@lucide/svelte/icons/crosshair";
 import Send from "@lucide/svelte/icons/send";
 import Handshake from "@lucide/svelte/icons/handshake";
 import MousePointerClick from "@lucide/svelte/icons/mouse-pointer-click";
 import {intlLocale,language,i18n,t,formatDate} from "#lib/i18n.svelte.ts";
 import {createLanguageSaver} from "#lib/language-save.ts";
 import PanelShell from "#lib/ui/PanelShell.svelte";
 import Toaster from "#lib/ui/Toaster.svelte";
 import ConfirmHost from "#lib/ui/ConfirmHost.svelte";
 import SignOutButton from "#lib/ui/SignOutButton.svelte";
 import {rpc,parseResponse} from "../lib/api";
 import {cloudPaths} from "../../../shared/panel-paths";
 import {panel} from "../lib/panel.svelte";
 import {setupDone} from "../lib/setup";
 import SetupGuide from "../lib/SetupGuide.svelte";
 let {children}:LayoutProps=$props();
 // The first steps show under the main pages of the menu, not on forms or
 // on one item's page (as in Pulse).
 const mainPages:ReadonlySet<string>=new Set(["/","/hunts","/channels","/affiliates","/stats"] satisfies RouteId[]);
 const setupLeft=$derived(!!panel.setup&&!setupDone(panel.setup)),guide=$derived(setupLeft&&mainPages.has(page.route.id??""));
 // The five tabs; a form opened from a tab keeps that tab lit.
 const under=(base:string)=>(path:string)=>path===base||path.startsWith(`${base}/`);
 const links=$derived([
  {href:resolve("/"),label:t("dealsOffers"),Icon:Tag,match:(path:string)=>path===resolve("/")||path===resolve("/offers/new")||path===resolve("/offers/video")||path===resolve("/settings")},
  {href:resolve("/hunts"),label:t("dealsHunts"),Icon:Crosshair,match:under(resolve("/hunts"))},
  {href:resolve("/channels"),label:t("dealsChannels"),Icon:Send,match:under(resolve("/channels"))},
  {href:resolve("/affiliates"),label:t("dealsAffiliates"),Icon:Handshake},
  {href:resolve("/stats"),label:t("dealsStats"),Icon:MousePointerClick},
 ]);
 // The tab's title: the page's name, then the app's.
 const pageTitles:Partial<Record<RouteId,()=>string>>={
  "/":()=>t("dealsOffers"),"/offers/new":()=>t("dealsPasteTitle"),"/hunts":()=>t("dealsHunts"),"/hunts/new":()=>t("dealsNewHuntTitle"),"/hunts/[id]":()=>t("dealsEditHunt"),
  "/channels":()=>t("dealsChannels"),"/channels/new":()=>t("dealsNewChannelTitle"),"/affiliates":()=>t("dealsAffiliates"),"/stats":()=>t("dealsStats"),
 };
 const title=$derived.by(()=>{const name=pageTitles[(page.route.id??"") as RouteId]?.();return name?`${name} · Blup Deals`:"Blup Deals";});
 // Signing out ends the Blup session, then opens the sign-in again (a
 // full load, so nothing of the account stays in memory).
 async function signOut(){try{const {dealsAuth}=await import("../lib/auth");await dealsAuth.signOut();}finally{location.href=resolve("/sign-in");}}
 let alive=true;onDestroy(()=>{alive=false;});
 const saver=createLanguageSaver((language)=>parseResponse(rpc.me.language.$put({json:{language}})),id=>alive&&panel.me?.id===id,()=>toast(()=>t("errorGeneric"),{kind:"error"}));
 $effect(()=>{document.documentElement.lang=intlLocale();});
 $effect(()=>{const me=panel.me,selected=language(),revision=i18n.revision;void revision;if(me)void saver.schedule(me.id,selected,me.language);});
</script>
<svelte:head>{#if !page.error&&page.route.id!=="/sign-in"}<title>{title}</title>{/if}<meta name="robots" content="noindex,nofollow" /></svelte:head>
<!-- An error page and the sign-in stand alone, outside the panel. -->
{#if page.error||page.route.id==="/sign-in"}{@render children()}
{:else if !panel.ready}<main class="p-10"><p>{t("loading")}</p></main>
{:else if !panel.me}<main class="mx-auto max-w-xl p-10"><p>{t("errorGeneric")}</p><Button class="mt-4" onclick={()=>location.reload()}>{t("retry")}</Button></main>
{:else}<PanelShell {links} home={resolve("/")} app={{current:"deals",links:{cloud:`${appURLs.cloud}${cloudPaths.account}`,pulse:appURLs.pulse,deals:resolve("/")}}} languages wide>
 {#snippet foot(compact)}<SignOutButton onclick={signOut} {compact}/>{/snippet}
 <!-- After losing the plan: read-only with the date the data goes; else
  no plan once the first steps are done (until then, their first step
  says it); else read-only for another reason. -->
 {#if panel.usage?.readOnlyUntil}<p class="mb-6 rounded-lg bg-amber-50 p-4 dark:bg-amber-950">{t("dealsReadOnlyUntil",{date:formatDate(panel.usage.readOnlyUntil,{dateStyle:"long"})})} <a href={`${appURLs.cloud}${cloudPaths.plan}`} class="underline">{t("subscription")}</a></p>
 {:else if !setupLeft&&!panel.usage?.plan}<p class="mb-6 rounded-lg bg-orange-50 p-4 dark:bg-orange-950">{t("dealsNeedPlan")} <a href={`${appURLs.cloud}${cloudPaths.plan}`} class="underline">{t("subscription")}</a></p>
 {:else if panel.usage?.plan&&panel.usage.readOnly}<p class="mb-6 rounded-lg bg-amber-50 p-4 dark:bg-amber-950">{t("dealsReadOnly")} <a href={`${appURLs.cloud}${cloudPaths.plan}`} class="underline">{t("subscription")}</a></p>{/if}
 {@render children()}
 <!-- The first steps, under the main pages only, until all are done. -->
 {#if guide&&panel.setup}<SetupGuide steps={panel.setup.steps}/>{/if}
 </PanelShell>{/if}
<Toaster /><ConfirmHost />
