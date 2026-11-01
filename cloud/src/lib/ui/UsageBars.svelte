<script lang="ts">
	import { bytes, megabytes, type Usage } from "#lib/usage.ts";
	import type { AIShare, Limits } from "#lib/api.ts";

	// The account's usage against its plan: storage and sites as
	// bars, then devices and the last sync. Only numbers, never the notes.
	let { usage, limits, ai }: { usage: Usage; limits: Limits; ai?: AIShare } = $props();

	const ago = (d: string) => {
		const min = Math.round((Date.now() - new Date(d).getTime()) / 60000);
		if (min < 1) return "agora";
		if (min < 60) return `há ${min} min`;
		if (min < 60 * 24) return `há ${Math.round(min / 60)} h`;
		return `há ${Math.round(min / 60 / 24)} dias`;
	};
	const bars = $derived([
		{ label: "Armazenamento", used: usage.storageBytes, max: limits.storageMB * 1024 * 1024, text: `${bytes(usage.storageBytes)} de ${megabytes(limits.storageMB)}` },
		{ label: "Sites", used: usage.sites, max: limits.sites, text: limits.sites < 0 ? `${usage.sites} · sem limite` : limits.sites === 0 ? "Não inclusos no plano" : `${usage.sites} de ${limits.sites}` },
		...(ai ? [{ label: "Uso de IA", used: ai.percent, max: ai.included ? 100 : 0, text: ai.included ? `${ai.percent}% deste mês · renova em ${new Date(ai.resetsAt).toLocaleDateString("pt-BR", { day: "numeric", month: "long", timeZone: "UTC" })}` : "Não incluso no plano" }] : []),
	]);
	const pct = (used: number, max: number) => (max > 0 ? Math.min(100, (used / max) * 100) : 0);
</script>

<div class="mt-3 grid gap-4">
	{#each bars as b (b.label)}
		<div>
			<div class="flex items-baseline justify-between gap-3 text-sm">
				<span class="font-medium">{b.label}</span>
				<span class="text-neutral-500">{b.text}</span>
			</div>
			{#if b.max > 0}
				<div class="mt-1.5 h-2 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
					<div class="h-full rounded-full {pct(b.used, b.max) >= 90 ? 'bg-red-500' : 'bg-blup'}" style:width="{pct(b.used, b.max)}%"></div>
				</div>
			{/if}
		</div>
	{/each}
	<p class="text-sm text-neutral-500">
		{usage.devices}{limits.devices >= 0 ? ` de ${limits.devices}` : ""} {usage.devices === 1 && limits.devices < 0 ? "computador com o Studio" : "computadores com o Studio"} ·
		{usage.lastSyncAt ? `última sincronização ${ago(usage.lastSyncAt)}` : "nunca sincronizou"}
	</p>
</div>
