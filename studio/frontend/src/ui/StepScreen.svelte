<script lang="ts">
  import type { Snippet } from "svelte";

  // A full screen of its own, centered: the creation and welcome screens.
  // A picture, the title and a line under it, the content, and the
  // buttons at the bottom (`actions`); `onClose` adds the close button.
  let { image = "", title, lead = "", closeLabel = "", onClose, children, actions }: {
    image?: string;
    title: string;
    lead?: string;
    closeLabel?: string;
    onClose?: () => void;
    children?: Snippet;
    actions?: Snippet;
  } = $props();
</script>

<div class="step-screen">
  {#if onClose}<button class="close btn ghost" onclick={onClose}>{closeLabel}</button>{/if}
  <div class="center">
    {#if image}<img class="picture" src={image} alt="" />{/if}
    <h1>{title}</h1>
    {#if lead}<p class="lead">{lead}</p>{/if}
    {@render children?.()}
    {#if actions}<div class="actions-row">{@render actions()}</div>{/if}
  </div>
</div>

<style>
  .step-screen { position: relative; height: 100%; overflow: auto; display: grid; place-items: center; padding: 3rem 1.5rem; }
  .close { position: absolute; inset-block-start: 1rem; inset-inline-end: 1rem; }
  .center { width: min(100%, 34rem); display: grid; justify-items: center; gap: 1rem; text-align: center; }
  .picture { width: min(100%, 24rem); max-height: 15rem; object-fit: contain; }
  h1 { font-size: 1.85rem; font-weight: 800; letter-spacing: -.03em; }
  .lead { color: var(--muted); font-size: 1.02rem; margin-block-start: -.4rem; }
  .center > :global(.card) { width: 100%; text-align: start; }
  .actions-row { display: flex; justify-content: space-between; align-items: center; gap: .75rem; width: 100%; margin-block-start: .5rem; }
</style>
