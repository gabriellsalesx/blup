<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
	import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
	import { confirmState, answer } from "#lib/confirm.svelte.ts";
	import Button from "./Button.svelte";
	import Field from "./Field.svelte";
	import Modal from "./Modal.svelte";

	// The one confirmation dialog (see confirm.svelte.ts). Esc or Cancel
	// answer no.
	const opts = $derived(confirmState.opts);
	const ready = $derived(!opts?.typeToConfirm || confirmState.typed.trim().toLowerCase() === opts.typeToConfirm.toLowerCase());
</script>

{#if opts}
	<Modal bind:open={confirmState.open} title={opts.title} onclose={() => answer(false)}>
		<div class="flex gap-3">
			{#if opts.danger}<span class="grid size-9 shrink-0 place-items-center rounded-full bg-red-100 text-red-600 dark:bg-red-950 dark:text-red-400"><TriangleAlert class="size-[18px]" /></span>{/if}
			<p class="text-sm text-neutral-500">{opts.message}</p>
		</div>
		<form class="mt-5 grid gap-3" onsubmit={(e) => { e.preventDefault(); if (ready) answer(true); }}>
			{#if opts.typeToConfirm}
				<Field label={t("typeConfirm",{word:opts.typeToConfirm})} autocomplete="off" bind:value={confirmState.typed} />
			{/if}
			<div class="flex justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => answer(false)}>{t("cancel")}</Button>
				<Button class={opts.danger ? "bg-red-700! text-white!" : ""} disabled={!ready}>{opts.confirmLabel ?? t("confirm")}</Button>
			</div>
		</form>
	</Modal>
{/if}
