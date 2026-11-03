<script lang="ts">
	import type { Snippet } from "svelte";

	// A bar of a chart that opens a small card above it (the public status
	// page's): on hover, on keyboard focus and, on a phone, with a tap (the
	// bar is a button). `align` keeps the card inside the screen at the ends
	// of a row; `label` is what a screen reader says for the bar.
	let { label, align = "center", class: extra = "", children, tip }: { label: string; align?: "center" | "start" | "end"; class?: string; children: Snippet; tip: Snippet } = $props();
	const place = { center: "left-1/2 -translate-x-1/2", start: "left-0", end: "right-0" } as const;
</script>

<button type="button" class="group relative flex cursor-pointer rounded-full outline-none focus-visible:ring-2 focus-visible:ring-neutral-400 {extra}" aria-label={label}>
	{@render children()}
	<span role="tooltip" class="pointer-events-none absolute bottom-[calc(100%+10px)] z-30 hidden w-max min-w-48 max-w-64 rounded-xl border border-neutral-200 bg-white px-3.5 pt-3 pb-2.5 text-left text-sm font-medium whitespace-normal text-neutral-900 shadow-lg group-hover:block group-focus:block dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-100 {place[align]}">{@render tip()}</span>
</button>
