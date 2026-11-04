<script lang="ts">
 import type {LayoutProps} from "./$types";
 import {resolve} from "$app/paths";
 import {page} from "$app/state";
 import "../app.css";
 import {onDestroy} from "svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Tag from "@lucide/svelte/icons/tag";
 import Crosshair from "@lucide/svelte/icons/crosshair";
 import Send from "@lucide/svelte/icons/send";
 import Handshake from "@lucide/svelte/icons/handshake";
 import MousePointerClick from "@lucide/svelte/icons/mouse-pointer-click";
 import {intlLocale,language,i18n,t} from "#lib/i18n.svelte.ts";
 import {createLanguageSaver} from "#lib/language-save.ts";
 import PanelShell from "#lib/ui/PanelShell.svelte";
 import Toaster from "#lib/ui/Toaster.svelte";
 import ConfirmHost from "#lib/ui/ConfirmHost.svelte";
 import SignOutButton from "#lib/ui/SignOutButton.svelte";
 import {cloudURL,rpc,parseResponse} from "../lib/api";
 import {cloudPaths,pulseAppURL} from "../../../shared/panel-paths";
 import {panel} from "../lib/panel.svelte";
 let {children}:LayoutProps=$props();
 // The five tabs; a form opened from a tab keeps that tab lit.
 const under=(base:string)=>(path:string)=>path===base||path.startsWith(`${base}/`);
 const links=$derived([
  {href:resolve("/"),label:t("dealsOffers"),Icon:Tag,match:(path:string)=>path===resolve("/")||path===resolve("/offers/new")||path===resolve("/start")},
  {href:resolve("/hunts"),label:t("dealsHunts"),Icon:Crosshair,match:under(resolve("/hunts"))},
  {href:resolve("/channels"),label:t("dealsChannels"),Icon:Send,match:under(resolve("/channels"))},
  {href:resolve("/affiliates"),label:t("dealsAffiliates"),Icon:Handshake},
  {href:resolve("/stats"),label:t("dealsStats"),Icon:MousePointerClick},
 ]);
 // Signing out ends the Blup session, then opens the sign-in again (a
 // full load, so nothing of the account stays in memory).
 async function signOut(){try{await fetch(`${cloudURL}/api/auth/sign-out`,{method:"POST",credentials:"include",headers:{"content-type":"application/json"},body:"{}"});}finally{location.href=resolve("/sign-in");}}
 let alive=true;onDestroy(()=>{alive=false;});
 const saver=createLanguageSaver((language)=>parseResponse(rpc.me.language.$put({json:{language}})),id=>alive&&panel.me?.id===id,()=>toast(()=>t("errorGeneric"),{kind:"error"}));
 $effect(()=>{document.documentElement.lang=intlLocale();});
 $effect(()=>{const me=panel.me,selected=language(),revision=i18n.revision;void revision;if(me)void saver.schedule(me.id,selected,me.language);});
</script>
<svelte:head><title>Blup Deals</title><meta name="robots" content="noindex,nofollow" /></svelte:head>
<!-- An error page and the sign-in stand alone, outside the panel. -->
{#if page.error||page.route.id==="/sign-in"}{@render children()}
{:else if !panel.ready}<main class="p-10"><p>{t("loading")}</p></main>
{:else if !panel.me}<main class="mx-auto max-w-xl p-10"><p>{t("errorGeneric")}</p><Button class="mt-4" onclick={()=>location.reload()}>{t("retry")}</Button></main>
{:else}<PanelShell {links} home={resolve("/")} app={{current:"deals",links:{cloud:`${cloudURL}${cloudPaths.account}`,pulse:pulseAppURL,deals:resolve("/")}}} languages wide>
 {#snippet foot(compact)}<SignOutButton onclick={signOut} {compact}/>{/snippet}
 {#if !panel.usage?.plan}<p class="mb-6 rounded-lg bg-orange-50 p-4 dark:bg-orange-950">{t("dealsNeedPlan")} <a href={`${cloudURL}${cloudPaths.plan}`} class="underline">{t("subscription")}</a></p>{/if}
 {@render children()}
 </PanelShell>{/if}
<Toaster /><ConfirmHost />
