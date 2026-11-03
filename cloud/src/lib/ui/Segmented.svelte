<script lang="ts" generics="T extends string">
	// A row of choices where one is on (the plan page's Monthly/Yearly).
	// When it is wider than the screen it slides sideways, with no scrollbar.
	let { label, options, value = $bindable(), onchange }: {
		label: string;
		options: readonly { value: T; label: string }[];
		value: T;
		onchange?: (value: T) => void;
	} = $props();
</script>

<div class="inline-flex max-w-full overflow-x-auto [scrollbar-width:none] rounded-full border border-neutral-200 bg-white p-1 text-sm font-semibold dark:border-neutral-800 dark:bg-neutral-900" role="radiogroup" aria-label={label}>
	{#each options as option (option.value)}
		<button
			type="button"
			role="radio"
			aria-checked={option.value === value}
			class="shrink-0 rounded-full px-4 py-1.5 whitespace-nowrap transition-colors {option.value === value ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white'}"
			onclick={() => {
				value = option.value;
				onchange?.(option.value);
			}}
		>{option.label}</button>
	{/each}
</div>
