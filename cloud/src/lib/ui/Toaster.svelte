<script lang="ts">
	import Check from "@lucide/svelte/icons/circle-check";
	import Alert from "@lucide/svelte/icons/circle-alert";
	import Info from "@lucide/svelte/icons/info";
	import X from "@lucide/svelte/icons/x";
	import { toasts, dismiss } from "#lib/toast.svelte.ts";
</script>

<div class="pointer-events-none fixed inset-x-0 bottom-5 z-50 grid justify-items-center gap-2" aria-live="polite">
	{#each toasts as n (n.id)}
		<div
			class="toast pointer-events-auto flex max-w-[min(34rem,calc(100%-2rem))] items-center gap-2.5 rounded-xl border border-neutral-200 bg-white py-2.5 pr-2.5 pl-3.5 text-sm shadow-[0_14px_40px_rgb(0_0_0/0.18)] dark:border-neutral-800 dark:bg-neutral-900"
			role={n.kind === "error" ? "alert" : "status"}
		>
			<span class="grid shrink-0 {n.kind === 'ok' ? 'text-green-600' : n.kind === 'error' ? 'text-red-600' : 'text-blup'}">
				{#if n.kind === "ok"}<Check class="size-[17px]" />{:else if n.kind === "error"}<Alert class="size-[17px]" />{:else}<Info class="size-[17px]" />{/if}
			</span>
			<span class="flex-1">{n.text}</span>
			{#if n.action}
				<button class="shrink-0 rounded-md px-2.5 py-1 font-bold text-blup hover:bg-neutral-100 dark:hover:bg-neutral-800" onclick={() => { n.action!.run(); dismiss(n.id); }}>{n.action.label}</button>
			{/if}
			<button class="grid shrink-0 place-items-center rounded-md p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white" title="Fechar" aria-label="Fechar" onclick={() => dismiss(n.id)}>
				<X class="size-3.5" />
			</button>
		</div>
	{/each}
</div>

<style>
	.toast { animation: up 0.18s ease-out; }
	@keyframes up { from { opacity: 0; transform: translateY(0.5rem); } }
</style>
