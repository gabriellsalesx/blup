<script lang="ts">
	import { page } from "$app/state";
	import ShieldX from "@lucide/svelte/icons/shield-x";
	import ThemePicker from "#lib/ui/ThemePicker.svelte";
	import { roleLabel } from "#lib/staff.ts";

	let { data, children } = $props();

	const links = $derived([
		{ href: "/staff", label: "Contas" },
		...(data.me?.role === "admin" ? [{ href: "/staff/equipe", label: "Equipe" }, { href: "/staff/registro", label: "Registro" }] : []),
	]);
	const here = (href: string) => (href === "/staff" ? page.url.pathname === "/staff" || page.url.pathname.startsWith("/staff/contas") : page.url.pathname.startsWith(href));
</script>

{#if data.denied}
	<main class="grid min-h-dvh place-items-center px-4">
		<div class="grid justify-items-center gap-3 text-center">
			<ShieldX class="size-10 text-red-600" />
			<h1 class="text-xl font-bold">Acesso negado</h1>
			<p class="max-w-sm text-sm text-neutral-500">Esta área é só da equipe do Blup. Entre pelo Cloudflare Access com um e-mail da equipe.</p>
		</div>
	</main>
{:else if data.me}
	<div class="min-h-dvh">
		<header class="flex items-center gap-6 border-b border-neutral-200 px-6 py-3 dark:border-neutral-800">
			<a href="/staff" class="flex items-center gap-2 font-bold"><img src="/blup.svg" alt="" width="26" height="26" />Staff</a>
			<nav class="flex gap-1 text-sm font-semibold">
				{#each links as l (l.href)}
					<a href={l.href} class="rounded-md px-3 py-1.5 {here(l.href) ? 'bg-neutral-100 text-neutral-900 dark:bg-neutral-800 dark:text-white' : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'}">{l.label}</a>
				{/each}
			</nav>
			<span class="flex-1"></span>
			<span class="text-sm text-neutral-500">{data.me.email} · {roleLabel(data.me.role)}</span>
			<ThemePicker />
		</header>
		<main class="mx-auto max-w-5xl px-6 py-8">{@render children()}</main>
	</div>
{/if}
