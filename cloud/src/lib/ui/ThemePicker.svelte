<script lang="ts">
	import Sun from "@lucide/svelte/icons/sun";
	import Moon from "@lucide/svelte/icons/moon";
	import Monitor from "@lucide/svelte/icons/monitor";
	import { theme, setTheme, type Theme } from "#lib/theme.svelte.ts";

	const options: { key: Theme; label: string; Icon: typeof Sun }[] = [
		{ key: "light", label: "Claro", Icon: Sun },
		{ key: "dark", label: "Escuro", Icon: Moon },
		{ key: "system", label: "Do sistema", Icon: Monitor },
	];
</script>

<div class="inline-flex gap-0.5 rounded-lg border border-neutral-200 bg-white p-0.5 dark:border-neutral-800 dark:bg-neutral-900" role="radiogroup" aria-label="Tema">
	{#each options as { key, label, Icon } (key)}
		<button
			role="radio"
			aria-checked={theme.value === key}
			title={label}
			aria-label={label}
			class="rounded-md p-1.5 {theme.value === key ? 'bg-neutral-100 text-neutral-900 dark:bg-neutral-800 dark:text-white' : 'text-neutral-400 hover:text-neutral-900 dark:hover:text-white'}"
			onclick={() => setTheme(key)}
		>
			<Icon class="size-3.5" />
		</button>
	{/each}
</div>
