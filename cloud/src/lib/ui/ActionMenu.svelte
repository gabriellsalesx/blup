<script lang="ts">
	import type { Component } from "svelte";
	import EllipsisVertical from "@lucide/svelte/icons/ellipsis-vertical";

	// The three dots of a row or card: a click opens its actions in a small
	// menu under them. Arrows move, Enter picks, Escape or a click outside
	// closes. `danger` paints the item red (deleting), set apart by a line.
	type Item = { label: string; Icon: Component; onselect: () => void; danger?: boolean };
	let { label, items }: { label: string; items: Item[] } = $props();
	const id = $props.id();
	let open = $state(false);

	// Each item sits right in the menu (its parent).
	function pick(item: Item, event: MouseEvent & { currentTarget: HTMLButtonElement }) {
		event.currentTarget.parentElement?.hidePopover();
		item.onselect();
	}
	function keys(event: KeyboardEvent & { currentTarget: HTMLDivElement }) {
		if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
		event.preventDefault();
		const buttons = [...event.currentTarget.querySelectorAll<HTMLButtonElement>("[role=menuitem]")];
		const at = buttons.findIndex((button) => button === document.activeElement);
		buttons[(at + (event.key === "ArrowDown" ? 1 : buttons.length - 1)) % buttons.length]?.focus();
	}
</script>

<button
	type="button"
	popovertarget="{id}-menu"
	aria-haspopup="menu"
	aria-expanded={open}
	aria-label={label}
	style:anchor-name="--menu-{id}"
	class="relative z-10 grid size-9 shrink-0 place-items-center rounded-lg text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white {open ? 'bg-neutral-100 text-neutral-900 dark:bg-neutral-800 dark:text-white' : ''}"
><EllipsisVertical class="size-5" /></button>
<div
	id="{id}-menu"
	popover
	role="menu"
	aria-label={label}
	tabindex="-1"
	style:position-anchor="--menu-{id}"
	class="action-menu rounded-xl border border-neutral-200 bg-white p-1.5 text-sm shadow-lg dark:border-neutral-800 dark:bg-neutral-900 dark:text-white"
	ontoggle={(event) => { open = event.newState === "open"; if (open) event.currentTarget.querySelector<HTMLButtonElement>("[role=menuitem]")?.focus(); }}
	onkeydown={keys}
>
	{#each items as item, index (item.label)}
		{#if item.danger && index > 0 && !items[index - 1]?.danger}<div role="separator" class="mx-1.5 my-1.5 border-t border-neutral-200 dark:border-neutral-800"></div>{/if}
		<button type="button" role="menuitem" class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left font-medium hover:bg-neutral-100 focus-visible:bg-neutral-100 focus-visible:outline-none dark:hover:bg-neutral-800 dark:focus-visible:bg-neutral-800 {item.danger ? 'text-red-600 dark:text-red-400' : ''}" onclick={(event) => pick(item, event)}><item.Icon class="size-4 shrink-0" />{item.label}</button>
	{/each}
</div>

<style>
	.action-menu {
		position: fixed;
		inset: auto;
		top: anchor(bottom);
		right: anchor(right);
		min-width: 11rem;
		margin-top: 0.25rem;
		position-try-fallbacks: flip-block;
	}
</style>
