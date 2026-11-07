<script lang="ts">
 import {appURLs} from "#lib/app-urls.ts";
 import type {LayoutProps} from "./$types";
 import {resolve} from "$app/paths";
 import {page} from "$app/state";
 import type {RouteId} from "$app/types";
 import "../app.css";
 import {onDestroy} from "svelte";
 import {toast} from "#lib/toast.svelte.ts";
 import PulseSelection from "#lib/ui/PulseSelection.svelte";
 import Button from "#lib/ui/Button.svelte";
 import Bell from "@lucide/svelte/icons/bell";
 import Activity from "@lucide/svelte/icons/activity";
 import Globe from "@lucide/svelte/icons/globe";
 import Folder from "@lucide/svelte/icons/folder";
 import Bug from "@lucide/svelte/icons/bug";
 import {intlLocale,language,i18n,t} from "#lib/i18n.svelte.ts";
 import {createLanguageSaver} from "#lib/language-save.ts";
 import PanelShell from "#lib/ui/PanelShell.svelte";
 import Toaster from "#lib/ui/Toaster.svelte";
 import ConfirmHost from "#lib/ui/ConfirmHost.svelte";
 import SignOutButton from "#lib/ui/SignOutButton.svelte";
 import {cloudURL,rpc,pulseApi,parseResponse} from "../lib/api";
 import {cloudPaths} from "../../../shared/panel-paths";
 import {panel} from "../lib/panel.svelte";
 import {setupDone} from "../lib/setup";
 import SetupGuide from "../lib/SetupGuide.svelte";
 let {children}:LayoutProps=$props();
 const links=$derived([{href:resolve("/"),label:t("pulseProjects"),Icon:Folder,match:(path:string)=>path===resolve("/")||path===resolve("/projects/new")},{href:resolve("/issues"),label:t("pulseIssues"),Icon:Bug},{href:resolve("/monitors"),label:t("pulseMonitors"),Icon:Activity,match:(path:string)=>path.startsWith(resolve("/monitors"))},{href:resolve("/status-pages"),label:t("pulsePages"),short:t("pulsePagesShort"),Icon:Globe,match:(path:string)=>path.startsWith(resolve("/status-pages"))},{href:resolve("/alerts"),label:t("pulseAlerts"),Icon:Bell,match:(path:string)=>path.startsWith(resolve("/alerts"))}]);
 // The first steps show under the main pages of the menu, not on forms or
 // on one item's page.
 const mainPages:ReadonlySet<string>=new Set(["/","/issues","/monitors","/status-pages","/alerts"] satisfies RouteId[]);
 const setupLeft=$derived(!!panel.setup&&!setupDone(panel.setup)),guide=$derived(setupLeft&&mainPages.has(page.route.id??""));
 let selectionBusy=$state(false),keep=$state({projects:[] as string[],monitors:[] as string[],pages:[] as string[]});
 async function saveSelection(){selectionBusy=true;try{await parseResponse(pulseApi.selection.$put({json:keep}));panel.usage=await parseResponse(pulseApi.usage.$get());location.reload();}catch{toast(()=>t("errorInvalid"),{kind:"error"});}finally{selectionBusy=false;}}
 // Signing out ends the Blup session, then opens the sign-in again (a
 // full load, so nothing of the account stays in memory).
 async function signOut(){try{await fetch(`${cloudURL}/api/auth/sign-out`,{method:"POST",credentials:"include",headers:{"content-type":"application/json"},body:"{}"});}finally{location.href=resolve("/sign-in");}}
 let alive=true;onDestroy(()=>{alive=false;});
 const saver=createLanguageSaver((language)=>parseResponse(rpc.me.language.$put({json:{language}})),id=>alive&&panel.me?.id===id,()=>toast(()=>t("errorGeneric"),{kind:"error"}));
 $effect(()=>{document.documentElement.lang=intlLocale();});
 $effect(()=>{const me=panel.me,selected=language(),revision=i18n.revision;void revision;if(me)void saver.schedule(me.id,selected,me.language);});
</script>
<svelte:head><title>Blup Pulse</title><meta name="robots" content="noindex,nofollow" /></svelte:head>
<!-- An error page and the sign-in stand alone, outside the panel. -->
{#if page.error||page.route.id==="/sign-in"}{@render children()}
{:else if !panel.ready}<main class="p-10"><p>{t("loading")}</p></main>
{:else if !panel.me}<main class="mx-auto max-w-xl p-10"><p>{t("errorGeneric")}</p><Button class="mt-4" onclick={()=>location.reload()}>{t("retry")}</Button></main>
{:else}<PanelShell {links} home={resolve("/")} app={{current:"pulse",links:{cloud:`${appURLs.cloud}${cloudPaths.account}`,pulse:resolve("/"),deals:appURLs.deals}}} languages wide>
 {#snippet foot(compact)}<SignOutButton onclick={signOut} {compact}/>{/snippet}
 {#if !setupLeft&&!panel.usage?.plan}<p class="mb-6 rounded-lg bg-emerald-50 p-4 dark:bg-emerald-950">{t("pulseNeedPlan")} <a href={`${appURLs.cloud}${cloudPaths.plan}`} class="underline">{t("subscription")}</a></p>{/if}
 {#if panel.usage?.readOnlyUntil}<p class="mb-6 rounded-lg bg-amber-50 p-4 dark:bg-amber-950">{t("pulseReadOnly",{date:new Intl.DateTimeFormat(intlLocale(),{dateStyle:"long"}).format(new Date(panel.usage.readOnlyUntil))})}</p>{/if}
 {#if panel.usage?.selectionRequired&&panel.usage.limits}<PulseSelection resources={panel.usage.resources} limits={panel.usage.limits} bind:keep/><Button busy={selectionBusy} onclick={saveSelection}>{t("save")}</Button>{/if}
 {@render children()}
 <!-- The first steps, under the main pages only, until all are done. -->
 {#if guide&&panel.setup}<SetupGuide steps={panel.setup.steps}/>{/if}
 </PanelShell>{/if}
<Toaster /><ConfirmHost />
