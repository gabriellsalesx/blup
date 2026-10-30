<script lang="ts">
	import ThemePicker from "#lib/ui/ThemePicker.svelte";

	let { children } = $props();
</script>

<!-- Minimal: only the Blup above the content, no bar on top. -->
<main class="grid min-h-dvh place-items-center px-4 py-16">
	<div class="grid w-full justify-items-center gap-8">
		<img src="/blup.svg" alt="Blup" width="64" height="64" />
		{@render children()}
	</div>
</main>
<div class="fixed top-4 right-4"><ThemePicker /></div>
