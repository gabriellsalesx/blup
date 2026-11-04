<script lang="ts">
 import {resolve} from "$app/paths";
 import {t} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Soon from "../../../lib/Soon.svelte";
</script>

<div class="mb-4"><BackLink href={resolve("/hunts")} label={t("dealsHunts")}/></div>
<PageHeader title={t("dealsNewHuntTitle")}/>
<Soon lead={t("dealsNewHuntLead")}/>
