<script lang="ts">
	import Lock from "@lucide/svelte/icons/lock";
	import Row from "#lib/ui/Row.svelte";

	// Sync: the space the notes use in the cloud (the sync itself comes
	// with the next Blup Studio).
	let { data } = $props();
	const limit = $derived(data.me.plan.limits.storageMB);
	const used = 0;
	const space = (mb: number) => (mb >= 1024 ? `${(mb / 1024).toLocaleString("pt-BR")} GB` : `${mb} MB`);
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Sync</h1>
<p class="mt-2 text-neutral-500">Suas notas em todos os seus computadores, sempre offline no seu e guardadas na nuvem criptografadas: nem o Blup consegue ler.</p>

<div class="mt-6">
	<Row title="Espaço usado" help={`${space(used)} de ${space(limit)} do plano ${data.me.plan.label}`}>
		<div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
			<div class="h-full rounded-full bg-blup" style:width="{Math.min(100, (used / limit) * 100)}%"></div>
		</div>
	</Row>
	<Row title="Criptografia de ponta a ponta" help="As notas saem do seu computador já embaralhadas, com uma chave que só você tem.">
		{#snippet action()}<Lock class="size-5 text-green-600" />{/snippet}
	</Row>
	<Row title="Como ligar" help="No Blup Studio, entre com a sua conta e escolha as notas que vão para a nuvem. Chega na próxima versão do Studio." />
</div>
