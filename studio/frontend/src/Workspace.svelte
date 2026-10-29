<script lang="ts">
  import { onDestroy } from "svelte";
  import { Open, Read, Write, StartPreview, StopPreview, Settings as LoadSettings } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Tree from "./Tree.svelte";
  import Editor from "./Editor.svelte";
  import LookPanel from "./LookPanel.svelte";
  import SectionsPanel from "./SectionsPanel.svelte";
  import CommentsPanel from "./CommentsPanel.svelte";
  import CookiesPanel from "./CookiesPanel.svelte";
  import PasswordPanel from "./PasswordPanel.svelte";
  import PublishPanel from "./PublishPanel.svelte";
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
  let status = $state<"saved" | "saving" | "unsaved">("saved");
  let error = $state("");
  let previewURL = $state("");
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
  ]);

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

<div class="ws" class:no-preview={!previewOn}>
  <header class="bar">
    <button class="btn ghost" onclick={back}><ArrowLeft size={16} aria-hidden="true" />{t.back}</button>
    <strong class="title">{site.title}</strong>
    <span class="status muted">{current ? (status === "saved" ? t.saved : status === "saving" ? t.saving : t.unsaved) : ""}</span>
    <span class="spacer"></span>
    <button class="btn ghost" onclick={togglePreview}>{previewOn ? "✓ " : ""}{t.preview}</button>
    <button class="btn ghost" disabled title={t.soon}>{t.sync}</button>
    <button class="btn" disabled title={t.soon}>{t.publish}</button>
  </header>

  <aside class="side">
    <nav>
      {#each screens as { key, label, Icon } (key)}
        <button class="nav" class:active={screen === key} onclick={async () => { await save(); screen = key; error = ""; }}><Icon size={17} strokeWidth={1.75} aria-hidden="true" />{label}</button>
      {/each}
    </nav>
    {#if screen === "write"}
      <h2>{t.notes}</h2>
      <Tree nodes={tree} {current} onPick={pick} />
    {/if}
  </aside>

  <section class="main" class:scroll={screen !== "write"}>
    {#if error}<p class="error">{t.error}: {error}</p>{/if}
    {#if screen === "write"}
      {#if current}
        {#key current}
          <Editor {text} onChange={changed} onSave={save} />
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
        {:else if screen === "publish"}<PublishPanel />
        {/if}
      {/key}
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
  nav { display: grid; gap: .1rem; margin-block-end: 1.25rem; }
  .nav { display: flex; align-items: center; gap: .6rem; padding: .45rem .6rem; border-radius: .45rem; font-size: .875rem; font-weight: 500; color: var(--muted); text-align: start; }
  .nav:hover { background: var(--soft); color: var(--text); }
  .nav.active { background: var(--soft); color: var(--text); font-weight: 600; }
  .main.scroll { overflow: auto; background: var(--bg); }
  .side h2 { padding-inline: .5rem; margin-block-end: .5rem; font-size: .7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); }
  .main { position: relative; overflow: hidden; background: var(--panel); }
  .preview { overflow: hidden; border-inline-start: 1px solid var(--line); background: #fff; }
  iframe { width: 100%; height: 100%; border: 0; }
  .hint { padding: 3rem 2rem; text-align: center; }
  .error { position: absolute; inset-inline: 1rem; inset-block-start: .75rem; z-index: 2; padding: .6rem .9rem; border-radius: .5rem; background: var(--soft); color: var(--red); font-size: .85rem; }
</style>
