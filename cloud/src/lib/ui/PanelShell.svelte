<script lang="ts">
	import type { Component, Snippet } from "svelte";
	import { page } from "$app/state";
	import BlupLogo from "./BlupLogo.svelte";
	import AppSwitcher from "./AppSwitcher.svelte";
	import LanguagePicker from "./LanguagePicker.svelte";
	import ThemePicker from "./ThemePicker.svelte";

	// The panels' frame (the account's, Pulse's and the staff's). On a
	// computer: the Blup logo and the menu on the left, the page on the
	// right. On a phone, as an app: a bar on top with the logo, language,
	// theme and signing out, and the menu as tabs at the bottom (`short` is a
	// tab's shorter name), both clear of the screen's notch and home bar
	// (safe areas). `tag` names the area under the logo; `foot` goes under
	// the menu (signing out; `compact` asks for its icon alone). With `app`
	// the logo switches between the account and Pulse panels.
	type Link = { href: string; label: string; short?: string; Icon: Component; match?: (path: string) => boolean };
	let { links, home, tag, foot, app, wide = false, languages=false, mint=false, children }: { links: Link[]; home: string; tag?: string; foot?: Snippet<[compact: boolean]>; app?: { current: "cloud" | "pulse"; links: Record<"cloud" | "pulse", string> }; wide?: boolean; languages?:boolean;mint?:boolean; children: Snippet } = $props();

	const here = (l: Link) => (l.match ? l.match(page.url.pathname) : page.url.pathname === l.href);
</script>

{#snippet logo()}
	{#if app}<AppSwitcher current={app.current} links={app.links} />
	{:else}<a href={home} class="inline-flex flex-col items-start gap-1" aria-label="Blup">
		<BlupLogo {mint} class="h-8 w-auto text-[#0e1b45] dark:text-white" />
		{#if tag}<span class="text-xs font-bold tracking-wider text-blup uppercase">{tag}</span>{/if}
	</a>{/if}
{/snippet}

<div class="min-h-dvh md:grid md:grid-cols-[17rem_1fr]">
	<header class="sticky top-0 z-40 border-b border-neutral-200 bg-neutral-50/90 pt-[env(safe-area-inset-top)] pr-[max(1rem,env(safe-area-inset-right))] pl-[max(1rem,env(safe-area-inset-left))] backdrop-blur-md md:hidden dark:border-neutral-800 dark:bg-neutral-950/90">
		<div class="flex min-h-16 items-center justify-between gap-3 py-2">
			<div class="min-w-0">{@render logo()}</div>
			<div class="flex shrink-0 items-center gap-1.5">{#if languages}<LanguagePicker />{/if}<ThemePicker />{@render foot?.(true)}</div>
		</div>
	</header>
	<aside class="hidden border-neutral-200 py-10 pr-6 pl-10 md:block md:border-r dark:border-neutral-800">
		{@render logo()}
		<nav class="mt-8 grid gap-1">
			{#each links as l (l.href)}
				<a href={l.href} aria-current={here(l) ? "page" : undefined} class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium {here(l) ? 'text-blup' : 'text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white'}">
					<l.Icon class="size-[18px]" />{l.label}
				</a>
			{/each}
		</nav>
		<div class="mt-6 border-t border-neutral-200 pt-6 dark:border-neutral-800">
			{@render foot?.(false)}
			<div class="mt-4 flex items-center gap-2 px-2.5">{#if languages}<LanguagePicker />{/if}<ThemePicker /></div>
		</div>
	</aside>
	<main class="px-[max(1.25rem,env(safe-area-inset-left))] pt-6 pb-[calc(6rem+env(safe-area-inset-bottom))] md:px-16 md:py-12 {wide ? 'max-w-6xl' : 'max-w-3xl'}">{@render children()}</main>
	<nav class="fixed inset-x-0 bottom-0 z-40 border-t border-neutral-200 bg-neutral-50/95 pr-[env(safe-area-inset-right)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)] backdrop-blur-md md:hidden dark:border-neutral-800 dark:bg-neutral-950/95" aria-label="Menu">
		<div class="grid" style:grid-template-columns="repeat({links.length},minmax(0,1fr))">
			{#each links as l (l.href)}
				{@const on = here(l)}
				<a href={l.href} aria-current={on ? "page" : undefined} class="flex min-h-14 flex-col items-center justify-center gap-1 px-1 py-2 text-[11px] font-medium {on ? 'text-blup' : 'text-neutral-500 active:text-neutral-900 dark:active:text-white'}">
					<span class="grid h-7 w-12 place-items-center rounded-full transition-colors {on ? 'bg-blup/15' : ''}"><l.Icon class="size-5" /></span>
					<span class="max-w-full truncate">{l.short ?? l.label}</span>
				</a>
			{/each}
		</div>
	</nav>
</div>
