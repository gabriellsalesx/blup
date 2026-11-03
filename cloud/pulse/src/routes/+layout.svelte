<script lang="ts">
 import "../app.css";
 import {onMount,onDestroy} from "svelte";
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
 import LanguagePicker from "#lib/ui/LanguagePicker.svelte";
 import ThemePicker from "#lib/ui/ThemePicker.svelte";
 import {cloudURL,rpc,pulseApi,parseResponse} from "../lib/api";
 import {panel} from "../lib/panel.svelte";
 let {children}=$props();
 const links=$derived([{href:"/",label:t("pulseProjects"),Icon:Folder},{href:"/problemas",label:t("pulseIssues"),Icon:Bug},{href:"/monitores",label:t("pulseMonitors"),Icon:Activity},{href:"/paginas",label:t("pulsePages"),Icon:Globe},{href:"/avisos",label:t("pulseAlerts"),Icon:Bell}]);
 let selectionBusy=$state(false),keep=$state({projects:[] as string[],monitors:[] as string[],pages:[] as string[]});
 async function saveSelection(){selectionBusy=true;try{await parseResponse(pulseApi.selection.$put({json:keep}));panel.usage=await parseResponse(pulseApi.usage.$get());location.reload();}catch{toast(()=>t("errorInvalid"),{kind:"error"});}finally{selectionBusy=false;}}
 let alive=true;onDestroy(()=>{alive=false;});
 const saver=createLanguageSaver((language)=>parseResponse(rpc.me.language.$put({json:{language}})),id=>alive&&panel.me?.id===id,()=>toast(()=>t("errorGeneric"),{kind:"error"}));
 onMount(()=>{void Promise.all([parseResponse(rpc.me.$get()),parseResponse(pulseApi.usage.$get())]).then(([me,usage])=>{panel.me=me;panel.usage=usage;}).catch(()=>{panel.error=true;}).finally(()=>{panel.ready=true;});});
 $effect(()=>{document.documentElement.lang=intlLocale();});
 $effect(()=>{const me=panel.me,selected=language(),revision=i18n.revision;void revision;if(me)void saver.schedule(me.id,selected,me.language);});
</script>
<svelte:head><title>Blup Pulse</title><meta name="robots" content="noindex,nofollow" /></svelte:head>
{#if !panel.ready}<main class="p-10"><p>{t("loading")}</p></main>
{:else if !panel.me}<main class="mx-auto max-w-xl p-10"><img src="/mascot.svg" alt="" class="mb-6 size-24" /><h1 class="text-3xl font-bold">Blup Pulse</h1><h2 class="mt-5 text-xl font-semibold">{t("pulseWelcome")}</h2><p class="my-4">{t("pulseLead")}</p><a class="font-semibold text-blup underline" href={`${cloudURL}/entrar?volta=/conta/pulse`}>{t("pulseLogin")}</a><div class="mt-6 flex gap-2"><LanguagePicker /><ThemePicker /></div></main>
{:else}<PanelShell {links} home="/" tag="Pulse" mint languages wide>
 {#snippet foot()}<a class="px-2.5 text-sm text-neutral-500 underline" href={`${cloudURL}/conta`}>{t("pulseCloud")}</a>{/snippet}
 {#if !panel.usage?.plan}<p class="mb-6 rounded-lg bg-emerald-50 p-4 dark:bg-emerald-950">{t("pulseNeedPlan")} <a href={`${cloudURL}/conta/plano`} class="underline">{t("subscription")}</a></p>{/if}
 {#if panel.usage?.readOnlyUntil}<p class="mb-6 rounded-lg bg-amber-50 p-4 dark:bg-amber-950">{t("pulseReadOnly",{date:new Intl.DateTimeFormat(intlLocale(),{dateStyle:"long"}).format(new Date(panel.usage.readOnlyUntil))})}</p>{/if}
 {#if panel.usage?.selectionRequired&&panel.usage.limits}<PulseSelection resources={panel.usage.resources} limits={panel.usage.limits} bind:keep/><Button busy={selectionBusy} onclick={saveSelection}>{t("save")}</Button>{/if}
 {@render children()}
 </PanelShell>{/if}
<Toaster /><ConfirmHost />
