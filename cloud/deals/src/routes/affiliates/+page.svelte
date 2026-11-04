<script lang="ts">
 import type {PageProps} from "./$types";
 import {invalidateAll} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import {hasPlan} from "../../lib/plan";
 import {stores} from "../../lib/catalog";
 import StoreCard from "../../lib/StoreCard.svelte";
 // A card per store, in the order a beginner needs them: Amazon and
 // Mercado Livre take one ID; Shopee and AliExpress take API keys.
 let {data}:PageProps=$props();
</script>

<PageHeader title={t("dealsAffiliates")} lead={t("dealsAffiliatesLead")}/>

<div class="mt-8 grid gap-6 lg:grid-cols-2">
 {#each stores as info(info.id)}
  <StoreCard store={info.id} saved={data.stores.find(s=>s.store===info.id)} writable={hasPlan()} onchange={()=>void invalidateAll()}/>
 {/each}
</div>

<p class="mt-8 text-sm text-neutral-500">{t("dealsKeysNote")}</p>
