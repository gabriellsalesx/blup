<script lang="ts">
 import ImageOff from "@lucide/svelte/icons/image-off";
 import {t,language} from "#lib/i18n.svelte.ts";
 import {money} from "#lib/money.ts";
 import {tiktokMark} from "./store-marks";
 // How a post looks in the channel: the photo, the title, the price, the
 // short link and, as the server writes it, the affiliate notice only when
 // the link carries the person's ID. `example` marks a made-up one. A
 // video's news (`author` set) shows the video's place on top, as Telegram
 // shows its cover, and says it is a new video. extra and socials are the
 // account's own text and networks, under the link; description and
 // coupon, the post's own; style, the account's layout (emoji: a mark for
 // each part, as the server writes it).
 let {title,image=null,price=null,currency=null,store,affiliate,example=false,author=null,extra=null,socials=[],description=null,coupon=null,style="classic",discount=null}:{title:string;image?:string|null;price?:number|null;currency?:string|null;store:string;affiliate:boolean;example?:boolean;author?:string|null;extra?:string|null;socials?:readonly string[];description?:string|null;coupon?:string|null;style?:"classic"|"emoji";discount?:number|null}=$props();
 const shown=$derived(author?t("dealsVideoNews",{title}):title);
 const priced=$derived(price!==null&&currency?money(price,currency,language()):null);
 let broken=$state(false);
</script>
<figure class="max-w-sm overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
 {#if author}<div class="relative grid aspect-video place-items-center bg-neutral-950 text-white" aria-hidden="true">
  <svg viewBox="0 0 24 24" class="size-10" fill="currentColor"><path d={tiktokMark.path}/></svg>
  <span class="absolute bottom-3 left-3 max-w-[85%] truncate text-xs font-semibold text-white/80">@{author}</span>
 </div>
 {:else if image&&!broken}<img src={image} alt="" class="aspect-video w-full bg-neutral-100 object-cover dark:bg-neutral-800" referrerpolicy="no-referrer" loading="lazy" onerror={()=>broken=true}/>
 {:else}<div class="grid aspect-video place-items-center bg-neutral-100 text-neutral-400 dark:bg-neutral-800" aria-hidden="true"><ImageOff class="size-8"/></div>{/if}
 <figcaption class="grid gap-1.5 p-4 text-sm">
  {#if example}<span class="text-xs font-bold tracking-wider text-blup uppercase">{t("dealsExample")}</span>{/if}
  {#if style==="emoji"}
   <p class="font-bold break-words">🔥 {[shown,discount?"📉":""].filter(Boolean).join(" ")}</p>
   {#if description}<p class="mt-1.5 break-words">{description}</p>{/if}
   {#if priced||coupon}<div class="mt-1.5 grid gap-0.5">{#if priced}<p>💵 {t("dealsPostValue")}: <span class="font-semibold">{priced}</span></p>{/if}{#if coupon}<p class="break-words">🧾 {t("dealsCoupon")}: {coupon}</p>{/if}</div>{/if}
   <p class="mt-1.5 break-all">📌 {author?t("dealsWatchVideo"):t("dealsPostLink")}: <span class="text-blup underline">deals.getblup.site/r/…</span></p>
  {:else}
   <p class="font-bold break-words">{shown}</p>
   {#if description}<p class="break-words">{description}</p>{/if}
   {#if priced}<p class="font-semibold">{priced}</p>{/if}
   {#if coupon}<p class="break-words">{t("dealsCoupon")}: {coupon}</p>{/if}
   <p class="break-all text-blup underline">{#if author}{t("dealsWatchVideo")}{:else}deals.getblup.site/r/…{/if}</p>
  {/if}
  {#if extra}<p class="break-words">{extra}</p>{/if}
  {#if socials.length}<p class="break-words text-blup">{socials.join(" · ")}</p>{/if}
  <p class="text-neutral-500">{[store,affiliate?(author?t("dealsVideoAd"):t("dealsAdNotice")):""].filter(Boolean).join(" · ")}</p>
 </figcaption>
</figure>
