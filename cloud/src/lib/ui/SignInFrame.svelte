<script lang="ts">
	import type { Snippet } from "svelte";
	import { MediaQuery } from "svelte/reactivity";
	import Check from "@lucide/svelte/icons/check";
	import { t } from "#lib/i18n.svelte.ts";
	import cloudScene from "#lib/assets/login-blups.svg?raw";
	import pulseScene from "#lib/assets/login-blups-pulse.svg?raw";
	import dealsScene from "#lib/assets/login-blups-deals.svg?raw";
	import BlupLogo from "./BlupLogo.svelte";
	import Scene from "./Scene.svelte";
	import LanguagePicker from "./LanguagePicker.svelte";
	import ThemePicker from "./ThemePicker.svelte";

	// Signing in. On a phone: a band in the app color with the logo, the
	// phrase and the gang, and the form on a sheet that covers its bottom
	// edge. From medium screens on: a card split in two, the pitch on the
	// left. The Cloud panel's is blue, the Pulse panel's green and the Deals
	// panel's coral, each with its own pitch and drawing; the form is the
	// same (one Blup account).
	type App = "cloud" | "pulse" | "deals";
	let { app, children }: { app: App; children: Snippet } = $props();

	const looks = {
		cloud: { brand: "bg-blup", title: "accessTitle", lead: "accessEncrypted", perks: ["accessFree", "accessPrivate", "accessSites"], scene: cloudScene, mint: false, coral: false },
		pulse: { brand: "bg-[#16846b]", title: "pulseAccessTitle", lead: "pulseAccessLead", perks: ["pulseAccessIssues", "pulseAccessUptime", "pulseAccessAlerts"], scene: pulseScene, mint: true, coral: false },
		deals: { brand: "bg-[#c2410c]", title: "dealsAccessTitle", lead: "dealsAccessLead", perks: ["dealsAccessChannels", "dealsAccessIds", "dealsAccessGames"], scene: dealsScene, mint: false, coral: true },
	} as const;
	const look = $derived(looks[app]);
	// Only the band of this screen size is drawn (Tailwind's md, 48rem): the
	// drawing and its animation are the heaviest part of the page.
	const wide = new MediaQuery("min-width: 48rem");
</script>


<div class="relative min-h-dvh {look.brand} md:grid md:place-items-center md:overflow-hidden md:bg-neutral-100 md:p-8 md:dark:bg-neutral-950">
	<!-- Soft lines in the background (large screens). -->
	<svg class="pointer-events-none absolute inset-0 hidden h-full w-full text-neutral-200 md:block dark:text-neutral-900" viewBox="0 0 1440 900" preserveAspectRatio="xMidYMid slice" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
		<path d="M-40 260C180 120 260 420 120 620S-60 880 120 960" />
		<path d="M1480 120C1300 260 1420 480 1260 600S1180 860 1380 960" />
		<path d="M380 -40C460 140 700 60 760 -60" />
	</svg>

	<div class="relative flex min-h-dvh w-full max-w-5xl flex-col md:grid md:min-h-[42rem] md:grid-cols-[minmax(0,26rem)_1fr] md:overflow-hidden md:rounded-3xl md:bg-white md:shadow-sm md:dark:bg-neutral-900">
		{#if !wide.current}
		<!-- Phone: the band in the app color. -->
		<header class="panel relative flex items-end gap-2 overflow-hidden px-6 pt-6 pb-12 text-white md:hidden">
			<div class="min-w-0 flex-1 self-start">
				<BlupLogo mint={look.mint} coral={look.coral} class="h-8 w-auto text-white" />
				<h2 class="mt-6 text-[1.7rem] leading-[1.15] font-extrabold tracking-tight">{t(look.title)}</h2>
			</div>
			<div class="-mr-4 -mb-6 w-[56%] shrink-0"><Scene svg={look.scene} /></div>
		</header>
		{:else}
		<!-- Large screens: the panel in the app color. -->
		<aside class="panel relative hidden flex-col {look.brand} px-10 pt-10 text-white md:flex">
			<BlupLogo mint={look.mint} coral={look.coral} class="h-9 w-auto self-start text-white" />
			<h2 class="mt-8 text-[2.1rem] leading-[1.15] font-extrabold tracking-tight">{t(look.title)}</h2>
			<p class="mt-4 text-lg text-white/85">{t(look.lead)}</p>
			<ul class="mt-6 grid gap-3">
				{#each look.perks as perk (perk)}
					<li class="flex items-center gap-3 font-medium"><span class="grid size-6 place-items-center rounded-full bg-white/20"><Check class="size-3.5" /></span>{t(perk)}</li>
				{/each}
			</ul>
			<div class="-mx-4 mt-auto pt-6"><Scene svg={look.scene} /></div>
		</aside>
		{/if}

		<!-- The form: a sheet on the phone, the right half on large screens. -->
		<main class="relative -mt-6 flex flex-1 flex-col rounded-t-3xl bg-white px-6 pt-10 pb-10 md:mt-0 md:grid md:place-items-center md:rounded-none md:px-16 md:py-12 dark:bg-neutral-900">
			<div class="w-full max-w-[400px]">{@render children()}</div>
		</main>
	</div>

	<div class="absolute top-4 right-4 z-10 flex items-center gap-2"><LanguagePicker /><ThemePicker /></div>
</div>

<style>
	/* A light grain on the color, drawn by the browser (no image file). */
	.panel::before {
		content: "";
		position: absolute;
		inset: 0;
		opacity: 0.12;
		pointer-events: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='.9' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
		mix-blend-mode: overlay;
	}
	.panel > :global(*) { position: relative; }
</style>
