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
 import connectingArt from "./assets/whatsapp-connecting.svg";
 // Connecting the account's WhatsApp to post in its Canais (the unofficial
 // way): the risks and the person's yes first, then "Generate QR code".
 // The QR on screen stays the same while its time counts down; when it runs
 // out, it blurs and a new one is a tap away. Meanwhile the state is asked
 // every few seconds, to see it connect; then the Canais the number owns
 // or runs. onadded runs once a Canal is in Blup Deals.
 let {onadded}:{onadded:()=>void|Promise<void>}=$props();
 type View=InferResponseType<typeof dealsApi.whatsapp.session.$get,200>;
 type Channels=InferResponseType<typeof dealsApi.whatsapp.channels.$get,200>["channels"];
 let view=$state<View|null>(null),accepted=$state(false),channels=$state<Channels|null>(null),adding=$state<string|null>(null);
 // The QR shown, until when it is good and how long it had; now ticks
 // every second for the count and the bar that empties with it (the
 // opposite of the "Connecting" bar, which fills).
 let shown=$state<{qr:string;until:number;total:number}|null>(null),now=$state(Date.now());
 const left=$derived(shown?Math.max(0,Math.ceil((shown.until-now)/1000)):0);
 const remaining=$derived(shown?Math.min(100,Math.max(0,(shown.until-now)/shown.total*100)):0);
 const stale=$derived(!!shown&&(left===0||view?.state==="expired"));
 const clock=(s:number)=>`${Math.floor(s/60)}:${String(s%60).padStart(2,"0")}`;
 // What a connection that ended says.
 const endedText={expired:"dealsWhatsappExpired",logged_out:"dealsWhatsappLoggedOut",banned:"dealsWhatsappBanned",replaced:"dealsWhatsappReplaced",outdated:"dealsWhatsappOutdated"} as const;
 const ended=(state:string):state is keyof typeof endedText=>Object.hasOwn(endedText,state);
 // The action running: only its button spins; the others wait, locked.
 type Action="load"|"generate"|"refresh"|"add"|"disconnect";
 let pending=$state<Action|null>(null);
 async function run(action:Action,fn:()=>Promise<void>){pending=action;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{pending=null;}}
 // From the QR read to the Canais on screen: "Connecting", with the
 // Blup's animation and a bar in two steps (signing in to WhatsApp, then
 // looking for the Canais). If the Canais can't be read, the list shows
 // with Refresh.
 let loadingChannels=false,channelsFailed=$state(false);
 const connecting=$derived(view?.state==="linking"||(view?.state==="connected"&&channels===null&&!channelsFailed));
 const progress=$derived(view?.state==="linking"?45:85);
 async function load(){
  view=await parseResponse(dealsApi.whatsapp.session.$get());
  if(view.state==="pairing"||view.state==="linking"||view.state==="disconnected")startPolling();else stopPolling();
  if(view.state==="linking")shown=null;
  if(view.state==="connected"){shown=null;if(channels===null&&!loadingChannels){loadingChannels=true;try{await loadChannels();}catch(e){channelsFailed=true;toast(dealsError(e),{kind:"error"});}finally{loadingChannels=false;}}}
 }
 // Right after the QR is read WhatsApp may still be finishing: a few tries.
 async function loadChannels(){
  for(let tries=0;;tries++){
   try{channels=(await parseResponse(dealsApi.whatsapp.channels.$get())).channels;return;}
   catch(e){if(tries>=3)throw e;await new Promise(r=>setTimeout(r,2000));}
  }
 }
 // A QR with time left: the one the Bridge has now, or a new connection
 // (the Bridge may take a moment to make the first one). Once one ran out,
 // a new connection: its first QR lasts a whole minute.
 const fresh=(v:View)=>v.state==="pairing"&&!!v.qr&&!!v.qrExpiresAt&&Date.parse(v.qrExpiresAt)-Date.now()>5000;
 const generate=()=>run("generate",async()=>{
  channels=null;
  const again=shown!==null;
  let next=again?null:await parseResponse(dealsApi.whatsapp.session.$get());
  if(!next||!fresh(next))next=await parseResponse(dealsApi.whatsapp.session.$post({json:{acceptRisk:true,fresh:again}}));
  for(let i=0;i<5&&next.state==="pairing"&&!fresh(next);i++){await new Promise(r=>setTimeout(r,1500));next=await parseResponse(dealsApi.whatsapp.session.$get());}
  view=next;
  if(next.qr&&next.qrExpiresAt){now=Date.now();const until=Date.parse(next.qrExpiresAt);shown={qr:next.qr,until,total:Math.max(1000,until-now)};}
  startPolling();
 });
 const refresh=()=>run("refresh",async()=>{await loadChannels();channelsFailed=false;});
 const add=(id:string)=>{adding=id;void run("add",async()=>{try{await parseResponse(dealsApi.whatsapp.channels.$post({json:{channel:id}}));await onadded();}finally{adding=null;}});};
 async function disconnect(){
  if(!await confirm({title:t("dealsWhatsappDisconnect"),message:t("dealsWhatsappDisconnectConfirm"),danger:true}))return;
  await run("disconnect",async()=>{await parseResponse(dealsApi.whatsapp.session.$delete());channels=null;accepted=false;await load();});
 }
 // While the QR waits, or the connection comes back, the state is asked
 // every three seconds (five minutes at most).
 let timer:ReturnType<typeof setInterval>|undefined,polls=0;
 function stopPolling(){if(timer!==undefined){clearInterval(timer);timer=undefined;}}
 function startPolling(){
  if(timer!==undefined)return;polls=0;
  timer=setInterval(()=>{if(++polls>100){stopPolling();return;}void load().catch(()=>{});},3000);
 }
 let tick:ReturnType<typeof setInterval>|undefined;
 onMount(()=>{void run("load",load);tick=setInterval(()=>{now=Date.now();},1000);});
 onDestroy(()=>{stopPolling();clearInterval(tick);});
