<script lang="ts">
 import type {PageProps} from "./$types";
 import type {Component} from "svelte";
 import {onMount} from "svelte";
 import {resolve} from "$app/paths";
 import {goto,invalidateAll} from "$app/navigation";
 import Send from "@lucide/svelte/icons/send";
 import Gamepad2 from "@lucide/svelte/icons/gamepad-2";
 import Tag from "@lucide/svelte/icons/tag";
 import ShoppingBag from "@lucide/svelte/icons/shopping-bag";
 import Link from "@lucide/svelte/icons/link";
 import Check from "@lucide/svelte/icons/check";
 import Clock from "@lucide/svelte/icons/clock";
 import KeyRound from "@lucide/svelte/icons/key-round";
 import RefreshCw from "@lucide/svelte/icons/refresh-cw";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import ChannelIcon from "#lib/ui/ChannelIcon.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import CodeBlock from "#lib/ui/CodeBlock.svelte";
 import DetailHeader from "#lib/ui/DetailHeader.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import LogoField from "#lib/ui/LogoField.svelte";
 import DomainCard from "#lib/ui/DomainCard.svelte";
 import {dealsApi,parseResponse} from "../../../lib/api";
 import {dealsError,failureText} from "../../../lib/errors";
 import {refreshUsage} from "../../../lib/panel.svelte";
 import {hasPlan} from "../../../lib/plan";
 import SiteForm from "../../../lib/SiteForm.svelte";
 import {getSite,uploadSiteLogo,removeSiteLogo,type SiteLanguage} from "../../../lib/site";
 // One channel, as Pulse's alerts: its state and the examples it can get
 // (the plain test, a free game, a Nuuvem sale, a store find, a pasted
 // link), each as it would really arrive, marked as an example. One round
 // a minute (on WhatsApp, one example at a time, in the Canal's 30-minute
 // turn), with a countdown to the next. A webhook also gets a new signing
 // secret here; the deals page is edited here (as Pulse's status page:
 // name, logos, domain); and any channel is removed here.
 let {data}:PageProps=$props();
 const channel=$derived(data.channel);
 const kindName=(type:NonNullable<typeof channel>["type"])=>type==="site"?t("dealsSiteKind"):({telegram:"Telegram",discord:"Discord",whatsapp:"WhatsApp",webhook:"Webhook"} as const)[type];
 // The deals page: settings or domain, one at a time (?tab=domain).
 let site=$derived(data.page?.site??null);
 const tabs=$derived([{value:"settings" as const,label:t("dealsSiteSettings")},{value:"domain" as const,label:t("dealsSiteDomainTab")}]);
 let tab=$derived<"settings"|"domain">(data.tab);
 async function saveSite(values:{title:string;description:string;language:SiteLanguage;showTitle:boolean;links:{label:string;url:string}[]}){await run(async()=>{await parseResponse(dealsApi.site.$patch({json:{title:values.title,description:values.description,language:values.language,showTitle:values.showTitle,links:values.links}}));site=await getSite();await invalidateAll();toast(()=>t("pulseToastSaved"));});}
 type Kind="test"|"game_free"|"nuuvem_deal"|"store_deal"|"product";
 const options:{value:Kind;label:()=>string;help:()=>string;Icon:Component<{class?:string}>;bg:string}[]=[
  {value:"test",label:()=>t("dealsSampleTest"),help:()=>t("dealsSampleTestHelp"),Icon:Send,bg:"bg-blup"},
  {value:"game_free",label:()=>t("dealsSampleGameFree"),help:()=>t("dealsSampleGameFreeHelp"),Icon:Gamepad2,bg:"bg-emerald-500"},
  {value:"nuuvem_deal",label:()=>t("dealsSampleNuuvem"),help:()=>t("dealsSampleNuuvemHelp"),Icon:Tag,bg:"bg-violet-500"},
  {value:"store_deal",label:()=>t("dealsSampleStore"),help:()=>t("dealsSampleStoreHelp"),Icon:ShoppingBag,bg:"bg-red-500"},
  {value:"product",label:()=>t("dealsSampleProduct"),help:()=>t("dealsSampleProductHelp"),Icon:Link,bg:"bg-amber-500"},
 ];
 let kinds=$state<Kind[]>([]),busy=$state(false),secret=$state<string|null>(null);
 // On WhatsApp, picking one leaves only that one.
 const single=$derived(channel?.type==="whatsapp");
 function toggle(kind:Kind){kinds=kinds.includes(kind)?kinds.filter(k=>k!==kind):single?[kind]:[...kinds,kind];}
 // When it can get examples again: the server's, then the last round's.
 let againAt=$derived(channel?.testAgainAt??0);
 let now=$state(Date.now());
 onMount(()=>{const timer=setInterval(()=>now=Date.now(),500);return()=>clearInterval(timer);});
 const waiting=$derived(now<againAt);
 const clock=(ms:number)=>{const s=Math.max(0,Math.ceil(ms/1000)),h=Math.floor(s/3600),m=Math.floor(s%3600/60),r=s%60;return h?`${h}:${String(m).padStart(2,"0")}:${String(r).padStart(2,"0")}`:`${m}:${String(r).padStart(2,"0")}`;};
 const reconnect=$derived(channel?.type==="whatsapp"&&!!channel.lastError?.startsWith("WHATSAPP_"));
 const status=$derived(!channel?"":reconnect?failureText(channel.lastError):!channel.active?t("dealsChannelOff"):channel.lastError?t("dealsChannelFailing"):t("dealsChannelActive"));

 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function send(){const current=channel;if(!current||!kinds.length)return;await run(async()=>{
  const result=await parseResponse(dealsApi.channels[":channel"].test.$post({param:{channel:current.id},json:{kinds:options.map(o=>o.value).filter(k=>kinds.includes(k))}}));
  againAt=result.testAgainAt;
  if(result.ok)toast(()=>t("dealsSamplesSent"));else toast(failureText(result.code,"dealsTestFailed"),{kind:"error"});
  await invalidateAll();againAt=Math.max(againAt,result.testAgainAt);
 });}
 async function rotate(){const current=channel;if(!current||!await confirm({title:t("dealsRotateSecret"),message:t("dealsRotateSecretConfirm")}))return;await run(async()=>{secret=(await parseResponse(dealsApi.channels[":channel"].secret.$post({param:{channel:current.id},json:{}}))).secret;});}
 async function remove(){const current=channel;if(!current||!await confirm({title:t("dealsRemoveChannelTitle"),message:t("dealsRemoveChannel"),danger:true}))return;await run(async()=>{await parseResponse(dealsApi.channels[":channel"].$delete({param:{channel:current.id}}));void refreshUsage();toast(()=>t("dealsRemoved"));await goto(resolve("/channels"),{invalidateAll:true});});}
 // What a webhook gets for each deal.
 const payloadSample=JSON.stringify({version:1,source:"blup-deals",event:"deal",test:false,language:"pt",text:"Oferta do dia: Smartwatch com GPS\nR$ 399,00 R$ 299,00 (25% de desconto)\nVer oferta: https://deals.getblup.site/r/AbCd2345",deal:{title:"Oferta do dia: Smartwatch com GPS",kind:"product",store:"amazon",price:29900,originalPrice:39900,currency:"BRL",discount:25,link:"https://deals.getblup.site/r/AbCd2345",image:"https://...",video:null,description:null,coupon:null,affiliate:true,credit:null},sentAt:"2026-11-05T12:00:00.000Z"},null,2);
