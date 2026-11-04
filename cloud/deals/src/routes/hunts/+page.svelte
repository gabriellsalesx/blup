<script lang="ts">
 import {goto} from "$app/navigation";
 import {resolve} from "$app/paths";
 import Plus from "@lucide/svelte/icons/plus";
 import Button from "#lib/ui/Button.svelte";
 import {hasPlan} from "../../lib/plan";
 import {t} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
</script>

<PageHeader title={t("dealsHunts")} lead={t("dealsHuntsLead")}>
 {#snippet aside()}{#if hasPlan()}<Button onclick={()=>goto(resolve("/hunts/new"))}><Plus class="size-4"/>{t("dealsNewHunt")}</Button>{/if}{/snippet}
</PageHeader>

<div class="mt-8"><List empty={t("dealsHuntsEmpty")}/></div>
