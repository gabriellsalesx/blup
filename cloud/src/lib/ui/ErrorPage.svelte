<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
 import BlupLogo from "./BlupLogo.svelte";
 import Scene from "./Scene.svelte";
 import ThemePicker from "./ThemePicker.svelte";
 import LanguagePicker from "./LanguagePicker.svelte";
 // The error page both panels share: a drawing for the status, what
 // happened, and a way back. Each panel says where "back" goes and what
 // the missing page sends people back to.
 let {status,app="Blup",home,homeLabel,notFoundBody="pageNotFoundBody",mint=false,coral=false,languages=true}:{status:number;app?:string;home:string;homeLabel:string;notFoundBody?:PlainMessageKey;mint?:boolean;coral?:boolean;languages?:boolean}=$props();
 const drawings=import.meta.glob<string>("../assets/erros/*.svg",{query:"?raw",import:"default",eager:true});
 const drawing=(name:string)=>drawings[`../assets/erros/${name}.svg`]??drawings["../assets/erros/generico.svg"]??"";
 type Kind={title:PlainMessageKey;text:PlainMessageKey;art:string;retry?:boolean};
 const kinds=$derived<Record<string,Kind>>({
  offline:{title:"pageOffline",text:"pageOfflineBody",art:"offline",retry:true},
  400:{title:"pageInvalid",text:"pageInvalidBody",art:"generico"},
  401:{title:"pageUnauthorized",text:"pageUnauthorizedBody",art:"403"},
  403:{title:"pageForbidden",text:"pageForbiddenBody",art:"403"},
  404:{title:"pageNotFound",text:notFoundBody,art:"404"},
  429:{title:"pageRate",text:"pageRateBody",art:"429",retry:true},
  500:{title:"pageServer",text:"pageServerBody",art:"500",retry:true},
 });
 const offline=typeof navigator!=="undefined"&&!navigator.onLine;
 const kind=$derived(offline?kinds.offline:kinds[status]??kinds[500]);
</script>
<svelte:head><title>{t(kind.title)} · {app}</title></svelte:head>
<main class="grid min-h-dvh place-items-center px-6 py-16">
 <div class="grid w-full max-w-md justify-items-center text-center">
  <a href={home} aria-label={app}><BlupLogo {mint} {coral} class="h-8 w-auto text-[#0e1b45] dark:text-white" /></a>
  <Scene svg={drawing(kind.art)} class="mt-8 w-full max-w-sm" />
  {#if !offline&&status}<p class="mt-6 text-sm font-bold tracking-wider text-blup">{t("errorNumber",{status})}</p>{/if}
  <h1 class="mt-2 text-3xl font-extrabold tracking-tight">{t(kind.title)}</h1>
  <p class="mt-2 text-neutral-500">{t(kind.text)}</p>
  <div class="mt-8 flex flex-wrap justify-center gap-3">
   {#if kind.retry}<button class="flex h-12 items-center rounded-full bg-neutral-900 px-6 font-semibold text-white hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200" onclick={()=>location.reload()}>{t("retry")}</button>{/if}
   <a class="flex h-12 items-center rounded-full border border-neutral-300 px-6 font-semibold hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-900" href={home}>{homeLabel}</a>
  </div>
 </div>
</main>
<div class="fixed top-4 right-4 flex items-center gap-2">{#if languages}<LanguagePicker />{/if}<ThemePicker /></div>
