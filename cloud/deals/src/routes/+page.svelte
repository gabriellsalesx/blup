<script lang="ts">
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import ClipboardPaste from "@lucide/svelte/icons/clipboard-paste";
 import Rocket from "@lucide/svelte/icons/rocket";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import List from "#lib/ui/List.svelte";
 import {hasPlan} from "../lib/plan";
 // The deals sent to the channels (empty in phase F0), with a way to paste
 // a link and to the first steps.
</script>

<PageHeader title={t("dealsOffers")} lead={t("dealsOffersLead")}>
 {#snippet aside()}{#if hasPlan()}<div class="flex flex-wrap gap-2"><Button variant="secondary" onclick={()=>goto(resolve("/start"))}><Rocket class="size-4"/>{t("dealsStart")}</Button><Button onclick={()=>goto(resolve("/offers/new"))}><ClipboardPaste class="size-4"/>{t("dealsPasteLink")}</Button></div>{/if}{/snippet}
</PageHeader>

<div class="mt-8"><List empty={t("dealsOffersEmpty")}/></div>
