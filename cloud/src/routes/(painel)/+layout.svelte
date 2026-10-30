<script lang="ts">
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import User from "@lucide/svelte/icons/user";
	import CreditCard from "@lucide/svelte/icons/credit-card";
	import RefreshCw from "@lucide/svelte/icons/refresh-cw";
	import Globe from "@lucide/svelte/icons/globe";
	import Power from "@lucide/svelte/icons/power";
	import { authClient } from "#lib/auth.ts";
	import ThemePicker from "#lib/ui/ThemePicker.svelte";

	// The account panel: the menu on the left, the page on the right.
	let { children } = $props();

	const links = [
		{ href: "/conta", label: "Perfil", Icon: User },
		{ href: "/conta/plano", label: "Plano", Icon: CreditCard },
		{ href: "/conta/sync", label: "Sync", Icon: RefreshCw },
		{ href: "/conta/sites", label: "Sites", Icon: Globe },
	];

	async function signOut() {
		await authClient.signOut();
		goto("/entrar", { replaceState: true });
	}
</script>

<div class="min-h-dvh md:grid md:grid-cols-[17rem_1fr]">
	<aside class="border-neutral-200 px-6 py-6 md:border-r md:py-10 md:pl-10 dark:border-neutral-800">
		<a href="/conta" class="flex items-center gap-2.5 text-lg font-bold"><img src="/blup.svg" alt="" width="30" height="30" />Blup Cloud</a>
		<nav class="mt-6 flex flex-wrap gap-1 md:mt-8 md:grid">
			{#each links as { href, label, Icon } (href)}
				<a {href} aria-current={page.url.pathname === href ? "page" : undefined} class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium {page.url.pathname === href ? 'text-blup' : 'text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white'}">
					<Icon class="size-[18px]" />{label}
				</a>
			{/each}
		</nav>
		<div class="mt-4 border-t border-neutral-200 pt-4 md:mt-6 md:pt-6 dark:border-neutral-800">
			<button class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white" onclick={signOut}><Power class="size-[18px]" />Sair</button>
			<div class="mt-4 px-2.5"><ThemePicker /></div>
		</div>
	</aside>
	<main class="max-w-3xl px-6 py-8 md:px-16 md:py-12">{@render children()}</main>
</div>
