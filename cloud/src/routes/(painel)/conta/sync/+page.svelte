<script lang="ts">
	import Lock from "@lucide/svelte/icons/lock";
	import Monitor from "@lucide/svelte/icons/monitor";
	import Row from "#lib/ui/Row.svelte";
	import Button from "#lib/ui/Button.svelte";
	import { api, type Device } from "#lib/api.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { bytes, megabytes } from "#lib/usage.ts";

	// Sync: the space the notes use in the cloud and the computers with Blup
	// Studio signed in (each can be disconnected).
	let { data } = $props();

	const limit = $derived(data.me.plan.limits.storageMB);
	const used = $derived(data.me.usage.storageBytes);
	const maxDevices = $derived(data.me.plan.limits.devices);
	let devices = $state<Device[] | null>(null);
	let busy = $state("");

	const when = (d: string) => new Date(d).toLocaleString("pt-BR", { dateStyle: "medium", timeStyle: "short" });
	async function load() {
		try {
			devices = (await api.devices()).devices;
		} catch {
			devices = [];
		}
	}
	load();

	async function disconnect(d: Device) {
		const ok = await confirm({ title: "Desconectar este computador?", message: `O Blup Studio em ${d.name} sai da sua conta e para de sincronizar. As notas continuam nele. Para voltar, é só entrar de novo.`, confirmLabel: "Desconectar", danger: true });
		if (!ok) return;
		busy = d.id;
		try {
			await api.disconnect(d.id);
			toast("Computador desconectado");
			await load();
		} catch {
			toast("Não deu para desconectar agora. Tente de novo.", { kind: "error" });
		} finally {
			busy = "";
		}
	}
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Sync</h1>
<p class="mt-2 text-neutral-500">Suas notas em todos os seus computadores, sempre offline no seu e guardadas na nuvem criptografadas: nem o Blup consegue ler.</p>

<div class="mt-6">
	<Row title="Espaço usado" help={`${bytes(used)} de ${megabytes(limit)} do plano ${data.me.plan.label}`}>
		<div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
			<div class="h-full rounded-full bg-blup" style:width="{Math.min(100, (used / (limit * 1024 * 1024)) * 100)}%"></div>
		</div>
	</Row>
	<Row
		title="Computadores conectados"
		help={devices === null ? "Carregando…" : `${devices.length}${maxDevices >= 0 ? ` de ${maxDevices}` : ""} no plano ${data.me.plan.label}${devices.length ? "" : ". No Blup Studio, toque em Entrar com Blup."}`}
	/>
	{#each devices ?? [] as d (d.id)}
		<Row title={d.name} help={`Conectado em ${when(d.createdAt)} · último uso ${when(d.lastSeen)}`}>
			{#snippet action()}
				<div class="flex items-center gap-3">
					<Monitor class="size-5 text-neutral-400" />
					<Button variant="secondary" busy={busy === d.id} onclick={() => disconnect(d)}>Desconectar</Button>
				</div>
			{/snippet}
		</Row>
	{/each}
	<Row title="Criptografia de ponta a ponta" help="As notas saem do seu computador já embaralhadas, com uma chave que só você tem.">
		{#snippet action()}<Lock class="size-5 text-green-600" />{/snippet}
	</Row>
</div>
