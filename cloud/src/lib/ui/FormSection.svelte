<script lang="ts">
	import type { Snippet } from "svelte";

	// One part of a long form: its name and what it is for on the left, the
	// fields in a box on the right (stacked on narrow screens).
	let { title, help, children }: { title: string; help?: string; children: Snippet } = $props();
</script>

<section class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,17rem)_minmax(0,1fr)] lg:gap-10">
	<div>
		<h2 class="font-bold">{title}</h2>
		{#if help}<p class="mt-1 text-sm text-neutral-500">{help}</p>{/if}
	</div>
	<div class="rounded-xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">{@render children()}</div>
</section>
