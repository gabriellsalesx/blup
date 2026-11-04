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
 import {refreshUsage} from "../../../lib/panel.svelte";
 import type {HuntInput} from "../../../lib/catalog";
 import HuntForm from "../../../lib/HuntForm.svelte";
 let {data}:PageProps=$props();
 let busy=$state(false);
 // Several channels to mirror: one mirror each, in turn; one that fails
 // (already mirrored, over the limit) is said and the rest still go.
 async function create({mirrors,...input}:HuntInput){busy=true;try{
  if(!mirrors){await parseResponse(dealsApi.hunts.$post({json:input}));}
  else{let made=0;for(const channel of mirrors){try{await parseResponse(dealsApi.hunts.$post({json:{...input,channel}}));made++;}catch(e){toast(`@${channel}: ${dealsError(e)}`,{kind:"error"});}}if(!made)return;}
  void refreshUsage();toast(()=>t("dealsHuntCreated"));await goto(resolve("/hunts"));
 }catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
</script>

<div class="mb-4"><BackLink href={resolve("/hunts")} label={t("dealsHunts")}/></div>
<PageHeader title={t("dealsNewHuntTitle")} lead={t("dealsNewHuntLead")}/>
<HuntForm channels={data.channels} affiliates={data.affiliates} {busy} submitLabel={t("dealsCreateHunt")} onsubmit={create}/>
