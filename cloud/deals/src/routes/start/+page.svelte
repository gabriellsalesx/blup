<script lang="ts">
 import type {Component} from "svelte";
 import {resolve} from "$app/paths";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import Hash from "@lucide/svelte/icons/hash";
 import Send from "@lucide/svelte/icons/send";
 import Handshake from "@lucide/svelte/icons/handshake";
 import Eye from "@lucide/svelte/icons/eye";
 import Power from "@lucide/svelte/icons/power";
 import {t} from "#lib/i18n.svelte.ts";
 import type {PlainMessageKey} from "#lib/i18n/catalog.ts";
 import Button from "#lib/ui/Button.svelte";
 import Card from "#lib/ui/Card.svelte";
 import PageHeader from "#lib/ui/PageHeader.svelte";
 import BackLink from "#lib/ui/BackLink.svelte";
 // The first-run assistant: topic, destination, affiliate IDs (can be
 // skipped), preview and turning it on. Phase F0 has only the steps; each
 // one's form comes later.
 type Step={id:"topic"|"channel"|"ids"|"preview"|"on";title:PlainMessageKey;text:PlainMessageKey;Icon:Component<{class?:string}>;skip?:boolean};
 const steps:Step[]=[
  {id:"topic",title:"dealsStepTopic",text:"dealsStepTopicText",Icon:Hash},
  {id:"channel",title:"dealsStepChannel",text:"dealsStepChannelText",Icon:Send},
  {id:"ids",title:"dealsStepIds",text:"dealsStepIdsText",Icon:Handshake,skip:true},
  {id:"preview",title:"dealsStepPreview",text:"dealsStepPreviewText",Icon:Eye},
  {id:"on",title:"dealsStepOn",text:"dealsStepOnText",Icon:Power},
 ];
 let at=$state(0);
 const step=$derived(steps[at]);
 const last=$derived(at===steps.length-1);
 let heading:HTMLHeadingElement|undefined=$state();
 // Moving between steps puts the focus on the new step's title.
 function go(to:number){at=Math.max(0,Math.min(steps.length-1,to));queueMicrotask(()=>heading?.focus());}
</script>

<div class="mb-4"><BackLink href={resolve("/")} label={t("dealsOffers")}/></div>
<PageHeader title={t("dealsStartTitle")} lead={t("dealsStartLead")}/>

<ol class="mt-8 flex flex-wrap gap-2" aria-label={t("dealsStartTitle")}>
 {#each steps as s,i(s.id)}
  <li><button type="button" aria-current={i===at?"step":undefined} onclick={()=>go(i)} class="flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-semibold {i===at?'border-blup bg-blup text-white':i<at?'border-neutral-300 text-blup dark:border-neutral-700':'border-neutral-300 text-neutral-500 dark:border-neutral-700'}">
   {#if i<at}<CircleCheck class="size-4"/>{:else}<span class="tabular-nums">{i+1}</span>{/if}{t(s.title)}
  </button></li>
 {/each}
</ol>

<div class="mt-6"><Card>
 <div class="flex items-start gap-4">
  <span class="grid size-12 shrink-0 place-items-center rounded-2xl border border-neutral-200 text-blup dark:border-neutral-700" aria-hidden="true"><step.Icon class="size-6"/></span>
  <div class="min-w-0">
   <p class="text-sm font-semibold text-neutral-500">{t("dealsStepOf",{step:String(at+1),total:String(steps.length)})}</p>
   <h2 bind:this={heading} tabindex="-1" class="mt-1 text-xl font-bold outline-none">{t(step.title)}</h2>
   <p class="mt-2 text-neutral-500">{t(step.text)}</p>
   <p class="mt-4 text-sm font-semibold">{t("dealsSoon")}</p>
  </div>
 </div>
 <div class="mt-6 flex flex-wrap justify-end gap-2">
  {#if at>0}<Button variant="secondary" onclick={()=>go(at-1)}>{t("back")}</Button>{/if}
  {#if step.skip}<Button variant="secondary" onclick={()=>go(at+1)}>{t("dealsSkip")}</Button>{/if}
  {#if !last}<Button onclick={()=>go(at+1)}>{t("next")}</Button>{:else}<Button disabled>{t("dealsStepOn")}</Button>{/if}
 </div>
</Card></div>
