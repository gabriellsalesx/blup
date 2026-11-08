<script lang="ts">
  import { fade, fly } from "svelte/transition";
  import CalendarClock from "@lucide/svelte/icons/calendar-clock";
  import CalendarPlus from "@lucide/svelte/icons/calendar-plus";
  import type { Note } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app } from "../lib/app.svelte";
  import { plural, t } from "../desk/i18n";

  // The note's details, from the side: its dates, words and links. The folder
  // and the tags are above the note itself.
  let { note, onprops, onremove }: {
    note: Note;
    onprops: (created: string, updated: string, tags: string[]) => void;
    onremove: () => void;
  } = $props();

  const tags = $derived(note.propTags ?? []);

  let confirming = $state(false);

  // Sliding it to the right takes it away with the finger.
  let panel = $state<HTMLElement>();
  let drag = $state<number | null>(null);
  let touch: { x: number; y: number; at: number } | null = null;
  let dragged = false;
  function down(e: PointerEvent) {
    touch = { x: e.clientX, y: e.clientY, at: e.timeStamp };
    dragged = false;
  }
  function move(e: PointerEvent) {
    if (!touch) return;
    const x = e.clientX - touch.x, y = e.clientY - touch.y;
    if (drag === null) {
      if (Math.abs(x) < 10 && Math.abs(y) < 10) return;
      if (x < 0 || Math.abs(y) > Math.abs(x) || (e.target as HTMLElement).closest("input")) { touch = null; return; }
      dragged = true;
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    }
    drag = Math.min(1, Math.max(0, 1 - x / (panel?.offsetWidth ?? 1)));
  }
  function up(e: PointerEvent) {
    const start = touch;
    touch = null;
    if (drag === null || !start) return;
    const flicked = (e.clientX - start.x) / (e.timeStamp - start.at) > 0.5;
    if (drag < 0.65 || flicked) app.back();
    else drag = null;
  }
  function cancel() {
    touch = null;
    drag = null;
  }
</script>

<button class="scrim" aria-label={t.common.close} style:opacity={drag ?? null} onclick={() => { if (!dragged) app.back(); }} onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={cancel} transition:fade={{ duration: 150 }}></button>
<aside bind:this={panel} class={{ moving: drag !== null }} aria-label={t.note.about} style:transform={drag !== null ? `translateX(${(1 - drag) * 100}%)` : null}
  onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={cancel} onclickcapture={(e) => { if (dragged) { e.stopPropagation(); e.preventDefault(); } }}
  transition:fly={{ x: 320, duration: 220, opacity: 1 }}>
  <h2>{note.title}</h2>
  <div class="scroll">
    <label class="row"><span><CalendarPlus size={17} />{t.note.created}</span>
      <input type="date" value={note.created} onchange={(e) => onprops(e.currentTarget.value, note.changed, tags)} />
    </label>
    <label class="row"><span><CalendarClock size={17} />{t.note.updated}</span>
      <input type="date" value={note.changed} onchange={(e) => onprops(note.created, e.currentTarget.value, tags)} />
    </label>
    <p class="facts">{plural(note.words, t.note.words1, t.note.wordsN)}</p>
    {#if note.links?.length}
      <div class="group"><h3>{t.note.links}</h3>{#each note.links as title (title)}<button class="link" onclick={() => app.openTitle(title)}>{title}</button>{/each}</div>
    {/if}
    {#if confirming}
      <button class="danger solid" onclick={onremove}>{t.common.toTrash}</button>
    {:else}
      <button class="danger" onclick={() => (confirming = true)}>{t.note.deleteNote}</button>
    {/if}
  </div>
</aside>

<style>
  .scrim { position: fixed; inset: 0; z-index: 20; background: rgb(0 0 0 / .5); touch-action: pan-y; transition: opacity .2s; }
  aside { position: fixed; inset-block: 0; inset-inline-end: 0; z-index: 21; display: flex; flex-direction: column; width: min(88%, 24rem); padding-block: var(--sat) max(var(--sab), var(--kb, 0px)); background: var(--panel); border-radius: 1.1rem 0 0 1.1rem; touch-action: pan-y; transition: transform .2s; }
  aside.moving { transition: none; }
  .scroll { flex: 1; min-height: 0; overflow-y: auto; touch-action: pan-y; display: grid; align-content: start; gap: 1rem; padding: .25rem 1.1rem 1.25rem; }
  h2 { flex: none; padding: 1rem 1.1rem .75rem; font-size: 1.15rem; font-weight: 700; overflow-wrap: anywhere; }
  .row { display: grid; gap: .4rem; min-width: 0; }
  .row > span { display: flex; align-items: center; gap: .45rem; font-size: .85rem; font-weight: 600; color: var(--muted); }
  .facts { font-size: .85rem; color: var(--muted); }
  .group { display: grid; gap: .1rem; }
  h3 { font-size: .8rem; font-weight: 600; color: var(--muted); }
  .link { min-height: 2.75rem; text-align: start; color: var(--blue-text); font-weight: 500; }
  .danger { min-height: 3rem; border-radius: .6rem; font-weight: 600; color: var(--red); border: 1px solid var(--line); }
  .danger.solid { background: var(--red); color: var(--bg); border-color: transparent; }
</style>
