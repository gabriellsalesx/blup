<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 import Badge from "#lib/ui/Badge.svelte";
 import type {IssueEvent} from "./issues";
 // The exceptions of one occurrence, the last raised first, each with its
 // stack newest call first. Only the app's own frames show until all are
 // asked for (when the app has none, all show). The file names, functions
 // and lines are what the SDK sent, cleaned on arrival.
 let {exceptions}:{exceptions:IssueEvent["event"]["exceptions"]}=$props();
 let all=$state(false);
 const list=$derived([...exceptions].reverse().map(e=>{const frames=[...e.frames].reverse(),own=frames.filter(f=>f.inApp);return {...e,frames:all||!own.length?frames:own,hidden:own.length&&!all?frames.length-own.length:0};}));
 const hidden=$derived(list.reduce((n,e)=>n+e.hidden,0));
 const where=(f:IssueEvent["event"]["exceptions"][number]["frames"][number])=>[f.filename||f.module,f.line,f.column].filter(v=>v!==null&&v!=="").join(":");
</script>

<div class="grid gap-6">
 {#each list as exception,index(index)}
  <div>
   <p class="font-mono text-sm break-words"><span class="font-bold text-red-600 dark:text-red-400">{exception.type}</span>{#if exception.value}<span class="text-neutral-700 dark:text-neutral-300">: {exception.value}</span>{/if}</p>
   {#if exception.frames.length}
    <ol class="mt-3 overflow-hidden rounded-lg border border-neutral-200 dark:border-neutral-800">
     {#each exception.frames as frame,i(i)}
      <li class="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-t border-neutral-200 px-3 py-2 font-mono text-[13px] first:border-t-0 dark:border-neutral-800 {frame.inApp?"bg-white dark:bg-neutral-900":"bg-neutral-50 text-neutral-500 dark:bg-neutral-950/60"}">
       <span class="min-w-0 break-all"><span class="font-semibold {frame.inApp?"text-neutral-900 dark:text-white":""}">{frame.function||t("pulseAnonymous")}</span>{#if where(frame)}<span class="text-neutral-500"> · {where(frame)}</span>{/if}</span>
       {#if frame.inApp}<Badge tone="neutral">{t("pulseInApp")}</Badge>{/if}
      </li>
     {/each}
    </ol>
   {:else}<p class="mt-2 text-sm text-neutral-500">{t("pulseNoStack")}</p>{/if}
  </div>
 {/each}
 {#if hidden||all}<button type="button" class="justify-self-start text-sm font-semibold text-neutral-600 underline hover:text-blup dark:text-neutral-300" onclick={()=>all=!all}>{all?t("pulseOwnFrames"):t("pulseAllFrames",{count:String(hidden)})}</button>{/if}
</div>
