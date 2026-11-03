<script lang="ts">
	import CopyButton from "./CopyButton.svelte";

	// A piece of code or a key to copy: monospace on the code background,
	// with the copy button in its corner. A command keeps its lines (and
	// scrolls sideways); a key breaks anywhere to fit.
	let { code, label, wrap = false }: { code: string; label?: string; wrap?: boolean } = $props();
</script>

<div class="relative">
	<pre class="rounded-lg bg-neutral-100 p-4 pr-28 font-mono text-[13px] leading-6 text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100 {wrap ? 'break-all whitespace-pre-wrap' : 'overflow-x-auto'}">{code}</pre>
	<div class="absolute top-2.5 right-2.5"><CopyButton text={code} {label} /></div>
</div>
