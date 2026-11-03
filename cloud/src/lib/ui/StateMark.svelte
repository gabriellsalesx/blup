<script lang="ts">
	import Check from "@lucide/svelte/icons/check";
	import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
	import X from "@lucide/svelte/icons/x";
	import Wrench from "@lucide/svelte/icons/wrench";

	// The state of an item as a colored circle with its Lucide mark, the same
	// as the public status page: resolved or up (green check), degraded
	// (amber alert), down (red x), maintenance (indigo wrench) and a past
	// maintenance (gray wrench). `label` is what a screen reader says.
	type Tone = "up" | "partial" | "down" | "maint" | "past";
	let { tone, label }: { tone: Tone; label: string } = $props();
	const look = {
		up: { Icon: Check, bg: "bg-emerald-500" },
		partial: { Icon: TriangleAlert, bg: "bg-amber-500" },
		down: { Icon: X, bg: "bg-red-500" },
		maint: { Icon: Wrench, bg: "bg-indigo-500" },
		past: { Icon: Wrench, bg: "bg-neutral-400 dark:bg-neutral-600" },
	} as const;
	const mark = $derived(look[tone]);
</script>

<span class="grid size-6 shrink-0 place-items-center rounded-full text-white {mark.bg}" role="img" aria-label={label}><mark.Icon class="size-3.5" strokeWidth={3} /></span>
