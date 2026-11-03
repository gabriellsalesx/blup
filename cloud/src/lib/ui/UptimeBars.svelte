<script lang="ts">
	import Tooltip from "./Tooltip.svelte";

	// Days as bars, oldest first: green all up, amber mostly up, red mostly
	// down and gray without checks. Each bar shows its tip on hover.
	type Day = { key: number; tip: string; up: number; down: number };
	let { days, label }: { days: Day[]; label: string } = $props();
	const tone = (d: Day) => {
		const total = d.up + d.down;
		if (!total) return "bg-neutral-200 dark:bg-neutral-700";
		return d.up === total ? "bg-emerald-500" : d.up / total >= 0.9 ? "bg-amber-500" : "bg-red-500";
	};
</script>

<div class="flex h-9 gap-[3px]" role="img" aria-label={label}>
	{#each days as day, index (day.key)}
		<Tooltip class="flex flex-1" text={day.tip} align={index < 5 ? "start" : index >= days.length - 5 ? "end" : "center"}><span class="w-full rounded-full {tone(day)}"></span></Tooltip>
	{/each}
</div>
