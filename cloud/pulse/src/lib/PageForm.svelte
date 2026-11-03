<script lang="ts">
 import {untrack} from "svelte";
 import Plus from "@lucide/svelte/icons/plus";
 import {t} from "#lib/i18n.svelte.ts";
 import Button from "#lib/ui/Button.svelte";
 import FloatField from "#lib/ui/FloatField.svelte";
 import FormSection from "#lib/ui/FormSection.svelte";
 import Switch from "#lib/ui/Switch.svelte";
 import SlugField,{type SlugState} from "./SlugField.svelte";
 import {pulseApi,parseResponse} from "./api";
 import type {Monitor} from "./status-pages";
 // A status page's title and address, then the monitors it shows, each
 // with the name visitors see. While the page is new the address follows
 // the title until it is typed, and is checked as it changes; once the
 // page exists it is fixed.
 type Values={title:string;slug:string;components:{monitorId:string;name:string}[]};
 let {monitors,initial,editing=false,busy=false,onsave,oncancel}:{monitors:Monitor[];initial?:Values;editing?:boolean;busy?:boolean;onsave:(values:Values)=>void;oncancel:()=>void}=$props();
 // The fields start from `initial` and are the form's own from then on;
 // a different page remounts the form ({#key} in the page).
 const start=untrack(()=>initial);
 let title=$state(start?.title??""),slug=$state(start?.slug??""),chosen=$state<Record<string,string>>(Object.fromEntries((start?.components??[]).map(c=>[c.monitorId,c.name])));
 let slugState=$state<SlugState>("idle"),slugTyped=$state(false);
 const valid=$derived(Object.values(chosen).every(v=>v.trim().length>0)&&(editing||slugState==="available"));
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
 function toggle(id:string,on:boolean){if(on)chosen[id]="";else delete chosen[id];}
 function submit(e:SubmitEvent){e.preventDefault();onsave({title,slug,components:Object.entries(chosen).map(([monitorId,name])=>({monitorId,name:name.trim()}))});}
</script>

<form class="grid grid-cols-[minmax(0,1fr)] gap-10" onsubmit={submit}>
 <FormSection title={t("pulseBasicInfo")} help={t("pulseBasicHelp")}>
  <div class="grid grid-cols-[minmax(0,1fr)] gap-6"><FloatField id="page-title" surface="bg-white dark:bg-neutral-900" label={t("pulsePageTitle")} required maxlength={100} bind:value={title} oninput={titled}/><SlugField bind:value={slug} state={slugState} fixed={editing} oninput={typed}/></div>
 </FormSection>
 <FormSection title={t("pulseComponents")} help={t("pulseComponentsHelp")}>
  <div class="-my-1 grid grid-cols-[minmax(0,1fr)]">{#each monitors as monitor(monitor.id)}<div class="grid gap-3 border-b border-neutral-200 py-3 last:border-b-0 sm:grid-cols-[1fr_minmax(0,16rem)] sm:items-center dark:border-neutral-800"><Switch tone="mint" label={monitor.name} checked={Object.hasOwn(chosen,monitor.id)} onchange={on=>toggle(monitor.id,on)}/>{#if Object.hasOwn(chosen,monitor.id)}<FloatField id={`public-${monitor.id}`} surface="bg-white dark:bg-neutral-900" label={t("pulsePublicName")} maxlength={80} required bind:value={chosen[monitor.id]}/>{/if}</div>{:else}<p class="py-4 text-center text-sm text-neutral-500">{t("pulseNoMonitors")}</p>{/each}</div>
 </FormSection>
 <div class="flex flex-wrap justify-end gap-2"><Button type="button" variant="secondary" onclick={oncancel}>{t("cancel")}</Button><Button {busy} disabled={!valid}>{#if !editing}<Plus class="size-4"/>{/if}{editing?t("save"):t("pulseCreatePage")}</Button></div>
</form>
