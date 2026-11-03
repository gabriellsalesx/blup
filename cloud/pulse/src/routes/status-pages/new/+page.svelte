<script lang="ts">
 import type {PageProps} from "./$types";
 import {resolve} from "$app/paths";
 import {goto} from "$app/navigation";
 import {t,language} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import PageForm from "../../../lib/PageForm.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 // A new status page; once made it opens, to add a domain or incidents.
 let {data}:PageProps=$props();
 let busy=$state(false);
 async function save(values:{title:string;slug:string;components:{monitorId:string;name:string;description:string}[];showTitle:boolean;links:{label:string;url:string}[]}){busy=true;try{const made=await parseResponse(pulseApi.pages.$post({json:{...values,language:language()}}));toast(()=>t("pulseToastPage"));await goto(resolve("/status-pages/[id]",{id:made.id}));}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
</script>

<div class="mb-4"><BackLink href={resolve("/status-pages")} label={t("pulsePages")}/></div>
<PageHeader title={t("pulseNewPage")} lead={t("pulsePublicPrivacy")}/>
<div class="mt-8"><PageForm monitors={data.monitors} {busy} onsave={save} oncancel={()=>goto(resolve("/status-pages"))}/></div>
