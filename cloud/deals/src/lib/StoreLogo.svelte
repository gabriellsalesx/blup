<script lang="ts">
 import mercadolivre from "./assets/mercadolivre.svg";
 import nuuvem from "./assets/nuuvem.png";
 import type {Store} from "./api";
 import {storeMarks} from "./store-marks";
 // A store's mark as an app tile: its own color behind the white mark
 // (Simple Icons). Mercado Livre is not in Simple Icons: its official mark
 // (the handshake), as its own site serves it, unchanged, on its yellow;
 // Nuuvem's is its own app icon (favicon-144.png from nuuvem.com),
 // unchanged, which is already a tile. AliExpress's mark is its name
 // written out, so it is drawn larger to be read.
 let {store,class:extra="size-14"}:{store:Store;class?:string}=$props();
 // Magalu and Terabyte wait for their official marks (TEMPORARY tile).
 const pending=$derived(store==="magalu"||store==="terabyte");
 const mark=$derived(store==="mercadolivre"||store==="nuuvem"||store==="magalu"||store==="terabyte"?null:storeMarks[store]);
</script>

{#if mark}
 <span class="grid shrink-0 place-items-center rounded-2xl {extra}" style:background-color={mark.color}><svg viewBox="0 0 24 24" class="fill-white {store==="aliexpress"?"size-[88%]":"size-1/2"}" role="img" aria-label={mark.title}><path d={mark.path}/></svg></span>
{:else if pending}
 <span class="grid shrink-0 place-items-center rounded-2xl bg-neutral-200 text-xl font-bold text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300 {extra}" aria-hidden="true">{store==="magalu"?"M":"T"}</span>
{:else if store==="nuuvem"}
 <img src={nuuvem} alt="Nuuvem" class="shrink-0 rounded-2xl object-cover {extra}"/>
{:else}
 <span class="grid shrink-0 place-items-center rounded-2xl bg-[#ffe600] {extra}"><img src={mercadolivre} alt="Mercado Livre" class="size-[78%]"/></span>
{/if}
