<script lang="ts">
 import ChevronRight from "@lucide/svelte/icons/chevron-right";
 import {t} from "#lib/i18n.svelte.ts";
 import Badge from "#lib/ui/Badge.svelte";
 import type {IssueEvent} from "./issues";
 // The exceptions of one occurrence, the last raised first, each with its
 // stack newest call first. Only the app's own frames show until all are
 // asked for (when the app has none, all show). A frame of the app that
 // came with its code opens to it, the line of the error marked; the first
 // one starts open. A frame of minified code that the release's source map
 // covers (`originals`, by "exception:frame" as stored) shows its original
 // file, line, function and code, and the minified place under it.
 // Everything here is what the SDK sent, cleaned on arrival.
 type Stored=IssueEvent["event"]["exceptions"][number]["frames"][number];
 type Frame={function:string;where:string;line:number|null;inApp:boolean;context?:{pre:string[];line:string;post:string[]};minified?:string};
 let {exceptions,originals={}}:{exceptions:IssueEvent["event"]["exceptions"];originals?:IssueEvent["originals"]}=$props();
 let all=$state(false);
 const placeOf=(file:string,line:number|null,column:number|null)=>[file,line,column].filter(v=>v!==null&&v!=="").join(":");
 const viewOf=(f:Stored,key:string):Frame=>{const o=originals[key];if(o)return {function:o.function||f.function,where:placeOf(o.file,o.line,o.column),line:o.line,inApp:o.inApp,context:o.context,minified:placeOf(f.filename,f.line,f.column)};return {function:f.function,where:placeOf(f.filename||f.module,f.line,f.column),line:f.line,inApp:f.inApp,context:f.context};};
 const list=$derived(exceptions.map((e,ei)=>({...e,frames:e.frames.map((f,fi)=>viewOf(f,`${ei}:${fi}`))})).reverse().map(e=>{const frames=[...e.frames].reverse(),own=frames.filter(f=>f.inApp);return {...e,frames:all||!own.length?frames:own,hidden:own.length&&!all?frames.length-own.length:0};}));
 const hidden=$derived(list.reduce((n,e)=>n+e.hidden,0));
 // Which frames are open, by exception and position; the first with code
 // starts open.
 const firstWithCode=$derived.by(()=>{for(const [e,exception] of list.entries()){const i=exception.frames.findIndex(f=>f.context);if(i>=0)return `${e}:${i}`;}return null;});
 let toggled=$state<Record<string,boolean>>({});
 const isOpen=(key:string)=>toggled[key]??key===firstWithCode;
 // The numbered lines of a frame's code, with the error's own line marked.
 const linesOf=(f:Frame)=>{const c=f.context;if(!c)return [];const start=(f.line??c.pre.length+1)-c.pre.length;return [...c.pre.map((text,i)=>({n:start+i,text,error:false})),{n:start+c.pre.length,text:c.line,error:true},...c.post.map((text,i)=>({n:start+c.pre.length+1+i,text,error:false}))];};
</script>

<!-- One column that never grows past the card: long code scrolls sideways. -->
<div class="grid grid-cols-[minmax(0,1fr)] gap-6">
 {#each list as exception,e(e)}
  <div class="min-w-0">
   <p class="font-mono text-sm break-words"><span class="font-bold text-red-600 dark:text-red-400">{exception.type}</span>{#if exception.value}<span class="text-neutral-700 dark:text-neutral-300">: {exception.value}</span>{/if}</p>
   {#if exception.frames.length}
    <ol class="mt-3 overflow-hidden rounded-lg border border-neutral-200 dark:border-neutral-800">
     {#each exception.frames as frame,i(i)}
      {@const key=`${e}:${i}`}
      <li class="border-t border-neutral-200 first:border-t-0 dark:border-neutral-800 {frame.inApp?"bg-white dark:bg-neutral-900":"bg-neutral-50 text-neutral-500 dark:bg-neutral-950/60"}">
       {#snippet head()}
        <span class="flex min-w-0 items-center gap-1.5 break-all">{#if frame.context}<ChevronRight class="size-3.5 shrink-0 transition-transform {isOpen(key)?"rotate-90":""}"/>{/if}<span><span class="font-semibold {frame.inApp?"text-neutral-900 dark:text-white":""}">{frame.function||t("pulseAnonymous")}</span>{#if frame.where}<span class="text-neutral-500"> · {frame.where}</span>{/if}{#if frame.minified}<span class="block text-xs text-neutral-500">{t("pulseMinifiedAt",{place:frame.minified})}</span>{/if}</span></span>
        <span class="flex gap-1.5">{#if frame.minified}<Badge tone="blue">{t("pulseSourceMapped")}</Badge>{/if}{#if frame.inApp}<Badge tone="neutral">{t("pulseInApp")}</Badge>{/if}</span>
       {/snippet}
       {#if frame.context}<button type="button" aria-expanded={isOpen(key)} onclick={()=>toggled[key]=!isOpen(key)} class="flex w-full flex-wrap items-center justify-between gap-x-4 gap-y-1 px-3 py-2 text-left font-mono text-[13px] hover:bg-neutral-50 dark:hover:bg-neutral-800/50">{@render head()}</button>
       {:else}<div class="flex w-full flex-wrap items-center justify-between gap-x-4 gap-y-1 px-3 py-2 font-mono text-[13px]">{@render head()}</div>{/if}
       {#if frame.context&&isOpen(key)}
        <div class="overflow-x-auto border-t border-neutral-200 bg-neutral-50 py-1.5 font-mono text-[13px] leading-6 dark:border-neutral-800 dark:bg-neutral-950">
         {#each linesOf(frame) as line(line.n)}
          <div class="flex min-w-max {line.error?"bg-red-500/10 text-red-700 dark:text-red-300":"text-neutral-700 dark:text-neutral-300"}"><span class="w-14 shrink-0 pr-4 text-right text-neutral-400 select-none {line.error?"border-l-2 border-red-500":"border-l-2 border-transparent"}">{line.n}</span><span class="pr-4 whitespace-pre">{line.text}</span></div>
         {/each}
        </div>
       {/if}
      </li>
     {/each}
    </ol>
   {:else}<p class="mt-2 text-sm text-neutral-500">{t("pulseNoStack")}</p>{/if}
  </div>
 {/each}
 {#if hidden||all}<button type="button" class="justify-self-start text-sm font-semibold text-neutral-600 underline hover:text-blup dark:text-neutral-300" onclick={()=>all=!all}>{all?t("pulseOwnFrames"):t("pulseAllFrames",{count:String(hidden)})}</button>{/if}
</div>