</script>

<!-- Only the states are announced, not the count. -->
<div>
 {#if !view}<p class="text-sm text-neutral-500">{t("dealsWhatsappQRWait")}</p>
 {:else if !view.ready}<p class="text-sm text-neutral-500">{t("dealsWhatsappUnavailable")}</p>
 {:else if connecting}
  <div class="flex flex-col items-center gap-4 py-2 text-center" role="status">
   <img src={connectingArt} alt="" width="320" height="180" class="h-auto w-full max-w-80"/>
   <p class="font-semibold">{t("dealsWhatsappConnecting")}</p>
   <div class="h-2 w-full max-w-sm overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800" role="progressbar" aria-label={t("dealsWhatsappConnecting")} aria-valuemin={0} aria-valuemax={100} aria-valuenow={progress}>
    <div class="h-full rounded-full bg-[#c2410c] transition-[width] duration-700 motion-reduce:transition-none" style:width="{progress}%"></div>
   </div>
   <p class="text-sm text-neutral-500">{view.state==="linking"?t("dealsWhatsappLinking"):t("dealsWhatsappLoadingChannels")}</p>
  </div>
 {:else if view.state==="connected"||view.state==="disconnected"}
  <p class="flex items-center gap-2 text-sm font-semibold" role="status"><Check class="size-4 text-emerald-600 dark:text-emerald-400"/>{t("dealsWhatsappConnected",{end:view.phoneEnd??"----"})}</p>
  {#if view.state==="disconnected"}<p class="mt-2 text-sm text-amber-700 dark:text-amber-300">{t("dealsWhatsappReconnecting")}</p>{/if}
  <div class="mt-5 flex items-center justify-between gap-3">
   <h3 class="font-semibold">{t("dealsWhatsappPick")}</h3>
   <Button variant="secondary" small onclick={refresh} busy={pending==="refresh"} disabled={pending!==null}>{#if pending!=="refresh"}<RefreshCw class="size-4"/>{/if}{t("dealsWhatsappRefresh")}</Button>
  </div>
  {#if channels&&!channels.length}<p class="mt-3 text-sm text-neutral-500">{t("dealsWhatsappNoChannels")}</p>
  {:else if channels}
   <ul class="mt-3 grid gap-2">
    {#each channels as channel(channel.id)}
     <li class="flex items-center justify-between gap-3 rounded-xl border border-neutral-200 px-4 py-3 dark:border-neutral-800">
      <span class="min-w-0"><span class="block truncate font-semibold">{channel.name}</span><span class="block text-sm text-neutral-500">{channel.role==="owner"?t("dealsWhatsappOwner"):t("dealsWhatsappAdmin")}</span></span>
      {#if channel.added}<span class="shrink-0 text-sm text-neutral-500">{t("dealsWhatsappAdded")}</span>
      {:else}<Button small busy={adding===channel.id} disabled={pending!==null||!hasPlan()} onclick={()=>add(channel.id)}>{t("dealsWhatsappAdd")}</Button>{/if}
     </li>
    {/each}
   </ul>
  {/if}
  <div class="mt-6 flex justify-end"><Button variant="secondary" onclick={disconnect} busy={pending==="disconnect"} disabled={pending!==null}>{t("dealsWhatsappDisconnect")}</Button></div>
 {:else if shown}
  <ol class="grid list-decimal gap-1.5 pl-5 text-sm text-neutral-600 dark:text-neutral-300"><li>{t("dealsWhatsappStep1")}</li><li>{t("dealsWhatsappStep2")}</li><li>{t("dealsWhatsappStep3")}</li></ol>
  <div class="mt-5 flex flex-col items-center gap-3">
   <div class="relative">
    <img src={shown.qr} alt={t("dealsWhatsappQRAlt")} width="256" height="256" class="size-64 rounded-xl bg-white p-3 transition {stale?'opacity-40 blur-sm':''}"/>
    {#if stale}<div class="absolute inset-0 grid place-items-center"><Button onclick={generate} busy={pending==="generate"} disabled={pending!==null}>{#if pending!=="generate"}<RefreshCw class="size-4"/>{/if}{t("dealsWhatsappRegenerate")}</Button></div>{/if}
   </div>
   <div class="h-2 w-64 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800" role="progressbar" aria-label={t("dealsWhatsappQRTime")} aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(stale?0:remaining)}>
    <div class="h-full rounded-full bg-[#c2410c] transition-[width] duration-1000 ease-linear motion-reduce:transition-none" style:width="{stale?0:remaining}%"></div>
   </div>
   {#if stale}<p class="text-sm text-neutral-500" role="status">{t("dealsWhatsappQRExpired")}</p>
   {:else}<p class="text-sm text-neutral-500 tabular-nums">{t("dealsWhatsappExpiresIn",{time:clock(left)})}</p>{/if}
  </div>
 {:else}
  {#if ended(view.state)}<p class="mb-4 text-sm font-semibold text-amber-700 dark:text-amber-300">{t(endedText[view.state])}</p>{/if}
  <div class="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
   <p class="flex items-center gap-2 font-semibold"><TriangleAlert class="size-4 shrink-0"/>{t("dealsWhatsappRiskTitle")}</p>
   <ul class="mt-2 grid list-disc gap-1.5 pl-5"><li>{t("dealsWhatsappRisk1")}</li><li>{t("dealsWhatsappRisk2")}</li><li>{t("dealsWhatsappRisk3")}</li><li>{t("dealsWhatsappRisk4")}</li></ul>
  </div>
  <div class="mt-4"><Switch label={t("dealsWhatsappAccept")} tone="coral" bind:checked={accepted}/></div>
  <div class="mt-5 flex justify-end"><Button onclick={generate} busy={pending==="generate"} disabled={pending!==null||!accepted||!hasPlan()}>{t("dealsWhatsappGenerate")}</Button></div>
 {/if}
</div>
