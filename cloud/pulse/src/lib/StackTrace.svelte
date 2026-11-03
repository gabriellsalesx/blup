<script lang="ts">
 import ChevronRight from "@lucide/svelte/icons/chevron-right";
 import {t} from "#lib/i18n.svelte.ts";
 import Badge from "#lib/ui/Badge.svelte";
 import type {IssueEvent} from "./issues";
 // The exceptions of one occurrence, the last raised first, each with its
 // stack newest call first. Only the app's own frames show until all are
 // asked for (when the app has none, all show). A frame of the app that
 // came with its code opens to it, the line of the error marked; the first
 // one starts open. Everything here is what the SDK sent, cleaned on
 // arrival.
 type Frame=IssueEvent["event"]["exceptions"][number]["frames"][number];
 let {exceptions}:{exceptions:IssueEvent["event"]["exceptions"]}=$props();
 let all=$state(false);
 const list=$derived([...exceptions].reverse().map(e=>{const frames=[...e.frames].reverse(),own=frames.filter(f=>f.inApp);return {...e,frames:all||!own.length?frames:own,hidden:own.length&&!all?frames.length-own.length:0};}));
 const hidden=$derived(list.reduce((n,e)=>n+e.hidden,0));
 // Which frames are open, by exception and position; the first with code
 // starts open.
 const firstWithCode=$derived.by(()=>{for(const [e,exception] of list.entries()){const i=exception.frames.findIndex(f=>f.context);if(i>=0)return `${e}:${i}`;}return null;});
 let toggled=$state<Record<string,boolean>>({});
 const isOpen=(key:string)=>toggled[key]??key===firstWithCode;
 const where=(f:Frame)=>[f.filename||f.module,f.line,f.column].filter(v=>v!==null&&v!=="").join(":");
 // The numbered lines of a frame's code, with the error's own line marked.
 const linesOf=(f:Frame)=>{if(!f.context)return [];const start=(f.line??f.context.pre.length+1)-f.context.pre.length;return [...f.context.pre.map((text,i)=>({n:start+i,text,error:false})),{n:start+f.context.pre.length,text:f.context.line,error:true},...f.context.post.map((text,i)=>({n:start+f.context!.pre.length+1+i,text,error:false}))];};
</script>

<div class="grid gap-6">
 {#each list as exception,e(e)}
  <div>
   <p class="font-mono text-sm break-words"><span class="font-bold text-red-600 dark:text-red-400">{exception.type}</span>{#if exception.value}<span class="text-neutral-700 dark:text-neutral-300">: {exception.value}</span>{/if}</p>
   {#if exception.frames.length}
    <ol class="mt-3 overflow-hidden rounded-lg border border-neutral-200 dark:border-neutral-800">
     {#each exception.frames as frame,i(i)}
      {@const key=`${e}:${i}`}
      <li class="border-t border-neutral-200 first:border-t-0 dark:border-neutral-800 {frame.inApp?"bg-white dark:bg-neutral-900":"bg-neutral-50 text-neutral-500 dark:bg-neutral-950/60"}">
       {#snippet head()}
        <span class="flex min-w-0 items-center gap-1.5 break-all">{#if frame.context}<ChevronRight class="size-3.5 shrink-0 transition-transform {isOpen(key)?"rotate-90":""}"/>{/if}<span><span class="font-semibold {frame.inApp?"text-neutral-900 dark:text-white":""}">{frame.function||t("pulseAnonymous")}</span>{#if where(frame)}<span class="text-neutral-500"> · {where(frame)}</span>{/if}</span></span>
        {#if frame.inApp}<Badge tone="neutral">{t("pulseInApp")}</Badge>{/if}
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
