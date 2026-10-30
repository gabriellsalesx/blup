<script lang="ts">
	import Check from "@lucide/svelte/icons/check";
	import Sparkles from "@lucide/svelte/icons/sparkles";
	import { authClient, message } from "#lib/auth.ts";
	import { api, brl, type Me, type PlanInfo } from "#lib/api.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import Button from "./Button.svelte";
	import Card from "./Card.svelte";

	// The plan: what the account has, and the way to subscribe or to manage
	// the subscription (Stripe's own pages).
	let me = $state<Me | null>(null);
	let plans = $state<PlanInfo[]>([]);
	let busy = $state(false);

	Promise.all([api.me(), api.plans()]).then(([m, p]) => { me = m; plans = p; }).catch(() => toast("Não deu para carregar o plano.", { kind: "error" }));

	const pro = $derived(plans.find((p) => p.name === "pro"));
	const paid = $derived(!!me && me.plan.name !== "free");
	const space = (mb: number) => (mb >= 1024 ? `${mb / 1024} GB` : `${mb} MB`);
	const when = (d: string) => new Date(d).toLocaleDateString("pt-BR", { day: "numeric", month: "long" });

	async function subscribe() {
		busy = true;
		const back = `${location.origin}/conta`;
		const { error } = await authClient.subscription.upgrade({ plan: "pro", successUrl: `${back}?assinatura=ok`, cancelUrl: back });
		busy = false;
		if (error) toast(message(error), { kind: "error" });
	}

	async function manage() {
		busy = true;
		const { error } = await authClient.subscription.billingPortal({ returnUrl: `${location.origin}/conta` });
		busy = false;
		if (error) toast(message(error), { kind: "error" });
	}
</script>

{#if me}
	<Card title={`Plano ${me.plan.label}`} help={paid ? (me.plan.renewsAt ? (me.plan.cancelsAtPeriodEnd ? `Termina em ${when(me.plan.renewsAt)}.` : `Renova em ${when(me.plan.renewsAt)}.`) : undefined) : `Até ${space(me.plan.limits.storageMB)} de notas e ${me.plan.limits.sites} site.`}>
		{#snippet action()}
			{#if paid}
				<Button variant="secondary" class="shrink-0 px-3 py-2 text-sm" {busy} onclick={manage}>Gerenciar</Button>
			{/if}
		{/snippet}
		{#if !paid && pro}
			<div class="mt-4 rounded-lg border border-blup/30 bg-blup/5 p-4">
				<div class="flex items-baseline justify-between gap-3">
					<span class="flex items-center gap-1.5 font-bold"><Sparkles class="size-4 text-blup" />{pro.label}</span>
					<span class="text-sm"><b class="text-lg">{brl(pro.priceCents)}</b> por mês</span>
				</div>
				<ul class="mt-3 grid gap-1.5 text-sm">
					{#each pro.perks as perk (perk)}<li class="flex items-start gap-2"><Check class="mt-0.5 size-4 shrink-0 text-green-600" />{perk}</li>{/each}
				</ul>
				<Button class="mt-4 w-full" {busy} onclick={subscribe}>Assinar o {pro.label}</Button>
			</div>
		{/if}
	</Card>
{/if}
