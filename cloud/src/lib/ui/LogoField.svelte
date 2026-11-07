<script lang="ts">
 import {onDestroy} from "svelte";
 import CloudUpload from "@lucide/svelte/icons/cloud-upload";
 import LoaderCircle from "@lucide/svelte/icons/loader-circle";
 import Trash2 from "@lucide/svelte/icons/trash-2";
 import {t} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import {optimizeLogo,LogoError,LOGO_INPUT_TYPES,type LogoVariant,type LogoProblem,type MadeLogo} from "#lib/logo-image.ts";
 // One logo of a public page: a dashed area to drop or pick an image
 // (made small in the browser and sent at once, by the panel's own
 // `upload`), then the logo on its own kind of background, to change or
 // remove.
 let {variant,label,preview,upload,remove,disabled=false}:{variant:LogoVariant;label:string;preview:string|null;upload:(logo:MadeLogo)=>Promise<unknown>;remove:()=>Promise<unknown>;disabled?:boolean}=$props();
 // What this field sent or removed wins over the preview it came with
 // (which the page does not refresh after a save).
 let local=$state<string|null|undefined>(undefined),busy=$state(false),over=$state(false);
 const shown=$derived(local===undefined?preview:local);
 const id=$derived(`logo-${variant}`);
 const problems:Record<LogoProblem,()=>string>={type:()=>t("pulseLogoErrorType"),size:()=>t("pulseLogoErrorSize"),unreadable:()=>t("pulseLogoErrorUnreadable"),heavy:()=>t("pulseLogoErrorHeavy")};
 // Addresses this field made for previews are freed when it goes.
 const made:string[]=[];
 onDestroy(()=>{for(const url of made)URL.revokeObjectURL(url);});
 async function take(file:File|undefined){
  if(!file||busy||disabled)return;busy=true;
  try{const logo=await optimizeLogo(file);await upload(logo);const url=URL.createObjectURL(logo.blob);made.push(url);local=url;toast(()=>t("pulseToastSaved"));}
  catch(e){toast(e instanceof LogoError?problems[e.problem]():errorMessage(e),{kind:"error"});}
  finally{busy=false;}
 }
 async function drop(){busy=true;try{await remove();local=null;toast(()=>t("pulseToastDeleted"));}catch(e){toast(errorMessage(e),{kind:"error"});}finally{busy=false;}}
 function picked(e:Event&{currentTarget:HTMLInputElement}){const file=e.currentTarget.files?.[0];e.currentTarget.value="";void take(file);}
 function dropped(e:DragEvent){e.preventDefault();over=false;void take(e.dataTransfer?.files[0]);}
</script>

<div>
 <p class="mb-2 text-sm font-medium text-neutral-600 dark:text-neutral-400">{label}</p>
 <input {id} type="file" accept={LOGO_INPUT_TYPES.join(",")} class="sr-only" disabled={disabled||busy} onchange={picked}/>
 {#if shown}
  <div class="flex flex-wrap items-center gap-3 rounded-lg border border-neutral-200 p-3 dark:border-neutral-800">
   <span class="grid h-16 min-w-0 flex-1 place-items-center rounded-md px-4 {variant==="light"?"bg-white":"bg-neutral-950"}"><img src={shown} alt={label} class="max-h-10 max-w-full object-contain"/></span>
   {#if !disabled}<span class="flex gap-2">
    <label for={id} class="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold transition-colors hover:bg-neutral-100 dark:border-neutral-700 dark:hover:bg-neutral-800">{#if busy}<LoaderCircle class="size-4 animate-spin"/>{:else}<CloudUpload class="size-4"/>{/if}{t("pulseLogoChange")}</label>
    <button type="button" class="inline-flex items-center gap-2 rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-semibold text-red-600 transition-colors hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:text-red-400 dark:hover:bg-neutral-800" disabled={busy} onclick={drop}><Trash2 class="size-4"/>{t("pulseLogoRemove")}</button>
   </span>{/if}
  </div>
 {:else}
  <label for={id} class="flex min-h-24 cursor-pointer items-center justify-center gap-2 rounded-lg border border-dashed px-4 text-neutral-500 transition-colors {over?"border-blup bg-blup/5 text-blup":"border-neutral-300 hover:border-neutral-400 dark:border-neutral-700 dark:hover:border-neutral-600"} {disabled?"pointer-events-none opacity-50":""}" ondragover={e=>{e.preventDefault();over=true;}} ondragleave={()=>over=false} ondrop={dropped}>
   {#if busy}<LoaderCircle class="size-5 animate-spin"/>{t("pulseLogoSaving")}{:else}<CloudUpload class="size-5"/>{t("pulseLogoDrop")}{/if}
  </label>
 {/if}
</div>
