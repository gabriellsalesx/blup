<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {invalidateAll,goto} from "$app/navigation";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import ExternalLink from "@lucide/svelte/icons/external-link";
 import Info from "@lucide/svelte/icons/info";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Button from "#lib/ui/Button.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import {dealsApi,parseResponse} from "../../../lib/api";
 import {dealsError} from "../../../lib/errors";
 import {hasPlan} from "../../../lib/plan";
 import {stores,storeName} from "../../../lib/catalog";
 import StoreLogo from "../../../lib/StoreLogo.svelte";
 import StoreForm from "../../../lib/StoreForm.svelte";
 // A store's own page, as an integration: its mark, what connecting does
 // and what it takes, the steps to get the details and, once Connect is
 // pressed (or after connecting), its settings; connected, it only manages
 // them (change or remove).
 let {data}:PageProps=$props();
 const info=$derived(stores.find(s=>s.id===data.store)!);
 const name=$derived(storeName[data.store]);
 const connected=$derived(!!data.saved?.configured);
 let adding=$state(false),busy=$state(false);
 const facts=$derived([
  {label:t("dealsFactNeeds"),value:info.fields.filter(f=>!f.optional).map(f=>f.label?t(f.label):f.text??f.id).join(", ")},
  {label:t("dealsFactCommission"),value:t("dealsPaidByStore")},
  {label:t("dealsFactLinks"),value:t("dealsLinksAuto")},
  {label:t("dealsFactPrice"),value:t("dealsFree")},
 ]);
 async function remove(){if(!await confirm({title:t("dealsRemoveIntegration"),message:t("dealsRemoveAffiliate"),danger:true}))return;busy=true;try{await parseResponse(dealsApi.affiliates[":store"].$delete({param:{store:data.store}}));toast(()=>t("dealsRemoved"));await goto(resolve("/affiliates"),{invalidateAll:true});}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 function saved(){adding=false;void invalidateAll();}
</script>

<div class="mb-4"><BackLink href={resolve("/affiliates")} label={t("dealsAffiliates")}/></div>

<header class="flex flex-col gap-5 sm:flex-row sm:items-start">
 <StoreLogo store={data.store} class="size-20 sm:size-24"/>
 <div class="min-w-0">
  <h1 class="text-2xl font-bold sm:text-3xl">{name}</h1>
  <p class="mt-1 text-neutral-600 dark:text-neutral-300">{t(info.about)}</p>
  <div class="mt-4">
   {#if connected}<span class="inline-flex items-center gap-1.5 rounded-lg bg-emerald-50 px-3 py-1.5 text-sm font-semibold text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400"><CircleCheck class="size-4"/>{t("dealsConfigured")}</span>
   {:else if hasPlan()&&!adding}<Button onclick={()=>adding=true}>{t("dealsIntegrate")}</Button>{/if}
  </div>
 </div>
</header>

<dl class="mt-8 grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-neutral-200 bg-neutral-200 sm:grid-cols-4 dark:border-neutral-800 dark:bg-neutral-800">
 {#each facts as fact(fact.label)}
  <div class="bg-white px-4 py-4 text-center dark:bg-neutral-900"><dt class="text-xs font-semibold tracking-wide text-neutral-500 uppercase">{fact.label}</dt><dd class="mt-1.5 text-sm font-semibold break-words">{fact.value}</dd></div>
 {/each}
</dl>

<p class="mt-6 flex items-start gap-3 rounded-xl border border-neutral-200 bg-white p-4 text-sm dark:border-neutral-800 dark:bg-neutral-900"><Info class="mt-0.5 size-4 shrink-0 text-neutral-500"/><span>{t("dealsCommissionNote",{store:name})} <a class="inline-flex items-center gap-1 font-semibold text-blup underline" href={info.site} target="_blank" rel="noopener noreferrer">{t("dealsStoreSite")}<ExternalLink class="size-3.5"/></a></span></p>

<div class="mt-10 grid grid-cols-[minmax(0,1fr)] gap-10">
 {#if connected||adding}
  <FormSection title={t("dealsSettings")} help={t("dealsSettingsHelp")}>
   <StoreForm store={data.store} saved={data.saved} writable={hasPlan()} onchange={saved} oncancel={connected?undefined:()=>adding=false}/>
  </FormSection>
 {/if}
 <FormSection title={t("dealsHowItWorks")} help={t("dealsHowItWorksHelp")}>
  <ol class="grid list-decimal gap-2 pl-5 text-sm text-neutral-600 dark:text-neutral-300">
   {#each info.steps as step(step)}<li>{t(step)}</li>{/each}
  </ol>
 </FormSection>
 {#if connected&&hasPlan()}
  <FormSection title={t("dealsRemoveIntegration")} help={t("dealsRemoveIntegrationHelp")}>
   <button type="button" class="inline-flex items-center gap-2 justify-self-start rounded-lg bg-red-600 px-4 py-2.5 font-semibold text-white transition-colors hover:bg-red-700 disabled:opacity-50" disabled={busy} onclick={remove}><Trash2 class="size-4"/>{t("dealsRemoveIntegration")}</button>
  </FormSection>
 {/if}
</div>
