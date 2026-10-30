<script lang="ts">
	import Row from "#lib/ui/Row.svelte";

	// Sites: publishing through the cloud (comes with the next Blup Studio).
	let { data } = $props();
	const sites = $derived(data.me.plan.limits.sites);
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Sites</h1>
<p class="mt-2 text-neutral-500">Seus sites no ar direto pela nuvem do Blup, com domínio próprio e sem configurar nada.</p>

<div class="mt-6">
	<Row title="Seus sites" help={`Nenhum site publicado pela nuvem ainda. O plano ${data.me.plan.label} permite ${sites < 0 ? "sites sem limite" : sites === 1 ? "1 site" : `${sites} sites`}.`} />
	<Row title="Como publicar" help="No Blup Studio, entre com a sua conta e toque em Publicar. Chega na próxima versão do Studio." />
</div>
