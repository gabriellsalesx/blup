<script lang="ts">
 import {resolve} from "$app/paths";
 import {page} from "$app/state";
 import {t} from "#lib/i18n.svelte.ts";
 import ErrorPage from "#lib/ui/ErrorPage.svelte";
 const signedOut=$derived(page.status===401);
</script>
<ErrorPage status={page.status} app="Blup Pulse" mint home={signedOut?resolve("/sign-in"):resolve("/")} homeLabel={signedOut?t("signIn"):t("pulseBackToProjects")} notFoundBody="pulseNotFoundBody"/>
