<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
</script>

<PageHeader title={t("dealsAffiliates")} lead={t("dealsAffiliatesLead")}/>

<div class="mt-8"><List empty={t("dealsAffiliatesEmpty")}/></div>
