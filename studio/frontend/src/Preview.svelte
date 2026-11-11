<script lang="ts">
  import { onDestroy, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { samePage, previewMessage } from "./previewurl";
  import { scheme } from "./theme.svelte";

  // The live preview without flashes: a rebuilt page loads in a hidden
  // frame, scrolls to where the reader was, and only then takes the place
  // of the visible one. Each frame's address carries a mark (_blup) that the
  // preview pages send back with their scroll (a script the theme adds in
  // preview mode only), so the Studio knows which frame is talking.
  // onNavigate tells the page the reader went to by a link inside.
  let { src, title, onNavigate }: { src: string; title: string; onNavigate?: (path: string) => void } = $props();

  let seq = 0;
  const mark = () => String(++seq);
  // The page takes the Studio's light or dark (the theme's preview script
  // reads _blup_scheme): a page in a frame doesn't always get the system's.
  const schemeName = () => (scheme.dark ? "dark" : "light");
  const withMark = (url: string, m: string) => url + (url.includes("?") ? "&" : "?") + "_blup=" + m + "&_blup_scheme=" + schemeName();

  const first = mark();
  let frames = $state<{ src: string; mark: string }[]>([{ src: withMark(untrack(() => src), first), mark: first }, { src: "", mark: "" }]);
  let active = $state(0);
  let els: HTMLIFrameElement[] = $state([]);
  let pending: { idx: number; y: number; timer: ReturnType<typeof setTimeout> } | null = null;
  let path = ""; // the page shown (the reader may follow links inside)
  let y = 0; // its scroll

  const origin = () => { try { return new URL(src).origin; } catch { return ""; } };

  function load(url: string, scroll: number) {
    if (!url) return;
    if (pending) clearTimeout(pending.timer);
    const idx = 1 - active;
    const m = mark();
    frames[idx] = { src: withMark(url, m), mark: m };
    // A page without the preview script (an error page): swap anyway.
    pending = { idx, y: scroll, timer: setTimeout(() => swap(idx), 4000) };
  }
  function swap(idx: number) {
    if (!pending || pending.idx !== idx) return;
    clearTimeout(pending.timer);
    pending = null;
    active = idx;
  }

  function onMessage(e: MessageEvent<unknown>) {
    // Only the preview's own pages talk to the Studio, and only in the
    // shape its script sends.
    if (e.origin !== origin()) return;
    const d = previewMessage(e.data);
    if (!d) return;
    const m = new URLSearchParams(d.q).get("_blup");
    // The frame it came from; its mark, when it has one, must be that
    // frame's (a page reached by a link inside has none).
    const idx = els.findIndex((el) => el && el.contentWindow === e.source);
    if (idx < 0 || (m && frames[idx].mark !== m)) return;
    if (pending && idx === pending.idx) {
      if (d.blup === "loaded") (e.source as Window | null)?.postMessage({ blup: "scrollTo", y: pending.y }, origin());
      else if (d.blup === "ready") { path = d.path; y = d.y; swap(idx); }
    } else if (idx === active) {
      if (d.blup === "loaded" && !m && d.path !== path) onNavigate?.(d.path);
      path = d.path;
      y = d.y;
    }
  }

  // A note was saved and the site rebuilt: same page, same place.
  const off = Events.On("previewChanged", () => load(path ? origin() + path : src, y));
  onDestroy(() => { off(); if (pending) clearTimeout(pending.timer); });

  // The Studio turned light or dark: the pages shown follow.
  $effect(() => {
    const v = schemeName();
    const o = untrack(origin);
    for (const el of untrack(() => els)) el?.contentWindow?.postMessage({ blup: "scheme", v }, o);
  });

  // Another note was opened: its page, from the top.
  let shown = untrack(() => src);
  $effect(() => {
    if (src && src !== shown) {
      const before = shown;
      shown = src;
      // The note of the page the reader just went to: already there. A
      // preview started again (a new address) has it on another server:
      // that one loads.
      if (path && samePage(src, before, path)) return;
      path = "";
      y = 0;
      load(src, 0);
    }
  });
</script>

<svelte:window onmessage={onMessage} />

<div class="frames">
  {#each frames as f, i (i)}
    {#if f.src}<iframe bind:this={els[i]} src={f.src} {title} class:hidden={i !== active} sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-downloads"></iframe>{/if}
  {/each}
</div>

<style>
  .frames { position: relative; width: 100%; height: 100%; }
  iframe { position: absolute; inset: 0; width: 100%; height: 100%; border: 0; }
  /* Transparent, not invisible: the browser keeps painting it, so it is
     ready the moment it swaps in. */
  .hidden { opacity: 0; pointer-events: none; z-index: -1; }
</style>
