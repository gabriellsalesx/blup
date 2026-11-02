<script lang="ts">
	// A colored dot of state: up (green), down (red) or unknown/paused (gray).
	let { state }: { state: "up" | "down" | "unknown" } = $props();
	const look = { up: "bg-emerald-500", down: "bg-red-500", unknown: "bg-neutral-400" };
</script>

<span class="inline-block size-2.5 shrink-0 rounded-full {look[state]}" aria-hidden="true"></span>
