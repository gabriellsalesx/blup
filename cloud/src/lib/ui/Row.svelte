<script lang="ts">
	import type { Snippet } from "svelte";

	// A line of a settings page: the title, what it is (or its value) and
	// the action on the right. Lines are separated by a rule.
	let { title, help, value, action, children }: { title: string; help?: string; value?: string; action?: Snippet; children?: Snippet } = $props();
</script>

<div class="flex flex-wrap items-center justify-between gap-x-8 gap-y-3 border-b border-neutral-200 py-5 last:border-b-0 dark:border-neutral-800">
	<div class="min-w-0 flex-1 basis-72">
		<h2 class="font-semibold">{title}</h2>
		{#if value}<p class="mt-0.5 text-lg">{value}</p>{/if}
		{#if help}<p class="mt-0.5 text-sm text-neutral-500">{help}</p>{/if}
		{@render children?.()}
	</div>
	{#if action}<div class="shrink-0">{@render action()}</div>{/if}
</div>
