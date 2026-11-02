<script lang="ts">
 import {t,formatDate} from "#lib/i18n.svelte.ts";
 import { onMount } from "svelte";
 import { goto } from "$app/navigation";
 import Globe from "@lucide/svelte/icons/globe";
 import Row from "#lib/ui/Row.svelte";
 import Button from "#lib/ui/Button.svelte";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import { api, type Site } from "#lib/api.ts";
 import { confirm } from "#lib/confirm.svelte.ts";
 import { toast } from "#lib/toast.svelte.ts";
 import { bytes } from "#lib/usage.ts";
 let {data}=$props();
 let result=$state<Awaited<ReturnType<typeof api.sites>>|null>(null);
 let busy=$state("");
 const limit=$derived(result?.limit??data.me.plan.limits.sites);
 const downgrade=$derived(result?.downgrade??null);
 const when=(date:string)=>formatDate(date);
 async function load(){try{result=await api.sites();}catch(error){toast(()=>errorMessage(error),{kind:"error"});}}
 onMount(()=>{void load();});
 async function choose(site:Site){
  busy=site.id;
  try{await api.selectRetained(site.id);await load();toast(()=>t("siteChoiceSaved"));}
  catch(error){await load();toast(()=>errorMessage(error),{kind:"error"});}
  finally{busy="";}
 }
 async function takeDown(site:Site){
  if(!await confirm({title:t("takeDownTitle",{site:`${site.slug}.getblup.site`}),message:t("takeDownBody"),confirmLabel:t("takeDown"),danger:true,typeToConfirm:site.slug}))return;
  busy=site.id;
  try{await api.takeDown(site.id);await load();toast(()=>t("siteDown"));}
  catch(error){toast(()=>errorMessage(error),{kind:"error"});}
  finally{busy="";}
 }
 const count=$derived(result===null?t("loading"):result.sites.length===1?t("siteCountOne",{limit}):t("siteCount",{count:result.sites.length,limit}));
</script>
<h1 class="text-3xl font-extrabold tracking-tight">{t("sites")}</h1>
<p class="mt-2 text-neutral-500">{t("sitesLead")}</p>
<div class="mt-6">
 <Row title={t("yourSites")}  help={count} />
 {#if downgrade}
  <Row title={downgrade.selectionOpen?t("chooseSite"):t("planChange")} help={downgrade.selectionOpen?t("chooseSiteUntil",{date:when(downgrade.deadline)}):limit===0?t("freeSitesUntil",{date:when(downgrade.deadline)}):t("sitesUntil",{date:when(downgrade.deadline)})}>
   {#if downgrade.selectionOpen && downgrade.selectedSiteId===null}
    <p class="mt-2 text-sm text-neutral-500">{t("defaultSite",{site:result?.sites.find(s=>downgrade.retainedSiteIds.includes(s.id))?.title??t("oldestSite")})}</p>
   {/if}
  </Row>
 {/if}
 {#if limit===0}
  <Row title={t("hosting")}  help={t("hostingHelp")} >
   {#snippet action()}<Button onclick={()=>goto("/conta/plano")}>{t("viewPlans")}</Button>{/snippet}
  </Row>
 {/if}
 {#each result?.sites??[] as site (site.id)}
  <Row title={site.title} help={t("updatedAt",{state:site.live?(site.files===1?t("siteFilesOne",{size:bytes(site.bytes)}):t("siteFiles",{count:site.files,size:bytes(site.bytes)})):site.published?t("siteOffline"):t("siteUnpublished"),date:when(site.updatedAt)})}>
   {#if site.live}<a class="mt-1 inline-flex items-center gap-1.5 text-sm font-semibold text-blup hover:underline" href={site.url} target="_blank" rel="noopener"><Globe class="size-4" />{site.url.replace(/^https?:\/\//,"").replace(/\/$/,"")}</a>{/if}
   {#if downgrade?.selectedSiteId===site.id}<p class="mt-2 text-sm font-semibold">{t("siteSelected")}</p>{/if}
   {#snippet action()}
    <div class="flex flex-wrap gap-2">
     {#if downgrade?.selectionOpen}<Button variant="secondary" disabled={busy!==""||downgrade.selectedSiteId===site.id} busy={busy===site.id} onclick={()=>choose(site)}>{downgrade.selectedSiteId===site.id?t("selected"):t("keepSite")}</Button>{/if}
     <Button variant="secondary" disabled={busy!==""} onclick={()=>takeDown(site)}>{t("takeDown")}</Button>
    </div>
   {/snippet}
  </Row>
 {/each}
 <Row title={t("howPublish")}  help={t("howPublishHelp")}  />
</div>
