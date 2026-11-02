<script lang="ts">
	import { page } from "$app/state";
	import BlupLogo from "#lib/ui/BlupLogo.svelte";
	import Scene from "#lib/ui/Scene.svelte";
	import ThemePicker from "#lib/ui/ThemePicker.svelte";

	// Every error page: a Blup drawing, what happened in plain words and the
	// way out. Without internet the browser says so before any status does.
	const drawings = import.meta.glob<string>("../lib/assets/erros/*.svg", { query: "?raw", import: "default", eager: true });
	const drawing = (name: string) => drawings[`../lib/assets/erros/${name}.svg`] ?? drawings["../lib/assets/erros/generico.svg"] ?? "";

	type Kind = { title: string; text: string; art: string; retry?: boolean };
	const kinds: Record<string, Kind> = {
		offline: { title: "Sem internet", text: "Parece que a conexão caiu. Confira o Wi-Fi e tente de novo.", art: "offline", retry: true },
		400: { title: "Pedido inválido", text: "Algo nesse endereço não está certo. Volte e tente de novo.", art: "generico" },
		401: { title: "Entre para continuar", text: "Essa página é da sua conta. Entre e ela abre.", art: "403" },
		403: { title: "Sem acesso", text: "Você não tem permissão para ver esta página.", art: "403" },
		404: { title: "Página não encontrada", text: "O endereço pode ter mudado ou nunca ter existido.", art: "404" },
		429: { title: "Calma aí", text: "Muitos pedidos em pouco tempo. Espere um minuto e tente de novo.", art: "429", retry: true },
		500: { title: "Algo deu errado", text: "Um erro do nosso lado. Já estamos de olho. Tente de novo em instantes.", art: "500", retry: true },
		503: { title: "Voltamos já", text: "O Blup está em manutenção rápida. Tente de novo em alguns minutos.", art: "500", retry: true },
	};

	const offline = typeof navigator !== "undefined" && !navigator.onLine;
	const k = $derived(offline ? kinds.offline : (kinds[page.status] ?? (page.status >= 500 ? kinds[500] : { title: "Algo deu errado", text: page.error?.message ?? "Tente de novo.", art: "generico", retry: true })));
	const onStaff = typeof location !== "undefined" && location.hostname.startsWith("staff.");
	const home = onStaff ? "/" : page.status === 401 ? "/entrar" : "/conta";
</script>

<svelte:head><title>{k.title} · Blup</title></svelte:head>

<main class="grid min-h-dvh place-items-center px-6 py-16">
	<div class="grid w-full max-w-md justify-items-center text-center">
		<a href={home} aria-label="Blup"><BlupLogo class="h-8 w-auto text-[#0e1b45] dark:text-white" /></a>
		<Scene svg={drawing(k.art)} class="mt-8 w-full max-w-sm" />
		{#if !offline && page.status}<p class="mt-6 text-sm font-bold tracking-wider text-blup">ERRO {page.status}</p>{/if}
		<h1 class="mt-2 text-3xl font-extrabold tracking-tight">{k.title}</h1>
		<p class="mt-2 text-neutral-500">{k.text}</p>
		<div class="mt-8 flex flex-wrap justify-center gap-3">
			{#if k.retry}
				<button class="flex h-12 items-center rounded-full bg-neutral-900 px-6 font-semibold text-white hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200" onclick={() => location.reload()}>Tentar de novo</button>
			{/if}
			<a class="flex h-12 items-center rounded-full border border-neutral-300 px-6 font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900" href={home}>{page.status === 401 ? "Entrar" : "Voltar ao início"}</a>
		</div>
	</div>
</main>
<div class="fixed top-4 right-4"><ThemePicker /></div>
