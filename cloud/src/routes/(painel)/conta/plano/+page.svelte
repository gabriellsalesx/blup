<script lang="ts">
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
	const current = $derived(data.me.plan);
	const paid = $derived(current.name !== "free");
	const when = (d: string) => new Date(d).toLocaleDateString("pt-BR", { day: "numeric", month: "long", year: "numeric" });

	// Back from Stripe's checkout.
	if (page.url.searchParams.get("assinatura") === "ok") toast("Assinatura feita. Bem-vindo ao Pro!");

	async function subscribe(plan: string) {
		busy = true;
		const back = `${location.origin}/conta/plano`;
		const { error } = await authClient.subscription.upgrade({ plan, successUrl: `${back}?assinatura=ok`, cancelUrl: back });
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

<div class="mt-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
	{#each data.plans as p (p.name)}
		{@const mine = p.name === current.name}
		<div class="flex flex-col rounded-xl border p-5 {mine ? 'border-blup ring-1 ring-blup' : 'border-neutral-200 dark:border-neutral-800'}">
			<div class="flex items-baseline justify-between gap-2">
				<h2 class="font-semibold">{p.label}</h2>
				{#if mine}<span class="rounded-full bg-blup/10 px-2 py-0.5 text-xs font-semibold text-blup">Seu plano</span>{/if}
			</div>
			<p class="mt-1"><b class="text-2xl">{money(p.priceCents, p.currency)}</b> <span class="text-sm text-neutral-500">por mês</span></p>
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
		<Row title="Assinatura" help={current.renewsAt ? (current.cancelsAtPeriodEnd ? `Cancelada: o ${current.label} vale até ${when(current.renewsAt)}.` : `Renova em ${when(current.renewsAt)}.`) : undefined}>
			{#snippet action()}<Button variant="secondary" {busy} onclick={manage}>Gerenciar</Button>{/snippet}
		</Row>
		<Row title="Faturas e pagamento" help="Baixe as faturas, troque o cartão ou cancele, na página segura do Stripe.">
			{#snippet action()}<Button variant="secondary" {busy} onclick={manage}>Abrir</Button>{/snippet}
		</Row>
	</div>
{/if}
