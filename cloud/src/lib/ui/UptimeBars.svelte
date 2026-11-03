<script lang="ts">
	import BarTip from "./BarTip.svelte";
	import DayTip, { type DayState } from "./DayTip.svelte";

	// Days as bars, oldest first: green all up, amber mostly up, red mostly
	// down, indigo for a day with scheduled maintenance (unless it went down
	// outside it) and gray without checks. Each bar opens its day's card.
	type Day = { key: number; state: DayState; name: string; detail: string; date: string };
	let { days, label }: { days: Day[]; label: string } = $props();
	const color: Record<DayState, string> = { up: "bg-emerald-500", partial: "bg-amber-500", down: "bg-red-500", maint: "bg-indigo-500", none: "bg-neutral-200 dark:bg-neutral-700" };
</script>

<div class="flex h-9 gap-[2px] sm:gap-[3px]" role="group" aria-label={label}>
	{#each days as day, index (day.key)}
		<BarTip class="flex-1" label={`${day.date}: ${day.name}`} align={index < 5 ? "start" : index >= days.length - 5 ? "end" : "center"}>
			<span class="w-full rounded-full {color[day.state]}"></span>
			{#snippet tip()}<DayTip state={day.state} name={day.name} detail={day.detail} date={day.date} />{/snippet}
		</BarTip>
	{/each}
</div>
