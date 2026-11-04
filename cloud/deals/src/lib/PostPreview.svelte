<script lang="ts">
 import ImageOff from "@lucide/svelte/icons/image-off";
 import {t,language} from "#lib/i18n.svelte.ts";
 import {money} from "#lib/money.ts";
 // How a post looks in the channel: the photo, the title, the price, the
 // short link and, as the server writes it, the affiliate notice only when
 // the link carries the person's ID. `example` marks a made-up one.
 let {title,image=null,price=null,currency=null,store,affiliate,example=false}:{title:string;image?:string|null;price?:number|null;currency?:string|null;store:string;affiliate:boolean;example?:boolean}=$props();
 let broken=$state(false);
</script>
<figure class="max-w-sm overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
 {#if image&&!broken}<img src={image} alt="" class="aspect-video w-full bg-neutral-100 object-cover dark:bg-neutral-800" referrerpolicy="no-referrer" loading="lazy" onerror={()=>broken=true}/>
 {:else}<div class="grid aspect-video place-items-center bg-neutral-100 text-neutral-400 dark:bg-neutral-800" aria-hidden="true"><ImageOff class="size-8"/></div>{/if}
 <figcaption class="grid gap-1.5 p-4 text-sm">
  {#if example}<span class="text-xs font-bold tracking-wider text-blup uppercase">{t("dealsExample")}</span>{/if}
  <p class="font-bold break-words">{title}</p>
  {#if price!==null&&currency}<p class="font-semibold">{money(price,currency,language())}</p>{/if}
  <p class="break-all text-blup underline">deals.getblup.site/r/…</p>
  <p class="text-neutral-500">{store}{#if affiliate} · {t("dealsAdNotice")}{/if}</p>
 </figcaption>
</figure>
