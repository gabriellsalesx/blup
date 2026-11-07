<script lang="ts">
 import {onMount} from "svelte";
 import {t} from "#lib/i18n.svelte.ts";
 import {appURLs} from "#lib/app-urls.ts";
 // Fixed destination after the ordinary Cloud sign-in; no external URL input.
 onMount(()=>window.location.replace(appURLs.pulse));
</script>
<p>{t("loading")}</p><a href={appURLs.pulse}>Blup Pulse</a>
