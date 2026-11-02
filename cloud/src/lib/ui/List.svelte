<script lang="ts">
	import type { Snippet } from "svelte";

	// A white box of rows (ListItem), with a message when it is empty.
	let { empty, children }: { empty?: string; children?: Snippet } = $props();
</script>

<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
	{@render children?.()}
	<p class="hidden px-5 py-10 text-center text-neutral-500 only:block">{empty ?? ""}</p>
</div>
