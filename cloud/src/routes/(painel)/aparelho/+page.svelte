<script lang="ts">
	import { onMount } from "svelte";
	import { page } from "$app/state";
	import MonitorSmartphone from "@lucide/svelte/icons/monitor-smartphone";
	import CircleCheck from "@lucide/svelte/icons/circle-check";
	import { authClient } from "#lib/auth.ts";
	import Button from "#lib/ui/Button.svelte";
	import FloatField from "#lib/ui/FloatField.svelte";

	// Blup Studio signing in through the browser: the Studio shows a code
	// and opens this page; the person checks it is the same code and
	// approves. Checking the code claims it for this session, so only this
	// person can approve it.
	let { data } = $props();

	type Step = "type" | "confirm" | "approved" | "denied";
	let step = $state<Step>("type");
	let code = $state(page.url.searchParams.get("user_code") ?? "");
	let error = $state("");
	let busy = $state(false);

	const clean = (c: string) => c.replace(/[^0-9a-z]/gi, "").toUpperCase();
	const shown = $derived(clean(code).replace(/^(.{4})(.+)$/, "$1-$2"));

	async function check(e?: SubmitEvent) {
		e?.preventDefault();
		error = "";
		busy = true;
		const { error: err } = await authClient.device({ query: { user_code: clean(code) } });
		busy = false;
		if (err) return (error = "Esse código não vale mais. Peça um novo no Blup Studio.");
		step = "confirm";
	}

	async function decide(approve: boolean) {
		error = "";
		busy = true;
		const { error: err } = approve ? await authClient.device.approve({ userCode: clean(code) }) : await authClient.device.deny({ userCode: clean(code) });
		busy = false;
		if (err && "code" in err && err.code === "DEVICES_LIMIT") return (error = `Seu plano ${data.me.plan.label} permite ${data.me.plan.limits.devices} ${data.me.plan.limits.devices === 1 ? "computador conectado" : "computadores conectados"}. Desconecte um em Sync ou mude de plano.`);
		if (err) return (error = "Esse código não vale mais. Peça um novo no Blup Studio.");
		step = approve ? "approved" : "denied";
	}

	// Opened from the Studio, the code comes in the address.
	onMount(() => {
		if (code) check();
	});
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Entrar no Blup Studio</h1>

<div class="mt-6 max-w-lg">
	{#if step === "type"}
		<p class="text-neutral-500">Digite o código que aparece no Blup Studio.</p>
		<form class="mt-6 grid gap-4" onsubmit={check}>
			<FloatField id="code" label="Código" class="font-mono tracking-[0.2em] uppercase" autocomplete="off" required bind:value={code} />
			{#if error}<p role="alert" class="text-sm text-red-600">{error}</p>{/if}
			<Button {busy} disabled={clean(code).length < 8}>Continuar</Button>
		</form>
	{:else if step === "confirm"}
		<div class="flex items-start gap-4">
			<MonitorSmartphone class="mt-1 size-6 shrink-0 text-blup" />
			<p>Um Blup Studio quer entrar na sua conta <span class="font-semibold">{data.me.email ?? "Blup Cloud"}</span> e sincronizar as suas notas.</p>
		</div>
		<p class="mt-6 text-sm text-neutral-500">Confira se este é o mesmo código que aparece no Studio:</p>
		<p class="mt-2 font-mono text-4xl font-bold tracking-[0.2em]">{shown}</p>
		<p class="mt-6 text-sm text-neutral-500">Só aprove se foi você que pediu agora, num computador seu. Se alguém mandou este link por mensagem, recuse.</p>
		{#if error}<p role="alert" class="mt-4 text-sm text-red-600">{error}</p>{/if}
		<div class="mt-6 flex gap-3">
			<Button {busy} onclick={() => decide(true)}>Aprovar</Button>
			<Button variant="secondary" disabled={busy} onclick={() => decide(false)}>Recusar</Button>
		</div>
	{:else if step === "approved"}
		<div class="flex items-start gap-4">
			<CircleCheck class="mt-1 size-6 shrink-0 text-green-600" />
			<p>Pronto. Pode voltar para o Blup Studio: ele já entrou na sua conta.</p>
		</div>
	{:else}
		<p>Pedido recusado. O Blup Studio não entrou na sua conta.</p>
	{/if}
</div>
