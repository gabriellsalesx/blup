<script lang="ts">
  import { onDestroy, tick } from "svelte";
  import { Open, Tree as LoadTree, ReadNote, SaveNote, StartPreview, StopPreview, PreviewPage, Settings as LoadSettings } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Tree from "./Tree.svelte";
  import Editor from "./Editor.svelte";
  import Props from "./Props.svelte";
  import Preview from "./Preview.svelte";
  import TerminalPanel from "./TerminalPanel.svelte";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import { Events } from "@wailsio/runtime";
  import LookPanel from "./LookPanel.svelte";
  import SectionsPanel from "./SectionsPanel.svelte";
  import CommentsPanel from "./CommentsPanel.svelte";
  import CookiesPanel from "./CookiesPanel.svelte";
  import PasswordPanel from "./PasswordPanel.svelte";
  import PublishPanel from "./PublishPanel.svelte";
  import TrashPanel from "./TrashPanel.svelte";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import PenLine from "@lucide/svelte/icons/pen-line";
  import Palette from "@lucide/svelte/icons/palette";
  import LayoutGrid from "@lucide/svelte/icons/layout-grid";
  import MessageCircle from "@lucide/svelte/icons/message-circle";
  import Cookie from "@lucide/svelte/icons/cookie";
  import KeyRound from "@lucide/svelte/icons/key-round";
  import Upload from "@lucide/svelte/icons/upload";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import { t } from "./i18n";

  let { site, onBack }: { site: SiteInfo; onBack: () => void } = $props();

  let tree = $state<Node[]>([]);
  let current = $state(""); // path of the open note
  let text = $state("");
  let noteProps = $state<Record<string, any>>({});
  let kind = $state("");
  let edits: Record<string, any> = {}; // properties changed since the last save
  let status = $state<"saved" | "saving" | "unsaved">("saved");
  let error = $state("");
  let previewURL = $state("");
  let previewSrc = $state(""); // the page shown: the open note's, when known
  let previewOn = $state(true);
  let previewBusy = $state(false);

  // The screen on the right of the menu: writing or a settings panel.
  let screen = $state("write");
  let settings = $state<Settings | null>(null);
  const reload = () => LoadSettings().then((s) => (settings = s)).catch((e) => (error = String(e)));
  const screens = $derived([
    { key: "write", label: t.nav.write, Icon: PenLine },
    { key: "look", label: t.nav.look, Icon: Palette },
    ...(site.model === "bento" ? [{ key: "sections", label: t.nav.sections, Icon: LayoutGrid }] : []),
    { key: "comments", label: t.nav.comments, Icon: MessageCircle },
    { key: "cookies", label: t.nav.cookies, Icon: Cookie },
    { key: "password", label: t.nav.password, Icon: KeyRound },
    { key: "publish", label: t.nav.publish, Icon: Upload },
    { key: "trash", label: t.nav.trash, Icon: Trash2 },
  ]);

  // The terminal at the bottom, and its height (dragging its top edge).
  let termOpen = $state(false);
  let termHeight = $state(300);
  function resizeTerm(e: MouseEvent) {
    const start = e.clientY, h = termHeight;
    const move = (m: MouseEvent) => (termHeight = Math.min(Math.max(140, h + start - m.clientY), window.innerHeight - 200));
    const up = () => { removeEventListener("mousemove", move); removeEventListener("mouseup", up); };
    addEventListener("mousemove", move);
    addEventListener("mouseup", up);
  }

  // Notes changed on disk (an agent in the terminal, another editor): the
  // tree follows; the open note too, unless it has changes of its own.
  let reloadN = $state(0);
  let outside = $state(false);
  const offNotes = Events.On("notesChanged", async (ev: any) => {
    const list = String(ev.data ?? "").split("\n");
    try { tree = (await LoadTree()).children ?? []; } catch {}
    if (!current || !list.some((p) => p.toLowerCase() === current.toLowerCase())) return;
    const note = await ReadNote(current).catch(() => null);
    if (!note || note.body === pending) return;
    if (status === "saved") applyOutside(note);
    else outside = true;
  });
  function applyOutside(note: { body: string; props: Record<string, any> | null; kind: string }) {
    text = note.body;
    pending = note.body;
    noteProps = note.props ?? {};
    edits = {};
    status = "saved";
    outside = false;
    reloadN++;
  }
  async function reloadOutside() {
    const note = await ReadNote(current).catch(() => null);
    if (note) applyOutside(note);
  }
  onDestroy(() => offNotes());

  let timer: ReturnType<typeof setTimeout> | undefined;
  let pending = ""; // text not saved yet

  Open(site.dir)
    .then((root) => {
      tree = root.children ?? [];
      reload();
      if (previewOn) startPreview();
    })
    .catch((e) => (error = String(e)));

  async function startPreview() {
    previewBusy = true;
    try {
      previewURL = await StartPreview();
      previewSrc = previewURL;
      if (current) follow(current);
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
      const changed = edits;
      edits = {};
      await SaveNote(current, pending, Object.entries(changed).map(([key, value]) => ({ key, value })));
      status = "saved";
    } catch (e) {
      status = "unsaved";
      error = String(e);
    }
  }

  // Saved by itself a moment after typing stops; the preview follows.
  function changed(v: string) {
    pending = v;
    later();
  }

  function editProp(key: string, value: any) {
    edits[key] = value;
    later();
  }

  function later() {
    status = "unsaved";
    clearTimeout(timer);
    timer = setTimeout(save, 300);
  }

  async function pick(path: string) {
    if (path === current) return;
    await save();
    try {
      const note = await ReadNote(path);
      text = note.body;
      noteProps = note.props ?? {};
      kind = note.kind;
      edits = {};
      pending = text;
      current = path;
      follow(path);
      status = "saved";
      error = "";
    } catch (e) {
      error = String(e);
    }
  }

  // The preview shows the page of the note being edited.
  async function follow(path: string) {
    if (!previewURL) return;
    const url = await PreviewPage(path).catch(() => "");
    if (url && path === current) previewSrc = url;
  }

  // The top buttons open the Publish screen and start right away.
  let autostart = $state("");
  async function go(where: string, start: string) {
    await save();
    autostart = start;
    screen = "";
    await tick();
    screen = where;
  }

  async function openNote(path: string) {
    screen = "write";
    await pick(path);
  }

  async function back() {
    await save();
    StopPreview();
    onBack();
  }

  onDestroy(() => clearTimeout(timer));
