<script lang="ts">
	import type { Snippet } from "svelte";

	// A form field with its explanation beside it (below it on narrow
	// screens), as in the "new monitor" form.
	let { help, children }: { help: string; children: Snippet } = $props();
</script>

<div class="grid gap-2 sm:grid-cols-[minmax(0,20rem)_1fr] sm:items-center sm:gap-5">
	<div>{@render children()}</div>
	<p class="text-sm text-neutral-500">{help}</p>
</div>
