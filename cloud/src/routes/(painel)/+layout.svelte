<script lang="ts">
	import { goto } from "$app/navigation";
	import User from "@lucide/svelte/icons/user";
	import CreditCard from "@lucide/svelte/icons/credit-card";
	import RefreshCw from "@lucide/svelte/icons/refresh-cw";
	import Globe from "@lucide/svelte/icons/globe";
	import Power from "@lucide/svelte/icons/power";
	import { authClient } from "#lib/auth.ts";
	import PanelShell from "#lib/ui/PanelShell.svelte";

	// The account panel.
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

<PanelShell {links} home="/conta">
	{#snippet foot()}
		<button class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white" onclick={signOut}><Power class="size-[18px]" />Sair</button>
	{/snippet}
	{@render children()}
</PanelShell>
