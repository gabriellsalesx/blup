<script lang="ts">
 import { onMount } from "svelte";
 import { goto } from "$app/navigation";
 import Globe from "@lucide/svelte/icons/globe";
 import Row from "#lib/ui/Row.svelte";
 import Button from "#lib/ui/Button.svelte";
 import { api, type Site } from "#lib/api.ts";
 import { confirm } from "#lib/confirm.svelte.ts";
 import { toast } from "#lib/toast.svelte.ts";
 import { bytes } from "#lib/usage.ts";
 let {data}=$props();
 let result=$state<Awaited<ReturnType<typeof api.sites>>|null>(null);
 let busy=$state("");
 const limit=$derived(result?.limit??data.me.plan.limits.sites);
 const downgrade=$derived(result?.downgrade??null);
 const when=(date:string)=>new Date(date).toLocaleString("pt-BR",{dateStyle:"medium",timeStyle:"short"});
 async function load(){try{result=await api.sites();}catch{toast("Não deu para carregar seus sites. Tente novamente.",{kind:"error"});}}
 onMount(()=>{void load();});
 async function choose(site:Site){
  busy=site.id;
  try{await api.selectRetained(site.id);await load();toast("Escolha salva. Você pode mudar até o fim do prazo.");}
  catch{await load();toast("Não deu para salvar. Confira se o prazo ainda está aberto.",{kind:"error"});}
  finally{busy="";}
 }
 async function takeDown(site:Site){
  if(!await confirm({title:`Tirar ${site.slug}.getblup.site do ar?`,message:"O endereço para de abrir e os arquivos publicados são apagados da nuvem. As notas continuam no seu computador.",confirmLabel:"Tirar do ar",danger:true,typeToConfirm:site.slug}))return;
  busy=site.id;
  try{await api.takeDown(site.id);await load();toast("Site tirado do ar");}
  catch{toast("Não deu para tirar o site do ar agora. Tente de novo.",{kind:"error"});}
  finally{busy="";}
 }
 const count=$derived(result===null?"Carregando…":`${result.sites.length} sites guardados; limite do plano: ${limit}.`);
</script>
<h1 class="text-3xl font-extrabold tracking-tight">Sites</h1>
<p class="mt-2 text-neutral-500">Seus sites publicados pelo Blup Studio. Os arquivos ficam preservados quando um site sai do ar por mudança de plano.</p>
<div class="mt-6">
 <Row title="Seus sites" help={count} />
 {#if downgrade}
  <Row title={downgrade.selectionOpen?"Escolha qual site continua no ar":"Mudança de plano"} help={downgrade.selectionOpen?`Escolha até ${when(downgrade.deadline)}. Você pode mudar de ideia até o prazo. Sem escolha, fica o site mais antigo; empate é resolvido pelo ID.`:limit===0?`O Grátis não inclui sites no ar. A carência termina em ${when(downgrade.deadline)}.`:`Após ${when(downgrade.deadline)}, ficam no ar os sites que cabem no plano.`}>
   {#if downgrade.selectionOpen && downgrade.selectedSiteId===null}
    <p class="mt-2 text-sm text-neutral-500">Sem escolha salva. O site padrão é {result?.sites.find(s=>downgrade.retainedSiteIds.includes(s.id))?.title??"o mais antigo"}.</p>
   {/if}
  </Row>
 {/if}
 {#if limit===0}
  <Row title="Hospedagem no seu plano" help="O plano Grátis não inclui hospedagem depois da carência. Os arquivos guardados não são apagados por essa mudança.">
   {#snippet action()}<Button onclick={()=>goto("/conta/plano")}>Ver os planos</Button>{/snippet}
  </Row>
 {/if}
 {#each result?.sites??[] as site (site.id)}
  <Row title={site.title} help={`${site.live?`${site.files} arquivos, ${bytes(site.bytes)}`:site.published?"Fora do ar pelo limite do plano; arquivos preservados":"Ainda não publicado"} · atualizado em ${when(site.updatedAt)}`}>
   {#if site.live}<a class="mt-1 inline-flex items-center gap-1.5 text-sm font-semibold text-blup hover:underline" href={site.url} target="_blank" rel="noopener"><Globe class="size-4" />{site.url.replace(/^https?:\/\//,"").replace(/\/$/,"")}</a>{/if}
   {#if downgrade?.selectedSiteId===site.id}<p class="mt-2 text-sm font-semibold">Escolhido para continuar no ar</p>{/if}
   {#snippet action()}
    <div class="flex flex-wrap gap-2">
     {#if downgrade?.selectionOpen}<Button variant="secondary" disabled={busy!==""||downgrade.selectedSiteId===site.id} busy={busy===site.id} onclick={()=>choose(site)}>{downgrade.selectedSiteId===site.id?"Escolhido":"Manter este site"}</Button>{/if}
     <Button variant="secondary" disabled={busy!==""} onclick={()=>takeDown(site)}>Tirar do ar</Button>
    </div>
   {/snippet}
  </Row>
 {/each}
 <Row title="Como publicar" help="No Blup Studio, abra o site, vá em Publicar e escolha Blup Cloud. Cada publicação manda só o que mudou." />
</div>
