<script lang="ts">
 import {appURLs} from "#lib/app-urls.ts";
 import type {Component} from "svelte";
 import {resolve} from "$app/paths";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import CircleDashed from "@lucide/svelte/icons/circle-dashed";
 import ChevronRight from "@lucide/svelte/icons/chevron-right";
 import ChevronDown from "@lucide/svelte/icons/chevron-down";
 import CreditCard from "@lucide/svelte/icons/credit-card";
 import FolderPlus from "@lucide/svelte/icons/folder-plus";
 import Bug from "@lucide/svelte/icons/bug";
 import Activity from "@lucide/svelte/icons/activity";
 import Globe from "@lucide/svelte/icons/globe";
 import {t} from "#lib/i18n.svelte.ts";
 import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
 import BrandIcon from "#lib/ui/BrandIcon.svelte";
 import {cloudPaths} from "../../../shared/panel-paths";
 import {cloudURL} from "./api";
 import type {SetupStep} from "./setup";
 // The first steps of a new account, under every page until all are
 // done. The open step is the first one left (another can be opened); the
 // plan comes first, and until it is active the others wait for it. The
 // whole box can be folded away, remembered on this device.
 let {steps}:{steps:{id:SetupStep;done:boolean}[]}=$props();
 type Step={title:PlainMessageKey;text:PlainMessageKey;action:PlainMessageKey;href:string;Icon?:Component<{class?:string}>};
 const content:Record<SetupStep,Step>={
  plan:{title:"pulseSetupPlan",text:"pulseSetupPlanText",action:"pulseSetupPlanAction",href:`${appURLs.cloud}${cloudPaths.plan}`,Icon:CreditCard},
  project:{title:"pulseSetupProject",text:"pulseSetupProjectText",action:"pulseCreateNewProject",href:resolve("/projects/new"),Icon:FolderPlus},
  event:{title:"pulseSetupEvent",text:"pulseSetupEventText",action:"pulseSetupEventAction",href:resolve("/"),Icon:Bug},
  monitor:{title:"pulseSetupMonitor",text:"pulseSetupMonitorText",action:"pulseCreateMonitor",href:resolve("/monitors/new"),Icon:Activity},
  alert:{title:"pulseSetupAlert",text:"pulseSetupAlertText",action:"pulseAddChannel",href:resolve("/alerts/new")},
  page:{title:"pulseSetupPage",text:"pulseSetupPageText",action:"pulseCreatePage",href:resolve("/status-pages/new"),Icon:Globe},
 };
 const total=$derived(steps.length),left=$derived(steps.filter(s=>!s.done).length);
 const planReady=$derived(steps.find(s=>s.id==="plan")?.done??false);
 // Follows the first step left when the steps change; a click opens another.
 let open=$derived<SetupStep|null>(steps.find(s=>!s.done)?.id??null);
 const FOLD="blup-pulse-setup-folded";
 let folded=$state(read());
 function read(){try{return localStorage.getItem(FOLD)==="1";}catch{return false;}}
 function fold(){folded=!folded;try{localStorage.setItem(FOLD,folded?"1":"0");}catch{/* private mode: only for now */}}
 // The ring: how much is done.
 const R=8,C=2*Math.PI*R;
</script>

<section class="mt-10 rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900" aria-label={t("pulseSetupTitle")}>
 <div class="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
  <h2 class="font-bold">{t("pulseSetupTitle")}</h2>
  <div class="flex items-center gap-3 text-sm text-neutral-500">
   <svg class="size-5 -rotate-90" viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="10" r={R} fill="none" stroke-width="2.5" class="stroke-neutral-200 dark:stroke-neutral-700"/><circle cx="10" cy="10" r={R} fill="none" stroke-width="2.5" stroke-linecap="round" class="stroke-blup transition-[stroke-dashoffset]" stroke-dasharray={C} stroke-dashoffset={C*left/Math.max(1,total)}/></svg>
   <span>{t("pulseSetupLeft",{left:String(left),total:String(total)})}</span>
   <button type="button" class="grid size-8 place-items-center rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-expanded={!folded} aria-label={folded?t("pulseSetupShow"):t("pulseSetupFold")} onclick={fold}><ChevronDown class="size-4 transition-transform {folded?'-rotate-90':''}"/></button>
  </div>
 </div>
 {#if !folded}
  <ol class="px-2 pb-2">
   {#each steps as step(step.id)}
    {@const c=content[step.id]}
    {@const locked=!planReady&&step.id!=="plan"}
    <li class="border-t border-neutral-200 first:border-t-0 dark:border-neutral-800">
     {#if open===step.id}
      <div class="my-1 grid gap-4 rounded-lg bg-neutral-50 p-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center dark:bg-neutral-800/40">
       <div class="min-w-0">
        <p class="flex items-center gap-3 font-semibold {step.done?'text-blup':''}">{#if step.done}<CircleCheck class="size-5 shrink-0 text-blup"/>{:else}<CircleDashed class="size-5 shrink-0 text-neutral-400"/>{/if}{t(c.title)}</p>
        <p class="mt-2 max-w-prose pl-8 text-sm text-neutral-500">{locked?t("pulseSetupLocked"):t(c.text)}</p>
        {#if !locked}<a href={c.href} class="mt-4 ml-8 inline-flex items-center gap-2 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900">{t(c.action)}</a>{/if}
       </div>
       <div class="hidden items-center gap-3 pr-2 sm:flex" aria-hidden="true">
        {#if c.Icon}<span class="grid size-16 place-items-center rounded-2xl border border-neutral-200 bg-white text-blup dark:border-neutral-700 dark:bg-neutral-900"><c.Icon class="size-7"/></span>
        {:else}<span class="grid size-16 place-items-center rounded-2xl border border-neutral-200 bg-white dark:border-neutral-700 dark:bg-neutral-900"><BrandIcon name="discord" class="size-7"/></span><span class="grid size-16 place-items-center rounded-2xl border border-neutral-200 bg-white dark:border-neutral-700 dark:bg-neutral-900"><BrandIcon name="telegram" class="size-7"/></span>{/if}
       </div>
      </div>
     {:else}
      <button type="button" class="flex w-full items-center gap-3 rounded-lg px-4 py-3.5 text-left font-semibold hover:bg-neutral-50 dark:hover:bg-neutral-800/40 {step.done?'text-blup':''}" onclick={()=>open=step.id}>
       {#if step.done}<CircleCheck class="size-5 shrink-0 text-blup"/>{:else}<CircleDashed class="size-5 shrink-0 text-neutral-400"/>{/if}
       <span class="min-w-0 flex-1">{t(c.title)}</span><ChevronRight class="size-4 shrink-0 text-neutral-400"/>
      </button>
     {/if}
    </li>
   {/each}
  </ol>
 {/if}
</section>
