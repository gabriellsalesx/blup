<script lang="ts">
	import ShieldX from "@lucide/svelte/icons/shield-x";
	import Users from "@lucide/svelte/icons/users";
	import UserCog from "@lucide/svelte/icons/user-cog";
	import ScrollText from "@lucide/svelte/icons/scroll-text";
	import PanelShell from "#lib/ui/PanelShell.svelte";
	import { roleLabel, staffHref } from "#lib/staff.ts";

	// The staff area: the same frame as the account panel.
	let { data, children } = $props();

	const links = $derived([
		{ href: staffHref("/staff"), label: "Contas", Icon: Users, match: (p: string) => p === staffHref("/staff") || p.startsWith(staffHref("/staff/contas")) },
		...(data.me?.role === "admin"
			? [
					{ href: staffHref("/staff/equipe"), label: "Equipe", Icon: UserCog },
					{ href: staffHref("/staff/registro"), label: "Registro", Icon: ScrollText },
				]
			: []),
	]);
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
	<PanelShell {links} home={staffHref("/staff")} tag="Staff" wide>
		{#snippet foot()}
			<p class="px-2.5 text-sm text-neutral-500"><span class="block truncate font-medium text-neutral-700 dark:text-neutral-300" title={data.me!.email}>{data.me!.email}</span>{roleLabel(data.me!.role)}</p>
		{/snippet}
		{@render children()}
	</PanelShell>
{/if}
