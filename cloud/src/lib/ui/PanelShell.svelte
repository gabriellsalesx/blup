<script lang="ts">
	import type { Component, Snippet } from "svelte";
	import { page } from "$app/state";
	import BlupLogo from "./BlupLogo.svelte";
	import ThemePicker from "./ThemePicker.svelte";

	// The panels' frame (the account's and the staff's): the Blup logo and
	// the menu on the left, the page on the right. `tag` names the area under
	// the logo; `foot` goes under the menu (signing out, who is signed in).
	type Link = { href: string; label: string; Icon: Component; match?: (path: string) => boolean };
	let { links, home, tag, foot, wide = false, children }: { links: Link[]; home: string; tag?: string; foot?: Snippet; wide?: boolean; children: Snippet } = $props();

	const here = (l: Link) => (l.match ? l.match(page.url.pathname) : page.url.pathname === l.href);
</script>

<div class="min-h-dvh md:grid md:grid-cols-[17rem_1fr]">
	<aside class="border-neutral-200 px-6 py-6 md:border-r md:py-10 md:pl-10 dark:border-neutral-800">
		<a href={home} class="inline-flex flex-col items-start gap-1" aria-label="Blup">
			<BlupLogo class="h-8 w-auto text-[#0e1b45] dark:text-white" />
			{#if tag}<span class="text-xs font-bold tracking-wider text-blup uppercase">{tag}</span>{/if}
		</a>
		<nav class="mt-6 flex flex-wrap gap-1 md:mt-8 md:grid">
			{#each links as l (l.href)}
				<a href={l.href} aria-current={here(l) ? "page" : undefined} class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium {here(l) ? 'text-blup' : 'text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white'}">
					<l.Icon class="size-[18px]" />{l.label}
				</a>
			{/each}
		</nav>
		<div class="mt-4 border-t border-neutral-200 pt-4 md:mt-6 md:pt-6 dark:border-neutral-800">
			{@render foot?.()}
			<div class="mt-4 px-2.5"><ThemePicker /></div>
		</div>
	</aside>
	<main class="px-6 py-8 md:px-16 md:py-12 {wide ? 'max-w-6xl' : 'max-w-3xl'}">{@render children()}</main>
</div>
