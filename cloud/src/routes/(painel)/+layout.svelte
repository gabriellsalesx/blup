<script lang="ts">
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
	import Power from "@lucide/svelte/icons/power";
	import { authClient } from "#lib/auth.ts";
	import PanelShell from "#lib/ui/PanelShell.svelte";

	// The account panel.
	let { children,data } = $props();
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
		{ href: "/conta", label: t("profile"), Icon: User },
		{ href: "/conta/plano", label: t("plan"), Icon: CreditCard },
		{ href: "/conta/sync", label: t("sync"), Icon: RefreshCw },
		{ href: "/conta/sites", label: t("sites"), Icon: Globe },
	]);

	async function signOut() {
		await authClient.signOut();
		goto("/entrar", { replaceState: true });
	}
</script>

<!-- The plans side by side need the wide page. -->
<PanelShell languages {links} home="/conta" app={{ current: "cloud", links: { cloud: "/conta", pulse: "https://pulse.getblup.io/" } }} wide={page.url.pathname === "/conta/plano"}>
	{#snippet foot()}
		<button class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 font-medium text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white" onclick={signOut}><Power class="size-[18px]" />{t("signOut")}</button>
	{/snippet}
	{@render children()}
</PanelShell>
