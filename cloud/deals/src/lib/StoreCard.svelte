<script lang="ts">
 import ExternalLink from "@lucide/svelte/icons/external-link";
 import {t} from "#lib/i18n.svelte.ts";
 import {confirm} from "#lib/confirm.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import {dealsApi,parseResponse,type Store,type StoreState} from "./api";
 import {dealsError} from "./errors";
 import {stores,storeName} from "./catalog";
 // One store's affiliate details: the steps to get them, the fields and
 // whether they are saved. What was saved never comes back (only a hint
 // the server chooses), so the fields start empty and saving replaces it.
 let {store,saved,writable,onchange}:{store:Store;saved:StoreState|undefined;writable:boolean;onchange:()=>void}=$props();
 const info=$derived(stores.find(s=>s.id===store)!);
 let values=$state<Record<string,string>>({}),busy=$state(false);
 const ready=$derived(info.fields.every(f=>f.optional||(values[f.id]??"").trim()));
 const v=(id:string)=>(values[id]??"").trim();
 // The body the server expects for each store.
 function body(){
  switch(store){
   case "amazon":return {tag:v("tag")};
   case "mercadolivre":return {word:v("word"),...(v("tool")?{tool:v("tool")}:{})};
   case "shopee":return {appId:v("appId"),secret:v("secret")};
   case "aliexpress":return {appKey:v("appKey"),appSecret:v("appSecret"),trackingId:v("trackingId")};
  }
 }
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function save(e:SubmitEvent){e.preventDefault();if(!ready)return;await run(async()=>{await parseResponse(dealsApi.affiliates[":store"].$put({param:{store},json:body()}));values={};toast(()=>t("dealsSaved"));onchange();});}
 async function remove(){if(!await confirm({title:t("dealsRemove"),message:t("dealsRemoveAffiliate"),danger:true}))return;await run(async()=>{await parseResponse(dealsApi.affiliates[":store"].$delete({param:{store}}));values={};toast(()=>t("dealsRemoved"));onchange();});}
</script>

<Card title={storeName[store]}>
 {#snippet action()}{#if saved?.configured}<Badge tone="green">{t("dealsConfigured")}</Badge>{:else}<Badge>{t("dealsNotConfigured")}</Badge>{/if}{/snippet}
 <ol class="mt-4 grid list-decimal gap-1.5 pl-5 text-sm text-neutral-600 dark:text-neutral-300">
  {#each info.steps as step(step)}<li>{t(step)}</li>{/each}
 </ol>
 <a class="mt-3 inline-flex items-center gap-1.5 text-sm font-semibold text-blup underline" href={info.site} target="_blank" rel="noopener noreferrer">{t("dealsStoreSite")}<ExternalLink class="size-3.5"/></a>
 {#if saved?.configured}<p class="mt-4 text-sm text-neutral-500">{#if saved.hint}<span class="font-mono">{saved.hint}</span> · {/if}{t("dealsSecretHidden")}</p>{/if}
 <form class="mt-5 grid gap-3" onsubmit={save}>
  {#each info.fields as field(field.id)}
   <FloatField id="{store}-{field.id}" surface="bg-white dark:bg-neutral-900" label={field.label?t(field.label):field.text??field.id} type={field.secret?"password":"text"} autocomplete="off" spellcheck={false} maxlength={200} required={!field.optional} disabled={!writable} bind:value={() => values[field.id] ?? "", (next) => (values[field.id] = next)}/>
  {/each}
  <div class="flex flex-wrap justify-end gap-2">
   {#if saved?.configured&&writable}<Button type="button" variant="secondary" {busy} onclick={remove}>{t("dealsRemove")}</Button>{/if}
   <Button {busy} disabled={!ready||!writable}>{t("dealsSave")}</Button>
  </div>
 </form>
</Card>
