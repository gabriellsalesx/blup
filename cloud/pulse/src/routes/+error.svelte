<script lang="ts">
 import {resolve} from "$app/paths";
 import {page} from "$app/state";
 import {t} from "#lib/i18n.svelte.ts";
 import ErrorPage from "#lib/ui/ErrorPage.svelte";
 import {cloudURL} from "../lib/api";
 import {cloudPaths,signInFor} from "../../../shared/panel-paths";
 // Signing in happens on the Cloud, which sends people back here after.
 const signedOut=$derived(page.status===401);
</script>
<ErrorPage status={page.status} app="Blup Pulse" mint home={signedOut?`${cloudURL}${signInFor(cloudPaths.pulse)}`:resolve("/")} homeLabel={signedOut?t("signIn"):t("pulseBackToProjects")} notFoundBody="pulseNotFoundBody"/>
