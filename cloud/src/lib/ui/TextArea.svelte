<script lang="ts">
	import type { HTMLTextareaAttributes } from "svelte/elements";

	// A tall text box with its label on the border, like FloatField.
	let { label, id, value = $bindable(""), surface = "bg-neutral-50 dark:bg-neutral-950", ...rest }: HTMLTextareaAttributes & { label: string; id: string; surface?: string } = $props();
</script>

<div class="relative">
	<label for={id} class="absolute -top-2.5 left-2 px-1 text-[13px] text-neutral-500 {surface}">{label}</label>
	<textarea {id} class="min-h-28 w-full rounded-md border border-neutral-300 p-3 outline-none focus:border-blup dark:border-neutral-700 {surface}" bind:value {...rest}></textarea>
</div>
