<script lang="ts">
	import type { Snippet } from "svelte";

	// A small rounded label: a state, a type, "Live".
	let { tone = "neutral", children }: { tone?: "neutral" | "green" | "amber" | "red" | "blue" | "indigo"; children: Snippet } = $props();
	const look = {
		neutral: "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400",
		green: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300",
		amber: "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300",
		red: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
		blue: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
		indigo: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
	};
</script>

<span class="inline-block rounded-full px-2 py-0.5 text-xs font-semibold {look[tone]}">{@render children()}</span>
