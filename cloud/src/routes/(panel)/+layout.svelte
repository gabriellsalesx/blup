<script lang="ts">
	import type { LayoutProps } from "./$types";
	import { resolve } from "$app/paths";
 import {i18n,t} from "#lib/i18n.svelte.ts";
 import {api} from "#lib/api.ts";
 import {onDestroy} from "svelte";
 import {createLanguageSaver} from "#lib/language-save.ts";
 import {toast} from "#lib/toast.svelte.ts";
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import User from "@lucide/svelte/icons/user";
	import CreditCard from "@lucide/svelte/icons/credit-card";
	import RefreshCw from "@lucide/svelte/icons/refresh-cw";
	import Globe from "@lucide/svelte/icons/globe";
	import { authClient } from "#lib/auth.ts";
	import PanelShell from "#lib/ui/PanelShell.svelte";
	import { pulseAppURL } from "../../../shared/panel-paths";
	import SignOutButton from "#lib/ui/SignOutButton.svelte";

	// The account panel.
	let { children,data }: LayoutProps = $props();
 let alive=true;
 onDestroy(()=>{alive=false;});
 const saver=createLanguageSaver(language=>api.setLanguage(language),id=>alive&&data.me.id===id,()=>toast(()=>t("languageSaveError"),{kind:"error"}));
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

	async function signOut() {
		await authClient.signOut();
		goto(resolve("/(auth)/sign-in"), { replace: true });
	}
</script>

<!-- The plans side by side need the wide page. -->
<PanelShell languages {links} home={resolve("/(panel)/account")} app={{ current: "cloud", links: { cloud: resolve("/(panel)/account"), pulse: pulseAppURL } }} wide={page.url.pathname === resolve("/(panel)/account/plan")}>
	{#snippet foot()}
		<SignOutButton onclick={signOut} />
	{/snippet}
	{@render children()}
</PanelShell>
