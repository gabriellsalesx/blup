<script lang="ts">
 import mercadolivre from "./assets/mercadolivre.svg";
 import type {Store} from "./api";
 import {storeMarks} from "./store-marks";
 // A store's mark as an app tile: its own color behind the white mark
 // (Simple Icons). Mercado Livre is not in Simple Icons: its official mark
 // (the handshake), as its own site serves it, unchanged, on its yellow. AliExpress's mark is its name written out, so it is
 // drawn larger to be read.
 let {store,class:extra="size-14"}:{store:Store;class?:string}=$props();
 const mark=$derived(store==="mercadolivre"?null:storeMarks[store]);
</script>

{#if mark}
 <span class="grid shrink-0 place-items-center rounded-2xl {extra}" style:background-color={mark.color}><svg viewBox="0 0 24 24" class="fill-white {store==="aliexpress"?"size-[88%]":"size-1/2"}" role="img" aria-label={mark.title}><path d={mark.path}/></svg></span>
{:else}
 <span class="grid shrink-0 place-items-center rounded-2xl bg-[#ffe600] {extra}"><img src={mercadolivre} alt="Mercado Livre" class="size-[78%]"/></span>
{/if}
