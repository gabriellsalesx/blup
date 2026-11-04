<script lang="ts">
 import type {PageProps} from "./$types";
 import {untrack,tick} from "svelte";
 import {resolve} from "$app/paths";
 import X from "@lucide/svelte/icons/x";
 import Plus from "@lucide/svelte/icons/plus";
 import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
 import Save from "@lucide/svelte/icons/save";
 import Globe from "@lucide/svelte/icons/globe";
 import {t,language} from "#lib/i18n.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
 import {dealsApi,parseResponse,type Settings} from "../../lib/api";
 import {dealsError} from "../../lib/errors";
 import {hasPlan} from "../../lib/plan";
 import {paceOptions,type DealsPace} from "../../lib/pace";
 import {socialMarks} from "../../lib/social-marks";
 import PostPreview from "../../lib/PostPreview.svelte";
 import {defaultHeadlines,headline,headlineKinds,tokenText,hasAddress,socialNetworks,socialURL,networkOf,HEADLINE_MAX,EXTRA_MAX,type HeadlineKind,type HeadlineToken,type SocialNetwork} from "../../../../shared/deals-post";
 // One setting for all the account's channels: how often the queue sends,
 // the opening line of each kind of deal (a default ready, changeable),
 // the extra text and the social networks (one of each), with a preview
 // of the post as it will read.
 let {data}:PageProps=$props();
 const saved:Settings=untrack(()=>data.settings)??{pace:0,headlines:{},extra:{text:"",on:false},socials:{links:[],on:false},behind:false};
 const defaults=$derived(defaultHeadlines[language()]);
 let pace=$state(String(saved.pace));
 // An empty line is the default one.
 let lines=$state<Record<HeadlineKind,string>>(untrack(()=>Object.fromEntries(headlineKinds.map(k=>[k,saved.headlines[k]??defaultHeadlines[language()][k]])) as Record<HeadlineKind,string>));
 // A first extra text ready (off until the person turns it on).
 let extra=$state(saved.extra.text||untrack(()=>t("dealsExtraSample"))),extraOn=$state(saved.extra.on);
 let links=$state<{network:SocialNetwork;url:string}[]>(saved.socials.links.map(l=>({network:l.network,url:l.url})));
 let socialsOn=$state(saved.socials.on),busy=$state(false);

 const paces=$derived(paceOptions().map(o=>({value:String(o.value),label:o.label})));
 const kindLabel:Record<HeadlineKind,PlainMessageKey>={game_free:"dealsHeadlineGameFree",nuuvem_deal:"dealsHeadlineNuuvem",store_deal:"dealsHeadlineStore",product:"dealsHeadlineProduct"};
 const tokenLabel:Record<HeadlineToken,PlainMessageKey>={name:"dealsTokenName",discount:"dealsTokenDiscount",store:"dealsTokenStore"};
 const tokens=["name","discount","store"] as const;
 const lineBad=(kind:HeadlineKind)=>!lines[kind].trim()||hasAddress(lines[kind]);
 const linkBad=(l:{network:SocialNetwork;url:string})=>!socialURL(l.network,l.url);
 const invalid=$derived(headlineKinds.some(lineBad)||hasAddress(extra)||links.some(linkBad));

 async function addToken(kind:HeadlineKind,token:HeadlineToken){
  const el=document.getElementById(`line-${kind}`) as HTMLInputElement|null,text=tokenText(token,language());
  const at=el?.selectionStart??lines[kind].length,end=el?.selectionEnd??at;
  lines[kind]=(lines[kind].slice(0,at)+text+lines[kind].slice(end)).slice(0,HEADLINE_MAX);
  await tick();el?.focus();el?.setSelectionRange(at+text.length,at+text.length);
 }
 async function addNetwork(network:SocialNetwork){
  if(links.some(l=>l.network===network))return;
  links.push({network,url:networkOf(network)?.prefix??""});
  await tick();const el=document.getElementById(`social-${network}`) as HTMLInputElement|null;el?.focus();el?.setSelectionRange(el.value.length,el.value.length);
 }
 // The preview: a made-up deal of each kind.
 const samples:Record<HeadlineKind,{name:string;discount:number|null;store:string;price:number|null}>={
  game_free:{name:"Dark Mystery",discount:100,store:"GamerPower",price:null},
  nuuvem_deal:{name:"Hollow Knight",discount:50,store:"Nuuvem",price:2399},
  store_deal:{name:"Fone Bluetooth",discount:40,store:"Shopee",price:8990},
  product:{name:"Air Fryer 4L",discount:null,store:"Amazon",price:29990},
 };
 let previewKind=$state<HeadlineKind>("game_free");
 const previewKinds=$derived(headlineKinds.map(value=>({value,label:t(kindLabel[value])})));
 const sample=$derived(samples[previewKind]);
 const shownSocials=$derived(socialsOn?links.filter(l=>!linkBad(l)).map(l=>networkOf(l.network)?.name??l.network):[]);

 async function save(){
  if(invalid)return;busy=true;
  try{
   const headlines=Object.fromEntries(headlineKinds.filter(k=>lines[k].trim()!==defaults[k]).map(k=>[k,lines[k].trim()]));
   await parseResponse(dealsApi.settings.$put({json:{pace:Number(pace) as DealsPace,headlines,extra:{text:extra.trim(),on:extraOn},socials:{links,on:socialsOn}}}));
   toast(()=>t("dealsSettingsSaved"));
  }catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}
 }
</script>

