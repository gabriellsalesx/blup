<script lang="ts">
 import {goto} from "$app/navigation";
 import ArrowLeft from "@lucide/svelte/icons/arrow-left";
 import {t,language} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import PageForm from "../../../lib/PageForm.svelte";
 import {pulseApi,parseResponse} from "../../../lib/api";
 // A new status page; once made it opens, to add a domain or incidents.
 let {data}=$props();
 let busy=$state(false);
 async function save(values:{title:string;slug:string;components:{monitorId:string;name:string}[]}){busy=true;try{const made=await parseResponse(pulseApi.pages.$post({json:{...values,language:language()}}));toast(()=>t("pulseToastPage"));await goto(`/paginas/${encodeURIComponent(made.id)}`);}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
</script>

<a href="/paginas" class="mb-4 inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><ArrowLeft class="size-4"/>{t("pulsePages")}</a>
<PageHeader title={t("pulseNewPage")} lead={t("pulsePublicPrivacy")}/>
<div class="mt-8"><PageForm monitors={data.monitors} {busy} onsave={save} oncancel={()=>goto("/paginas")}/></div>
