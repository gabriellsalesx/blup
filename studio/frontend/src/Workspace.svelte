<script lang="ts">
  import { onDestroy } from "svelte";
  import { Open, Read, Write, StartPreview, StopPreview } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Tree from "./Tree.svelte";
  import Editor from "./Editor.svelte";
  import { t } from "./i18n";

  let { site, onBack }: { site: SiteInfo; onBack: () => void } = $props();

  let tree = $state<Node[]>([]);
  let current = $state(""); // path of the open note
  let text = $state("");
  let status = $state<"saved" | "saving" | "unsaved">("saved");
  let error = $state("");
  let previewURL = $state("");
  let previewOn = $state(true);
  let previewBusy = $state(false);

  let timer: ReturnType<typeof setTimeout> | undefined;
  let pending = ""; // text not saved yet

  Open(site.dir)
    .then((root) => {
      tree = root.children ?? [];
      if (previewOn) startPreview();
    })
    .catch((e) => (error = String(e)));

  async function startPreview() {
    previewBusy = true;
    try {
      previewURL = await StartPreview();
    } catch (e) {
      error = String(e);
      previewOn = false;
    } finally {
      previewBusy = false;
    }
  }

  function togglePreview() {
    previewOn = !previewOn;
    if (previewOn && !previewURL) startPreview();
  }

  async function save() {
    if (status === "saved" || !current) return;
    clearTimeout(timer);
    status = "saving";
    try {
      await Write(current, pending);
      status = "saved";
    } catch (e) {
      status = "unsaved";
      error = String(e);
    }
  }

  // Saved by itself a moment after typing stops; the preview follows.
  function changed(v: string) {
    pending = v;
    status = "unsaved";
    clearTimeout(timer);
    timer = setTimeout(save, 700);
  }

  async function pick(path: string) {
    if (path === current) return;
    await save();
    try {
      text = await Read(path);
      pending = text;
      current = path;
      status = "saved";
      error = "";
    } catch (e) {
      error = String(e);
    }
  }

  async function back() {
    await save();
    StopPreview();
    onBack();
  }

  onDestroy(() => clearTimeout(timer));
</script>

<div class="ws" class:no-preview={!previewOn}>
  <header class="bar">
    <button class="btn ghost" onclick={back}>← {t.back}</button>
    <strong class="title">{site.title}</strong>
    <span class="status muted">{current ? (status === "saved" ? t.saved : status === "saving" ? t.saving : t.unsaved) : ""}</span>
    <span class="spacer"></span>
    <button class="btn ghost" onclick={togglePreview}>{previewOn ? "✓ " : ""}{t.preview}</button>
    <button class="btn ghost" disabled title={t.soon}>{t.sync}</button>
    <button class="btn" disabled title={t.soon}>{t.publish}</button>
  </header>

  <aside class="side">
    <h2>{t.notes}</h2>
    <Tree nodes={tree} {current} onPick={pick} />
  </aside>

  <section class="main">
    {#if error}<p class="error">{t.error}: {error}</p>{/if}
    {#if current}
      {#key current}
        <Editor {text} onChange={changed} onSave={save} />
      {/key}
    {:else}
      <p class="hint muted">{t.pickNote}</p>
    {/if}
  </section>

  {#if previewOn}
    <section class="preview">
      {#if previewURL}
        <iframe src={previewURL} title={t.preview}></iframe>
      {:else}
        <p class="hint muted">{previewBusy ? t.previewStarting : t.previewOff}</p>
      {/if}
    </section>
  {/if}
</div>

<style>
  .ws { height: 100%; display: grid; grid-template-columns: 16rem minmax(0, 1fr) minmax(0, 1fr); grid-template-rows: auto minmax(0, 1fr); }
  .ws.no-preview { grid-template-columns: 16rem minmax(0, 1fr); }
  .bar { grid-column: 1 / -1; display: flex; align-items: center; gap: .75rem; padding: .6rem 1rem; border-block-end: 1px solid var(--line); background: var(--panel); }
  .title { font-size: .95rem; }
  .status { font-size: .8rem; }
  .spacer { flex: 1; }
  .side { overflow: auto; padding: 1rem .5rem; border-inline-end: 1px solid var(--line); }
  .side h2 { padding-inline: .5rem; margin-block-end: .5rem; font-size: .7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); }
  .main { position: relative; overflow: hidden; background: var(--panel); }
  .preview { overflow: hidden; border-inline-start: 1px solid var(--line); background: #fff; }
  iframe { width: 100%; height: 100%; border: 0; }
  .hint { padding: 3rem 2rem; text-align: center; }
  .error { position: absolute; inset-inline: 1rem; inset-block-start: .75rem; z-index: 2; padding: .6rem .9rem; border-radius: .5rem; background: var(--soft); color: var(--red); font-size: .85rem; }
</style>
