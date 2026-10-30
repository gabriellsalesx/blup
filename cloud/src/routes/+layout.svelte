<script lang="ts">
	import "../app.css";
	import Toaster from "#lib/ui/Toaster.svelte";
	import ConfirmHost from "#lib/ui/ConfirmHost.svelte";

	let { children } = $props();
</script>

{@render children()}
<Toaster />
<ConfirmHost />
