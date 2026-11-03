<script lang="ts">
	import type { Snippet } from "svelte";
	import ExternalLink from "@lucide/svelte/icons/external-link";

	// The top of one item's page (a monitor, a status page): its name with
	// an optional badge, its address underlined, opening in a new tab, and
	// its actions on the right (under it on phones). The address turns the
	// panel's color on hover (mint on Pulse, blue on Cloud).
	let { title, url, badge, actions }: { title: string; url?: string; badge?: Snippet; actions?: Snippet } = $props();
</script>

<header class="mt-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
	<div class="min-w-0">
		<h1 class="flex flex-wrap items-center gap-3 text-2xl font-bold"><span class="truncate">{title}</span>{@render badge?.()}</h1>
		{#if url}<a class="mt-1 inline-flex max-w-full items-center gap-1 text-sm text-neutral-500 underline transition-colors hover:text-blup" href={url} target="_blank" rel="noopener noreferrer"><span class="truncate">{url}</span><ExternalLink class="size-3.5 shrink-0" /></a>{/if}
	</div>
	{#if actions}<div class="flex shrink-0 flex-wrap gap-2">{@render actions()}</div>{/if}
</header>
