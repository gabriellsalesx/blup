<script lang="ts">
	import type { Snippet } from "svelte";
	import type { HTMLButtonAttributes } from "svelte/elements";
	import LoaderCircle from "@lucide/svelte/icons/loader-circle";

	// The panel's button: primary (dark) or secondary (outlined). While busy
	// it shows a spinner and can't be pressed again.
	let { variant = "primary", busy = false, disabled = false, class: extra = "", children, ...rest }: HTMLButtonAttributes & {
		variant?: "primary" | "secondary";
		busy?: boolean;
		children: Snippet;
	} = $props();

	const look = {
		primary: "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900",
		secondary: "border border-neutral-300 dark:border-neutral-700",
	};
</script>

<button
	class="flex items-center justify-center gap-2 rounded-lg px-4 py-2.5 font-semibold disabled:opacity-50 {look[variant]} {extra}"
	disabled={disabled || busy}
	{...rest}
>
	{#if busy}<LoaderCircle class="size-4 animate-spin" />{/if}
	{@render children()}
</button>
