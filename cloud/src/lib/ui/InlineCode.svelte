<script lang="ts">
	import type { Snippet } from "svelte";
	// A bit of code inside a sentence (the start of a key, a file name):
	// monospace on the code background, nothing to copy.
	let { children }: { children: Snippet } = $props();
</script>

<code class="rounded-md bg-neutral-100 px-1.5 py-0.5 font-mono text-[0.9em] text-neutral-900 dark:bg-neutral-800 dark:text-neutral-100">{@render children()}</code>
