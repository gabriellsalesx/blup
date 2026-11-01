<script lang="ts">
  import type { Snippet } from "svelte";

  // One option among a few, shown as a card: the title, a line about it,
  // and the blue ring when chosen. `children` goes below (a picture...).
  let { title, help = "", on = false, disabled = false, onclick, children }: {
    title: string;
    help?: string;
    on?: boolean;
    disabled?: boolean;
    onclick: () => void;
    children?: Snippet;
  } = $props();
</script>

<button class={["choice", { on }]} {disabled} aria-pressed={on} {onclick}>
  {@render children?.()}
  <b>{title}</b>
  {#if help}<span>{help}</span>{/if}
</button>
