<script lang="ts">
	import type { HTMLInputAttributes } from "svelte/elements";

	// A text field with its label above.
	let { label, value = $bindable(""), class: extra = "", ...rest }: HTMLInputAttributes & { label?: string } = $props();
</script>

<label class="grid gap-1.5 text-sm font-medium">
	{#if label}{label}{/if}
	<input
		class="rounded-lg border border-neutral-300 bg-white px-3 py-2.5 text-base outline-none focus:border-blup dark:border-neutral-700 dark:bg-neutral-900 {extra}"
		bind:value
		{...rest}
	/>
</label>
