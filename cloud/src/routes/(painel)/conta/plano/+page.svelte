<script lang="ts">
 import {planLabel,planPerks} from "../../../../../shared/plan-i18n";
 import {stripeLocale} from "../../../../../shared/language";
 import {t,language,intlLocale} from "#lib/i18n.svelte.ts";
	import { onMount } from "svelte";
	import { page } from "$app/state";
	import Check from "@lucide/svelte/icons/check";
	import { authClient, message } from "#lib/auth.ts";
	import { money } from "#lib/api.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Row from "#lib/ui/Row.svelte";

	// The plan: the one the account has, the others side by side, and
	// Stripe's own pages to pay and to manage the subscription.
	let { data } = $props();
	let busy = $state(false);
	// Monthly and annual amounts come from Stripe, independently.
	let yearly = $state(false);
	const current = $derived(data.me.plan);
	const paid = $derived(current.name !== "free");
	// The owner of Blup has the top plan as a gift: nothing to pay or manage.
	const gift = $derived(!!current.gift);
	const when = (d: string) => new Date(d).toLocaleDateString(intlLocale(), { day: "numeric", month: "long", year: "numeric" });

	// Back from Stripe's checkout.
	onMount(() => {
		if (page.url.searchParams.get("assinatura") === "ok") toast(()=>t("paymentDone"));
	});

	async function subscribe(plan: string) {
		const chosen=data.plans.find(p=>p.name===plan)?.prices[yearly?"yearly":"monthly"];
		if(!chosen){toast(()=>t("pricingUnavailable"),{kind:"error"});return;}
		busy = true;
		const back = `${location.origin}/conta/plano`;
		try{
			const { error } = await authClient.subscription.upgrade({ plan, locale:stripeLocale[language()],annual: yearly, successUrl: `${back}?assinatura=ok`, cancelUrl: back });
			if (error) toast(()=>message(error), { kind: "error" });
		}catch{toast(()=>t("errorPayment"),{kind:"error"});}
		finally{busy=false;}
	}

	async function manage() {
		busy = true;
		try{
			const { error } = await authClient.subscription.billingPortal({locale:stripeLocale[language()], returnUrl: `${location.origin}/conta/plano` });
			if (error) toast(()=>message(error), { kind: "error" });
		}catch{toast(()=>t("errorPayment"),{kind:"error"});}
		finally{busy=false;}
	}
</script>

<h1 class="text-3xl font-extrabold tracking-tight">{t("plan")}</h1>
<p class="mt-2 text-neutral-500">{t("planLead")}</p>

<div class="mt-8 inline-flex rounded-full border border-neutral-200 p-1 text-sm font-semibold dark:border-neutral-800" role="radiogroup" aria-label={t("billing")} >
	<button role="radio" aria-checked={!yearly} class="rounded-full px-4 py-1.5 {!yearly ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-500'}" onclick={() => (yearly = false)}>{t("monthly")}</button>
	<button role="radio" aria-checked={yearly} class="rounded-full px-4 py-1.5 {yearly ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-500'}" onclick={() => (yearly = true)}>{t("yearly")}</button>
</div>

<div class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
	{#each data.plans as p (p.name)}
		{@const mine = p.name === current.name}
		{@const price = p.prices[yearly ? "yearly" : "monthly"]}
		<div class="flex flex-col rounded-xl border p-5 {mine || (p.suggested && !paid) ? 'border-blup ring-1 ring-blup' : 'border-neutral-200 dark:border-neutral-800'}">
			<div class="flex items-baseline justify-between gap-2">
				<h2 class="font-semibold">{planLabel(p.name,language())}</h2>
				{#if mine}<span class="rounded-full bg-blup/10 px-2 py-0.5 text-xs font-semibold text-blup">{gift ? t("gift") : t("yourPlan")}</span>
				{:else if p.suggested}<span class="rounded-full bg-blup px-2 py-0.5 text-xs font-semibold text-white">{t("suggested")}</span>{/if}
			</div>
			{#if p.name==="free"}
				<p class="mt-1"><b class="text-2xl">{planLabel("free",language())}</b></p>
			{:else if price}
				<p class="mt-1"><b class="text-2xl">{money(price.unitAmount,price.currency,language())}</b> <span class="text-sm text-neutral-500">{yearly?t("perYear"):t("perMonth")}</span></p>
			{:else}
				<p class="mt-1 text-sm text-neutral-500">{t("pricingUnavailable")}</p>
			{/if}
			<ul class="mt-4 grid flex-1 content-start gap-1.5 text-sm">
				{#each planPerks(language(),p.name,p.limits) as perk (perk)}<li class="flex items-start gap-2"><Check class="mt-0.5 size-4 shrink-0 text-green-600" />{perk}</li>{/each}
			</ul>
			{#if !mine && p.name!=="free" && !paid}
				{#if data.me.email}
					<Button class="mt-5" {busy} disabled={!price} onclick={() => subscribe(p.name)}>{t("subscribe",{plan:planLabel(p.name,language())})}</Button>
				{:else}
					<p class="mt-5 text-sm text-neutral-500">{t("linkBeforeSubscribe")} <a class="font-semibold text-blup hover:underline" href="/conta">{t("linkBeforeSubscribeAction")}</a></p>
				{/if}
			{/if}
		</div>
	{/each}
</div>

{#if paid}
	<div class="mt-8">
		{#if gift}
			<Row title={t("subscription")} help={t("giftSubscription",{plan:planLabel(current.name,language())})} />
		{:else}
			<Row title={t("subscription")} help={current.renewsAt ? (current.cancelsAtPeriodEnd ? t("planCanceledUntil",{plan:planLabel(current.name,language()),date:when(current.renewsAt)}) : t("renews",{date:when(current.renewsAt)})) : undefined}>
				{#snippet action()}<Button variant="secondary" {busy} onclick={manage}>{t("manage")}</Button>{/snippet}
			</Row>
		{/if}
		<Row title={t("aiUsage")}  help={data.me.ai.included ? t("aiMonth",{percent:data.me.ai.percent,date:new Date(data.me.ai.resetsAt).toLocaleDateString(intlLocale(),{day:"numeric",month:"long",timeZone:"UTC"})}) : undefined}>
			<div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
				<div class="h-full rounded-full {data.me.ai.percent >= 90 ? 'bg-red-500' : 'bg-blup'}" style:width="{data.me.ai.percent}%"></div>
			</div>
		</Row>
		{#if !gift}
			<Row title={t("invoices")}  help={t("invoicesHelp")} >
				{#snippet action()}<Button variant="secondary" {busy} onclick={manage}>{t("open")}</Button>{/snippet}
			</Row>
		{/if}
	</div>
{/if}