<!-- A network's mark; the person's own site, the globe. -->
{#snippet mark(id:SocialNetwork,size:string)}{#if id==="site"}<Globe class={size} aria-hidden="true"/>{:else}<svg viewBox="0 0 24 24" class={size} fill="currentColor" aria-hidden="true"><path d={socialMarks[id]}/></svg>{/if}{/snippet}

<div class="mb-4"><BackLink href={resolve("/")} label={t("dealsOffers")}/></div>
<PageHeader title={t("dealsSettings")} lead={t("dealsSettingsLead")}/>

<div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-6 lg:grid-cols-[minmax(0,1fr)_22rem] lg:items-start">
 <div class="grid min-w-0 grid-cols-[minmax(0,1fr)] gap-6">
  <Card title={t("dealsPace")} help={t("dealsPaceHelp")}>
   <div class="mt-5"><Segmented label={t("dealsPace")} options={paces} bind:value={pace}/></div>
  </Card>

  <Card title={t("dealsHeadlines")} help={t("dealsHeadlinesHelp")}>
   <div class="mt-5 grid gap-6">
    {#each headlineKinds as kind(kind)}
     <div onfocusin={()=>previewKind=kind}>
      <FloatField id={`line-${kind}`} surface="bg-white dark:bg-neutral-900" label={t(kindLabel[kind])} maxlength={HEADLINE_MAX} bind:value={lines[kind]} aria-invalid={lineBad(kind)}/>
      {#if hasAddress(lines[kind])}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsNoAddress")}</p>{/if}
      <div class="mt-2 flex flex-wrap items-center gap-2">
       {#each tokens as token(token)}
        <button type="button" class="inline-flex items-center gap-1 rounded-full border border-neutral-300 px-3 py-1 text-xs font-semibold hover:border-[#c2410c] hover:text-[#c2410c] dark:border-neutral-700" aria-label={t("dealsTokenAdd",{token:t(tokenLabel[token])})} onclick={()=>addToken(kind,token)}><Plus class="size-3.5"/>{t(tokenLabel[token])}</button>
       {/each}
       {#if lines[kind].trim()!==defaults[kind]}<button type="button" class="ml-auto inline-flex items-center gap-1 text-xs font-semibold text-neutral-500 hover:text-neutral-900 dark:hover:text-white" onclick={()=>lines[kind]=defaults[kind]}><RotateCcw class="size-3.5"/>{t("dealsHeadlineReset")}</button>{/if}
      </div>
     </div>
    {/each}
   </div>
  </Card>

  <Card title={t("dealsExtra")} help={t("dealsExtraHelp")}>
   <div class="mt-5 grid gap-4">
    <Switch label={t("dealsShowInPosts")} tone="coral" bind:checked={extraOn}/>
    <div>
     <FloatField id="extra-text" surface="bg-white dark:bg-neutral-900" label={t("dealsExtraField")} maxlength={EXTRA_MAX} bind:value={extra} aria-invalid={hasAddress(extra)}/>
     {#if hasAddress(extra)}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsNoAddress")}</p>{/if}
    </div>
   </div>
  </Card>

  <Card title={t("dealsSocials")} help={t("dealsSocialsHelp")}>
   <div class="mt-5 grid gap-4">
    <Switch label={t("dealsShowInPosts")} tone="coral" bind:checked={socialsOn}/>
    {#each links as link,i(link.network)}
     {@const net=networkOf(link.network)}
     <div>
      <div class="flex items-center gap-2">
       <span class="grid size-14 shrink-0 place-items-center rounded-xl bg-neutral-100 dark:bg-neutral-800" aria-hidden="true">{@render mark(link.network,"size-5")}</span>
       <div class="min-w-0 flex-1"><FloatField id={`social-${link.network}`} surface="bg-white dark:bg-neutral-900" label={net?.name??link.network} type="url" inputmode="url" autocomplete="off" maxlength={300} bind:value={links[i].url} aria-invalid={linkBad(link)}/></div>
       <button type="button" class="grid size-10 shrink-0 place-items-center rounded-lg text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white" aria-label={`${t("remove")}: ${net?.name??""}`} onclick={()=>links.splice(i,1)}><X class="size-4"/></button>
      </div>
      {#if linkBad(link)}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsSocialInvalid",{network:net?.name??""})}</p>{/if}
     </div>
    {/each}
    <div class="flex flex-wrap gap-2">
     {#each socialNetworks as n(n.id)}
      {@const added=links.some(l=>l.network===n.id)}
      <button type="button" class="inline-flex items-center gap-2 rounded-full border border-neutral-300 px-3 py-1.5 text-sm font-semibold enabled:hover:border-[#c2410c] enabled:hover:text-[#c2410c] disabled:opacity-40 dark:border-neutral-700" disabled={added} aria-pressed={added} onclick={()=>addNetwork(n.id)}>{@render mark(n.id,"size-4")}{n.name}</button>
     {/each}
    </div>
   </div>
  </Card>

  <div class="flex justify-end"><Button {busy} disabled={invalid||!hasPlan()} onclick={save}><Save class="size-4"/>{t("save")}</Button></div>
 </div>

 <aside class="grid min-w-0 gap-3 lg:sticky lg:top-6">
  <p class="text-sm font-semibold">{t("dealsPreviewOf")}</p>
  <Segmented label={t("dealsPreviewOf")} options={previewKinds} bind:value={previewKind}/>
  <PostPreview example title={headline(lines[previewKind].trim()||defaults[previewKind],sample)||sample.name} price={sample.price} currency={sample.price===null?null:"BRL"} store={sample.store} affiliate={previewKind!=="game_free"} extra={extraOn&&extra.trim()&&!hasAddress(extra)?extra.trim():null} socials={shownSocials}/>
 </aside>
</div>
