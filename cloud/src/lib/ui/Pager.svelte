<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
	import ChevronLeft from "@lucide/svelte/icons/chevron-left";
	import ChevronRight from "@lucide/svelte/icons/chevron-right";

	// Previous and next under a list (see pager.svelte.ts). Hidden when
	// everything fits on one page.
	let { pager }: { pager: { page: number; hasPrev: boolean; hasNext: boolean; busy: boolean; prev: () => void; next: () => void } } = $props();

	const btn = "flex items-center gap-1 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold hover:bg-neutral-100 disabled:opacity-40 dark:border-neutral-700 dark:hover:bg-neutral-900";
</script>

{#if pager.hasPrev || pager.hasNext}
	<nav class="mt-4 flex items-center justify-between gap-3" aria-label={t("pagination")}>
		<button class={btn} disabled={!pager.hasPrev || pager.busy} onclick={() => pager.prev()}><ChevronLeft class="size-4" />{t("previous")}</button>
		<span class="text-sm text-neutral-500">{t("pageNumber",{number:pager.page})}</span>
		<button class={btn} disabled={!pager.hasNext || pager.busy} onclick={() => pager.next()}>{t("next")}<ChevronRight class="size-4" /></button>
	</nav>
{/if}
