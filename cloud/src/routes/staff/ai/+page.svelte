<script lang="ts">
	import type { PageProps } from "./$types";
	import { untrack } from "svelte";
	import { replaceState } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { staffHref, type AIModels } from "#lib/staff.ts";
	import { aiTabs, tabLabels, tabAbout, tabFeatures, featureInfo, markName, type AITab } from "#lib/staff-ai.ts";
	import PageHeader from "#lib/ui/PageHeader.svelte";
	import Segmented from "#lib/ui/Segmented.svelte";
	import List from "#lib/ui/List.svelte";
	import ListItem from "#lib/ui/ListItem.svelte";
	import AIMark from "#lib/ui/AIMark.svelte";

	// The AI lists, as Pulse's alerts: one tab per subject, each list a row
	// with its mark (Blup Cloud, OpenRouter or Deals), how many models it
	// has and which one is marked. A click opens the list to change it.
	let { data }: PageProps = $props();
	type Feature = AIModels["features"][number];
	const tabs = aiTabs.map((value) => ({ value, label: tabLabels[value] }));
	let tab = $state<AITab>(untrack(() => data.tab));
	// The tab goes in the address, so a reload (or a link) opens it.
	function show(next: AITab) {
		const url = new URL(page.url.href);
		url.searchParams.set("tab", next);
		replaceState(url, {});
	}
	const rows = $derived(tabFeatures[tab].map((name) => data.ai?.features.find((f) => f.feature === name)).filter((f): f is Feature => !!f));
	const count = (n: number) => (n === 1 ? "1 modelo" : `${n} modelos`);
	const summary = (f: Feature) => (f.models.length ? `${count(f.models.length)} · ${f.recommended ? `${markName(f.kind)}: ${f.recommended}` : `nenhum ${markName(f.kind).toLowerCase()}`}` : "Nenhum modelo ainda");
</script>

<PageHeader title="IA" lead="Os modelos do OpenRouter que o Blup Studio e o Blup Deals usam. Só o dono do Blup vê esta página." />

<div class="mt-8"><Segmented label="Assunto" options={tabs} bind:value={tab} onchange={show} /></div>

{#if data.ai}
	{#if !data.ai.catalog || !data.ai.speechCatalog}<p class="mt-6 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300">O catálogo do OpenRouter não respondeu agora: os preços ficam com "?" e os IDs não são conferidos.</p>{/if}
	<p class="mt-6 text-sm text-neutral-500">{tabAbout[tab]}</p>
	<div class="mt-4">
		<List empty="Nenhuma lista aqui.">
			{#each rows as f (f.feature)}
				{@const info = featureInfo[f.feature]}
				{@const href = staffHref(resolve("/staff/ai/[feature]", { feature: f.feature }))}
				<ListItem {href} label={info.title}>
					<div class="flex items-center gap-4">
						<span class="grid size-11 shrink-0 place-items-center rounded-xl border border-neutral-200 bg-white dark:border-neutral-700 dark:bg-neutral-900"><AIMark kind={info.mark} class="size-6" /></span>
						<div class="min-w-0 flex-1">
							<p class="truncate font-semibold">{info.title}</p>
							<p class="truncate text-sm text-neutral-500">{summary(f)}</p>
						</div>
					</div>
				</ListItem>
			{/each}
		</List>
	</div>
{/if}
