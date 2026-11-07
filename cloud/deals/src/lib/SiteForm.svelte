<script lang="ts">
 import {untrack,type Snippet} from "svelte";
 import Plus from "@lucide/svelte/icons/plus";
 import X from "@lucide/svelte/icons/x";
 import {t,language} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import TextArea from "#lib/ui/TextArea.svelte";
 import Select from "#lib/ui/Select.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import SlugField,{type SlugState} from "#lib/ui/SlugField.svelte";
 import {slugState,SITE_HOST,type SiteLanguage} from "./site";
 // The deals page, as Pulse's status page form: its name, address,
 // description and language; then its header (the name beside the logo or
 // not, and the logos once the page exists: `logos`, each sent as it is
 // picked) and up to five links. While the page is new the address follows
 // the name until it is typed, and is checked as it changes; once the page
 // exists it is fixed.
 type Link={label:string;url:string};
 type Values={title:string;slug:string;description:string;language:SiteLanguage;showTitle:boolean;links:Link[]};
 let {initial,editing=false,busy=false,disabled=false,logos,onsave}:{initial?:Values;editing?:boolean;busy?:boolean;disabled?:boolean;logos?:Snippet;onsave:(values:Values)=>void}=$props();
 const start=untrack(()=>initial);
 let title=$state(start?.title??""),slug=$state(start?.slug??""),description=$state(start?.description??""),lang=$state<SiteLanguage>(start?.language??language());
 let showTitle=$state(start?.showTitle??true),links=$state<Link[]>(start?.links.map(l=>({...l}))??[]);
 let address=$state<SlugState>("idle"),slugTyped=false;
 const languages=[{value:"pt" as const,label:"Português"},{value:"en" as const,label:"English"},{value:"es" as const,label:"Español"}];
 // A link needs a name and a full https address.
 const linkOk=(l:Link)=>{if(!l.label.trim())return false;try{const u=new URL(l.url.trim());return u.protocol==="https:"&&u.hostname.includes(".")&&!u.username&&!u.password;}catch{return false;}};
 const valid=$derived(title.trim().length>0&&links.every(linkOk)&&(editing||address==="available"));
 const slugify=(text:string)=>text.normalize("NFD").replace(/[̀-ͯ]/g,"").toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"").slice(0,40).replace(/-+$/,"");
 let timer:ReturnType<typeof setTimeout>|undefined,asked=0;
 // Asks the server a moment after typing stops; only the latest answer counts.
 function check(){
  clearTimeout(timer);const value=slug,ticket=++asked;
  if(!value){address="idle";return;}
  if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value)||value.length<3){address="invalid";return;}
  address="checking";
  timer=setTimeout(async()=>{try{const answer=await slugState(value);if(ticket===asked)address=answer;}catch{if(ticket===asked)address="idle";}},350);
 }
 function titled(){if(!editing&&!slugTyped){slug=slugify(title);check();}}
 function typed(){slugTyped=slug.length>0;check();}
 function submit(e:SubmitEvent){e.preventDefault();onsave({title:title.trim(),slug,description:description.trim(),language:lang,showTitle,links:links.map(l=>({label:l.label.trim(),url:l.url.trim()}))});}
</script>

<form class="grid grid-cols-[minmax(0,1fr)] gap-10" onsubmit={submit}>
 <FormSection title={t("dealsSiteBasic")} help={t("dealsSiteBasicHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6">
   <FloatField id="site-title" surface="bg-white dark:bg-neutral-900" label={t("dealsSiteName")} required maxlength={80} {disabled} bind:value={title} oninput={titled}/>
   <SlugField prefix={SITE_HOST} bind:value={slug} state={address} fixed={editing} oninput={typed}/>
   <TextArea id="site-description" surface="bg-white dark:bg-neutral-900" label={t("dealsDescription")} maxlength={300} rows={3} {disabled} bind:value={description}/>
   <div><Select surface="bg-white dark:bg-neutral-900" label={t("dealsSiteLanguage")} options={languages} {disabled} bind:value={lang}/><p class="mt-2 px-1 text-sm text-neutral-500">{t("dealsSiteLanguageHelp")}</p></div>
  </div>
 </FormSection>
 <FormSection title={t("pulsePageHeader")} help={t("pulsePageHeaderHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6">
   <Switch tone="coral" label={t("pulseShowTitle")} {disabled} checked={showTitle} onchange={on=>showTitle=on}/>
   {#if logos}{@render logos()}<p class="text-sm text-neutral-500">{t("pulseLogoHelp")}</p>
   {:else}<p class="text-sm text-neutral-500">{t("pulseLogoAfterCreate")}</p>{/if}
  </div>
 </FormSection>
 <FormSection title={t("pulseNavLinks")} help={t("pulseNavLinksHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-3">
   {#each links as link,i(i)}
    <div class="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3 sm:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]">
     <FloatField id={`link-label-${i}`} surface="bg-white dark:bg-neutral-900" label={t("pulseLinkLabel")} maxlength={40} required {disabled} bind:value={link.label}/>
     <div class="col-span-2 row-start-2 sm:col-span-1 sm:row-start-auto"><FloatField id={`link-url-${i}`} surface="bg-white dark:bg-neutral-900" label={t("pulseLinkURL")} type="url" maxlength={2048} required {disabled} bind:value={link.url}/>{#if link.url.trim()&&link.url.trim()!=="https://"&&!linkOk({label:"x",url:link.url})}<p class="mt-1 px-3 text-sm text-red-600 dark:text-red-400">{t("pulseLinkInvalid")}</p>{/if}</div>
     <button type="button" class="grid size-14 place-items-center rounded-md border border-neutral-300 text-neutral-500 hover:bg-neutral-100 hover:text-red-600 dark:border-neutral-700 dark:hover:bg-neutral-800" aria-label={t("pulseRemoveLink")} {disabled} onclick={()=>links.splice(i,1)}><X class="size-4"/></button>
    </div>
   {/each}
   {#if links.length<5}<Button small variant="secondary" type="button" class="justify-self-start" {disabled} onclick={()=>links.push({label:"",url:"https://"})}><Plus class="size-4"/>{t("pulseAddLink")}</Button>{/if}
  </div>
 </FormSection>
 <div class="flex justify-end"><Button {busy} disabled={!valid||disabled}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("dealsSiteCreate")}</Button></div>
</form>
