<script lang="ts">
	// A switch that turns something on and off (role=switch), with its label
	// beside it. `tone` is the color when on: the Blup blue, the Pulse mint
	// or the Deals coral (#c2410c: white text on it passes AA).
	let { label, checked = $bindable(false), tone = "blup", disabled = false, onchange }: {
		label: string;
		checked?: boolean;
		tone?: "blup" | "mint" | "coral";
		disabled?: boolean;
		onchange?: (checked: boolean) => void;
	} = $props();
	const on = { blup: "bg-blup", mint: "bg-emerald-600", coral: "bg-[#c2410c]" };
</script>

<button
	type="button"
	role="switch"
	aria-checked={checked}
	{disabled}
	class="inline-flex items-center gap-2.5 text-sm font-semibold disabled:opacity-50"
	onclick={() => {
		checked = !checked;
		onchange?.(checked);
	}}
>
	<span class="relative inline-flex h-6 w-11 shrink-0 rounded-full transition-colors {checked ? on[tone] : 'bg-neutral-300 dark:bg-neutral-700'}">
		<span class="absolute top-0.5 left-0.5 size-5 rounded-full bg-white shadow transition-transform {checked ? 'translate-x-5' : ''}"></span>
	</span>
	{label}
</button>
