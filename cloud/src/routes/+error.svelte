<script lang="ts">
 import {page} from "$app/state";
 import {t} from "#lib/i18n.svelte.ts";
 import BlupLogo from "#lib/ui/BlupLogo.svelte";
 import Scene from "#lib/ui/Scene.svelte";
 import ThemePicker from "#lib/ui/ThemePicker.svelte";
 import LanguagePicker from "#lib/ui/LanguagePicker.svelte";
 import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
 const drawings=import.meta.glob<string>("../lib/assets/erros/*.svg",{query:"?raw",import:"default",eager:true});
 const drawing=(name:string)=>drawings[`../lib/assets/erros/${name}.svg`]??drawings["../lib/assets/erros/generico.svg"]??"";
 type Kind={title:PlainMessageKey;text:PlainMessageKey;art:string;retry?:boolean};
 const kinds:Record<string,Kind>={
  offline:{title:"pageOffline",text:"pageOfflineBody",art:"offline",retry:true},
  400:{title:"pageInvalid",text:"pageInvalidBody",art:"generico"},
  401:{title:"pageUnauthorized",text:"pageUnauthorizedBody",art:"403"},
  403:{title:"pageForbidden",text:"pageForbiddenBody",art:"403"},
  404:{title:"pageNotFound",text:"pageNotFoundBody",art:"404"},
  429:{title:"pageRate",text:"pageRateBody",art:"429",retry:true},
  500:{title:"pageServer",text:"pageServerBody",art:"500",retry:true},
 };
 const offline=typeof navigator!=="undefined"&&!navigator.onLine;
 const kind=$derived(offline?kinds.offline:kinds[page.status]??kinds[500]);
 const onStaff=typeof location!=="undefined"&&location.hostname.startsWith("staff.");
 const home=onStaff?"/":page.status===401?"/entrar":"/conta";
</script>
<svelte:head><title>{t(kind.title)} · Blup</title></svelte:head>
<main class="grid min-h-dvh place-items-center px-6 py-16">
 <div class="grid w-full max-w-md justify-items-center text-center">
  <a href={home} aria-label="Blup"><BlupLogo class="h-8 w-auto text-[#0e1b45] dark:text-white" /></a>
  <Scene svg={drawing(kind.art)} class="mt-8 w-full max-w-sm" />
  {#if !offline&&page.status}<p class="mt-6 text-sm font-bold tracking-wider text-blup">{t("errorNumber",{status:page.status})}</p>{/if}
  <h1 class="mt-2 text-3xl font-extrabold tracking-tight">{t(kind.title)}</h1>
  <p class="mt-2 text-neutral-500">{t(kind.text)}</p>
  <div class="mt-8 flex flex-wrap justify-center gap-3">
   {#if kind.retry}<button class="flex h-12 items-center rounded-full bg-neutral-900 px-6 font-semibold text-white hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200" onclick={()=>location.reload()}>{t("retry")}</button>{/if}
   <a class="flex h-12 items-center rounded-full border border-neutral-300 px-6 font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900" href={home}>{page.status===401?t("signIn"):t("backAccount")}</a>
  </div>
 </div>
</main>
<div class="fixed top-4 right-4 flex items-center gap-2">{#if !onStaff}<LanguagePicker />{/if}<ThemePicker /></div>
