<script lang="ts" generics="T extends string|number">
	import ChevronDown from "@lucide/svelte/icons/chevron-down";
	import Check from "@lucide/svelte/icons/check";

	// The panel's dropdown, in place of the system one: the same tall box as
	// FloatField, with its label on the border, and a list that opens under
	// it. Arrows move, Enter or Space picks, Escape closes.
	type Option = { value: T; label: string };
	let { label, options, value = $bindable(), surface = "bg-neutral-50 dark:bg-neutral-950", disabled = false, onchange }: {
		label: string;
		options: readonly Option[];
		value: T;
		surface?: string;
		disabled?: boolean;
		onchange?: (value: T) => void;
	} = $props();
	const id = $props.id();
	let open = $state(false), active = $state(0), list = $state<HTMLElement>(), button = $state<HTMLButtonElement>();
	const current = $derived(options.find((o) => o.value === value));

	function show() {
		active = Math.max(0, options.findIndex((o) => o.value === value));
		list?.showPopover();
	}
	function pick(option: Option) {
		value = option.value;
		list?.hidePopover();
		button?.focus();
		onchange?.(option.value);
	}
	function keys(event: KeyboardEvent) {
		if (event.key === "ArrowDown" || event.key === "ArrowUp") {
			event.preventDefault();
			if (!open) return show();
			active = (active + (event.key === "ArrowDown" ? 1 : options.length - 1)) % options.length;
		} else if ((event.key === "Enter" || event.key === " ") && open) {
			event.preventDefault();
			const option = options[active];
			if (option) pick(option);
		}
	}
</script>

<div class="relative">
	<span class="pointer-events-none absolute -top-2.5 left-2 z-10 px-1 text-[13px] text-neutral-500 {surface}">{label}</span>
	<button
		bind:this={button}
		type="button"
		{disabled}
		popovertarget="{id}-list"
		aria-haspopup="listbox"
		aria-expanded={open}
		aria-label="{label}: {current?.label ?? ''}"
		style:anchor-name="--select-{id}"
		class="flex min-h-14 w-full items-center justify-between gap-3 rounded-md border px-3 text-left text-base transition-colors disabled:opacity-50 {surface} {open ? 'border-blup' : 'border-neutral-300 hover:border-neutral-400 dark:border-neutral-700'}"
		onkeydown={keys}
	>
		<span class="truncate">{current?.label ?? ""}</span>
		<ChevronDown class="size-4 shrink-0 text-neutral-500 transition-transform {open ? 'rotate-180' : ''}" />
	</button>
	<div
		bind:this={list}
		id="{id}-list"
		popover
		role="listbox"
		aria-label={label}
		style:position-anchor="--select-{id}"
		class="select-list rounded-xl border border-neutral-200 bg-white p-1.5 text-sm shadow-lg dark:border-neutral-800 dark:bg-neutral-900 dark:text-white"
		ontoggle={(event) => (open = event.newState === "open")}
	>
		{#each options as option, index (option.value)}
			<button
				type="button"
				role="option"
				aria-selected={option.value === value}
				class="flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2.5 text-left {index === active ? 'bg-neutral-100 dark:bg-neutral-800' : ''}"
				onmouseenter={() => (active = index)}
				onclick={() => pick(option)}
			>
				<span class="truncate">{option.label}</span>
				{#if option.value === value}<Check class="size-4 shrink-0 text-blup" />{/if}
			</button>
		{/each}
	</div>
</div>

<style>
	.select-list {
		position: fixed;
		inset: auto;
		top: anchor(bottom);
		left: anchor(left);
		width: anchor-size(width);
		margin-top: 0.375rem;
		max-height: 18rem;
		overflow: auto;
		position-try-fallbacks: flip-block;
	}
</style>