</script>

<svelte:window onkeydown={(e) => { if ((e.ctrlKey || e.metaKey) && e.key === "`") { e.preventDefault(); termOpen = !termOpen; } }} />

<div class="ws" class:no-preview={!previewOn}>
  <header class="bar">
    <button class="btn ghost" onclick={back}><ArrowLeft size={16} aria-hidden="true" />{t.back}</button>
    <strong class="title">{site.title}</strong>
    <span class="status muted">{current ? (status === "saved" ? t.saved : status === "saving" ? t.saving : t.unsaved) : ""}</span>
    <span class="spacer"></span>
    <button class="btn ghost" class:on={termOpen} onclick={() => (termOpen = !termOpen)} title="Ctrl+`"><SquareTerminal size={15} />{t.term.button}</button>
    <button class="btn ghost" onclick={togglePreview}>{previewOn ? "✓ " : ""}{t.preview}</button>
    <button class="btn ghost" onclick={() => go("publish", "sync")}>{t.sync}</button>
    <button class="btn" onclick={() => go("publish", "publish")}>{t.publish}</button>
  </header>

  <aside class="side">
    <nav>
      {#each screens as { key, label, Icon } (key)}
        <button class="nav" class:active={screen === key} onclick={async () => { await save(); autostart = ""; screen = key; error = ""; }}><Icon size={17} strokeWidth={1.75} aria-hidden="true" />{label}</button>
      {/each}
    </nav>
    {#if screen === "write"}
      <h2>{t.notes}</h2>
      <Tree nodes={tree} {current} onPick={pick} />
    {/if}
  </aside>

  <section class="main" class:scroll={screen !== "write"}>
    {#if error}<p class="error">{t.error}: {error}</p>{/if}
    {#if outside}<div class="outside"><span>{t.term.outside}</span><button class="btn ghost" onclick={reloadOutside}>{t.term.reload}</button><button class="btn ghost" onclick={() => (outside = false)}>{t.term.keep}</button></div>{/if}
    {#if screen === "write"}
      {#if current}
        {#key current + ":" + reloadN}
          <div class="note">
            <Props props={noteProps} {kind} isIndex={current.split("/").pop() === "_index.md"} onEdit={editProp} />
            <div class="body"><Editor {text} onChange={changed} onSave={save} /></div>
          </div>
        {/key}
      {:else}
        <p class="hint muted">{t.pickNote}</p>
      {/if}
    {:else if settings}
      {#key screen}
        {#if screen === "look"}<LookPanel {settings} {reload} />
        {:else if screen === "sections"}<SectionsPanel {settings} {reload} onTree={(n) => (tree = n.children ?? [])} />
        {:else if screen === "comments"}<CommentsPanel {settings} {reload} />
        {:else if screen === "cookies"}<CookiesPanel {settings} {reload} onOpenNote={openNote} />
        {:else if screen === "password"}<PasswordPanel {settings} {reload} />
        {:else if screen === "publish"}<PublishPanel autostart={autostart} />
        {:else if screen === "trash"}<TrashPanel {reload} onTree={(n) => (tree = n.children ?? [])} />
        {/if}
      {/key}
    {/if}
  </section>

  {#if previewOn}
    <section class="preview">
      {#if previewURL}
        <Preview src={previewSrc || previewURL} title={t.preview} />
      {:else}
        <p class="hint muted">{previewBusy ? t.previewStarting : t.previewOff}</p>
      {/if}
    </section>
  {/if}
  {#if termOpen}
    <div class="term" style:height="{termHeight}px">
      <div class="grip" role="separator" aria-orientation="horizontal" onmousedown={resizeTerm}></div>
      <TerminalPanel onClose={() => (termOpen = false)} />
    </div>
  {/if}
</div>

<style>
  .ws { height: 100%; display: grid; grid-template-columns: 16rem minmax(0, 1fr) minmax(0, 1fr); grid-template-rows: auto minmax(0, 1fr) auto; }
  .side { grid-row: 2 / 4; }
  .term { grid-column: 2 / -1; position: relative; min-height: 0; }
  .grip { position: absolute; z-index: 3; inset-inline: 0; inset-block-start: -3px; height: 6px; cursor: row-resize; }
  .btn.on { background: var(--soft); }
  .outside { position: absolute; z-index: 2; inset-inline: 1rem; inset-block-start: .75rem; display: flex; gap: .5rem; align-items: center; padding: .5rem .75rem; border-radius: .5rem; background: var(--soft); font-size: .85rem; }
  .outside span { flex: 1; }
  .ws.no-preview { grid-template-columns: 16rem minmax(0, 1fr); }
  .bar { grid-column: 1 / -1; display: flex; align-items: center; gap: .75rem; padding: .6rem 1rem; border-block-end: 1px solid var(--line); background: var(--panel); }
  .title { font-size: .95rem; }
  .status { font-size: .8rem; }
  .spacer { flex: 1; }
  .side { overflow: auto; padding: 1rem .5rem; border-inline-end: 1px solid var(--line); }
  nav { display: grid; gap: .1rem; margin-block-end: 1.25rem; }
  .nav { display: flex; align-items: center; gap: .6rem; padding: .45rem .6rem; border-radius: .45rem; font-size: .875rem; font-weight: 500; color: var(--muted); text-align: start; }
  .nav:hover { background: var(--soft); color: var(--text); }
  .nav.active { background: var(--soft); color: var(--text); font-weight: 600; }
  .note { height: 100%; display: flex; flex-direction: column; }
  .body { flex: 1; min-height: 0; }
  .main.scroll { overflow: auto; background: var(--bg); }
  .side h2 { padding-inline: .5rem; margin-block-end: .5rem; font-size: .7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); }
  .main { position: relative; overflow: hidden; background: var(--panel); }
  .preview { overflow: hidden; border-inline-start: 1px solid var(--line); background: var(--bg); }
  iframe { width: 100%; height: 100%; border: 0; }
  .hint { padding: 3rem 2rem; text-align: center; }
  .error { position: absolute; inset-inline: 1rem; inset-block-start: .75rem; z-index: 2; padding: .6rem .9rem; border-radius: .5rem; background: var(--soft); color: var(--red); font-size: .85rem; }
</style>
