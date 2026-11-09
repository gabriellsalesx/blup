<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import ChevronLeft from "@lucide/svelte/icons/chevron-left";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import { CalendarService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message } from "../lib/app.svelte";
  import { t } from "../desk/i18n";
  import { boardKind, fileTitle } from "../desk/boards";
  import "../desk/desk.css";

  // A canvas, a drawing or a database, over the screen it came from.
  let { path }: { path: string } = $props();
  const kind = $derived(boardKind(path));
  const notes = $derived(app.notes.map((n) => ({ name: n.title, path: n.id })));
  const fail = (e: string) => app.say({ text: message(e) });
  let save: (() => Promise<void>) | undefined;
  // The sync brought a new version of the file: the view opens it again
  // (it only says so when nothing of its own is waiting to be saved).
  let version = $state(0);
  const reopen = () => version++;

  async function remove() {
    try {
      await save?.();
      const where = await CalendarService.Delete(path);
      app.back();
      await app.refresh();
      app.sync();
      app.say({ text: t.common.inTrash, action: { label: t.common.undo, run: () => app.restore(where) } });
    } catch (e) { fail(String(e)); }
  }
  onDestroy(() => { void app.refresh().then(() => app.sync()); });

  // A field that gets the focus makes the browser scroll the page up to it,
  // even what never scrolls, and the screen moves away from over the keyboard.
  onMount(() => {
    const hold = (e: Event) => {
      const el = e.target instanceof Document ? document.scrollingElement : (e.target as HTMLElement);
      if (el && el.matches("html, body, #app, main, .layer, .stage, .canvas, .draw, .board") && !el.closest(".db")) el.scrollTop = el.scrollLeft = 0;
    };
    addEventListener("scroll", hold, true);
    return () => removeEventListener("scroll", hold, true);
  });
</script>

<div class="top">
  <button aria-label={t.common.back} onclick={() => app.back()}><ChevronLeft size={26} /></button>
  <h1>{fileTitle(path)}</h1>
  <button aria-label={t.common.toTrash} onclick={remove}><Trash2 size={22} /></button>
</div>
<div class="stage">
  <!-- Each kind loads only when one of its own opens. -->
  {#key version}
  {#if kind === "canvas"}
    {#await import("../desk/CanvasView.svelte") then { default: CanvasView }}
      <CanvasView {path} {notes} onStatus={() => {}} onError={fail} onOpenNote={(p) => app.open(p)} onOutside={reopen} bindSave={(s) => (save = s)} />
    {/await}
  {:else if kind === "drawing"}
    {#await import("../desk/DrawingView.svelte") then { default: DrawingView }}
      <DrawingView {path} onStatus={() => {}} onError={fail} onOutside={reopen} bindSave={(s) => (save = s)} />
    {/await}
  {:else if kind === "database"}
    {#await import("../desk/DatabaseView.svelte") then { default: DatabaseView }}
      <DatabaseView {path} {notes} onOpenNote={(p) => app.open(p)} onStatus={() => {}} onError={fail} bindSave={(s) => (save = s)} />
    {/await}
  {/if}
  {/key}
</div>

<style>
  .top { flex: none; display: flex; align-items: center; gap: .25rem; padding: calc(var(--sat) + .35rem) .5rem .35rem; border-block-end: 1px solid var(--line); background: var(--bg); }
  .top button { flex: none; display: grid; place-items: center; width: 2.75rem; height: 2.75rem; color: var(--text); }
  h1 { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 1.1rem; font-weight: 700; }
  .stage { position: relative; flex: 1; min-height: 0; overflow: hidden; background: var(--bg); }
</style>
