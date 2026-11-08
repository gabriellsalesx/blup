<script lang="ts">
  import Compare from "@lucide/svelte/icons/git-compare-arrows";
  import Sheet from "./Sheet.svelte";
  import { app } from "../lib/app.svelte";
  import { t } from "../desk/i18n";

  // "Compare from scratch", asked in a sheet over the Cloud card: the phone
  // forgets the last sync and compares file by file with Blup Cloud. It
  // accepts the version the cloud shows now, so it only ever runs from here,
  // after the person read what it does. Confirming goes back to the card,
  // which shows the sync under way and its result.
  function confirm() {
    app.back();
    app.sync(true);
  }
</script>

<Sheet title={t.cloud.freshTitle} onclose={() => app.back()}>
  <div class="body">
    <p class="lead"><span class="ico"><Compare size={22} /></span>{t.cloud.freshConfirm}</p>
    <button class="yes" disabled={app.syncing} onclick={confirm}>{t.cloud.fresh}</button>
    <button class="no" onclick={() => app.back()}>{t.common.cancel}</button>
  </div>
</Sheet>

<style>
  .body { display: grid; gap: .9rem; padding-block-end: .5rem; }
  .lead { display: flex; gap: .75rem; align-items: flex-start; margin: 0; font-size: 1rem; line-height: 1.45; }
  .ico { color: var(--blue); flex-shrink: 0; }
  button { min-height: 3rem; border-radius: .6rem; font-size: 1rem; font-weight: 600; }
  button:disabled { opacity: .5; }
  .yes { background: var(--blue); color: #fff; }
  .no { background: transparent; color: var(--text); border: 1px solid var(--line); }
</style>
