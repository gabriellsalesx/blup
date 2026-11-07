<script lang="ts" generics="N extends SocialNetwork">
 import {tick} from "svelte";
 import X from "@lucide/svelte/icons/x";
 import {t} from "#lib/i18n.svelte.ts";
 import FloatField from "#lib/ui/FloatField.svelte";
 import {socialMarks} from "./social-marks";
 import {networkOf,socialURL,type SocialNetwork} from "../../../shared/deals-post";
 // The social networks, as Blup Studio asks for them: a button per network
 // (one of each), which opens its address already started with the
 // network's own site; each with its mark and an X to take it out.
 let {networks,links=$bindable(),disabled=false}:{networks:readonly N[];links:{network:N;url:string}[];disabled?:boolean}=$props();
 const bad=(l:{network:N;url:string})=>!socialURL(l.network,l.url)&&l.url.trim()!==networkOf(l.network)?.prefix;
 async function add(network:N){
  if(links.some(l=>l.network===network))return;
  links.push({network,url:networkOf(network)?.prefix??""});
  await tick();const el=document.getElementById(`site-social-${network}`) as HTMLInputElement|null;el?.focus();el?.setSelectionRange(el.value.length,el.value.length);
 }
</script>

{#snippet mark(id:N,size:string)}{#if id!=="site"}<svg viewBox="0 0 24 24" class={size} fill="currentColor" aria-hidden="true"><path d={socialMarks[id as Exclude<SocialNetwork,"site">]}/></svg>{/if}{/snippet}

<div class="grid grid-cols-[minmax(0,1fr)] gap-4">
 {#each links as link,i(link.network)}
  {@const net=networkOf(link.network)}
  <div>
   <div class="flex items-center gap-2">
    <span class="grid size-14 shrink-0 place-items-center rounded-xl bg-neutral-100 dark:bg-neutral-800" aria-hidden="true">{@render mark(link.network,"size-5")}</span>
    <div class="min-w-0 flex-1"><FloatField id={`site-social-${link.network}`} surface="bg-white dark:bg-neutral-900" label={net?.name??link.network} type="url" inputmode="url" autocomplete="off" maxlength={300} {disabled} bind:value={links[i].url} aria-invalid={bad(link)}/></div>
    <button type="button" class="grid size-10 shrink-0 place-items-center rounded-lg text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white" aria-label={`${t("remove")}: ${net?.name??""}`} {disabled} onclick={()=>links.splice(i,1)}><X class="size-4"/></button>
   </div>
   {#if bad(link)}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsSocialInvalid",{network:net?.name??""})}</p>{/if}
  </div>
 {/each}
 <div class="flex flex-wrap gap-2">
  {#each networks as id(id)}
   {@const added=links.some(l=>l.network===id)}
   <button type="button" class="inline-flex items-center gap-2 rounded-full border border-neutral-300 px-3 py-1.5 text-sm font-semibold enabled:hover:border-[#c2410c] enabled:hover:text-[#c2410c] disabled:opacity-40 dark:border-neutral-700" disabled={added||disabled} aria-pressed={added} onclick={()=>add(id)}>{@render mark(id,"size-4")}{networkOf(id)?.name??id}</button>
  {/each}
 </div>
</div>
