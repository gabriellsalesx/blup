<script lang="ts">
  import { t } from "./i18n";
  import { noteTip } from "./notetip.svelte";

  // A note's dates beside its row in the file list; kept inside the
  // window when the row is near its right edge.
  let box = $state<HTMLDivElement>();
  const left = $derived(box ? Math.min(noteTip.x, innerWidth - box.offsetWidth - 8) : noteTip.x);
</script>

{#if noteTip.shown}
  <div bind:this={box} class="note-tip" role="tooltip" style:left="{left}px" style:top="{noteTip.y}px">
    <span>{t.side.modifiedAt.replace("{date}", noteTip.modified)}</span>
    {#if noteTip.created}<span>{t.side.createdAt.replace("{date}", noteTip.created)}</span>{/if}
  </div>
{/if}

<style>
  .note-tip { position: fixed; z-index: 90; translate: 0 -50%; display: grid; justify-items: center; gap: .1rem; padding: .55rem .9rem; border-radius: .6rem; background: rgb(0 0 0 / .88); color: #fff; font-size: .82rem; line-height: 1.35; white-space: nowrap; pointer-events: none; box-shadow: 0 6px 18px rgb(0 0 0 / .25); }
</style>
