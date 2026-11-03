<script lang="ts">
	import ChevronsUpDown from "@lucide/svelte/icons/chevrons-up-down";
	import ChevronsDownUp from "@lucide/svelte/icons/chevrons-down-up";
	import Check from "@lucide/svelte/icons/check";
	import { t } from "#lib/i18n.svelte.ts";
	import BlupLogo from "./BlupLogo.svelte";

	// The logo of the account and Pulse panels: a click opens both panels
	// and picks the other one, like switching organizations.
	type App = "cloud" | "pulse";
	let { current, links }: { current: App; links: Record<App, string> } = $props();
	const id = $props.id();
	let open = $state(false);
	const apps = $derived([
		{ id: "cloud" as const, tag: "Cloud", hint: t("appCloudHint"), mint: false, color: "text-[#3b82f6]" },
		{ id: "pulse" as const, tag: "Pulse", hint: t("appPulseHint"), mint: true, color: "text-[#16846b]" },
	]);
	const here = $derived(apps.find((a) => a.id === current) ?? apps[0]);
</script>

<button
	type="button"
	popovertarget="{id}-apps"
	aria-haspopup="menu"
	aria-expanded={open}
	aria-label={t("appSwitch")}
	style:anchor-name="--apps-{id}"
	class="-mx-2 -my-1.5 flex items-center gap-3 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-neutral-100 dark:hover:bg-neutral-900 {open ? 'bg-neutral-100 dark:bg-neutral-900' : ''}"
>
	<span class="inline-flex flex-col items-start gap-1">
		<BlupLogo mint={here.mint} class="h-8 w-auto text-[#0e1b45] dark:text-white" />
		<span class="text-xs font-bold tracking-wider uppercase {here.color}">{here.tag}</span>
	</span>
	{#if open}<ChevronsDownUp class="size-4 shrink-0 text-neutral-400" />{:else}<ChevronsUpDown class="size-4 shrink-0 text-neutral-400" />{/if}
</button>
<div
	id="{id}-apps"
	popover
	role="menu"
	aria-label={t("appSwitch")}
	style:position-anchor="--apps-{id}"
	class="apps rounded-xl border border-neutral-200 bg-white p-1.5 shadow-lg dark:border-neutral-800 dark:bg-neutral-900 dark:text-white"
	ontoggle={(event) => (open = event.newState === "open")}
>
	{#each apps as app (app.id)}
		<a
			role="menuitem"
			href={links[app.id]}
			aria-current={app.id === current ? "page" : undefined}
			class="flex items-center gap-3 rounded-lg px-3 py-2.5 hover:bg-neutral-100 dark:hover:bg-neutral-800"
		>
			<span class="inline-flex w-20 shrink-0 flex-col items-start gap-0.5">
				<BlupLogo mint={app.mint} class="h-5 w-auto text-[#0e1b45] dark:text-white" />
				<span class="text-[11px] font-bold tracking-wider uppercase {app.color}">{app.tag}</span>
			</span>
			<span class="min-w-0 flex-1 text-sm text-neutral-500">{app.hint}</span>
			{#if app.id === current}<Check class="size-4 shrink-0 {app.color}" />{/if}
		</a>
	{/each}
</div>

<style>
	.apps {
		position: fixed;
		inset: auto;
		top: anchor(bottom);
		left: anchor(left);
		width: 18rem;
		margin-top: 0.5rem;
		position-try-fallbacks: flip-block;
	}
</style>
