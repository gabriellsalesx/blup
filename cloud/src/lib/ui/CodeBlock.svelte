<script lang="ts">
	import CopyButton from "./CopyButton.svelte";

	// A piece of code or a key to copy: monospace on the code background,
	// its lines kept as they are (a long one scrolls sideways, also on a
	// phone) and the copy button in its own column, so the code never runs
	// under it.
	let { code, label }: { code: string; label?: string } = $props();
</script>

<div class="flex items-start gap-2 rounded-lg bg-neutral-100 dark:bg-neutral-950">
	<pre class="min-w-0 flex-1 overflow-x-auto py-4 pl-4 font-mono text-[13px] leading-6 whitespace-pre text-neutral-900 dark:text-neutral-100">{code}</pre>
	<div class="shrink-0 py-2.5 pr-2.5"><CopyButton text={code} {label} /></div>
</div>
