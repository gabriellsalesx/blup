<script lang="ts">
	import Power from "@lucide/svelte/icons/power";
	import { t } from "#lib/i18n.svelte.ts";

	// The panels' "Sign out" under the menu: red on hover and while pressed.
	let { onclick }: { onclick: () => void } = $props();
</script>

<button type="button" class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium text-neutral-600 transition-colors hover:text-red-600 focus-visible:text-red-600 active:text-red-700 dark:text-neutral-400 dark:hover:text-red-400 dark:focus-visible:text-red-400 dark:active:text-red-500" {onclick}><Power class="size-[18px]" />{t("signOut")}</button>
