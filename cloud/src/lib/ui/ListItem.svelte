<script lang="ts">
	import type { Snippet } from "svelte";
	import { cardLink } from "#lib/card.ts";

	// A row of a List: what it is on the left and its actions on the right
	// (below it on narrow screens). `selected` shades the chosen row; more
	// content (details that open) goes in `extra`, under the row. With
	// `href` the whole row opens it; links and the actions inside stay
	// clickable on their own (`relative z-10`). `menu` (the three dots)
	// always stays at the right end.
	let { selected = false, href, label, children, actions, menu, extra }: { selected?: boolean; href?: string; label?: string; children: Snippet; actions?: Snippet; menu?: Snippet; extra?: Snippet } = $props();
</script>

<!-- The row's link covers it: a click anywhere opens it. -->
<div class="relative border-b border-neutral-200 px-5 py-4 last-of-type:border-b-0 dark:border-neutral-800 {selected ? 'bg-neutral-50 dark:bg-neutral-800/50' : ''} {href ? 'cursor-pointer transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/60' : ''}">
	{#if href}<a {href} class={cardLink} aria-label={label}></a>{/if}
	<div class="flex items-center gap-3">
		<div class="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-3">
			<div class="min-w-0 flex-1 basis-full sm:basis-0">{@render children()}</div>
			{#if actions}<div class="relative z-10 flex flex-wrap gap-2">{@render actions()}</div>{/if}
		</div>
		{#if menu}<div class="relative z-10 shrink-0">{@render menu()}</div>{/if}
	</div>
	{@render extra?.()}
</div>
