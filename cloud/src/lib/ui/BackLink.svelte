<script lang="ts">
	import ArrowLeft from "@lucide/svelte/icons/arrow-left";

	// The "← list" link at the top of a page opened from a list.
	let { href, label }: { href: string; label: string } = $props();
</script>

<a {href} class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><ArrowLeft class="size-4" />{label}</a>
