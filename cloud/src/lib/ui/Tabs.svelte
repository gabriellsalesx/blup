<script lang="ts" generics="T extends string">
	// Tabs with an underline under the chosen one, scrolling sideways when
	// they don't fit (narrow screens). Arrows move between them.
	let { label, tabs, value = $bindable() }: { label: string; tabs: readonly { value: T; label: string }[]; value: T } = $props();
	function keys(event: KeyboardEvent, index: number) {
		if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") return;
		event.preventDefault();
		const next = tabs[(index + (event.key === "ArrowRight" ? 1 : tabs.length - 1)) % tabs.length];
		if (!next) return;
		value = next.value;
		const list = (event.currentTarget as HTMLElement).parentElement;
		list?.querySelector<HTMLElement>(`[data-tab="${next.value}"]`)?.focus();
	}
</script>

<div class="flex overflow-x-auto border-b border-neutral-200 dark:border-neutral-800" role="tablist" aria-label={label}>
	{#each tabs as tab, index (tab.value)}
		<button
			type="button"
			role="tab"
			data-tab={tab.value}
			aria-selected={tab.value === value}
			tabindex={tab.value === value ? 0 : -1}
			class="-mb-px shrink-0 border-b-2 px-4 py-2.5 text-sm font-semibold whitespace-nowrap transition-colors {tab.value === value ? 'border-blup text-neutral-900 dark:text-white' : 'border-transparent text-neutral-500 hover:text-neutral-900 dark:hover:text-white'}"
			onclick={() => (value = tab.value)}
			onkeydown={(event) => keys(event, index)}
		>{tab.label}</button>
	{/each}
</div>