</script>

{#if channel}
 <div class="mb-4"><BackLink href={resolve("/channels")} label={t("dealsChannels")}/></div>
 {#if site}<DetailHeader title={site.title} url={site.url}/><p class="mt-2 text-neutral-500">{t("dealsSiteLead")}</p>
 {:else}<PageHeader title={channel.label} lead={t("dealsChannelLead")}/>{/if}
 <div class="mt-6 flex flex-wrap items-center gap-3">
  <span class="grid size-10 shrink-0 place-items-center rounded-full border border-neutral-200 dark:border-neutral-700"><ChannelIcon type={channel.type} class="size-5"/></span>
  <Badge>{kindName(channel.type)}</Badge>
  <span class="text-sm {!channel.active||channel.lastError?'text-amber-700 dark:text-amber-300':'text-neutral-500'}">{status}</span>
  {#if reconnect&&hasPlan()}<Button variant="secondary" onclick={()=>goto(`${resolve("/channels/new")}?kind=whatsapp`)}><RefreshCw class="size-4"/>{t("dealsWhatsappReconnect")}</Button>{/if}
 </div>

 {#if site}
  {@const current=site}
  <div class="mt-8"><Segmented label={t("dealsSiteKind")} options={tabs} bind:value={tab}/></div>
  {#if tab==="domain"}
   <DomainCard eligible={hasPlan()} help={t("dealsSiteDomainHelp")} removeAsk={t("dealsSiteDomainRemoveAsk")} errorText={dealsError} load={()=>parseResponse(dealsApi.site.domain.$get())} connect={hostname=>parseResponse(dealsApi.site.domain.$post({json:{hostname}}))} remove={()=>parseResponse(dealsApi.site.domain.$delete())}/>
  {:else}
   <div class="mt-8">{#key current.id}<SiteForm editing {busy} disabled={!hasPlan()} initial={{title:current.title,slug:current.slug,description:current.description,language:current.language,showTitle:current.showTitle,links:current.links}} onsave={saveSite}>
    {#snippet logos()}
     <LogoField variant="light" label={t("pulseLogoLight")} preview={data.page?.logos.light??null} disabled={!hasPlan()} upload={logo=>uploadSiteLogo("light",logo)} remove={()=>removeSiteLogo("light")}/>
     <LogoField variant="dark" label={t("pulseLogoDark")} preview={data.page?.logos.dark??null} disabled={!hasPlan()} upload={logo=>uploadSiteLogo("dark",logo)} remove={()=>removeSiteLogo("dark")}/>
    {/snippet}
   </SiteForm>{/key}</div>
  {/if}
 {:else if hasPlan()&&!reconnect}
  <div class="mt-8"><Card title={t("dealsChooseSamples")} help={t("dealsChooseSamplesHelp")}>
   <div class="mt-5 grid grid-cols-[minmax(0,1fr)] gap-3 sm:grid-cols-2 xl:grid-cols-3" role={single?"radiogroup":"group"} aria-label={t("dealsChooseSamples")}>
    {#each options as option(option.value)}
     {@const on=kinds.includes(option.value)}
     <button type="button" role={single?"radio":"checkbox"} aria-checked={on} class="flex items-center gap-3 rounded-xl border p-4 text-left transition-colors {on?'border-blup bg-orange-50/60 dark:bg-orange-950/30':'border-neutral-200 hover:border-neutral-300 dark:border-neutral-800 dark:hover:border-neutral-700'}" disabled={waiting||busy} onclick={()=>toggle(option.value)}>
      <span class="grid size-9 shrink-0 place-items-center rounded-full text-white {option.bg}"><option.Icon class="size-4"/></span>
      <span class="min-w-0 flex-1"><span class="block font-semibold">{option.label()}</span><span class="block text-sm text-neutral-500">{option.help()}</span></span>
      <span class="grid size-5 shrink-0 place-items-center border {single?'rounded-full':'rounded'} {on?'border-blup bg-blup text-white':'border-neutral-300 dark:border-neutral-600'}">{#if on}<Check class="size-3.5"/>{/if}</span>
     </button>
    {/each}
   </div>
   <p class="mt-5 flex items-start gap-2 text-sm text-neutral-500"><Clock class="mt-0.5 size-4 shrink-0"/>{single?t("dealsSampleOneWhatsapp"):t("dealsSamplesTiming")}</p>
   {#if !channel.active}<p class="mt-3 text-sm text-amber-700 dark:text-amber-300">{t("dealsChannelOff")}</p>{/if}
   <div class="mt-5 flex justify-end"><Button onclick={send} {busy} disabled={!kinds.length||waiting}>{#if waiting}<Clock class="size-4"/>{t("dealsTestAgainIn",{time:clock(againAt-now)})}{:else}<Send class="size-4"/>{t("dealsSendSamples")}{/if}</Button></div>
  </Card></div>
 {/if}

 {#if !site||tab==="settings"}<div class="mt-10 grid grid-cols-[minmax(0,1fr)] gap-10">
  {#if channel.type==="webhook"}
   <FormSection title={t("dealsSigningSecret")} help={t("dealsSigningSecretHelp")}>
    {#if secret}
     <div class="grid grid-cols-[minmax(0,1fr)] gap-3"><p class="font-semibold">{t("dealsNewSecretReady")}</p><CodeBlock code={secret}/><p class="text-sm text-neutral-500">{t("dealsWebhookSecretOnce")}</p></div>
    {:else if hasPlan()}<Button variant="secondary" {busy} onclick={rotate}><KeyRound class="size-4"/>{t("dealsRotateSecret")}</Button>{/if}
   </FormSection>
   <FormSection title="JSON" help={t("dealsWebhookSample")}><CodeBlock code={payloadSample}/></FormSection>
  {/if}
  <FormSection title={t("dealsRemoveChannelTitle")} help={t("dealsRemoveChannelHelp")}>
   <button type="button" class="inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2.5 font-semibold text-white transition-colors hover:bg-red-700 disabled:opacity-50" disabled={busy} onclick={remove}><Trash2 class="size-4"/>{t("dealsRemoveThisChannel")}</button>
  </FormSection>
 </div>{/if}

 <p class="mt-10 text-xs text-neutral-500">{t("dealsChannelsPrivacy")}</p>
{/if}
