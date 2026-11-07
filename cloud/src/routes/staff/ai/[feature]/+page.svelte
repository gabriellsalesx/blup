<script lang="ts">
	import type { PageProps } from "./$types";
	import { untrack } from "svelte";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import ArrowUp from "@lucide/svelte/icons/arrow-up";
	import ArrowDown from "@lucide/svelte/icons/arrow-down";
	import Trash from "@lucide/svelte/icons/trash-2";
	import Plus from "@lucide/svelte/icons/plus";
	import Star from "@lucide/svelte/icons/star";
	import StarOff from "@lucide/svelte/icons/star-off";
	import { staffApi, staffHref, type AIFeature, type AIModels } from "#lib/staff.ts";
	import { featureInfo, markName, tabLabels, tabOf } from "#lib/staff-ai.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { confirm } from "#lib/confirm.svelte.ts";
	import BackLink from "#lib/ui/BackLink.svelte";
	import Button from "#lib/ui/Button.svelte";
	import List from "#lib/ui/List.svelte";
	import ListItem from "#lib/ui/ListItem.svelte";
	import ActionMenu from "#lib/ui/ActionMenu.svelte";
	import AIMark from "#lib/ui/AIMark.svelte";

	// One AI list, as one of Pulse's alert projects: its models in order,
	// each with its menu (mark it as the recommended or main one, move it,
	// remove it), every change saved at once. "+ Adicionar modelo" opens the
	// page to add one. It starts empty: nothing comes by default.
	let { data, params }: PageProps = $props();
	const name = $derived(params.feature as AIFeature);
	const info = $derived(featureInfo[name]);
	const tab = $derived(tabOf(name));
	let ai = $state<AIModels | null>(untrack(() => data.ai));
	const list = $derived(ai?.features.find((f) => f.feature === name) ?? null);
	const voice = $derived(list?.kind === "speech");
	const role = $derived(markName(list?.kind ?? "chat"));
	const addHref = $derived(staffHref(resolve("/staff/ai/[feature]/new", { feature: name })));
	const price = (n: number | null) => (n === null ? "?" : `US$ ${n.toLocaleString("pt-BR", { maximumFractionDigits: 3 })}`);

	// Saves the list as it is now and reads it back.
	async function save(models: string[], marked: string | null, done: string) {
		try {
			await staffApi.setModels(name, models, marked && models.includes(marked) ? marked : null);
			ai = await staffApi.ai();
			toast(done);
		} catch {
			toast("Não deu certo.", { kind: "error" });
		}
	}
	const ids = () => list?.models.map((m) => m.id) ?? [];
	function move(i: number, by: number) {
		const l = ids(), j = i + by;
		[l[i], l[j]] = [l[j], l[i]];
		void save(l, list?.recommended ?? null, "Ordem salva");
	}
	function mark(id: string) {
		const on = list?.recommended === id;
		void save(ids(), on ? null : id, on ? `${role} desmarcado` : `Marcado como ${role.toLowerCase()}`);
	}
	async function remove(id: string) {
		if (!(await confirm({ title: "Tirar o modelo?", message: `${id} deixa de ${voice ? "aparecer no Blup Studio" : "ser usado"} nesta lista.`, confirmLabel: "Tirar", danger: true }))) return;
		void save(ids().filter((m) => m !== id), list?.recommended === id ? null : (list?.recommended ?? null), "Modelo removido");
	}
</script>

<div class="mb-4"><BackLink href={`${staffHref(resolve("/staff/ai"))}?tab=${tab}`} label={`IA · ${tabLabels[tab]}`} /></div>

<div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
	<div class="flex min-w-0 flex-1 items-start gap-4">
		<span class="grid size-14 shrink-0 place-items-center rounded-2xl border border-neutral-200 bg-white dark:border-neutral-700 dark:bg-neutral-900"><AIMark kind={info.mark} class="size-8" /></span>
		<div class="min-w-0">
			<p class="text-sm font-semibold text-neutral-500">{tabLabels[tab]}</p>
			<h1 class="text-3xl font-extrabold tracking-tight break-words">{info.title}</h1>
			<p class="mt-2 text-neutral-500">{info.lead}</p>
		</div>
	</div>
	{#if list && list.models.length < list.max}<div class="shrink-0"><Button onclick={() => goto(addHref)}><Plus class="size-4" />Adicionar modelo</Button></div>{/if}
</div>

{#if list}
	<p class="mt-8 text-sm text-neutral-500">{voice ? `Aparecem no Blup Studio nesta ordem. Pelo menu de cada modelo, marque o ${role.toLowerCase()}; sem marcar, nenhum aparece assim.` : `Pelo menu de cada modelo, marque o ${role.toLowerCase()}, que responde primeiro. Se ele falhar, os outros assumem de cima para baixo.`} Até {list.max} modelos.</p>
	<div class="mt-4">
		<List empty={voice ? "Nenhum modelo ainda: o Blup Studio não mostra vozes daqui. Use \"Adicionar modelo\"." : "Nenhum modelo ainda: esta função fica desligada. Use \"Adicionar modelo\"."}>
			{#each list.models as m, i (m.id)}
				{@const on = list.recommended === m.id}
				<ListItem label={m.id}>
					<div class="flex items-center gap-3">
						<span class="grid size-9 shrink-0 place-items-center rounded-full text-sm font-semibold {on ? 'bg-amber-100 text-amber-600 dark:bg-amber-950 dark:text-amber-400' : 'bg-neutral-100 text-neutral-500 dark:bg-neutral-800'}">{#if on}<Star class="size-4 fill-current" />{:else}{i + 1}{/if}</span>
						<div class="min-w-0 flex-1">
							<p class="flex flex-wrap items-center gap-2 font-semibold"><span class="break-all">{m.id}</span>{#if on}<span class="rounded-full bg-blup/10 px-2 py-0.5 text-xs font-semibold text-blup">{role}</span>{/if}</p>
							<p class="text-sm text-neutral-500">{voice ? `${m.name ?? ""} · ${price(m.perThousand)} por mil caracteres` : `${m.name ?? ""} · entrada ${price(m.prompt)} e saída ${price(m.completion)} por milhão de tokens`}</p>
						</div>
					</div>
					{#snippet menu()}
						<ActionMenu label={`Ações de ${m.id}`} items={[
							{ label: on ? `Desmarcar como ${role.toLowerCase()}` : `Marcar como ${role.toLowerCase()}`, Icon: on ? StarOff : Star, onselect: () => mark(m.id) },
							...(i > 0 ? [{ label: "Subir", Icon: ArrowUp, onselect: () => move(i, -1) }] : []),
							...(i < list.models.length - 1 ? [{ label: "Descer", Icon: ArrowDown, onselect: () => move(i, 1) }] : []),
							{ label: "Tirar da lista", Icon: Trash, danger: true, onselect: () => void remove(m.id) },
						]} />
					{/snippet}
				</ListItem>
			{/each}
		</List>
	</div>
{/if}
