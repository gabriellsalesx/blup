<script lang="ts">
	import type { HTMLInputAttributes } from "svelte/elements";

	// A tall field whose label sits inside and floats onto the border while
	// focused or filled (the sign-in page's).
	// `surface` is the background behind it (the floating label covers the
	// border with it).
	let { label, id, value = $bindable(""), surface = "bg-neutral-50 dark:bg-neutral-950", class: extra = "", ...rest }: HTMLInputAttributes & { label: string; id: string; surface?: string } = $props();
	let focused = $state(false);
	const up = $derived(focused || String(value ?? "").length > 0);
</script>

<div class="relative flex min-h-14 flex-col justify-center rounded-md border px-3 transition-colors {surface} {focused ? 'border-blup' : 'border-neutral-300 dark:border-neutral-700'}">
	<label
		for={id}
		class="pointer-events-none absolute left-2 px-1 transition-all duration-200 {surface} {up ? '-top-2.5 text-[13px]' : 'top-1/2 -translate-y-1/2 text-base'} {focused ? 'text-blup' : 'text-neutral-500'}"
	>{label}</label>
	<input
		{id}
		class="w-full border-none bg-transparent p-0 text-base outline-none {extra}"
		bind:value
		onfocus={() => (focused = true)}
		onblur={() => (focused = false)}
		{...rest}
	/>
</div>
