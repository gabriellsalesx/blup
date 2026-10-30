<script lang="ts">
	import type { Snippet } from "svelte";

	// A dialog over the page (the browser's own <dialog>: Esc closes it and
	// focus stays inside).
	let { open = $bindable(false), title, children }: { open?: boolean; title: string; children: Snippet } = $props();

	// Re-runs when `open` changes.
	function show(el: HTMLDialogElement) {
		if (open && !el.open) el.showModal();
		else if (!open && el.open) el.close();
	}
</script>

<dialog
	{@attach show}
	onclose={() => (open = false)}
	class="m-auto w-[min(28rem,calc(100%-2rem))] rounded-xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl backdrop:bg-black/40 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-100"
>
	<h2 class="text-lg font-bold">{title}</h2>
	<div class="mt-3">{@render children()}</div>
</dialog>
