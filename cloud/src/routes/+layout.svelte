<script lang="ts">
	import "../app.css";
	import Toaster from "#lib/ui/Toaster.svelte";

	let { children } = $props();
</script>

{@render children()}
<Toaster />
