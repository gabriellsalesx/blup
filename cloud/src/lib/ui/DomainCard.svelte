<script lang="ts" module>
 // A page's own domain as the Worker shows it (Pulse's status pages and
 // the Deals page answer the same way).
 export type DomainInfo={hostname:string;status:string;records:{type:string;name:string;value:string}[]}|null;
</script>

<script lang="ts">
 import {onMount} from "svelte";
 import {t} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import FloatField from "./FloatField.svelte";
 import Button from "./Button.svelte";
 // The domain of a public page: the records to create while it waits,
 // checked again every minute; linked, checked or switched off with the
 // panel's own calls.
 let {eligible,help,removeAsk,load,connect,remove,errorText}:{eligible:boolean;help:string;removeAsk:string;load:()=>Promise<DomainInfo>;connect:(hostname:string)=>Promise<DomainInfo>;remove:()=>Promise<unknown>;errorText:(e:unknown)=>string}=$props();
 let info=$state<DomainInfo>(null),hostname=$state(""),busy=$state(false);
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(errorText(e),{kind:"error"});}finally{busy=false;}}
 async function refresh(){info=await load();if(info)hostname=info.hostname;}
 async function submit(e:SubmitEvent){e.preventDefault();await run(async()=>{info=await connect(hostname);});}
 async function drop(){if(!await confirm({title:t("pulseDomainRemove"),message:removeAsk,danger:true}))return;await run(async()=>{await remove();info=null;hostname="";toast(()=>t("pulseToastDeleted"));});}
 onMount(()=>{void run(refresh);const timer=setInterval(()=>{if(!busy&&info&&info.status!=="active"&&document.visibilityState==="visible")void run(refresh);},60000);return()=>clearInterval(timer);});
</script>
<section class="mt-6 rounded-xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"><h3 class="font-bold">{t("pulseDomain")}</h3><p class="mt-0.5 text-sm text-neutral-500">{help}</p>
 {#if info}<p class="my-2 break-all font-semibold">{info.hostname}</p><p class="mb-3 text-sm" role="status">{t(info.status==="active"?"pulseActive":info.status==="pending_ownership"?"pulseDomainProof":info.status==="disabled"?"pulseDomainDisabled":info.status==="error"?"pulseDomainError":"pulseDomainPending")}</p><div class="grid gap-3">{#each info.records as record(`${record.type}:${record.name}:${record.value}`)}<div class="break-all rounded-lg bg-neutral-100 p-3 text-sm dark:bg-neutral-800"><b>{record.type}</b><code class="my-1 block">{record.name}</code><code>{record.value}</code></div>{/each}</div><div class="mt-4 flex flex-wrap gap-2"><Button small variant="secondary" {busy} onclick={()=>run(refresh)}>{t("pulseDomainCheck")}</Button><Button small variant="secondary" {busy} onclick={drop}>{t("pulseDomainRemove")}</Button></div>{/if}
 {#if eligible&&(!info||info.status==="disabled")}<form class="mt-4 grid gap-3 sm:grid-cols-[1fr_auto] sm:items-center" onsubmit={submit}><FloatField id="page-domain" surface="bg-white dark:bg-neutral-900" label={t("pulseDomain")} required maxlength={253} bind:value={hostname}/><Button {busy} class="h-14">{t("pulseDomainConnect")}</Button></form>{/if}
</section>
