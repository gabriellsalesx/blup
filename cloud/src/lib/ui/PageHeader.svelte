<script lang="ts">
	import type { Snippet } from "svelte";

	// The top of a panel page: its title, a line about it and, on the
	// right, something extra (a usage bar, an action).
	let { title, lead, aside }: { title: string; lead?: string; aside?: Snippet } = $props();
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
	<div class="min-w-0">
		<h1 class="text-3xl font-extrabold tracking-tight break-words">{title}</h1>
		{#if lead}<p class="mt-2 text-neutral-500">{lead}</p>{/if}
	</div>
	{@render aside?.()}
</div>
