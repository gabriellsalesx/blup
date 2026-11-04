<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import {dealsApi,parseResponse,type Store,type StoreState} from "./api";
 import {dealsError} from "./errors";
 import {stores} from "./catalog";
 // One store's affiliate fields: what was saved never comes back (only a
 // hint the server chooses), so they start empty and saving replaces it.
 // `oncancel` closes a form not saved yet.
 let {store,saved,writable,onchange,oncancel}:{store:Store;saved:StoreState|undefined;writable:boolean;onchange:()=>void;oncancel?:()=>void}=$props();
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
   case "nuuvem":return {siteId:v("siteId")};
  }
 }
 async function run(fn:()=>Promise<void>){busy=true;try{await fn();}catch(e){toast(dealsError(e),{kind:"error"});}finally{busy=false;}}
 async function save(e:SubmitEvent){e.preventDefault();if(!ready)return;await run(async()=>{await parseResponse(dealsApi.affiliates[":store"].$put({param:{store},json:body()}));values={};toast(()=>t("dealsSaved"));onchange();});}
</script>

{#if saved?.configured}<p class="mb-4 flex flex-wrap gap-x-2 text-sm text-neutral-500">{#if saved.hint}<span class="font-mono">{saved.hint}</span><span aria-hidden="true">·</span>{/if}<span>{t("dealsSecretHidden")}</span></p>{/if}
<form class="grid grid-cols-[minmax(0,1fr)] gap-3" onsubmit={save}>
 {#each info.fields as field(field.id)}
  <FloatField id="{store}-{field.id}" surface="bg-white dark:bg-neutral-900" label={field.label?t(field.label):field.text??field.id} type={field.secret?"password":"text"} autocomplete="off" spellcheck={false} maxlength={200} required={!field.optional} disabled={!writable} bind:value={() => values[field.id] ?? "", (next) => (values[field.id] = next)}/>
 {/each}
 <div class="flex flex-wrap justify-end gap-2">
  {#if oncancel}<Button type="button" variant="secondary" onclick={oncancel}>{t("cancel")}</Button>{/if}
  <Button {busy} disabled={!ready||!writable}>{t("dealsSave")}</Button>
 </div>
</form>
