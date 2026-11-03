<script lang="ts">
	import { onDestroy } from "svelte";
	import Copy from "@lucide/svelte/icons/copy";
	import Check from "@lucide/svelte/icons/check";
	import { t } from "#lib/i18n.svelte.ts";
	import { toast } from "#lib/toast.svelte.ts";
	import { copyText } from "#lib/clipboard.ts";

	// Copies `text` and says so on the button itself: for two seconds it
	// turns green, with a check and "Copied" (the toast still shows too).
	// `size` "xs" is the small one next to an id. `label` is what a screen
	// reader says before copying.
	let { text, label, size = "small", title }: { text: string; label?: string; size?: "small" | "xs"; title?: string } = $props();
	let copied = $state(false), timer: ReturnType<typeof setTimeout> | undefined;
	async function copy() {
		if (!(await copyText(text))) return toast(() => t("copyFailed"), { kind: "error" });
		copied = true;
		toast(() => t("copied"));
		clearTimeout(timer);
		timer = setTimeout(() => (copied = false), 2000);
	}
	onDestroy(() => clearTimeout(timer));
	const look = $derived(copied ? "border-emerald-500 bg-emerald-50 text-emerald-700 dark:border-emerald-500 dark:bg-emerald-950 dark:text-emerald-300" : "border-neutral-300 hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800");
</script>

<button type="button" class="inline-flex shrink-0 items-center gap-1.5 border font-semibold transition-colors {size === 'xs' ? 'rounded-md px-2 py-1 text-xs' : 'rounded-lg px-3 py-1.5 text-sm'} {look}" aria-label={copied ? t("copied") : (label ?? t("copy"))} {title} onclick={copy}>
	{#if copied}<Check class={size === "xs" ? "size-3.5" : "size-4"} />{:else}<Copy class={size === "xs" ? "size-3.5" : "size-4"} />{/if}
	<span aria-live="polite">{copied ? t("copied") : t("copy")}</span>
</button>
