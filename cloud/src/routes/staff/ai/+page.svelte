<script lang="ts">
	import type { PageProps } from "./$types";
	import { untrack } from "svelte";
	import ArrowUp from "@lucide/svelte/icons/arrow-up";
	import ArrowDown from "@lucide/svelte/icons/arrow-down";
	import Trash from "@lucide/svelte/icons/trash-2";
	import { staffApi, StaffError, type AIFeature, type AIModels } from "#lib/staff.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import Button from "#lib/ui/Button.svelte";
	import Row from "#lib/ui/Row.svelte";
	import Field from "#lib/ui/Field.svelte";
	import PageHeader from "#lib/ui/PageHeader.svelte";

	// The AI models of each feature, in order, on OpenRouter: the first
	// answers; when it fails (down, limit, refusal) the next one takes over.
	// What a request costs is what the model that answered charges, out of
	// the person's plan allowance.
	let { data }: PageProps = $props();
	let ai = $state<AIModels | null>(untrack(() => data.ai));
	const names: Record<AIFeature, { title: string; help: string }> = {
		deals_describe: { title: "Blup Deals: descrição do post", help: "O botão \"Escrever com IA\" do Colar link e do Vídeo novo, e a descrição automática nas caçadas. Texto curto: um modelo barato basta." },
		deals_curate: { title: "Blup Deals: triagem da caçada de lojas", help: "Nome curto, se combina com a busca e categoria, para até 10 ofertas de uma vez. Responde em JSON: o modelo precisa seguir o formato." },
		deals_extract: { title: "Blup Deals: leitura do espelho", help: "Cupom, descrição com palavras próprias, nome curto e categoria de um post espelhado. Responde em JSON." },
		deals_suggest: { title: "Blup Deals: sugestão de buscas", help: "Oito buscas a partir do assunto do canal. Responde em JSON." },
		deals_translate: { title: "Blup Deals: tradução do post", help: "A frase, a descrição e o texto extra para um canal em outro idioma. Responde em JSON." },
	};
	// The list being edited, per feature (ids only), and the one being added.
	let lists = $state<Record<string, string[]>>(untrack(() => Object.fromEntries((data.ai?.features ?? []).map((f) => [f.feature, f.models.map((m) => m.id)]))));
	let adding = $state<Record<string, string>>({});
	let busy = $state<string | null>(null);
	const valid = /^[a-z0-9][a-z0-9._-]{0,63}\/[a-z0-9][a-z0-9._:-]{0,99}$/;
	const price = (n: number | null) => (n === null ? "?" : `US$ ${n.toLocaleString("pt-BR", { maximumFractionDigits: 3 })}`);

	function move(feature: string, i: number, by: number) {
		const l = lists[feature], j = i + by;
		if (j < 0 || j >= l.length) return;
		[l[i], l[j]] = [l[j], l[i]];
	}
	function add(feature: string) {
		const id = (adding[feature] ?? "").trim().toLowerCase();
		if (!valid.test(id)) return toast("Use o ID do modelo no OpenRouter, como deepseek/deepseek-v4.1-flash.", { kind: "error" });
		if (lists[feature].includes(id)) return toast("Esse modelo já está na lista.", { kind: "error" });
		if (lists[feature].length >= 5) return toast("No máximo 5 modelos.", { kind: "error" });
		lists[feature].push(id);
		adding[feature] = "";
	}
	async function save(feature: AIFeature, models: string[]) {
		busy = feature;
		try {
			await staffApi.setModels(feature, models);
			ai = await staffApi.ai();
			lists[feature] = ai.features.find((f) => f.feature === feature)?.models.map((m) => m.id) ?? models;
			toast("Modelos salvos");
		} catch (e) {
			toast(e instanceof StaffError && e.code === "UNKNOWN_MODEL" ? "Algum modelo não existe no OpenRouter. Confira o ID." : "Não deu certo.", { kind: "error" });
		} finally { busy = null; }
	}
	const saved = (feature: string) => ai?.features.find((f) => f.feature === feature);
</script>

<PageHeader title="IA" lead="Os modelos que cada função usa no OpenRouter, em ordem. O primeiro responde; se ele falhar (fora do ar, limite, recusa), o próximo assume. A cobrança é pelo modelo que respondeu, na cota de IA do plano de cada cliente." />

{#if ai}
	{#if !ai.catalog}<p class="mt-6 text-sm text-amber-800 dark:text-amber-300">O catálogo do OpenRouter não respondeu agora: os preços ficam com "?" e os IDs não são conferidos.</p>{/if}
	<div class="mt-6">
		{#each ai.features as f (f.feature)}
			{@const list = lists[f.feature] ?? []}
			{@const known = new Map(saved(f.feature)?.models.map((m) => [m.id, m]) ?? [])}
			<Row title={names[f.feature].title} help={names[f.feature].help}>
				<ol class="mt-3 grid gap-2 text-sm">
					{#each list as id, i (id)}
						{@const m = known.get(id)}
						<li class="flex flex-wrap items-center gap-3 rounded-lg border border-neutral-200 px-3 py-2.5 dark:border-neutral-800">
							<span class="w-16 shrink-0 text-xs font-semibold text-neutral-500">{i === 0 ? "Principal" : `Reserva ${i}`}</span>
							<span class="min-w-0 flex-1"><b class="break-all">{id}</b>{#if m}<br /><span class="text-xs text-neutral-500">{m.name ?? ""} · entrada {price(m.prompt)} e saída {price(m.completion)} por milhão de tokens</span>{:else}<br /><span class="text-xs text-neutral-500">Salve para ver o preço</span>{/if}</span>
							<span class="flex gap-1">
								<button type="button" class="grid size-8 place-items-center rounded-lg hover:bg-neutral-100 disabled:opacity-30 dark:hover:bg-neutral-800" aria-label="Subir {id}" disabled={i === 0} onclick={() => move(f.feature, i, -1)}><ArrowUp class="size-4" /></button>
								<button type="button" class="grid size-8 place-items-center rounded-lg hover:bg-neutral-100 disabled:opacity-30 dark:hover:bg-neutral-800" aria-label="Descer {id}" disabled={i === list.length - 1} onclick={() => move(f.feature, i, 1)}><ArrowDown class="size-4" /></button>
								<button type="button" class="grid size-8 place-items-center rounded-lg text-red-700 hover:bg-red-50 disabled:opacity-30 dark:text-red-400 dark:hover:bg-red-950/40" aria-label="Tirar {id}" disabled={list.length === 1} onclick={() => lists[f.feature].splice(i, 1)}><Trash class="size-4" /></button>
							</span>
						</li>
					{/each}
				</ol>
				<form class="mt-3 flex flex-wrap items-end gap-3" onsubmit={(e) => { e.preventDefault(); add(f.feature); }}>
					<div class="min-w-64 flex-1"><Field label="Adicionar modelo (ID do OpenRouter)" placeholder="deepseek/deepseek-v4.1-flash" bind:value={adding[f.feature]} /></div>
					<Button variant="secondary" disabled={list.length >= 5}>Adicionar</Button>
				</form>
				<div class="mt-4 flex flex-wrap justify-end gap-2">
					<Button variant="secondary" disabled={busy !== null} onclick={() => save(f.feature, [...f.defaults])}>Voltar ao padrão ({f.defaults.join(", ")})</Button>
					<Button busy={busy === f.feature} disabled={!list.length || busy !== null} onclick={() => save(f.feature, [...list])}>Salvar</Button>
				</div>
			</Row>
		{/each}
	</div>
{/if}
