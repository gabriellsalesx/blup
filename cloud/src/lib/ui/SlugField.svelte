<script lang="ts" module>
 export type SlugState="idle"|"checking"|"available"|"taken"|"reserved"|"invalid";
</script>

<script lang="ts">
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import CircleX from "@lucide/svelte/icons/circle-x";
 import LoaderCircle from "@lucide/svelte/icons/loader-circle";
 import {t} from "#lib/i18n.svelte.ts";
 // The public address of a page (a Pulse status page, the Deals page): its
 // host in front of what is typed, a mark that says whether it is free, and
 // the full address below. Once the page exists it is only shown.
 let {value=$bindable(),state,prefix,fixed=false,surface="bg-white dark:bg-neutral-900",oninput}:{value:string;state:SlugState;prefix:string;fixed?:boolean;surface?:string;oninput?:()=>void}=$props();
 const message=$derived(fixed?t("pulseSlugFixed"):state==="checking"?t("pulseSlugChecking"):state==="taken"?t("pulseSlugTaken"):state==="reserved"?t("pulseSlugReserved"):state==="invalid"&&value?t("pulseSlugInvalid"):t("pulseSlugHelp"));
 const bad=$derived(!fixed&&(state==="taken"||state==="reserved"||(state==="invalid"&&!!value)));
</script>

<div>
 <div class="relative">
 <span class="pointer-events-none absolute -top-2.5 left-2 z-10 px-1 text-[13px] text-neutral-500 {surface}">{t("pulseSlugLabel")}</span>
 <div class="flex min-h-14 items-stretch overflow-hidden rounded-md border {bad?'border-red-500':'border-neutral-300 focus-within:border-blup dark:border-neutral-700'} {surface}">
  <span class="flex shrink-0 items-center border-r border-neutral-200 bg-neutral-50 px-3 text-sm text-neutral-500 dark:border-neutral-800 dark:bg-neutral-800/60">{prefix}</span>
  <input id="page-slug" class="min-w-0 flex-1 bg-transparent px-3 text-base outline-none disabled:text-neutral-500" aria-label={t("pulseSlugLabel")} aria-invalid={bad} aria-describedby="page-slug-help" autocomplete="off" spellcheck="false" required maxlength={40} disabled={fixed} bind:value {oninput}/>
  {#if !fixed&&state!=="idle"}<span class="flex shrink-0 items-center pr-3" aria-hidden="true">{#if state==="checking"}<LoaderCircle class="size-5 animate-spin text-neutral-400"/>{:else if state==="available"}<CircleCheck class="size-5 text-emerald-500"/>{:else if bad}<CircleX class="size-5 text-red-500"/>{/if}</span>{/if}
 </div>
 </div>
 <p id="page-slug-help" class="mt-2 px-1 text-sm {bad?'text-red-600 dark:text-red-400':state==='available'&&!fixed?'text-emerald-700 dark:text-emerald-400':'text-neutral-500'}" aria-live="polite">{state==="available"&&!fixed?`${t("pulseSlugAvailable")}: ${prefix}${value}/`:message}</p>
</div>
