<script lang="ts">
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
	// Yearly: ten months' price for twelve.
	let yearly = $state(false);
	const current = $derived(data.me.plan);
	const paid = $derived(current.name !== "free");
	// The owner of Blup has the top plan as a gift: nothing to pay or manage.
	const gift = $derived(!!current.gift);
	const when = (d: string) => new Date(d).toLocaleDateString("pt-BR", { day: "numeric", month: "long", year: "numeric" });

	// Back from Stripe's checkout.
	onMount(() => {
		if (page.url.searchParams.get("assinatura") === "ok") toast(`Assinatura feita. Bem-vindo ao ${data.me.plan.label}!`);
	});

	async function subscribe(plan: string) {
		busy = true;
		const back = `${location.origin}/conta/plano`;
		const { error } = await authClient.subscription.upgrade({ plan, annual: yearly, successUrl: `${back}?assinatura=ok`, cancelUrl: back });
		busy = false;
		if (error) toast(message(error), { kind: "error" });
	}

	async function manage() {
		busy = true;
		const { error } = await authClient.subscription.billingPortal({ returnUrl: `${location.origin}/conta/plano` });
		busy = false;
		if (error) toast(message(error), { kind: "error" });
	}
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Plano</h1>
<p class="mt-2 text-neutral-500">Notas sincronizadas com criptografia de ponta a ponta e sites no ar sem esforço. Cancele quando quiser.</p>

<div class="mt-8 inline-flex rounded-full border border-neutral-200 p-1 text-sm font-semibold dark:border-neutral-800" role="radiogroup" aria-label="Cobrança">
	<button role="radio" aria-checked={!yearly} class="rounded-full px-4 py-1.5 {!yearly ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-500'}" onclick={() => (yearly = false)}>Mensal</button>
	<button role="radio" aria-checked={yearly} class="rounded-full px-4 py-1.5 {yearly ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-500'}" onclick={() => (yearly = true)}>Anual <span class="font-normal {yearly ? '' : 'text-green-600'}">2 meses grátis</span></button>
</div>

<div class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
	{#each data.plans as p (p.name)}
		{@const mine = p.name === current.name}
		<div class="flex flex-col rounded-xl border p-5 {mine || (p.suggested && !paid) ? 'border-blup ring-1 ring-blup' : 'border-neutral-200 dark:border-neutral-800'}">
			<div class="flex items-baseline justify-between gap-2">
				<h2 class="font-semibold">{p.label}</h2>
				{#if mine}<span class="rounded-full bg-blup/10 px-2 py-0.5 text-xs font-semibold text-blup">{gift ? "Cortesia" : "Seu plano"}</span>
				{:else if p.suggested}<span class="rounded-full bg-blup px-2 py-0.5 text-xs font-semibold text-white">Mais escolhido</span>{/if}
			</div>
			{#if yearly && p.yearlyCents}
				<p class="mt-1"><b class="text-2xl">{money(p.yearlyCents, p.currency)}</b> <span class="text-sm text-neutral-500">por ano</span></p>
				<p class="text-sm text-neutral-500">{money(Math.round(p.yearlyCents / 12), p.currency)} por mês</p>
			{:else}
				<p class="mt-1"><b class="text-2xl">{money(p.priceCents, p.currency)}</b> <span class="text-sm text-neutral-500">por mês</span></p>
			{/if}
			<ul class="mt-4 grid flex-1 content-start gap-1.5 text-sm">
				{#each p.perks as perk (perk)}<li class="flex items-start gap-2"><Check class="mt-0.5 size-4 shrink-0 text-green-600" />{perk}</li>{/each}
			</ul>
			{#if !mine && p.priceCents && !paid}
				{#if data.me.email}
					<Button class="mt-5" {busy} onclick={() => subscribe(p.name)}>Assinar o {p.label}</Button>
				{:else}
					<p class="mt-5 text-sm text-neutral-500"><a class="font-semibold text-blup hover:underline" href="/conta">Vincule um e-mail</a> para assinar: é por ele que chegam os recibos.</p>
				{/if}
			{/if}
		</div>
	{/each}
</div>

{#if paid}
	<div class="mt-8">
		{#if gift}
			<Row title="Assinatura" help={`${current.label} de cortesia para o dono do Blup, sem cobrança e sem data para acabar.`} />
		{:else}
			<Row title="Assinatura" help={current.renewsAt ? (current.cancelsAtPeriodEnd ? `Cancelada: o ${current.label} vale até ${when(current.renewsAt)}.` : `Renova em ${when(current.renewsAt)}.`) : undefined}>
				{#snippet action()}<Button variant="secondary" {busy} onclick={manage}>Gerenciar</Button>{/snippet}
			</Row>
		{/if}
		<Row title="Uso de IA" help={data.me.ai.included ? `${data.me.ai.percent}% do uso de IA deste mês (narração). Renova em ${new Date(data.me.ai.resetsAt).toLocaleDateString("pt-BR", { day: "numeric", month: "long", timeZone: "UTC" })}.` : undefined}>
			<div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
				<div class="h-full rounded-full {data.me.ai.percent >= 90 ? 'bg-red-500' : 'bg-blup'}" style:width="{data.me.ai.percent}%"></div>
			</div>
		</Row>
		{#if !gift}
			<Row title="Faturas e pagamento" help="Baixe as faturas, troque o cartão ou cancele, na página segura do Stripe.">
				{#snippet action()}<Button variant="secondary" {busy} onclick={manage}>Abrir</Button>{/snippet}
			</Row>
		{/if}
	</div>
{/if}
