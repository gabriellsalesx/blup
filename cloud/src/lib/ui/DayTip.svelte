<script module lang="ts">
	export type DayState = "up" | "partial" | "down" | "maint" | "none";
</script>

<script lang="ts">
	import Circle from "@lucide/svelte/icons/circle";
	import StateMark from "./StateMark.svelte";

	// What a day's card says: the state's mark (our colored circles, or the
	// empty one without data) with its name, how it went when not all up,
	// and the date under a line.
	let { state, name, detail = "", date }: { state: DayState; name: string; detail?: string; date: string } = $props();
</script>

<span class="flex items-center gap-2 font-semibold">{#if state === "none"}<Circle class="size-6 shrink-0 text-neutral-400" strokeWidth={2} aria-hidden="true" />{:else}<StateMark tone={state} label={name} />{/if}<span class="min-w-0 break-words">{name}</span></span>
{#if detail}<span class="mt-1.5 block font-normal text-neutral-500">{detail}</span>{/if}
<span class="mt-2.5 block border-t border-neutral-200 pt-2 text-center text-[13px] font-normal text-neutral-500 dark:border-neutral-800">{date}</span>
