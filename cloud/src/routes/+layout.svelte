<script lang="ts">
	import type { LayoutProps } from "./$types";
	import "../app.css";
	import {afterNavigate} from "$app/navigation";
	import {page} from "$app/state";
	import {i18n,intlLocale} from "#lib/i18n.svelte.ts";
	import Toaster from "#lib/ui/Toaster.svelte";
	import ConfirmHost from "#lib/ui/ConfirmHost.svelte";

	let { children }: LayoutProps = $props();
	const isStaff=(url:Readonly<Pick<URL,"hostname"|"pathname">>)=>url.hostname.startsWith("staff.")||url.pathname==="/staff"||url.pathname.startsWith("/staff/");
	i18n.staff=isStaff(page.url);
	afterNavigate(({to})=>{i18n.staff=isStaff(to?.url??page.url);});
	const documentLanguage=$derived(intlLocale());
	$effect(()=>{document.documentElement.lang=documentLanguage;});
</script>

{@render children()}
<Toaster />
<ConfirmHost />
