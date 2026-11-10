<script lang="ts">
	import type { LayoutProps } from "./$types";
	import { resolve } from "$app/paths";
	import Users from "@lucide/svelte/icons/users";
	import UserCog from "@lucide/svelte/icons/user-cog";
	import ScrollText from "@lucide/svelte/icons/scroll-text";
	import Sparkles from "@lucide/svelte/icons/sparkles";
	import Ticket from "@lucide/svelte/icons/ticket";
	import PanelShell from "#lib/ui/PanelShell.svelte";
	import { roleLabel, staffHref } from "#lib/staff.ts";

	// The staff area: the same frame as the account panel.
	let { data, children }: LayoutProps = $props();

	// An account's page, any account (a route's parameter can't be empty:
	// the address of one, without its id).
	const accounts = staffHref(resolve("/staff/accounts/[id]", { id: "_" })).slice(0, -1);
	const links = $derived([
		{ href: staffHref(resolve("/staff")), label: "Contas", Icon: Users, match: (p: string) => p === staffHref(resolve("/staff")) || p.startsWith(accounts) },
		// AppSumo's codes: the whole staff sees the counts.
		{ href: staffHref(resolve("/staff/appsumo")), label: "AppSumo", Icon: Ticket },
		...(data.me?.role === "admin" ? [{ href: staffHref(resolve("/staff/team")), label: "Equipe", Icon: UserCog }] : []),
		// The AI settings: the owner (STAFF_OWNER_EMAIL) only.
		...(data.me?.owner ? [{ href: staffHref(resolve("/staff/ai")), label: "IA", Icon: Sparkles, match: (p: string) => p.startsWith(staffHref(resolve("/staff/ai"))) }] : []),
		...(data.me?.role === "admin" ? [{ href: staffHref(resolve("/staff/log")), label: "Registro", Icon: ScrollText }] : []),
	]);
</script>

{#if data.me}
	{@const me = data.me}
	<PanelShell {links} home={staffHref(resolve("/staff"))} tag="Staff" wide>
		{#snippet foot()}
			<p class="px-2.5 text-sm text-neutral-500"><span class="block truncate font-medium text-neutral-700 dark:text-neutral-300" title={me.email}>{me.email}</span>{roleLabel(me.role)}</p>
		{/snippet}
		{@render children()}
	</PanelShell>
{/if}
