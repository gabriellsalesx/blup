<script lang="ts">
 import {untrack} from "svelte";
 import Plus from "@lucide/svelte/icons/plus";
 import X from "@lucide/svelte/icons/x";
 import ChevronDown from "@lucide/svelte/icons/chevron-down";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import SlugField,{type SlugState} from "#lib/ui/SlugField.svelte";
 import LogoField from "#lib/ui/LogoField.svelte";
 import {uploadLogo,removeLogo} from "./logo-image";
 import {pulseApi,parseResponse} from "./api";
 import type {Monitor} from "./status-pages";
 // A status page's title and address, the monitors it shows (each with
 // the name visitors see and, if wanted, what it is: a help icon on the
 // public page), then its header: the title shown or not, its
 // logos (once the page exists: each is sent as it is picked) and up to
 // five links. While the page is new the address follows the title until
 // it is typed, and is checked as it changes; once the page exists it is
 // fixed.
 type Link={label:string;url:string};
 type Component={monitorId:string;name:string;description:string};
 type Values={title:string;slug:string;components:Component[];showTitle:boolean;links:Link[]};
 type Logos={page:string;light:string|null;dark:string|null};
 let {monitors,initial,logos,editing=false,busy=false,onsave,oncancel}:{monitors:Monitor[];initial?:Values;logos?:Logos;editing?:boolean;busy?:boolean;onsave:(values:Values)=>void;oncancel:()=>void}=$props();
 // The fields start from `initial` and are the form's own from then on;
 // a different page remounts the form ({#key} in the page).
 const start=untrack(()=>initial);
 let showTitle=$state(start?.showTitle??true),links=$state<Link[]>(start?.links.map(l=>({...l}))??[]);
 let title=$state(start?.title??""),slug=$state(start?.slug??""),chosen=$state<Record<string,{name:string;description:string}>>(Object.fromEntries((start?.components??[]).map(c=>[c.monitorId,{name:c.name,description:c.description}])));
 let slugState=$state<SlugState>("idle"),slugTyped=$state(false);
 // A link needs a name and a full https address.
 const linkOk=(l:Link)=>{if(!l.label.trim())return false;try{const u=new URL(l.url.trim());return u.protocol==="https:"&&u.hostname.includes(".")&&!u.username&&!u.password;}catch{return false;}};
 const valid=$derived(Object.values(chosen).every(v=>v.name.trim().length>0)&&links.every(linkOk)&&(editing||slugState==="available"));
 const slugify=(text:string)=>text.normalize("NFD").replace(/[̀-ͯ]/g,"").toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"").slice(0,40).replace(/-+$/,"");
 let timer:ReturnType<typeof setTimeout>|undefined,asked=0;
 // Asks the server a moment after typing stops; only the latest answer counts.
 function check(){
  clearTimeout(timer);const value=slug,ticket=++asked;
  if(!value){slugState="idle";return;}
  if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value)||value.length<3){slugState="invalid";return;}
  slugState="checking";
  timer=setTimeout(async()=>{try{const result=await parseResponse(pulseApi.pages.slug.$get({query:{slug:value}}));if(ticket===asked)slugState=result.state;}catch{if(ticket===asked)slugState="idle";}},350);
 }
 function titled(){if(!editing&&!slugTyped){slug=slugify(title);check();}}
 function typed(){slugTyped=slug.length>0;check();}
 // Each chosen service folds to one line; one just switched on opens to be
 // filled, and one without its public name stays open.
 let opened=$state<Record<string,boolean>>({});
 const isOpen=(id:string)=>opened[id]??!chosen[id]?.name.trim();
 function toggle(id:string,on:boolean){if(on){chosen[id]={name:"",description:""};opened[id]=true;}else{delete chosen[id];delete opened[id];}}
 function submit(e:SubmitEvent){e.preventDefault();onsave({title,slug,components:Object.entries(chosen).map(([monitorId,c])=>({monitorId,name:c.name.trim(),description:c.description.trim()})),showTitle,links:links.map(l=>({label:l.label.trim(),url:l.url.trim()}))});}
</script>

