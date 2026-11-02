<script lang="ts">
 import Globe from "@lucide/svelte/icons/globe";
 import {language,setLanguage,t} from "#lib/i18n.svelte.ts";
 const id=$props.id();
 const choices=[{code:"pt",label:"Português"},{code:"en",label:"English"},{code:"es",label:"Español"}] as const;
</script>
<div class="inline-flex">
 <button type="button" popovertarget={id} aria-label={`${t("language")}: ${language()}`} style:anchor-name={`--language-${id}`} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-sm font-semibold text-neutral-700 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-200"><Globe class="size-4" /><span class="uppercase">{language()}</span></button>
 <div {id} popover role="menu" style:position-anchor={`--language-${id}`} class="language-menu rounded-xl border border-neutral-200 bg-white p-1.5 text-sm shadow-lg dark:border-neutral-800 dark:bg-neutral-900 dark:text-white">
  {#each choices as choice (choice.code)}<button role="menuitemradio" lang={choice.code} aria-checked={language()===choice.code} class="block w-full rounded-lg px-3 py-2 text-left hover:bg-neutral-100 dark:hover:bg-neutral-800" onclick={(event)=>{setLanguage(choice.code);const menu=event.currentTarget.closest("[popover]");if(menu instanceof HTMLElement)menu.hidePopover();}}>{choice.label}</button>{/each}
 </div>
</div>
<style>.language-menu{position:fixed;inset:auto;top:3.5rem;right:1rem;top:anchor(bottom);right:anchor(right);margin-top:.5rem;min-width:10rem;position-try-fallbacks:flip-inline,flip-block;}</style>
