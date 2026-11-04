<script lang="ts">
 import {onMount} from "svelte";
 import Sparkles from "@lucide/svelte/icons/sparkles";
 import {t,language} from "#lib/i18n.svelte.ts";
 import {money} from "#lib/money.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import {dealsApi,parseResponse,type AIUsage} from "./api";
 import {dealsError} from "./errors";
 import {hasAddress} from "../../../shared/deals-post";
 // What a post may add about its product: a few words (typed, or written
 // by the AI with the plan's allowance, shown in percent of the month)
 // and the coupon. No addresses: the link is the post's own.
 let {title,store,price=null,currency=null,description=$bindable(""),coupon=$bindable("")}:{title:string;store:string;price?:number|null;currency?:string|null;description?:string;coupon?:string}=$props();
 let ai=$state<AIUsage|null>(null),writing=$state(false);
 onMount(async()=>{try{ai=await parseResponse(dealsApi.ai.$get());}catch{ai=null;}});
 async function write(){writing=true;try{
  const r=await parseResponse(dealsApi.offers.describe.$post({json:{title,store,...(price!==null&&currency?{price:money(price,currency,language())}:{})}}));
  description=r.text;
  try{ai=await parseResponse(dealsApi.ai.$get());}catch{}
 }catch(e){toast(dealsError(e),{kind:"error"});}finally{writing=false;}}
</script>

<div class="grid gap-4">
 <div>
  <TextArea id="post-description" surface="bg-white dark:bg-neutral-900" label={t("dealsDescription")} maxlength={300} rows={3} bind:value={description} aria-invalid={hasAddress(description)}/>
  {#if hasAddress(description)}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsNoAddress")}</p>{/if}
  <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-2">
   {#if ai?.available}
    <Button type="button" variant="secondary" busy={writing} disabled={ai.percent>=100||!title.trim()} onclick={write}><Sparkles class="size-4"/>{t("dealsWriteWithAI")}</Button>
    <span class="text-sm text-neutral-500">{t("dealsAIUsage",{percent:ai.percent})}</span>
   {:else if ai}<span class="text-sm text-neutral-500">{t("dealsAINotInPlan")}</span>{/if}
  </div>
 </div>
 <div>
  <FloatField id="post-coupon" surface="bg-white dark:bg-neutral-900" label={t("dealsCouponField")} autocomplete="off" maxlength={60} bind:value={coupon} aria-invalid={hasAddress(coupon)}/>
  {#if hasAddress(coupon)}<p class="mt-1.5 text-sm text-red-700 dark:text-red-400">{t("dealsNoAddress")}</p>{/if}
 </div>
</div>
