<script lang="ts">
  import type { Snippet } from "svelte";
  import { fade, fly } from "svelte/transition";
  import { t } from "../desk/i18n";

  // behind: another sheet is open over this one, and only that one shows.
  let { title, onclose, children, top, tall = false, behind = false }: { title: string; onclose: () => void; children: Snippet; top?: Snippet; tall?: boolean; behind?: boolean } = $props();

  // The title (and top, a search field say) stays where it is: only what is
  // under it scrolls. Sliding the sheet down takes it away with the finger;
  // a long list scrolls first, and the title always slides. A tall sheet is a
  // list that grows and shrinks: it keeps its size, and only its title slides.
  let sheet = $state<HTMLDivElement>();
  let body = $state<HTMLDivElement>();
  let dy = $state(0);
  let sliding = $state(false);
  const share = $derived(sheet && dy ? 1 - Math.min(1, dy / sheet.offsetHeight) : 1);

  // Once: a second touch while it goes must not take the note away too.
  let left = false;
  function leave() {
    if (left) return;
    left = true;
    onclose();
    // Still open (what closes it did not go through): it can be used again.
    setTimeout(() => { left = false; dy = 0; }, 700);
  }

  $effect(() => {
    const el = sheet;
    if (!el) return;
    let from: { x: number; y: number; at: number } | null = null;
    let inBody = false;
    const start = (e: TouchEvent) => {
      if ((e.target as Element).closest(".sheet") !== el) return; // one opened over this
      inBody = !!body?.contains(e.target as Node);
      from = { x: e.touches[0].clientX, y: e.touches[0].clientY, at: e.timeStamp };
    };
    const move = (e: TouchEvent) => {
      if (!from) return;
      const x = e.touches[0].clientX - from.x, y = e.touches[0].clientY - from.y;
      if (!sliding) {
        if (Math.abs(x) < 8 && Math.abs(y) < 8) return;
        if (y < 0 || (inBody && (tall || (body?.scrollTop ?? 0) > 0)) || Math.abs(x) > Math.abs(y)) { from = null; return; }
        sliding = true;
      }
      e.preventDefault();
      dy = Math.max(0, y);
    };
    const end = (e: TouchEvent) => {
      const began = from;
      from = null;
      if (!sliding || !began) return;
      sliding = false;
      const flicked = dy / (e.timeStamp - began.at) > 0.5;
      if (flicked || dy > el.offsetHeight * 0.3) leave();
      else dy = 0;
    };
    const cancel = () => {
      from = null;
      sliding = false;
      dy = 0;
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

<button class={["scrim", { behind }]} aria-label={t.common.close} style:opacity={share < 1 ? share : null} onclick={leave} transition:fade={{ duration: 150 }}></button>
<div bind:this={sheet} class={["sheet", { sliding, tall, behind }]} role="dialog" aria-modal="true" aria-label={title} style:transform={dy ? `translateY(${dy}px)` : null} transition:fly={{ y: 320, duration: 220 }}>
  <div class="head">
    <span class="grip"></span>
    <h2>{title}</h2>
    {@render top?.()}
  </div>
  <div class="body" bind:this={body}>{@render children()}</div>
</div>

<style>
  .scrim, .sheet { visibility: visible; }
  .scrim.behind, .sheet.behind { visibility: hidden; }
  .scrim { position: fixed; inset: 0; z-index: 20; background: rgb(0 0 0 / .5); transition: opacity .2s; }
  .sheet { position: fixed; inset-inline: 0; inset-block-end: var(--kb, 0px); z-index: 21; display: flex; flex-direction: column; max-height: calc(80% - var(--kb, 0px)); background: var(--panel); border-radius: 1.1rem 1.1rem 0 0; transition: transform .2s; }
  .sheet.tall { height: min(62%, calc(100% - var(--kb, 0px) - var(--sat) - 5rem)); }
  .head { flex: none; display: grid; gap: .8rem; padding: .6rem 1.1rem .8rem; }
  .body { flex: 1 1 auto; min-height: 0; overflow-y: auto; overscroll-behavior: contain; display: grid; align-content: start; gap: 1rem; padding: .3rem 1.1rem calc(1.1rem + var(--sab)); }
  .sheet.sliding { transition: none; }
  .grip { justify-self: center; width: 2.4rem; height: .25rem; border-radius: 999px; background: var(--line); }
  h2 { font-size: 1.05rem; font-weight: 700; }
</style>
