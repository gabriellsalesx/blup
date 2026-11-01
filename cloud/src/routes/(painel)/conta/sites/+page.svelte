<script lang="ts">
	import { goto } from "$app/navigation";
	import Globe from "@lucide/svelte/icons/globe";
	import Row from "#lib/ui/Row.svelte";
	import Button from "#lib/ui/Button.svelte";
	import { api, type Site } from "#lib/api.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { bytes } from "#lib/usage.ts";

	// Sites: published through the cloud from Blup Studio (paid plans), at
	// name.getblup.site. Each one can be taken down here.
	let { data } = $props();
	const limit = $derived(data.me.plan.limits.sites);
	let sites = $state<Site[] | null>(null);
	let busy = $state("");

	const when = (d: string) => new Date(d).toLocaleString("pt-BR", { dateStyle: "medium", timeStyle: "short" });
	async function load() {
		try {
			sites = (await api.sites()).sites;
		} catch {
			sites = [];
		}
	}
	load();

	async function takeDown(s: Site) {
		const ok = await confirm({ title: `Tirar ${s.slug}.getblup.site do ar?`, message: "O endereço para de abrir e os arquivos publicados são apagados da nuvem. As notas continuam no seu computador; para voltar, é só publicar de novo pelo Blup Studio.", confirmLabel: "Tirar do ar", danger: true, typeToConfirm: s.slug });
		if (!ok) return;
		busy = s.id;
		try {
			await api.takeDown(s.id);
			toast("Site tirado do ar");
			await load();
		} catch {
			toast("Não deu para tirar o site do ar agora. Tente de novo.", { kind: "error" });
		} finally {
			busy = "";
		}
	}

	const count = $derived(sites === null ? "Carregando…" : `${sites.length}${limit > 0 ? ` de ${limit}` : ""} no plano ${data.me.plan.label}`);
</script>

<h1 class="text-3xl font-extrabold tracking-tight">Sites</h1>
<p class="mt-2 text-neutral-500">Seus sites no ar direto pela nuvem do Blup, sem configurar nada, no endereço seusite.getblup.site. Os arquivos vão minificados e compactados, e o site abre rápido no mundo todo.</p>

<div class="mt-6">
	{#if limit === 0}
		<Row title="Seus sites" help={`O plano ${data.me.plan.label} não inclui sites no ar pelo Blup. Assine o Pro ou um plano maior em Plano.`}>
			{#snippet action()}<Button onclick={() => goto("/conta/plano")}>Ver os planos</Button>{/snippet}
		</Row>
	{:else}
		<Row title="Seus sites" help={count} />
		{#each sites ?? [] as s (s.id)}
			<Row title={s.title} help={`${s.live ? `${s.files} arquivos, ${bytes(s.bytes)}` : "Ainda não publicado"} · atualizado em ${when(s.updatedAt)}`}>
				<a class="mt-1 inline-flex items-center gap-1.5 text-sm font-semibold text-blup hover:underline" href={s.url} target="_blank" rel="noopener"><Globe class="size-4" />{s.url.replace(/^https?:\/\//, "").replace(/\/$/, "")}</a>
				{#snippet action()}
					<Button variant="secondary" busy={busy === s.id} onclick={() => takeDown(s)}>Tirar do ar</Button>
				{/snippet}
			</Row>
		{/each}
		<Row title="Como publicar" help="No Blup Studio, abra o site, vá em Publicar e escolha Blup Cloud. Cada publicação manda só o que mudou." />
	{/if}
</div>
