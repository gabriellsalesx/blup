<script lang="ts">
  import type { Snippet } from "svelte";
  import { app } from "../lib/app.svelte";

  // The open note lies over the screen it came from: sliding it to the right
  // takes it away with the finger and shows that screen under it.
  let { children }: { children: Snippet } = $props();

  let layer = $state<HTMLDivElement>();
  let dx = $state(0);
  let sliding = $state(false);
  const share = $derived(layer && dx ? 1 - Math.min(1, dx / layer.offsetWidth) : 0);

  // What sliding the note away goes back to: another note is drawn as it is
  // under this one (App.svelte); a board, by its name here. Either way not
  // the list this layer lies over, which back doesn't reach.
  const ghost = $derived.by(() => {
    const below = app.below;
    if (below?.board && below.board !== app.place.board) return { title: below.board.split("/").pop()?.replace(/\.[^.]+$/, "") ?? "", text: "" };
    return null;
  });

  $effect(() => {
    const el = layer;
    if (!el) return;
    let from: { x: number; y: number; at: number } | null = null;
    let leaving = false;
    const start = (e: TouchEvent) => {
      if (leaving) return; // on its way out: this touch is not a new slide
      const target = e.target as HTMLElement;
      const selecting = !(getSelection()?.isCollapsed ?? true);
      // A sheet or panel on its way out still takes the touch.
      if (app.place.sheet || app.place.pick || selecting || target.closest("input, .tools, .cm-md-table, .cm-image.focused, .canvas, .draw, .lanes") || el.querySelector(".sheet, aside")) return;
      from = { x: e.touches[0].clientX, y: e.touches[0].clientY, at: e.timeStamp };
    };
    const move = (e: TouchEvent) => {
      if (!from) return;
      const x = e.touches[0].clientX - from.x, y = e.touches[0].clientY - from.y;
      if (!sliding) {
        if (Math.abs(x) < 10 && Math.abs(y) < 10) return;
        // A finger that rested first is selecting text.
        if (x < 0 || Math.abs(y) > Math.abs(x) || e.timeStamp - from.at > 350) { from = null; return; }
        sliding = true;
      }
      e.preventDefault();
      dx = Math.max(0, x);
    };
    const end = (e: TouchEvent) => {
      const began = from;
      from = null;
      if (!sliding || !began) return;
      sliding = false;
      if (dx > el.offsetWidth * 0.35 || dx / (e.timeStamp - began.at) > 0.5) {
        dx = el.offsetWidth;
        leaving = true;
        setTimeout(() => {
          app.back();
          // Still here (the step back was inside this note): it comes back.
          setTimeout(() => { leaving = false; dx = 0; }, 450);
        }, 200);
      } else dx = 0;
    };
    const cancel = () => {
      if (leaving) return;
      from = null;
      sliding = false;
      dx = 0;
    };
    el.addEventListener("touchstart", start, { passive: true });
    el.addEventListener("touchmove", move, { passive: false }); // it has to hold the scroll back
    // On the window: what is under the finger may be gone from the page by
    // the time it lifts, and the end of the touch would never reach here.
    addEventListener("touchend", end);
    addEventListener("touchcancel", cancel);
    return () => {
      el.removeEventListener("touchstart", start);
      el.removeEventListener("touchmove", move);
      removeEventListener("touchend", end);
      removeEventListener("touchcancel", cancel);
    };
  });
</script>

{#if dx && ghost}
  <div class="ghost" aria-hidden="true">
    <div class="bar"></div>
    <p class="title">{ghost.title}</p>
    {#if ghost.text}<p class="text">{ghost.text}</p>{/if}
  </div>
{/if}
{#if dx}<div class="shade" style:opacity={share * 0.3}></div>{/if}
<div bind:this={layer} class={["layer", { sliding, moved: dx > 0 }]} style:transform={dx ? `translateX(${dx}px)` : null}>
  {@render children()}
</div>

<style>
  /* The note back goes to, drawn like the editor's page (Editor.svelte). */
  .ghost { position: absolute; inset: 0; z-index: 18; overflow: hidden; background: var(--bg); pointer-events: none; }
  .ghost .bar { height: calc(3.4rem + var(--sat)); border-block-end: 1px solid var(--line); }
  .ghost .title { padding: 2.6rem 1.5rem .4rem; font-size: 1.6rem; font-weight: 700; letter-spacing: -.02em; color: var(--text); }
  .ghost .text { padding: .5rem 1.1rem; color: var(--text); line-height: 1.5; white-space: pre-line; }
  .shade { position: absolute; inset: 0; z-index: 18; background: #000; pointer-events: none; }
  .layer { position: absolute; inset: 0; z-index: 19; display: flex; flex-direction: column; background: var(--bg); transition: transform .2s; }
  .layer.sliding { transition: none; }
  .layer.moved { box-shadow: -8px 0 24px rgb(0 0 0 / .25); }
</style>
