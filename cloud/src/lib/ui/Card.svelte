<script lang="ts">
	import type { Snippet } from "svelte";

	// A box of the panel, with an optional title, help and action on top.
	let { title, help, action, children }: { title?: string; help?: string; action?: Snippet; children?: Snippet } = $props();
</script>

<section class="rounded-xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
	{#if title || action}
		<div class="flex items-start justify-between gap-4">
			<div>
				{#if title}<h2 class="font-bold">{title}</h2>{/if}
				{#if help}<p class="mt-0.5 text-sm text-neutral-500">{help}</p>{/if}
			</div>
			{@render action?.()}
		</div>
	{/if}
	{@render children?.()}
</section>
