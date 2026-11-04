<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t} from "#lib/i18n.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import {dealsApi,parseResponse} from "../../../lib/api";
 import {dealsError} from "../../../lib/errors";
 import type {HuntInput} from "../../../lib/catalog";
 import HuntForm from "../../../lib/HuntForm.svelte";
 // Changing a hunt's filters and channels (its kind stays).
 let {data}:PageProps=$props();
 let busy=$state(false);
 async function save({channelIds,filters}:HuntInput){const hunt=data.hunt;if(!hunt)return;busy=true;try{await parseResponse(dealsApi.hunts[":hunt"].$patch({param:{hunt:hunt.id},json:{channelIds,filters}}));toast(()=>t("dealsSaved"));await goto(resolve("/hunts"));}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
</script>

<div class="mb-4"><BackLink href={resolve("/hunts")} label={t("dealsHunts")}/></div>
<PageHeader title={t("dealsEditHunt")}/>
{#if data.hunt}{#key data.hunt.id}<HuntForm channels={data.channels} configured={data.configured} hunt={data.hunt} {busy} submitLabel={t("dealsSave")} onsubmit={save}/>{/key}{/if}
