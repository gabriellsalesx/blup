<script lang="ts">
 import {onMount} from "svelte";
 import {t} from "#lib/i18n.svelte.ts";
 import {pulseAppURL} from "../../../../../shared/panel-paths";
 // Fixed destination after the ordinary Cloud sign-in; no external URL input.
 onMount(()=>window.location.replace(pulseAppURL));
</script>
<p>{t("loading")}</p><a href={pulseAppURL}>Blup Pulse</a>
