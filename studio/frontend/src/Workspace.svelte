<script lang="ts">
  import { onDestroy, tick } from "svelte";
  import { HyprlandSetup, OpenURL, Sites, CreateNote, CreateFolder, Open, Tree as LoadTree, ReadNote, SaveNote, StartPreview, StopPreview, PreviewPage, Settings as LoadSettings } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Tree from "./Tree.svelte";
  import Editor from "./Editor.svelte";
  import Props from "./Props.svelte";
  import Preview from "./Preview.svelte";
  import TerminalPanel from "./TerminalPanel.svelte";
  import FileMenu from "./FileMenu.svelte";
  import SettingsModal, { DOCS } from "./SettingsModal.svelte";
  import Gear from "@lucide/svelte/icons/settings";
  import CircleHelp from "@lucide/svelte/icons/circle-question-mark";
  import { look, setPref } from "./theme.svelte";
  import { ACTIONS, comboOf, keysOf, type Action } from "./keys";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import PanelLeft from "@lucide/svelte/icons/panel-left";
  import Files from "@lucide/svelte/icons/files";
  import Search from "@lucide/svelte/icons/search";
  import SquarePen from "@lucide/svelte/icons/square-pen";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import ChevronsUpDown from "@lucide/svelte/icons/chevrons-up-down";
  import ChevronsDownUp from "@lucide/svelte/icons/chevrons-down-up";
  import { Events } from "@wailsio/runtime";
  import LookPanel from "./LookPanel.svelte";
  import SectionsPanel from "./SectionsPanel.svelte";
  import CommentsPanel from "./CommentsPanel.svelte";
  import CookiesPanel from "./CookiesPanel.svelte";
  import PasswordPanel from "./PasswordPanel.svelte";
  import PublishPanel from "./PublishPanel.svelte";
  import TrashPanel from "./TrashPanel.svelte";
  import AuthorsPanel from "./AuthorsPanel.svelte";
  import AttachmentsPanel from "./AttachmentsPanel.svelte";
  import DeleteSite from "./DeleteSite.svelte";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Users from "@lucide/svelte/icons/users";
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

  let { site, onBack, onSwitch }: { site: SiteInfo; onBack: () => void; onSwitch: (s: SiteInfo) => void } = $props();

  // The files panel, like Obsidian's: tabs (files, search), tools (new
  // note, new folder, expand, collapse), the tree and the site switcher.
  let sideOpen = $state(true);
  let sideTab = $state<"files" | "search">("files");
  let openMap = $state<Record<string, boolean>>({});
  let folder = $state(""); // where new notes go: the folder clicked last
  let creating = $state<"note" | "folder" | null>(null);
  let newName = $state("");
  let query = $state("");
  let switching = $state(false);
  let others = $state<SiteInfo[]>([]);

  const flat = (nodes: Node[]): Node[] => nodes.flatMap((n) => (n.dir ? flat(n.children ?? []) : [n]));
  const folders = (nodes: Node[]): string[] => nodes.flatMap((n) => (n.dir ? [n.path, ...folders(n.children ?? [])] : []));
  const found = $derived.by(() => (query.trim() ? flat(tree).filter((n) => n.name.toLowerCase().includes(query.trim().toLowerCase())).slice(0, 80) : []));
  function expandAll() { for (const f of folders(tree)) openMap[f] = true; }
  const anyOpen = $derived(Object.values(openMap).some(Boolean));
  function reveal(path: string) {
    const parts = path.split("/").slice(0, -1);
    for (let i = 1; i <= parts.length; i++) openMap[parts.slice(0, i).join("/")] = true;
    folder = parts.join("/");
  }
  function startCreate(kind: "note" | "folder") { newName = ""; creating = kind; }

  // The right-click menu of a note or folder.
  let menu = $state<{ node: Node; x: number; y: number } | null>(null);
  async function openMenu(node: Node, e: MouseEvent) {
    await save();
    menu = { node, x: e.clientX, y: e.clientY };
  }
  // After a rename, move, copy or delete: the tree follows, and so does the
  // open note (its new path, or nothing when it went to the trash).
  async function menuChanged({ from, to, open }: { from: string; to: string | null; open?: string }) {
    menu = null;
    error = "";
    tree = (await LoadTree().catch(() => null))?.children ?? tree;
    const inside = from && (current === from || current.startsWith(from + "/"));
    if (inside && to === null) { current = ""; text = ""; pending = ""; status = "saved"; }
    else if (inside && to) { current = to + current.slice(from.length); reveal(current); follow(current); }
    if (to && !open) reveal(to);
    if (open) { reveal(open); screen = "write"; await pick(open); }
  }
  async function menuNew(kind: "note" | "folder", f: string) {
    folder = f;
    openMap[f] = true;
    startCreate(kind);
  }
  async function finishCreate() {
    const kind = creating, name = newName.trim();
    creating = null;
    if (!kind || !name) return;
    try {
      const rel = kind === "note" ? await CreateNote(folder, name) : await CreateFolder(folder, name);
      tree = (await LoadTree()).children ?? [];
      if (kind === "note") { reveal(rel); screen = "write"; await pick(rel); }
      else { reveal(rel + "/x"); folder = rel; }
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }
  async function loadSites() { others = ((await Sites().catch(() => [])) ?? []).filter((o) => o.dir !== site.dir); }

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
  let previewOn = $state(!look.previewOff);

  // The files panel and the preview are as wide as the user drags them
  // (their edge, like Obsidian), within limits that keep the editor usable;
  // saved in the Studio's choices. A double-click goes back to the default.
  const SIDE = 256, SIDE_MIN = 180, SIDE_MAX = 480, PREV_MIN = 300, EDITOR_MIN = 360, RIBBON = 48;
  let sideW = $state(look.sideWidth || SIDE);
  let prevW = $state(look.previewWidth || 0); // 0: half of the room
  let resizing = $state(false);
  let previewEl = $state<HTMLElement>();
  const room = () => window.innerWidth - RIBBON;
  const clampSide = (w: number) => Math.round(Math.min(Math.max(w, SIDE_MIN), SIDE_MAX, room() - EDITOR_MIN - (previewOn ? PREV_MIN : 0)));
  const clampPrev = (w: number) => Math.round(Math.min(Math.max(w, PREV_MIN), room() - (sideOpen ? sideW : 0) - EDITOR_MIN));
  function drag(e: MouseEvent, which: "side" | "preview") {
    e.preventDefault();
    const x0 = e.clientX, w0 = which === "side" ? sideW : prevW || previewEl?.getBoundingClientRect().width || PREV_MIN;
    resizing = true;
    const move = (m: MouseEvent) => {
      if (which === "side") sideW = clampSide(w0 + m.clientX - x0);
      else prevW = clampPrev(w0 - (m.clientX - x0));
    };
    const up = () => {
      resizing = false;
      removeEventListener("mousemove", move);
      removeEventListener("mouseup", up);
      setPref(which === "side" ? "sideWidth" : "previewWidth", which === "side" ? sideW : prevW);
    };
    addEventListener("mousemove", move);
    addEventListener("mouseup", up);
  }
  function resetWidth(which: "side" | "preview") {
    if (which === "side") { sideW = SIDE; setPref("sideWidth", 0); }
    else { prevW = 0; setPref("previewWidth", 0); }
  }
  // The Studio's shortcuts (Settings > Shortcuts).
  function shortcut(e: KeyboardEvent) {
    const combo = comboOf(e);
    if (!combo) return;
    const a = ACTIONS.find((x) => keysOf(look.keys, x.key) === combo)?.key as Action | undefined;
    if (!a) return;
    e.preventDefault();
    if (a === "terminal") termOpen = !termOpen;
    else if (a === "preview") togglePreview();
    else if (a === "sidebar") sideOpen = !sideOpen;
    else if (a === "settings") prefsOpen = true;
    else if (a === "search") { sideOpen = true; sideTab = "search"; }
    else if (a === "newNote") { sideOpen = true; sideTab = "files"; screen = "write"; startCreate("note"); }
  }

  // A smaller window keeps the editor's room.
  function fit() {
    sideW = clampSide(sideW);
    if (prevW) prevW = clampPrev(prevW);
  }
  let prefsOpen = $state(false);
  let prefsTab = $state("general");
  let deletingSite = $state(""); // a site in the delete window (from the switcher)

  // Once: the tip about the shortcut (Alt+Space by default) that shows and
  // hides the Studio from anywhere. On a Hyprland that isn't Omarchy, the
  // user pastes a line first: the tip leads there.
  const showTip = $derived(!!look.toggleKey && !look.shortcutTip);
  let manual = $state(false);
  HyprlandSetup(look.toggleKey ?? "").then((h) => (manual = h.on && !h.auto)).catch(() => {});
  function okTip() { setPref("shortcutTip", true); }
  function chooseTip() { setPref("shortcutTip", true); prefsTab = "keys"; prefsOpen = true; }
  let previewBusy = $state(false);

  // The screen on the right of the menu: writing or a settings panel.
  let screen = $state("write");
  let settings = $state<Settings | null>(null);
  const reload = () => LoadSettings().then((s) => (settings = s)).catch((e) => (error = String(e)));
  const screens = $derived([
    { key: "write", label: t.nav.write, Icon: PenLine },
    { key: "look", label: t.nav.look, Icon: Palette },
    { key: "authors", label: t.nav.authors, Icon: Users },
    { key: "attach", label: t.nav.attach, Icon: Paperclip },
    ...(site.model === "bento" ? [{ key: "sections", label: t.nav.sections, Icon: LayoutGrid }] : []),
    { key: "comments", label: t.nav.comments, Icon: MessageCircle },
    { key: "cookies", label: t.nav.cookies, Icon: Cookie },
    { key: "password", label: t.nav.password, Icon: KeyRound },
    { key: "publish", label: t.nav.publish, Icon: Upload },
    { key: "trash", label: t.nav.trash, Icon: Trash2 },
  ]);

  // The terminal at the bottom, and its height (dragging its top edge).
  let termOpen = $state(false);
  // Once opened, the terminals stay alive: closing the panel only hides it.
  let termStarted = $state(false);
  $effect(() => { if (termOpen) termStarted = true; });
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
    // Our own saves come back here too: only a text that is neither what
    // we wrote last nor what is being typed came from outside.
    if (!note || same(note.body, pending) || same(note.body, written)) return;
    if (status === "saved") applyOutside(note);
    else outside = true;
  });
  function applyOutside(note: { body: string; props: Record<string, any> | null; kind: string }) {
    text = note.body;
    pending = note.body;
    written = note.body;
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
  let written = ""; // the text the editor wrote to disk last
  const same = (a: string, b: string) => a.trimEnd() === b.trimEnd();

  Open(site.dir)
    .then((root) => {
      tree = root.children ?? [];
      reload();
      // The note open when the Studio last closed on this site.
      const last = look.lastNotes?.[site.dir];
      if (last && flat(tree).some((n) => n.path === last)) pick(last);
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
      written = pending;
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
      written = text;
      current = path;
      if (look.lastNotes?.[site.dir] !== path) setPref("lastNotes", { ...(look.lastNotes ?? {}), [site.dir]: path });
      reveal(path);
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

<svelte:window onresize={fit} onkeydown={shortcut} onclick={() => (switching = false)} />

<div class="ws" class:no-preview={!previewOn} class:no-side={!sideOpen} class:resizing style:--side="{sideW}px" style:--prev={prevW ? `${prevW}px` : "minmax(0, 1fr)"}>
  <!-- The ribbon, like Obsidian's: icons only, the name on hover. -->
  <nav class="ribbon">
    <button class="rib" title={t.side.toggle} onclick={() => (sideOpen = !sideOpen)}><PanelLeft size={19} strokeWidth={1.75} /></button>
    <span class="sep"></span>
    {#each screens as { key, label, Icon } (key)}
      <button class="rib" class:active={screen === key} title={label} onclick={async () => { await save(); autostart = ""; screen = key; error = ""; }}><Icon size={19} strokeWidth={1.75} /></button>
    {/each}
    <span class="grow"></span>
    <button class="rib" class:active={termOpen} title={t.term.button + (keysOf(look.keys, "terminal") ? ` (${keysOf(look.keys, "terminal")})` : "")} onclick={() => (termOpen = !termOpen)}><SquareTerminal size={19} strokeWidth={1.75} /></button>
  </nav>

  {#if sideOpen}
    <aside class="side">
      <div class="resizer right" role="separator" aria-orientation="vertical" aria-label={t.side.resize} title={t.side.resize} onmousedown={(e) => drag(e, "side")} ondblclick={() => resetWidth("side")}></div>
      <div class="side-tabs">
        <button class="tab" class:on={sideTab === "files"} title={t.side.files} onclick={() => (sideTab = "files")}><Files size={17} strokeWidth={1.75} /></button>
        <button class="tab" class:on={sideTab === "search"} title={t.side.search} onclick={() => (sideTab = "search")}><Search size={17} strokeWidth={1.75} /></button>
      </div>
      {#if sideTab === "files"}
        <div class="side-tools">
          <button title={t.side.newNote} onclick={() => startCreate("note")}><SquarePen size={17} strokeWidth={1.75} /></button>
          <button title={t.side.newFolder} onclick={() => startCreate("folder")}><FolderPlus size={17} strokeWidth={1.75} /></button>
          <!-- One button: collapse when a folder is open, else expand. -->
          {#if anyOpen}
            <button title={t.side.collapse} onclick={() => (openMap = {})}><ChevronsDownUp size={17} strokeWidth={1.75} /></button>
          {:else}
            <button title={t.side.expand} onclick={expandAll}><ChevronsUpDown size={17} strokeWidth={1.75} /></button>
          {/if}
        </div>
        {#if creating}
          <div class="create">
            <small class="muted">{creating === "note" ? t.side.noteIn : t.side.folderIn} {folder || site.title}</small>
            <!-- svelte-ignore a11y_autofocus -->
            <input type="text" autofocus bind:value={newName} placeholder={creating === "note" ? t.side.notePh : t.side.folderPh}
              onkeydown={(e) => { if (e.key === "Enter") finishCreate(); if (e.key === "Escape") creating = null; }} onblur={() => setTimeout(() => (creating = null), 150)} />
          </div>
        {/if}
        <div class="files">
          <Tree nodes={tree} {current} {folder} open={openMap} onPick={(p) => { screen = "write"; pick(p); }} onFolder={(p) => (folder = p)} onMenu={openMenu} />
        </div>
      {:else}
        <div class="side-search">
          <!-- svelte-ignore a11y_autofocus -->
          <input type="text" autofocus bind:value={query} placeholder={t.side.searchPh} />
          <ul>
            {#each found as n (n.path)}
              <li><button class="hit" class:active={n.path === current} onclick={() => { screen = "write"; pick(n.path); }}><span>{n.name}</span><small class="muted">{n.path.split("/").slice(0, -1).join(" / ")}</small></button></li>
            {/each}
            {#if query.trim() && !found.length}<li class="muted none">{t.side.nothing}</li>{/if}
          </ul>
        </div>
      {/if}
      <!-- The site switcher, like Obsidian's vault switcher. -->
      <div class="switcher">
        <button class="vault" onclick={(e) => { e.stopPropagation(); switching = !switching; if (switching) loadSites(); }}><ChevronsUpDown size={15} /><span>{site.title}</span></button>
        <button class="foot" title={t.prefs.helpIcon} onclick={() => OpenURL(DOCS)}><CircleHelp size={17} strokeWidth={1.75} /></button>
        <button class="foot" title={t.prefs.gear} onclick={() => (prefsOpen = true)}><Gear size={17} strokeWidth={1.75} /></button>
        {#if switching}
          <div class="vault-menu" role="menu">
            {#each others as o (o.dir)}
              <div class="vault-row">
                <button onclick={async () => { await save(); StopPreview(); onSwitch(o); }}>{o.title}</button>
                <button class="vault-del" title={t.del.deleteSite.replace("{name}", o.title)} aria-label={t.del.deleteSite.replace("{name}", o.title)} onclick={(e) => { e.stopPropagation(); switching = false; deletingSite = o.dir; }}><Trash2 size={15} /></button>
              </div>
            {/each}
            <button class="all" onclick={back}>{t.side.allSites}</button>
          </div>
        {/if}
      </div>
    </aside>
  {/if}

  <header class="bar">
    {#if screen === "write" && current}
      <span class="crumbs muted">{current.split("/").slice(0, -1).join(" / ")}{current.includes("/") ? " / " : ""}</span><strong class="title">{current.split("/").pop()?.replace(/\.md$/, "")}</strong>
      <span class="status muted">{status === "saved" ? t.saved : status === "saving" ? t.saving : t.unsaved}</span>
    {:else}
      <strong class="title">{site.title}</strong>
    {/if}
    <span class="spacer"></span>
    <button class="btn ghost" onclick={togglePreview}>{previewOn ? "✓ " : ""}{t.preview}</button>
    <button class="btn ghost" onclick={() => go("publish", "sync")}>{t.sync}</button>
    <button class="btn" onclick={() => go("publish", "publish")}>{t.publish}</button>
  </header>

  <section class="main" class:scroll={screen !== "write"}>
    {#if error}<p class="error">{t.error}: {error}</p>{/if}
    {#if outside}<div class="outside"><span>{t.term.outside}</span><button class="btn ghost" onclick={reloadOutside}>{t.term.reload}</button><button class="btn ghost" onclick={() => (outside = false)}>{t.term.keep}</button></div>{/if}
    {#if screen === "write"}
      {#if current}
        {#key current + ":" + reloadN}
          <div class="note">
            <Props props={noteProps} {kind} isIndex={current.split("/").pop() === "_index.md"} onEdit={editProp} />
            <div class="body"><Editor {text} note={current} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onChange={changed} onSave={save} onError={(e) => (error = e)} /></div>
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
        {:else if screen === "authors"}<AuthorsPanel onTree={async () => (tree = (await LoadTree()).children ?? [])} />
        {:else if screen === "attach"}<AttachmentsPanel onOpenNote={openNote} />
        {:else if screen === "trash"}<TrashPanel {reload} onTree={(n) => (tree = n.children ?? [])} />
        {/if}
      {/key}
    {/if}
  </section>

  {#if previewOn}
    <section class="preview" bind:this={previewEl}>
      <div class="resizer left" role="separator" aria-orientation="vertical" aria-label={t.side.resize} title={t.side.resize} onmousedown={(e) => drag(e, "preview")} ondblclick={() => resetWidth("preview")}></div>
      {#if previewURL}
        <Preview src={previewSrc || previewURL} title={t.preview} />
      {:else}
        <p class="hint muted">{previewBusy ? t.previewStarting : t.previewOff}</p>
      {/if}
    </section>
  {/if}
  {#if prefsOpen}
    <SettingsModal start={prefsTab} onClose={() => { prefsOpen = false; prefsTab = "general"; }} onPublish={() => { prefsOpen = false; screen = "publish"; }} />
  {/if}
  {#if deletingSite}<DeleteSite dir={deletingSite} onClose={() => (deletingSite = "")} onDone={() => { deletingSite = ""; loadSites(); }} />{/if}
  {#if showTip}
    <!-- Centered like a modal, so it isn't missed. -->
    <div class="tip-shade" role="presentation"></div>
    <div class="tip" role="dialog" aria-modal="true" aria-label={t.keys.tipTitle}>
      <img src="/blups/blup.svg" alt="" />
      <div>
        <b>{t.keys.tipTitle}</b>
        <p>{(manual ? t.keys.tipManual : t.keys.tipText).replace("{keys}", (look.toggleKey ?? "").replace("Space", t.keys.space))}</p>
        <div class="tip-actions">
          {#if manual}<button class="btn" onclick={chooseTip}>{t.keys.tipHow}</button>
          {:else}<button class="btn" onclick={okTip}>{t.keys.tipOk}</button><button class="btn ghost" onclick={chooseTip}>{t.keys.tipOther}</button>{/if}
        </div>
      </div>
    </div>
  {/if}
  {#if menu}
    <FileMenu node={menu.node} x={menu.x} y={menu.y} {current} folders={folders(tree)} onClose={() => (menu = null)} onChanged={menuChanged} onNew={menuNew} />
  {/if}
  {#if termStarted}
    <div class="term" class:min={!termOpen} style:height="{termHeight}px">
      <div class="grip" role="separator" aria-orientation="horizontal" onmousedown={resizeTerm}></div>
      <TerminalPanel visible={termOpen} onClose={() => (termOpen = false)} />
    </div>
  {/if}
</div>

<style>
  /* Columns: ribbon, files, editor, preview. The top bar starts after the
     files; the terminal sits under the editor and the preview. */
  .ws { height: 100%; display: grid; grid-template-columns: 3rem var(--side) minmax(0, 1fr) var(--prev); grid-template-rows: auto minmax(0, 1fr) auto; }
  .ws.no-preview { grid-template-columns: 3rem var(--side) minmax(0, 1fr); }
  .ws.no-side { grid-template-columns: 3rem minmax(0, 1fr) var(--prev); }
  /* The edges to drag: a thin line that lights up under the mouse. */
  .tip-shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .45); }
  .tip { position: fixed; z-index: 41; inset: 0; margin: auto; height: fit-content; display: grid; justify-items: center; gap: .9rem; width: min(26rem, calc(100% - 2rem)); padding: 1.75rem 1.5rem 1.5rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); text-align: center; }
  .tip img { width: 4rem; height: 4rem; }
  .tip b { font-size: 1.1rem; }
  .tip p { margin-block: .4rem 1rem; font-size: .9rem; color: var(--muted); }
  .tip-actions { display: flex; flex-wrap: wrap; justify-content: center; gap: .5rem; }
  .resizer { position: absolute; z-index: 5; inset-block: 0; width: 7px; cursor: col-resize; }
  .resizer.right { inset-inline-end: -4px; }
  .resizer.left { inset-inline-start: 0; } /* the preview cuts what sticks out */
  .resizer.left::after { inset-inline-start: 0; }
  .resizer::after { content: ""; position: absolute; inset-block: 0; inset-inline-start: 3px; width: 2px; background: transparent; transition: background .1s; }
  .resizer:hover::after, .ws.resizing .resizer:active::after { background: var(--blue); }
  .ws.resizing { cursor: col-resize; user-select: none; }
  .ws.resizing :global(iframe) { pointer-events: none; }
  .ws.no-side.no-preview { grid-template-columns: 3rem minmax(0, 1fr); }
  .ribbon { grid-row: 1 / -1; grid-column: 1; display: flex; flex-direction: column; align-items: center; gap: .2rem; padding-block: .6rem; border-inline-end: 1px solid var(--line); background: var(--panel); }
  .rib { display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .45rem; color: var(--muted); }
  .rib:hover { background: var(--soft); color: var(--text); }
  .rib.active { background: var(--soft); color: var(--text); }
  .sep { width: 1.4rem; height: 1px; margin-block: .35rem; background: var(--line); }
  .grow { flex: 1; }
  .side { position: relative; grid-row: 1 / -1; grid-column: 2; display: flex; flex-direction: column; min-height: 0; border-inline-end: 1px solid var(--line); background: var(--panel); }
  .side-tabs { display: flex; gap: .2rem; padding: .55rem .6rem; border-block-end: 1px solid var(--line); }
  .tab { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .tab:hover, .tab.on { background: var(--soft); color: var(--text); }
  .side-tools { display: flex; justify-content: center; gap: .3rem; padding: .45rem; }
  .side-tools button { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .side-tools button:hover { background: var(--soft); color: var(--text); }
  .create { display: grid; gap: .25rem; padding: .25rem .6rem .5rem; }
  .create small { font-size: .72rem; }
  .files { flex: 1; min-height: 0; overflow: auto; padding: 0 .4rem .75rem; }
  .side-search { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: .5rem; padding: .6rem; }
  .side-search ul { list-style: none; padding: 0; overflow: auto; }
  .hit { display: grid; width: 100%; text-align: start; padding: .35rem .5rem; border-radius: .35rem; font-size: .875rem; }
  .hit small { font-size: .72rem; }
  .hit:hover, .hit.active { background: var(--soft); }
  .none { padding: .5rem; font-size: .85rem; }
  .switcher { position: relative; display: flex; align-items: center; gap: .15rem; padding: .45rem .6rem; border-block-start: 1px solid var(--line); }
  .foot { display: grid; place-items: center; flex: none; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .foot:hover { background: var(--soft); color: var(--text); }
  .vault { display: flex; align-items: center; gap: .45rem; flex: 1; min-width: 0; padding: .4rem .5rem; border-radius: .4rem; font-size: .875rem; font-weight: 600; text-align: start; }
  .vault span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vault:hover { background: var(--soft); }
  .vault-menu { position: absolute; z-index: 6; inset-inline: .6rem; inset-block-end: calc(100% - .2rem); display: grid; padding: .3rem; border: 1px solid var(--line); border-radius: .55rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .18); }
  .vault-menu button { text-align: start; padding: .45rem .6rem; border-radius: .35rem; font-size: .875rem; }
  .vault-menu button:hover { background: var(--soft); }
  .vault-row { display: flex; align-items: center; }
  .vault-row > button:first-child { flex: 1; min-width: 0; text-align: start; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vault-del { display: grid; place-items: center; flex: none; width: 1.9rem !important; padding: .35rem !important; color: var(--muted); }
  .vault-del:hover { color: var(--red); }
  .vault-menu .all { border-block-start: 1px solid var(--line); margin-block-start: .2rem; padding-block-start: .55rem; color: var(--muted); }
  .bar { grid-row: 1; grid-column: 3 / -1; display: flex; align-items: center; gap: .6rem; padding: .55rem 1rem; border-block-end: 1px solid var(--line); background: var(--panel); min-width: 0; }
  .ws.no-side .bar { grid-column: 2 / -1; }
  .crumbs { font-size: .85rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 40%; }
  .title { font-size: .95rem; white-space: nowrap; }
  .status { font-size: .8rem; }
  .spacer { flex: 1; }
  .main { grid-row: 2; position: relative; overflow: hidden; background: var(--panel); }
  .main.scroll { overflow: auto; background: var(--bg); }
  .preview { position: relative; grid-row: 2; overflow: hidden; border-inline-start: 1px solid var(--line); background: var(--bg); }
  .term { grid-row: 3; grid-column: 3 / -1; position: relative; min-height: 0; }
  .ws.no-side .term { grid-column: 2 / -1; }
  .term.min { display: none; }
  .grip { position: absolute; z-index: 3; inset-inline: 0; inset-block-start: -3px; height: 6px; cursor: row-resize; }
  .outside { position: absolute; z-index: 2; inset-inline: 1rem; inset-block-start: .75rem; display: flex; gap: .5rem; align-items: center; padding: .5rem .75rem; border-radius: .5rem; background: var(--soft); font-size: .85rem; }
  .outside span { flex: 1; }
  .note { height: 100%; display: flex; flex-direction: column; }
  .body { flex: 1; min-height: 0; }
  .hint { padding: 3rem 2rem; text-align: center; }
  .error { position: absolute; inset-inline: 1rem; inset-block-start: .75rem; z-index: 2; padding: .6rem .9rem; border-radius: .5rem; background: var(--soft); color: var(--red); font-size: .85rem; }
</style>
