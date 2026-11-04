<script lang="ts" module>
 export type SendMode="now"|"queue"|"schedule";
</script>
<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 import Segmented from "#lib/ui/Segmented.svelte";
 import DateTimeField from "#lib/ui/DateTimeField.svelte";
 import {paceEvery} from "./pace";
 // When a pasted deal or a video goes: now (outside the queue), in the
 // queue (the next free time of the account's pace) or at a chosen time.
 let {mode=$bindable("now"),sendAt=$bindable(),min,max,pace}:{mode?:SendMode;sendAt:number;min:number;max:number;pace:number}=$props();
 const options=$derived([{value:"now" as const,label:t("dealsWhenNow")},{value:"queue" as const,label:t("dealsWhenQueue")},{value:"schedule" as const,label:t("dealsSchedule")}]);
 const help=$derived(mode==="now"?t("dealsWhenNowHelp"):mode==="queue"?t("dealsWhenQueueHelp",{pace:paceEvery(pace)}):t("dealsWhenScheduleHelp"));
</script>

<p class="text-sm font-semibold">{t("dealsWhen")}</p>
<div class="mt-2"><Segmented label={t("dealsWhen")} {options} bind:value={mode}/></div>
<p class="mt-2 text-sm text-neutral-500">{help}</p>
{#if mode==="schedule"}<div class="mt-4 max-w-sm"><DateTimeField label={t("dealsSendAt")} surface="bg-white dark:bg-neutral-900" {min} {max} bind:value={sendAt}/></div>{/if}
