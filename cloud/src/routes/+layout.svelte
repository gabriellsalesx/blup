<script lang="ts">
	import "../app.css";
	import Toaster from "#lib/ui/Toaster.svelte";
	import ThemePicker from "#lib/ui/ThemePicker.svelte";

	let { children } = $props();
</script>

<!-- Minimal: only the Blup above the content, no bar on top. -->
<main class="min-h-dvh grid place-items-center px-4 py-16">
	<div class="grid w-full justify-items-center gap-8">
		<img src="/blup.svg" alt="Blup" width="72" height="72" />
		{@render children()}
	</div>
</main>
<div class="fixed top-4 right-4"><ThemePicker /></div>
<Toaster />
