<script lang="ts">
 import {onMount} from "svelte";
 import type {InferResponseType} from "hono/client";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import FloatField from "#lib/ui/FloatField.svelte";
 import Button from "#lib/ui/Button.svelte";
 import {pulseApi,parseResponse} from "./api";
 const api=pulseApi.pages[":page"].domain;
 type Domain=InferResponseType<typeof api.$get,200>;
 let {page,eligible}:{page:string;eligible:boolean}=$props();
 let info=$state<Domain>(null),hostname=$state(""),busy=$state(false),error=$state<unknown>(null);
 async function run(fn:()=>Promise<void>){busy=true;error=null;try{await fn();}catch(e){error=e;}finally{busy=false;}}
 async function refresh(){info=await parseResponse(api.$get({param:{page}}));if(info)hostname=info.hostname;}
 async function connect(e:SubmitEvent){e.preventDefault();await run(async()=>{info=await parseResponse(api.$post({param:{page},json:{hostname}}));});}
 async function remove(){if(!await confirm({title:t("pulseDomainRemove"),message:t("pulseDomainRemoveAsk"),danger:true}))return;await run(async()=>{await parseResponse(api.$delete({param:{page}}));info=null;hostname="";});}
 onMount(()=>{void run(refresh);const timer=setInterval(()=>{if(!busy&&info&&info.status!=="active"&&document.visibilityState==="visible")void run(refresh);},60000);return()=>clearInterval(timer);});
</script>
<section class="my-6 rounded-xl border border-neutral-200 p-5 dark:border-neutral-800"><h3 class="font-semibold">{t("pulseDomain")}</h3><p class="my-3 text-sm text-neutral-500">{t("pulseDomainHelp")}</p>
 {#if error}<p class="my-3 text-sm text-red-600" role="alert">{errorMessage(error)}</p>{/if}
 {#if info}<p class="my-2 break-all font-semibold">{info.hostname}</p><p class="mb-3 text-sm" role="status">{t(info.status==="active"?"pulseActive":info.status==="pending_ownership"?"pulseDomainProof":info.status==="disabled"?"pulseDomainDisabled":info.status==="error"?"pulseDomainError":"pulseDomainPending")}</p><div class="grid gap-3">{#each info.records as record(`${record.type}:${record.name}:${record.value}`)}<div class="break-all rounded-lg bg-neutral-100 p-3 text-sm dark:bg-neutral-900"><b>{record.type}</b><code class="my-1 block">{record.name}</code><code>{record.value}</code></div>{/each}</div><div class="mt-4 flex flex-wrap gap-2"><Button variant="secondary" {busy} onclick={()=>run(refresh)}>{t("pulseDomainCheck")}</Button><Button variant="secondary" {busy} onclick={remove}>{t("pulseDomainRemove")}</Button></div>{/if}
 {#if eligible&&(!info||info.status==="disabled")}<form class="mt-4 grid gap-3" onsubmit={connect}><FloatField id="pulse-domain" label={t("pulseDomain")} required maxlength={253} bind:value={hostname}/><Button {busy}>{t("pulseDomainConnect")}</Button></form>{/if}
</section>
