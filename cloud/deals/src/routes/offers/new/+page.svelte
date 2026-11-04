<script lang="ts">
 import {resolve} from "$app/paths";
 import {t} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Soon from "../../../lib/Soon.svelte";
</script>

<div class="mb-4"><BackLink href={resolve("/")} label={t("dealsOffers")}/></div>
<PageHeader title={t("dealsPasteTitle")}/>
<Soon lead={t("dealsPasteLead")}/>
