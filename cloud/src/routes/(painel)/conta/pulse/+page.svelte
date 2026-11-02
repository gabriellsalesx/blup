<script lang="ts">
 import {onMount} from "svelte";
 import {t} from "#lib/i18n.svelte.ts";
 // Fixed destination after the ordinary Cloud sign-in; no external URL input.
 onMount(()=>window.location.replace("https://pulse.getblup.io/"));
</script>
<p>{t("loading")}</p><a href="https://pulse.getblup.io/">Blup Pulse</a>
