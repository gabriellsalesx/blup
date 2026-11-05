<script lang="ts">
 import {onMount,onDestroy} from "svelte";
 import {type InferResponseType} from "hono/client";
 import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
 import RefreshCw from "@lucide/svelte/icons/refresh-cw";
 import Check from "@lucide/svelte/icons/check";
 import {t} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import {dealsApi,parseResponse} from "./api";
 import {dealsError} from "./errors";
 import {hasPlan} from "./plan";
 // Connecting the account's WhatsApp to post in its Canais (the unofficial
 // way): the risks and the person's yes first, then the QR (asked again
 // every few seconds while it waits), then the Canais the number owns or
 // runs. onadded runs once a Canal is in Blup Deals.
 let {onadded}:{onadded:()=>void|Promise<void>}=$props();
 type View=InferResponseType<typeof dealsApi.whatsapp.session.$get,200>;
 type Channels=InferResponseType<typeof dealsApi.whatsapp.channels.$get,200>["channels"];
 let view=$state<View|null>(null),accepted=$state(false),busy=$state(false),channels=$state<Channels|null>(null),adding=$state<string|null>(null);
 // What a connection that ended says.
 const endedText={expired:"dealsWhatsappExpired",logged_out:"dealsWhatsappLoggedOut",banned:"dealsWhatsappBanned",replaced:"dealsWhatsappReplaced",outdated:"dealsWhatsappOutdated"} as const;
 const ended=(state:string):state is keyof typeof endedText=>Object.hasOwn(endedText,state);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function load(){
  view=await parseResponse(dealsApi.whatsapp.session.$get());
  if(view.state==="pairing"||view.state==="disconnected")startPolling();else stopPolling();
  if(view.state==="connected"&&channels===null)await loadChannels();
 }
 async function loadChannels(){channels=(await parseResponse(dealsApi.whatsapp.channels.$get())).channels;}
 const connect=()=>run(async()=>{channels=null;view=await parseResponse(dealsApi.whatsapp.session.$post({json:{acceptRisk:true}}));startPolling();});
 const refresh=()=>run(loadChannels);
 const add=(id:string)=>{adding=id;void run(async()=>{try{await parseResponse(dealsApi.whatsapp.channels.$post({json:{channel:id}}));await onadded();}finally{adding=null;}});};
 async function disconnect(){
  if(!await confirm({title:t("dealsWhatsappDisconnect"),message:t("dealsWhatsappDisconnectConfirm"),danger:true}))return;
  await run(async()=>{await parseResponse(dealsApi.whatsapp.session.$delete());channels=null;accepted=false;await load();});
 }
 // While the QR waits, or the connection comes back, the state is asked
 // every three seconds (five minutes at most).
 let timer:ReturnType<typeof setInterval>|undefined,polls=0;
 function stopPolling(){if(timer!==undefined){clearInterval(timer);timer=undefined;}}
 function startPolling(){
  if(timer!==undefined)return;polls=0;
  timer=setInterval(()=>{if(++polls>100){stopPolling();return;}void load().catch(()=>{});},3000);
 }
 onMount(()=>{void run(load);});
 onDestroy(stopPolling);
</script>

<div aria-live="polite">
 {#if !view}<p class="text-sm text-neutral-500">{t("dealsWhatsappQRWait")}</p>
 {:else if !view.ready}<p class="text-sm text-neutral-500">{t("dealsWhatsappUnavailable")}</p>
 {:else if view.state==="pairing"}
  <ol class="grid list-decimal gap-1.5 pl-5 text-sm text-neutral-600 dark:text-neutral-300"><li>{t("dealsWhatsappStep1")}</li><li>{t("dealsWhatsappStep2")}</li><li>{t("dealsWhatsappStep3")}</li></ol>
  <div class="mt-5 flex flex-col items-center gap-2">
   <!-- The QR renews itself while it waits: no deadline to show. -->
   {#if view.qr}<img src={view.qr} alt={t("dealsWhatsappQRAlt")} width="256" height="256" class="size-64 rounded-xl bg-white p-3"/>
   {:else}<p class="py-24 text-sm text-neutral-500">{t("dealsWhatsappQRWait")}</p>{/if}
  </div>
 {:else if view.state==="connected"||view.state==="disconnected"}
  <p class="flex items-center gap-2 text-sm font-semibold"><Check class="size-4 text-emerald-600 dark:text-emerald-400"/>{t("dealsWhatsappConnected",{end:view.phoneEnd??"----"})}</p>
  {#if view.state==="disconnected"}<p class="mt-2 text-sm text-amber-700 dark:text-amber-300">{t("dealsWhatsappReconnecting")}</p>{/if}
  <div class="mt-5 flex items-center justify-between gap-3">
   <h3 class="font-semibold">{t("dealsWhatsappPick")}</h3>
   <Button variant="secondary" small onclick={refresh} {busy}><RefreshCw class="size-4"/>{t("dealsWhatsappRefresh")}</Button>
  </div>
  {#if channels&&!channels.length}<p class="mt-3 text-sm text-neutral-500">{t("dealsWhatsappNoChannels")}</p>
  {:else if channels}
   <ul class="mt-3 grid gap-2">
    {#each channels as channel(channel.id)}
     <li class="flex items-center justify-between gap-3 rounded-xl border border-neutral-200 px-4 py-3 dark:border-neutral-800">
      <span class="min-w-0"><span class="block truncate font-semibold">{channel.name}</span><span class="block text-sm text-neutral-500">{channel.role==="owner"?t("dealsWhatsappOwner"):t("dealsWhatsappAdmin")}</span></span>
      {#if channel.added}<span class="shrink-0 text-sm text-neutral-500">{t("dealsWhatsappAdded")}</span>
      {:else}<Button small busy={adding===channel.id} disabled={busy||!hasPlan()} onclick={()=>add(channel.id)}>{t("dealsWhatsappAdd")}</Button>{/if}
     </li>
    {/each}
   </ul>
  {/if}
  <div class="mt-6 flex justify-end"><Button variant="secondary" onclick={disconnect} {busy}>{t("dealsWhatsappDisconnect")}</Button></div>
 {:else}
  {#if ended(view.state)}<p class="mb-4 text-sm font-semibold text-amber-700 dark:text-amber-300">{t(endedText[view.state])}</p>{/if}
  <div class="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
   <p class="flex items-center gap-2 font-semibold"><TriangleAlert class="size-4 shrink-0"/>{t("dealsWhatsappRiskTitle")}</p>
   <ul class="mt-2 grid list-disc gap-1.5 pl-5"><li>{t("dealsWhatsappRisk1")}</li><li>{t("dealsWhatsappRisk2")}</li><li>{t("dealsWhatsappRisk3")}</li><li>{t("dealsWhatsappRisk4")}</li></ul>
  </div>
  <div class="mt-4"><Switch label={t("dealsWhatsappAccept")} tone="coral" bind:checked={accepted}/></div>
  <div class="mt-5 flex justify-end"><Button onclick={connect} {busy} disabled={!accepted||!hasPlan()}>{view.state==="none"?t("dealsWhatsappConnect"):t("dealsWhatsappNewQR")}</Button></div>
 {/if}
</div>
