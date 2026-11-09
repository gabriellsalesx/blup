<script lang="ts">
	import type { LayoutProps } from "./$types";
	import { resolve } from "$app/paths";
 import {i18n,t} from "#lib/i18n.svelte.ts";
 import {api} from "#lib/api.ts";
 import {onDestroy,onMount} from "svelte";
 import {createLanguageSaver} from "#lib/language-save.ts";
 import {refreshOnReturn} from "#lib/refresh-on-return.ts";
 import {refreshAll} from "$app/navigation";
 import type {RouteId} from "$app/types";
 import {toast} from "#lib/toast.svelte.ts";
	import { page } from "$app/state";
	import User from "@lucide/svelte/icons/user";
	import CreditCard from "@lucide/svelte/icons/credit-card";
	import RefreshCw from "@lucide/svelte/icons/refresh-cw";
	import Globe from "@lucide/svelte/icons/globe";
	import { authClient } from "#lib/auth.ts";
	import PanelShell from "#lib/ui/PanelShell.svelte";
	import { appURLs } from "#lib/app-urls.ts";
	import SignOutButton from "#lib/ui/SignOutButton.svelte";

	// The account panel.
	let { children,data }: LayoutProps = $props();
 let alive=true;
 onDestroy(()=>{alive=false;});
 const saver=createLanguageSaver(language=>api.setLanguage(language),id=>alive&&data.me.id===id,()=>toast(()=>t("languageSaveError"),{kind:"error"}));
 // Back to the tab after a while: the account, its plan, storage, devices
 // and sites load again (not the plans page, whose choices would reset).
 const fresh:ReadonlySet<string>=new Set(["/(panel)/account","/(panel)/account/sync","/(panel)/account/sites"] satisfies RouteId[]);
 onMount(()=>refreshOnReturn(refreshAll,()=>fresh.has(page.route.id??"")));
 // Synchronize an external account preference. The saver only changes
 // its non-reactive queue; it does not assign Svelte state in this effect.
 $effect(()=>{
  // Retry a previously failed save when the same language is selected again.
  void i18n.revision;
  void saver.schedule(data.me.id,i18n.locale,data.me.language);
 });

	const links = $derived([
		{ href: resolve("/(panel)/account"), label: t("profile"), Icon: User },
		{ href: resolve("/(panel)/account/plan"), label: t("plan"), Icon: CreditCard },
		{ href: resolve("/(panel)/account/sync"), label: t("sync"), Icon: RefreshCw },
		{ href: resolve("/(panel)/account/sites"), label: t("sites"), Icon: Globe },
	]);

	// Signing out ends the session, then opens the sign-in with a full load,
	// so nothing of the account stays in memory (the same as Pulse and Deals).
	async function signOut() {
		try { await authClient.signOut(); }
		finally { location.replace(resolve("/(auth)/sign-in")); }
	}
</script>

<!-- The plans side by side need the wide page. -->
<PanelShell languages {links} home={resolve("/(panel)/account")} app={{ current: "cloud", links: { cloud: resolve("/(panel)/account"), pulse: appURLs.pulse, deals: appURLs.deals } }} wide={page.url.pathname === resolve("/(panel)/account/plan")}>
	{#snippet foot(compact)}
		<SignOutButton onclick={signOut} {compact} />
	{/snippet}
	{@render children()}
</PanelShell>
