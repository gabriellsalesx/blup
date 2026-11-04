<script lang="ts">
 import X from "@lucide/svelte/icons/x";
 import {t} from "#lib/i18n.svelte.ts";
 // A field whose words become tags: a comma or Enter closes one (a pasted
 // "a, b, c" makes three), Backspace on an empty field takes the last one
 // back, and pressing a tag removes it. add checks and cleans each word
 // (it may ask the server, so words are added one after another while the
 // person keeps typing); null leaves it out. At most max tags.
 let {id,label,tags=$bindable([]),max,maxlength=60,add,surface="bg-white dark:bg-neutral-900"}:{id:string;label:string;tags?:string[];max:number;maxlength?:number;add:(word:string)=>string|null|Promise<string|null>;surface?:string}=$props();
 let text=$state(""),focused=$state(false),input=$state<HTMLInputElement>();
 const full=$derived(tags.length>=max);
 const up=$derived(focused||tags.length>0||text.length>0);
 let queue=Promise.resolve();
 const commit=(raw:string)=>queue=queue.then(()=>addAll(raw));
 async function addAll(raw:string){
  for(const part of raw.split(",").map(w=>w.trim().replace(/\s+/g," ")).filter(Boolean)){
   if(tags.length>=max)break;
   const word=await add(part);
   if(word&&!tags.some(t=>t.toLowerCase()===word.toLowerCase()))tags.push(word);
  }
 }
 async function onkeydown(e:KeyboardEvent){
  if(e.key===","||e.key==="Enter"){e.preventDefault();const raw=text;text="";await commit(raw);}
  else if(e.key==="Backspace"&&!text&&tags.length){e.preventDefault();text=tags.pop()??"";}
 }
 async function onpaste(e:ClipboardEvent){const pasted=e.clipboardData?.getData("text")??"";if(!pasted.includes(","))return;e.preventDefault();await commit(text+pasted);text="";}
 async function onblur(){focused=false;if(text.trim()){const raw=text;text="";await commit(raw);}}
</script>

<div class="relative flex min-h-14 flex-wrap items-center gap-1.5 rounded-md border px-3 py-2.5 transition-colors {surface} {focused?'border-blup':'border-neutral-300 dark:border-neutral-700'}">
 <label for={id} class="pointer-events-none absolute left-2 px-1 transition-all duration-200 {surface} {up?'-top-2.5 text-[13px]':'top-1/2 -translate-y-1/2 text-base'} {focused?'text-blup':'text-neutral-500'}">{label}</label>
 {#each tags as tag,i(tag)}
  <button type="button" class="inline-flex max-w-full items-center gap-1 rounded-full bg-neutral-100 py-1 pr-2 pl-3 text-sm font-semibold hover:bg-red-50 hover:text-red-700 dark:bg-neutral-800 dark:hover:bg-red-950/50 dark:hover:text-red-300" aria-label={`${t("remove")}: ${tag}`} onclick={()=>{tags.splice(i,1);input?.focus();}}><span class="truncate">{tag}</span><X class="size-3.5 shrink-0"/></button>
 {/each}
 {#if !full}<input bind:this={input} {id} class="min-w-32 flex-1 border-none bg-transparent p-0 text-base outline-none" autocomplete="off" spellcheck={false} {maxlength} bind:value={text} {onkeydown} {onpaste} onfocus={()=>focused=true} {onblur}/>{/if}
</div>
