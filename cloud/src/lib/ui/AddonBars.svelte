<script lang="ts">
	import type { Addons } from "#lib/staff.ts";
	import { formatDate } from "#lib/i18n.svelte.ts";
	import { bytes } from "#lib/usage.ts";

	// The staff's view of an account's add-ons: what Blup Pulse and Blup
	// Deals allow on the plan and how much is in use, as the Cloud's bars.
	let { addons }: { addons: Addons } = $props();
	const n = (v: number) => v.toLocaleString("pt-BR");
	const of = (used: number, max: number) => `${n(used)} de ${n(max)}`;
	const products = $derived([
		...(addons.pulse
			? [{
					name: "Blup Pulse",
					help: `Erros de ${formatDate(`${addons.pulse.used.month}-01T00:00:00Z`, { month: "long", year: "numeric", timeZone: "UTC" })}, guardados por ${addons.pulse.limits.retentionDays} dias. ${bytes(addons.pulse.used.bytes)} em uso.`,
					bars: [
						{ label: "Erros recebidos no mês", used: addons.pulse.used.accepted, max: addons.pulse.limits.events },
						{ label: "Projetos", used: addons.pulse.used.projects, max: addons.pulse.limits.projects },
						{ label: "Monitores", used: addons.pulse.used.monitors, max: addons.pulse.limits.monitors },
						{ label: "Páginas de status", used: addons.pulse.used.pages, max: addons.pulse.limits.pages },
					],
				}]
			: []),
		...(addons.deals
			? [{
					name: "Blup Deals",
					help: `Os posts do dia recomeçam em ${formatDate(addons.deals.used.resetsAt, { day: "numeric", month: "long", hour: "2-digit", minute: "2-digit", timeZone: "UTC" })} (UTC). Histórico de ${addons.deals.limits.historyDays} dias.`,
					bars: [
						{ label: "Posts hoje", used: addons.deals.used.postsToday, max: addons.deals.limits.postsPerDay },
						{ label: "Canais", used: addons.deals.used.channels, max: addons.deals.limits.channels },
						{ label: "Caçadas", used: addons.deals.used.hunts, max: addons.deals.limits.hunts },
						{ label: "Espelhos", used: addons.deals.used.mirrors, max: addons.deals.limits.mirrors },
					],
				}]
			: []),
	]);
	const pct = (used: number, max: number) => (max > 0 ? Math.min(100, (used / max) * 100) : 0);
</script>

{#if products.length}
	<div class="mt-3 grid gap-6">
		{#each products as p (p.name)}
			<div>
				<p class="text-sm font-semibold">{p.name}</p>
				<p class="text-sm text-neutral-500">{p.help}</p>
				<div class="mt-3 grid gap-4">
					{#each p.bars as bar (bar.label)}
						<div>
							<div class="flex items-baseline justify-between gap-3 text-sm"><span class="font-medium">{bar.label}</span><span class="text-neutral-500">{of(bar.used, bar.max)}</span></div>
							<div class="mt-1.5 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full {pct(bar.used, bar.max) >= 90 ? 'bg-red-500' : 'bg-blup'}" style:width="{pct(bar.used, bar.max)}%"></div></div>
						</div>
					{/each}
				</div>
			</div>
		{/each}
	</div>
{:else}
	<p class="mt-3 text-sm text-neutral-500">O plano não inclui Blup Pulse nem Blup Deals.</p>
{/if}
