<script lang="ts">
  import { Page } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { OpenBlupCloud } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupAccount as Account } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import HardDrive from "@lucide/svelte/icons/hard-drive";
  import X from "@lucide/svelte/icons/x";
  import { t } from "./i18n";
  import { bytes, overLimit } from "./cloudhistory";
  import HistoryPurge from "./HistoryPurge.svelte";

  // Above the plan's storage on Blup Cloud (after a smaller plan, or
  // full): how much, the grace if one runs, and the ways out: change plan
  // or delete the old history. Deleting and editing notes keep syncing.
  let { account = $bindable() }: { account: Account | null } = $props();
  const q = t.quota;
  let closed = $state(false);
  const over = $derived(overLimit(account));
  const day = (d: Date) => d.toLocaleDateString(t.locale, { day: "numeric", month: "long", year: "numeric" });
</script>

{#if over && !closed}
  <div class="banner" role="status">
    <span class="icon"><HardDrive size={18} /></span>
    <div class="text">
      <b>{q.title}</b>
      <span>{q.used.replace("{used}", bytes(over.used)).replace("{limit}", bytes(over.limit))}
        {#if over.graceUntil}{q.grace.replace("{date}", day(over.graceUntil)).replace("{limit}", bytes(over.graceLimit))}{:else}{q.after}{/if}
        {q.free}</span>
    </div>
    <div class="actions">
      <button class="btn" onclick={() => OpenBlupCloud(Page.PagePlan)}>{q.plans}</button>
      <HistoryPurge bind:account compact />
    </div>
    <button class="close" title={q.close} aria-label={q.close} onclick={() => (closed = true)}><X size={16} /></button>
  </div>
{/if}

<style>
  .banner { display: flex; align-items: center; gap: .75rem; flex-wrap: wrap; margin: .75rem 1rem 0; padding: .75rem 1rem; border: 1px solid color-mix(in srgb, var(--red) 40%, var(--line)); border-radius: .7rem; background: color-mix(in srgb, var(--red) 8%, var(--panel)); }
  .icon { display: grid; color: var(--red); }
  .text { flex: 1; min-width: 14rem; display: grid; gap: .2rem; font-size: .875rem; }
  .actions { display: flex; gap: .5rem; flex-wrap: wrap; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
</style>
