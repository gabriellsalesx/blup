<script lang="ts">
	import type { Snippet } from "svelte";

	// The top of a panel page: its title, a line about it and, on the
	// right, something extra (a usage bar, an action). A long line wraps;
	// the extra stays on the right (under it only on phones).
	let { title, lead, aside }: { title: string; lead?: string; aside?: Snippet } = $props();
</script>

<div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
	<div class="min-w-0 flex-1">
		<h1 class="text-3xl font-extrabold tracking-tight break-words">{title}</h1>
		{#if lead}<p class="mt-2 text-neutral-500">{lead}</p>{/if}
	</div>
	{#if aside}<div class="shrink-0">{@render aside()}</div>{/if}
</div>
