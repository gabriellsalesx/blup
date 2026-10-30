<script lang="ts">
	// A number with its label (the staff's overview).
	let { label, value, hint }: { label: string; value: number | string; hint?: string } = $props();
</script>

<div class="rounded-xl border border-neutral-200 px-4 py-3 dark:border-neutral-800">
	<p class="text-sm text-neutral-500">{label}</p>
	<p class="mt-1 text-2xl font-extrabold tabular-nums">{typeof value === "number" ? value.toLocaleString("pt-BR") : value}</p>
	{#if hint}<p class="text-xs text-neutral-500">{hint}</p>{/if}
</div>
