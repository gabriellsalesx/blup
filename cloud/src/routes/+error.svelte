<script lang="ts">
 import {resolve} from "$app/paths";
 import {page} from "$app/state";
 import {t} from "#lib/i18n.svelte.ts";
 import {staffHref} from "#lib/staff.ts";
 import ErrorPage from "#lib/ui/ErrorPage.svelte";
 const onStaff=typeof location!=="undefined"&&location.hostname.startsWith("staff.");
 const home=onStaff?staffHref(resolve("/staff")):page.status===401?resolve("/(auth)/sign-in"):resolve("/(panel)/account");
</script>
<ErrorPage status={page.status} {home} homeLabel={page.status===401?t("signIn"):t("backAccount")} languages={!onStaff}/>
