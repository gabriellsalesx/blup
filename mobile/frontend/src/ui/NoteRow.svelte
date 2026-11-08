<script lang="ts">
  import { tick } from "svelte";
  import Folder from "@lucide/svelte/icons/folder";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import { NotesService, type Note } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message, when } from "../lib/app.svelte";

  let { note }: { note: Note } = $props();
  const tags = $derived(note.tags ?? []);

  // Sliding the note to the left turns what is under it red; let go past
  // REACH of the width and it goes to the trash, with Undo on the toast.
  const REACH = 0.55;
  let row = $state<HTMLDivElement>();
  let width = $state(1);
  let dx = $state(0);
  let sliding = $state(false);
  let height = $state<number>();
  const progress = $derived(Math.min(1, -dx / (width * REACH)));
  let from: { x: number; y: number } | null = null;
  let slid = false; // the click after a slide is not a tap

  function down(e: PointerEvent) {
    from = { x: e.clientX, y: e.clientY };
    slid = false;
  }
  function move(e: PointerEvent) {
    if (!from) return;
    const x = e.clientX - from.x, y = e.clientY - from.y;
    if (!sliding) {
      if (Math.abs(x) < 10 && Math.abs(y) < 10) return;
      if (Math.abs(y) > Math.abs(x)) { from = null; return; } // the list is scrolling
      if (x > 0) { from = null; slid = true; return; } // the side menu's, not the note's
      sliding = slid = true;
      width = row?.offsetWidth || 1; // read when it slides: no observer on every row
      row?.setPointerCapture(e.pointerId);
    }
    dx = Math.min(0, x);
  }
  function up() {
    from = null;
    if (!sliding) return;
    sliding = false;
    if (progress >= 1) remove();
    else dx = 0;
  }
  function cancel() {
    from = null;
    sliding = false;
    dx = 0;
  }
  async function remove() {
    NotesService.HapticStrong();
    dx = -width;
    height = row?.offsetHeight;
    await tick();
    requestAnimationFrame(() => (height = 0));
    setTimeout(async () => {
      try {
        await app.trashNote(note.id);
      } catch (e) {
        dx = 0;
        height = undefined;
        app.say({ text: message(e) });
      }
    }, 220);
  }
</script>

<div class="row" role="presentation" bind:this={row} style:height={height === undefined ? null : `${height}px`} style:--p={progress}
  onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={cancel}>
  <span class={["under", { ready: progress >= 1 }]}><Trash2 size={22} /></span>
  <button class={["note", { sliding }]} style:transform={dx ? `translateX(${dx}px)` : null} onclick={() => { if (!slid) app.open(note.id); }}>
    <span class="top"><b>{note.title}</b><time>{when(note.updated)}</time></span>
    {#if note.preview}<span class="preview">{note.preview}</span>{/if}
    {#if note.folder || tags.length}
      <span class="meta">
        {#if note.folder}<span class="where"><Folder size={13} />{note.folder}</span>{/if}
        {#each tags.slice(0, 2) as tag (tag)}<span class="tag">#{tag}</span>{/each}
      </span>
    {/if}
  </button>
</div>

<style>
  .row { position: relative; overflow: hidden; touch-action: pan-y; transition: height .2s; }
  .under { position: absolute; inset: 0; display: flex; align-items: center; justify-content: flex-end; padding-inline-end: 1.5rem; color: #fff; background: color-mix(in srgb, var(--red) calc(var(--p) * 100%), var(--soft)); }
  .under :global(svg) { opacity: var(--p); transition: scale .15s; }
  .under.ready :global(svg) { scale: 1.3; }
  .note { position: relative; display: grid; gap: .3rem; width: 100%; padding: .9rem 1rem; text-align: start; border-block-end: 1px solid var(--line); background: var(--bg); transition: transform .2s; }
  .note.sliding { transition: none; }
  .note:active { background: var(--soft); }
  .top { display: flex; align-items: baseline; justify-content: space-between; gap: .75rem; }
  b { font-weight: 600; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  time { flex: none; font-size: .8rem; color: var(--muted); }
  .preview { color: var(--muted); font-size: .9rem; line-height: 1.35; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .meta { display: flex; flex-wrap: wrap; align-items: center; gap: .6rem; font-size: .8rem; color: var(--muted); }
  .where { display: inline-flex; align-items: center; gap: .25rem; }
  .tag { color: var(--blue-text); }
</style>
