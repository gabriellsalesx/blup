<script lang="ts">
 import {resolve} from "$app/paths";
 import {refreshAll} from "$app/navigation";
 import type {InferResponseType} from "hono/client";
 import {parseResponse} from "hono/client";
 import {planLabel,planPerks} from "../../../../../shared/plan-i18n";
 import type {PlanName} from "../../../../../shared/pulse-plans";
 import {t,language,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {rpc,money} from "#lib/api.ts";
 import Button from "#lib/ui/Button.svelte";
 import Row from "#lib/ui/Row.svelte";
 import PulseSelection from "#lib/ui/PulseSelection.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import Check from "@lucide/svelte/icons/check";
 let {data}=$props();
 type Preview=InferResponseType<typeof rpc.billing.preview.$post,200>;
 type BillingState=InferResponseType<typeof rpc.billing.state.$get,200>;
 type Result=InferResponseType<typeof rpc.billing.confirm.$post,200>;
 let busy=$state(false),error=$state<unknown>(null),preview=$state<Preview|null>(null),result=$state<Result|null>(null);
 // They start from the current plan: its billing period and whether it has Pulse.
 let yearly=$derived(data.me.plan.billingInterval==="year"),withPulse=$derived<Record<string,boolean>>({max:false,ultra:false,...(data.me.plan.pulse.enabled?{[data.me.plan.name]:true}:{})}),billing=$derived<BillingState|null>(data.billing);
 let keep=$state({projects:[] as string[],monitors:[] as string[],pages:[] as string[]});
 const current=$derived(data.me.plan),paid=$derived(current.name!=="free"),gift=$derived(current.gift);
 const bases=$derived(data.plans.filter(p=>p.name===p.basePlan));
 const selectionValid=$derived(!preview?.quote.selectionRequired||!!preview.quote.limits&&keep.projects.length>0&&keep.projects.length<=preview.quote.limits.projects&&keep.monitors.length<=preview.quote.limits.monitors&&keep.pages.length<=preview.quote.limits.pages);
 const when=(value:string)=>formatDate(new Date(value));
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function refresh(){billing=await parseResponse(rpc.billing.state.$get());}
 async function choose(plan:PlanName){await run(async()=>{result=null;if(!paid){if(plan==="free")return;const response=await parseResponse(rpc.billing.checkout.$post({json:{plan,annual:yearly}}));location.assign(response.url);return;}preview=await parseResponse(rpc.billing.preview.$post({json:{plan,annual:yearly}}));keep={projects:[],monitors:[],pages:[]};});}
 async function confirmChange(){const value=preview;if(!value)return;await run(async()=>{result=await parseResponse(rpc.billing.confirm.$post({json:{ticket:value.ticket,...(value.quote.selectionRequired?{keep}: {})}}));preview=null;await refresh();await refreshAll();});}
 async function portal(){await run(async()=>{const response=await parseResponse(rpc.billing.portal.$post({json:{}}));location.assign(response.url);});}
 async function undo(){await run(async()=>{await parseResponse(rpc.billing.pending.$delete());await refresh();});}
 async function cancelCheckout(){await run(async()=>{await parseResponse(rpc.billing.checkout.$delete());await refresh();});}
</script>
<PageHeader title={t("plan")} lead={t("planLead")}/>
{#if error}<p class="my-4 text-red-600" role="alert">{errorMessage(error)}</p>{/if}
{#if result}<div class="my-5 rounded-xl border border-blup p-5" role="status"><p>{result.state==="scheduled"?t("billingScheduled",{date:when(result.effectiveAt)}):t("billingWaiting")}</p>{#if result.paymentUrl}<a class="mt-3 block text-blup underline" href={result.paymentUrl}>{t("billingPay")}</a>{/if}</div>{/if}
{#if billing?.paymentUrl}<a class="my-4 block text-blup underline" href={billing.paymentUrl}>{t("billingPay")}</a>{/if}
{#if billing?.checkout}<div class="my-5 grid gap-3 rounded-xl border p-5"><a class="text-blup underline" href={billing.checkout.url}>{t("billingResumeCheckout")}</a><Button variant="secondary" {busy} onclick={cancelCheckout}>{t("billingCancelCheckout")}</Button></div>{/if}
{#if billing?.change}<div class="my-5 grid gap-3 rounded-xl border p-5"><p>{t("billingScheduled",{date:when(billing.change.effectiveAt)})} {planLabel(billing.change.plan,language())}</p>{#if Date.parse(billing.change.effectiveAt)>Date.now()}<Button variant="secondary" {busy} onclick={undo}>{t("billingCancelPending")}</Button>{/if}</div>{/if}
<div class="mt-8"><Segmented label={t("billing")} options={[{value:"month",label:t("monthly")},{value:"year",label:t("yearly")}]} bind:value={()=>yearly?"year":"month",(v)=>yearly=v==="year"}/></div>
<div class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">{#each bases as base(base.name)}
 {@const pulseable=base.name==="max"||base.name==="ultra"}{@const pulse=pulseable&&!!withPulse[base.name]}
 {@const p=data.plans.find(p=>p.name===(pulse?`${base.name}_pulse`:base.name))??base}
 {@const mine=p.name===current.billingName}{@const price=p.prices[yearly?"yearly":"monthly"]}{@const perks=planPerks(language(),p.name,p.limits)}{@const baseCount=planPerks(language(),base.name,base.limits).length}
 <div class="flex flex-col rounded-xl border bg-white p-5 transition-colors dark:bg-neutral-900 {mine?'border-blup ring-1 ring-blup':pulse?'border-emerald-500 ring-1 ring-emerald-500':'border-neutral-200 dark:border-neutral-800'}"><h2 class="font-semibold">{planLabel(p.name,language())}</h2>{#if mine}<span class="mt-1 text-xs text-blup">{gift?t("gift"):t("yourPlan")}</span>{/if}
 {#if p.name==="free"}<p class="mt-2 text-2xl font-bold">{planLabel("free",language())}</p>{:else if price}<p class="mt-2"><b class="text-2xl">{money(price.unitAmount,price.currency,language())}</b> <span class="text-sm text-neutral-500">{yearly?t("perYear"):t("perMonth")}</span></p>{:else}<p class="mt-2 text-sm">{t("pricingUnavailable")}</p>{/if}
 {#if pulseable}<div class="mt-4 border-t border-neutral-200 pt-4 dark:border-neutral-800"><Switch tone="mint" label={t("billingAddPulse")} checked={pulse} onchange={(on)=>withPulse={...withPulse,[base.name]:on}}/></div>{/if}
 <div class="my-5 flex-1"><ul class="grid content-start gap-2 text-sm">{#each perks.slice(0,baseCount) as perk(perk)}<li class="flex gap-2"><Check class="mt-0.5 size-4 shrink-0 text-green-600"/>{perk}</li>{/each}</ul>{#if pulse}<div class="mt-4 rounded-lg bg-emerald-50 p-3 dark:bg-emerald-950/40"><p class="mb-2 text-sm font-bold text-emerald-700 dark:text-emerald-300">Blup Pulse</p><ul class="grid gap-2 text-sm">{#each perks.slice(baseCount+1) as perk(perk)}<li class="flex gap-2"><Check class="mt-0.5 size-4 shrink-0 text-emerald-600"/>{perk}</li>{/each}</ul></div>{/if}</div>
 {#if !gift&&p.name!=="free"&&(!mine||yearly!==(current.billingInterval==="year"))}{#if data.me.email}<Button {busy} disabled={!price||!!billing?.change||!!billing?.checkout} onclick={()=>choose(p.name)}>{paid?t("billingChange"):t("subscribe",{plan:planLabel(p.name,language())})}</Button>{:else}<a class="text-sm text-blup underline" href={resolve("/(panel)/account")}>{t("linkBeforeSubscribeAction")}</a>{/if}{/if}</div>
 {/each}</div>
{#if preview}<section class="mt-8 rounded-xl border border-blup p-6"><h2 class="text-xl font-bold">{t("billingReview")}</h2><p class="my-3">{planLabel(preview.quote.target,language())}. {t("billingEffective",{date:when(preview.quote.effectiveAt)})}</p>{#if preview.quote.mode==="immediate"}<dl class="my-5 grid grid-cols-2 gap-3"><dt>{t("billingCredit")}</dt><dd>{money(preview.quote.credit,preview.quote.currency,language())}</dd><dt>{t("billingDebit")}</dt><dd>{money(preview.quote.debit,preview.quote.currency,language())}</dd><dt class="font-bold">{t("billingDue")}</dt><dd class="font-bold">{money(preview.quote.amountDue,preview.quote.currency,language())}</dd></dl>{/if}{#if preview.quote.target!=="free"}<p>{t("billingRecurring")}: {money(preview.quote.recurringAmount,preview.quote.currency,language())} {preview.quote.annual?t("perYear"):t("perMonth")}</p>{/if}{#if preview.quote.selectionRequired&&preview.quote.limits}<PulseSelection resources={preview.quote.resources} limits={preview.quote.limits} bind:keep/>{/if}<div class="mt-5 flex flex-wrap gap-3"><Button {busy} disabled={!selectionValid} onclick={confirmChange}>{t("billingConfirm")}</Button><Button variant="secondary" disabled={busy} onclick={()=>preview=null}>{t("cancel")}</Button></div></section>{/if}
{#if paid}<div class="mt-8">{#if gift}<Row title={t("subscription")} help={t("giftSubscription",{plan:planLabel(current.billingName,language())})}/>{:else}<Row title={t("subscription")} help={current.renewsAt?(current.cancelsAtPeriodEnd?t("planCanceledUntil",{plan:planLabel(current.billingName,language()),date:when(current.renewsAt)}):t("renews",{date:when(current.renewsAt)})):undefined}>{#snippet action()}<div class="flex flex-wrap gap-2"><Button variant="secondary" {busy} onclick={portal}>{t("manage")}</Button>{#if !billing?.change}<Button variant="secondary" {busy} onclick={()=>choose(current.cancelsAtPeriodEnd?current.billingName:"free")}>{current.cancelsAtPeriodEnd?t("billingRestore"):t("billingCancel")}</Button>{/if}</div>{/snippet}</Row>{/if}<Row title={t("aiUsage")}><div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full bg-blup" style:width="{data.me.ai.percent}%"></div></div></Row>{#if !gift}<Row title={t("invoices")} help={t("invoicesHelp")}>{#snippet action()}<Button variant="secondary" {busy} onclick={portal}>{t("open")}</Button>{/snippet}</Row>{/if}</div>{/if}
