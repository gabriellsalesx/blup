<script lang="ts">
	import type { LayoutProps } from "./$types";
	import SignInFrame from "#lib/ui/SignInFrame.svelte";
	let { children }: LayoutProps = $props();
</script>

<SignInFrame app="cloud">{@render children()}</SignInFrame>
