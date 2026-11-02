<script lang="ts">
 import {page} from "$app/state";
 import {t} from "#lib/i18n.svelte.ts";
</script>
<svelte:head><title>Blup Pulse</title></svelte:head>
<main class="mx-auto max-w-xl p-10"><h1 class="text-3xl font-bold">{page.status}</h1><p class="my-5">{t("errorGeneric")}</p><a class="text-blup underline" href="/">{t("pulseProjects")}</a></main>
