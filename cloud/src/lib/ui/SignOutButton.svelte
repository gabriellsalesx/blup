<script lang="ts">
	import Power from "@lucide/svelte/icons/power";
	import { t } from "#lib/i18n.svelte.ts";

	// The panels' "Sign out" under the menu: red on hover and while pressed.
	// `compact` is its icon alone (the phone's top bar).
	let { onclick, compact = false }: { onclick: () => void; compact?: boolean } = $props();
</script>

<button type="button" aria-label={compact ? t("signOut") : undefined} class="flex items-center gap-2.5 rounded-lg {compact ? 'p-2' : 'px-2.5 py-1.5'} font-medium text-neutral-600 transition-colors hover:text-red-600 focus-visible:text-red-600 active:text-red-700 dark:text-neutral-400 dark:hover:text-red-400 dark:focus-visible:text-red-400 dark:active:text-red-500" {onclick}><Power class="size-[18px]" />{#if !compact}{t("signOut")}{/if}</button>
