<script lang="ts">
	// Days as bars, oldest first: green all up, amber mostly up, red mostly
	// down and gray without checks. Each bar says its day and share on hover.
	type Day = { key: number; label: string; up: number; down: number };
	let { days, label }: { days: Day[]; label: string } = $props();
	const tone = (d: Day) => {
		const total = d.up + d.down;
		if (!total) return "bg-neutral-200 dark:bg-neutral-700";
		return d.up === total ? "bg-emerald-500" : d.up / total >= 0.9 ? "bg-amber-500" : "bg-red-500";
	};
</script>

<div class="flex h-9 gap-[3px]" role="img" aria-label={label}>
	{#each days as day (day.key)}<span class="flex-1 rounded-full {tone(day)}" title={day.label}></span>{/each}
</div>
