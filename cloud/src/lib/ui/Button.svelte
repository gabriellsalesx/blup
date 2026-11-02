<script lang="ts">
	import type { Snippet } from "svelte";
	import type { HTMLButtonAttributes } from "svelte/elements";
	import LoaderCircle from "@lucide/svelte/icons/loader-circle";

	// The panel's button: primary (dark) or secondary (outlined); `pill` is
	// the tall rounded one of the sign-in page. While busy it shows a spinner
	// and can't be pressed again.
	let { variant = "primary", pill = false, small = false, busy = false, disabled = false, class: extra = "", children, ...rest }: HTMLButtonAttributes & {
		variant?: "primary" | "secondary";
		pill?: boolean;
		small?: boolean;
		busy?: boolean;
		children: Snippet;
	} = $props();

	const look = {
		primary: "bg-neutral-900 text-white hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200",
		secondary: "border border-neutral-300 hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900",
	};
</script>

<button
	class="flex items-center justify-center gap-2 font-semibold transition-colors disabled:opacity-50 {pill ? 'h-[52px] rounded-full px-6 text-[15px]' : small ? 'rounded-lg px-3 py-1.5 text-sm' : 'rounded-lg px-4 py-2.5'} {look[variant]} {extra}"
	disabled={disabled || busy}
	{...rest}
>
	{#if busy}<LoaderCircle class="size-4 animate-spin" />{/if}
	{@render children()}
</button>
