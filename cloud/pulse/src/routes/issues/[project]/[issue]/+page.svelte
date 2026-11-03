<script lang="ts">
 import type {PageProps} from "./$types";
 import {refreshAll} from "$app/navigation";
 import CircleCheck from "@lucide/svelte/icons/circle-check";
 import EyeOff from "@lucide/svelte/icons/eye-off";
 import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
 import {t,intlLocale,formatDate} from "#lib/i18n.svelte.ts";
 import {errorMessage} from "#lib/i18n/errors.ts";
 import {toast} from "#lib/toast.svelte.ts";
 import BackLink from "#lib/ui/BackLink.svelte";
 import DetailHeader from "#lib/ui/DetailHeader.svelte";
 import Card from "#lib/ui/Card.svelte";
 import Badge from "#lib/ui/Badge.svelte";
 import Button from "#lib/ui/Button.svelte";
 import OccurrenceBars from "../../../../lib/OccurrenceBars.svelte";
 import StackTrace from "../../../../lib/StackTrace.svelte";
 import {pulseApi,parseResponse} from "../../../../lib/api";
 import {panel} from "../../../../lib/panel.svelte";
 import {ago} from "../../../../lib/monitor";
 import {issuesOf,stateTone,stateName,levelTone,levelName,type IssueState} from "../../../../lib/issues";
 // One issue in full: what it is, how often and since when, the days it
 // happened, the exception of an occurrence (the latest, or the one chosen
 // below) and its latest occurrences. Resolving, ignoring and reopening
 // are on top.
 let {data}:PageProps=$props();
 let busy=$state(false);
 const issue=$derived(data.issue);
 const writable=$derived(panel.usage?.readOnly===false&&!!data.project?.writable);
 const days=$derived(Math.min(30,panel.usage?.limits?.retentionDays??30));
 // The occurrence shown: the latest until another is chosen; another issue
 // starts again from its latest.
 let chosen=$derived.by<string|null>(()=>data.events[0]?.id??null);
 const shown=$derived(data.events.find(e=>e.id===chosen)??data.events[0]??null);
 const stats=$derived(issue?[
  {label:t("pulseOccurrences"),value:issue.count.toLocaleString(intlLocale()),hint:t("pulseLastDays",{days:String(panel.usage?.limits?.retentionDays??30)})},
  {label:t("pulseFirstSeen"),value:ago(issue.firstAt),hint:formatDate(new Date(issue.firstAt))},
  {label:t("pulseLastSeen"),value:ago(issue.lastAt),hint:formatDate(new Date(issue.lastAt))},
  {label:t("pulseFieldEnvironment"),value:issue.environment||"-",hint:`${t("pulseFieldRelease")}: ${issue.release||"-"}`},
 ]:[]);
 async function change(state:IssueState){const project=data.project,current=issue;if(!project||!current)return;busy=true;try{await parseResponse(pulseApi.projects[":project"].issues[":issue"].$patch({param:{project:project.id,issue:current.id},json:{state}}));toast(()=>t("pulseToastIssue"));await refreshAll();}catch(err){toast(errorMessage(err),{kind:"error"});}finally{busy=false;}}
</script>

{#if !data.project||!issue}<p class="rounded-xl border border-neutral-200 bg-white px-5 py-10 text-center text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900">{t("pulseIssueNotFound")}</p>
{:else}
 <BackLink href={issuesOf(data.project.id)} label={t("pulseIssues")}/>
 <DetailHeader title={issue.title} wrap>
  {#snippet badge()}<Badge tone={levelTone(issue.level)}>{levelName(issue.level)}</Badge><Badge tone={stateTone[issue.state]}>{stateName(issue.state)}</Badge>{/snippet}
  {#snippet actions()}{#if writable}{#if issue.state==="open"}<Button small variant="secondary" {busy} onclick={()=>change("ignored")}><EyeOff class="size-4"/>{t("pulseIgnore")}</Button><Button small {busy} onclick={()=>change("resolved")}><CircleCheck class="size-4"/>{t("pulseResolve")}</Button>{:else}<Button small variant="secondary" {busy} onclick={()=>change("open")}><RotateCcw class="size-4"/>{t("pulseReopen")}</Button>{/if}{/if}{/snippet}
 </DetailHeader>
 <p class="mt-1 text-sm text-neutral-500">{[data.project.name,shown?.event.platform].filter(Boolean).join(" · ")}</p>

 <div class="mt-8 grid grid-cols-[minmax(0,1fr)] gap-6">
  <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
   {#each stats as stat(stat.label)}<div class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><p class="text-sm text-neutral-500">{stat.label}</p><p class="mt-1 text-lg font-bold break-words sm:text-xl">{stat.value}</p><p class="mt-0.5 truncate text-xs text-neutral-500" title={stat.hint}>{stat.hint}</p></div>{/each}
  </div>

  <Card title={t("pulseOccurrencesPerDay")}>
   <div class="mt-4"><OccurrenceBars days={data.days} range={days} label={t("pulseOccurrencesPerDay")}/></div>
  </Card>

  <Card title={t("pulseException")} help={shown?t("pulseOccurrenceAt",{date:formatDate(new Date(shown.receivedAt))}):undefined}>
   <div class="mt-4">{#if shown}<StackTrace exceptions={shown.event.exceptions}/>{#if shown.event.message&&!shown.event.exceptions.length}<p class="font-mono text-sm break-words">{shown.event.message}</p>{/if}{:else}<p class="text-sm text-neutral-500">{t("pulseNoEvents")}</p>{/if}</div>
  </Card>

  <section class="overflow-hidden rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
   <h2 class="px-5 pt-5 font-bold">{t("pulseEvents")}</h2>
   <div class="mt-3 overflow-auto"><table class="w-full text-left text-sm">
    <thead class="border-y border-neutral-200 whitespace-nowrap text-neutral-500 dark:border-neutral-800"><tr><th class="px-5 py-2.5 font-medium">{t("pulseTime")}</th><th class="px-5 py-2.5 font-medium">{t("pulseFieldEnvironment")}</th><th class="px-5 py-2.5 font-medium">{t("pulseFieldRelease")}</th><th class="px-5 py-2.5 font-medium">{t("pulseLevel")}</th></tr></thead>
    <tbody>
     {#each data.events as item(item.id)}
      <tr class="cursor-pointer border-b border-neutral-200 last:border-b-0 hover:bg-neutral-50 dark:border-neutral-800 dark:hover:bg-neutral-800/60 {item.id===shown?.id?"bg-neutral-50 dark:bg-neutral-800/40":""}" onclick={()=>chosen=item.id}>
       <td class="px-5 py-2.5 whitespace-nowrap"><button type="button" class="text-left font-medium {item.id===shown?.id?"text-blup":""}" aria-pressed={item.id===shown?.id} onclick={()=>chosen=item.id}>{formatDate(new Date(item.receivedAt))}</button></td>
       <td class="px-5 py-2.5 whitespace-nowrap">{item.event.environment||"-"}</td>
       <td class="px-5 py-2.5 whitespace-nowrap">{item.event.release||"-"}</td>
       <td class="px-5 py-2.5"><Badge tone={levelTone(item.event.level)}>{levelName(item.event.level)}</Badge></td>
      </tr>
     {:else}<tr><td colspan="4" class="px-5 py-10 text-center text-neutral-500">{t("pulseNoEvents")}</td></tr>{/each}
    </tbody>
   </table></div>
  </section>
  <p class="text-xs text-neutral-500">{t("pulsePrivacy")}</p>
 </div>
{/if}
