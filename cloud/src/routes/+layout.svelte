<script lang="ts">
	import "../app.css";

	let { children } = $props();
</script>

<div class="min-h-dvh flex flex-col">
	<header class="flex items-center gap-2.5 px-6 py-5">
		<img src="/blup.svg" alt="" width="32" height="32" />
		<span class="font-bold tracking-tight">Blup Cloud</span>
	</header>
	<main class="flex-1 grid place-items-start justify-center px-4 pt-8 pb-16">
		{@render children()}
	</main>
</div>