<form class="grid grid-cols-[minmax(0,1fr)] gap-10" onsubmit={submit}>
 <FormSection title={t("pulseBasicInfo")} help={t("pulseBasicHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6"><FloatField id="page-title" surface="bg-white dark:bg-neutral-900" label={t("pulsePageTitle")} required maxlength={100} bind:value={title} oninput={titled}/><SlugField prefix="pulse.getblup.site/" bind:value={slug} state={slugState} fixed={editing} oninput={typed}/></div>
 </FormSection>
 <FormSection title={t("pulseComponents")} help={t("pulseComponentsHelp")}>
  <div class="-my-1 grid grid-cols-[minmax(0,1fr)]">
   {#each monitors as monitor(monitor.id)}
    {@const item=chosen[monitor.id]}
    <div class="border-b border-neutral-200 py-4 last:border-b-0 dark:border-neutral-800">
     <div class="flex items-center gap-3">
      <div class="min-w-0 flex-1"><Switch tone="mint" label={monitor.name} checked={!!item} onchange={on=>toggle(monitor.id,on)}/></div>
      {#if item}<button type="button" class="flex min-w-0 shrink items-center gap-2 rounded-md px-2 py-1 text-sm text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white" aria-expanded={isOpen(monitor.id)} aria-controls={`service-${monitor.id}`} aria-label={isOpen(monitor.id)?t("pulseCollapse"):t("pulseExpand")} onclick={()=>opened[monitor.id]=!isOpen(monitor.id)}><span class="hidden truncate sm:inline">{item.name.trim()||t("pulsePublicName")}</span>{#if item.description.trim()}<span class="hidden text-xs sm:inline">· {t("pulseHasExplanation")}</span>{/if}<ChevronDown class="size-4 shrink-0 transition-transform {isOpen(monitor.id)?"rotate-180":""}"/></button>{/if}
     </div>
     {#if item&&isOpen(monitor.id)}
      <div id={`service-${monitor.id}`} class="mt-4 grid grid-cols-[minmax(0,1fr)] gap-4 sm:pl-14">
       <FloatField id={`public-${monitor.id}`} surface="bg-white dark:bg-neutral-900" label={t("pulsePublicName")} maxlength={80} required bind:value={chosen[monitor.id].name}/>
       <div><FloatField id={`explain-${monitor.id}`} surface="bg-white dark:bg-neutral-900" label={t("pulseExplanation")} maxlength={200} bind:value={chosen[monitor.id].description}/><p class="mt-2 px-3 text-sm text-neutral-500">{t("pulseExplanationHelp")}</p></div>
      </div>
     {/if}
    </div>
   {:else}<p class="py-4 text-center text-sm text-neutral-500">{t("pulseNoMonitors")}</p>{/each}
  </div>
 </FormSection>
 <FormSection title={t("pulsePageHeader")} help={t("pulsePageHeaderHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6">
   <Switch tone="mint" label={t("pulseShowTitle")} checked={showTitle} onchange={on=>showTitle=on}/>
   {#if logos}
    <LogoField variant="light" label={t("pulseLogoLight")} preview={logos.light} upload={logo=>uploadLogo(logos.page,"light",logo)} remove={()=>removeLogo(logos.page,"light")}/>
    <LogoField variant="dark" label={t("pulseLogoDark")} preview={logos.dark} upload={logo=>uploadLogo(logos.page,"dark",logo)} remove={()=>removeLogo(logos.page,"dark")}/>
    <p class="text-sm text-neutral-500">{t("pulseLogoHelp")}</p>
   {:else}<p class="text-sm text-neutral-500">{t("pulseLogoAfterCreate")}</p>{/if}
  </div>
 </FormSection>
 <FormSection title={t("pulseNavLinks")} help={t("pulseNavLinksHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-3">
   {#each links as link,i(i)}
    <div class="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3 sm:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]">
     <FloatField id={`link-label-${i}`} surface="bg-white dark:bg-neutral-900" label={t("pulseLinkLabel")} maxlength={40} required bind:value={link.label}/>
     <div class="col-span-2 row-start-2 sm:col-span-1 sm:row-start-auto"><FloatField id={`link-url-${i}`} surface="bg-white dark:bg-neutral-900" label={t("pulseLinkURL")} type="url" maxlength={2048} required bind:value={link.url}/>{#if link.url.trim()&&link.url.trim()!=="https://"&&!linkOk({label:"x",url:link.url})}<p class="mt-1 px-3 text-sm text-red-600 dark:text-red-400">{t("pulseLinkInvalid")}</p>{/if}</div>
     <button type="button" class="grid size-14 place-items-center rounded-md border border-neutral-300 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:border-neutral-700 dark:hover:bg-neutral-800" aria-label={t("pulseRemoveLink")} onclick={()=>links.splice(i,1)}><X class="size-4"/></button>
    </div>
   {/each}
   {#if links.length<5}<Button small variant="secondary" type="button" class="justify-self-start" onclick={()=>links.push({label:"",url:"https://"})}><Plus class="size-4"/>{t("pulseAddLink")}</Button>{/if}
  </div>
 </FormSection>
 <div class="flex flex-wrap justify-end gap-2"><Button type="button" variant="secondary" onclick={oncancel}>{t("cancel")}</Button><Button {busy} disabled={!valid}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("pulseCreatePage")}</Button></div>
</form>
