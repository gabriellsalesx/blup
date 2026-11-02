<script lang="ts">
	import type { Snippet } from "svelte";

	// A row of a List: what it is on the left and its actions on the right
	// (below it on narrow screens). `selected` shades the chosen row; more
	// content (details that open) goes in `extra`, under the row.
	let { selected = false, children, actions, extra }: { selected?: boolean; children: Snippet; actions?: Snippet; extra?: Snippet } = $props();
</script>

<div class="border-b border-neutral-200 px-5 py-4 last-of-type:border-b-0 dark:border-neutral-800 {selected ? 'bg-neutral-50 dark:bg-neutral-800/50' : ''}">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div class="min-w-0 flex-1 basis-full sm:basis-0">{@render children()}</div>
		{#if actions}<div class="flex flex-wrap gap-2">{@render actions()}</div>{/if}
	</div>
	{@render extra?.()}
</div>
