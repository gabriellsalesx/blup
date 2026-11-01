<script lang="ts">
  import { onDestroy, tick, untrack } from "svelte";
  import { HyprlandSetup, OpenURL, Sites, CreateNote, CreateFolder, CreateCanvas, CreateDrawing, CreateDatabase, SaveDatabaseSchema, DailyNote, RenameItem, MoveItem, Open, Tree as LoadTree, ReadNote, SaveNote, StartPreview, StopPreview, PreviewPage, NoteAt, Settings as LoadSettings, CloudHistory, BlupAccount, SetSyncRemote } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Tree from "./Tree.svelte";
  import Editor from "./Editor.svelte";
  import CanvasView from "./CanvasView.svelte";
  import DrawingView from "./DrawingView.svelte";
  import DatabaseView from "./DatabaseView.svelte";
  import DatabaseNew from "./DatabaseNew.svelte";
  import Database from "@lucide/svelte/icons/database";
  import CalendarDays from "@lucide/svelte/icons/calendar-days";
  import type { DBSchema } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Template } from "./db";
  import { boardKind, fileTitle, type NewKind } from "./boards";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import Props from "./Props.svelte";
  import NoteProps from "./NoteProps.svelte";
  import Backlinks from "./Backlinks.svelte";
  import GraphView from "./GraphView.svelte";
  import Waypoints from "@lucide/svelte/icons/waypoints";
  import Preview from "./Preview.svelte";
  import TerminalPanel from "./TerminalPanel.svelte";
  import HistoryModal from "./HistoryModal.svelte";
  import FileMenu from "./FileMenu.svelte";
  import SettingsModal, { DOCS } from "./SettingsModal.svelte";
  import Gear from "@lucide/svelte/icons/settings";
  import CircleHelp from "@lucide/svelte/icons/circle-question-mark";
  import { look, setPref } from "./theme.svelte";
  import { ACTIONS, comboOf, keysOf, type Action } from "./keys";
  import { toast } from "./toast.svelte";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import PanelLeft from "@lucide/svelte/icons/panel-left";
  import Files from "@lucide/svelte/icons/files";
  import Search from "@lucide/svelte/icons/search";
  import SquarePen from "@lucide/svelte/icons/square-pen";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import ChevronsUpDown from "@lucide/svelte/icons/chevrons-up-down";
  import ChevronsDownUp from "@lucide/svelte/icons/chevrons-down-up";
  import { Events } from "@wailsio/runtime";
  import CloudCheck from "@lucide/svelte/icons/cloud-check";
  import CloudAlert from "@lucide/svelte/icons/cloud-alert";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
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
  // A new note or folder is named right where it will be: a field in its
  // folder in the tree (opened), not at the top.
  function startCreate(kind: "note" | "folder") {
    if (folder) openMap[folder] = true;
    sideOpen = true; sideTab = "files";
    creating = kind;
  }

  // A new canvas or drawing, right away, no name to ask: "Drawing
  // 2026-11-01 14.05.33", in its own folder at the top (or in the folder
  // whose menu made it), then open.
  async function makeBoard(kind: "canvas" | "drawing", where?: string) {
    const b = t.boards;
    const p = (n: number) => String(n).padStart(2, "0"), d = new Date();
    const stamp = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}.${p(d.getMinutes())}.${p(d.getSeconds())}`;
    const name = `${kind === "canvas" ? b.canvasName : b.drawingName} ${stamp}`;
    const dir = where ?? (kind === "canvas" ? b.canvasFolder : b.drawingFolder);
    const rel = await (kind === "canvas" ? CreateCanvas : CreateDrawing)(dir, name);
    tree = (await LoadTree()).children ?? [];
    return rel;
  }
  // The note of the day: made (or opened) in the daily folder, with the
  // time added at its end and the cursor there, ready to write.
  let atEnd = $state(false);
  async function dailyNote() {
    try {
      await save();
      const rel = await DailyNote(t.daily.folder, t.daily.tag);
      tree = (await LoadTree()).children ?? [];
      reveal(rel);
      screen = "write";
      atEnd = true;
      if (rel === current) { const note = await ReadNote(rel); applyOutside(note); }
      else await pick(rel);
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }

  // A new database: its template and name are asked (DatabaseNew), then
  // it opens.
  let dbNew = $state(false);
  async function newDatabase(title: string, tpl: Template) {
    dbNew = false;
    try {
      const rel = await CreateDatabase("", title, tpl.schema);
      tree = (await LoadTree()).children ?? [];
      reveal(rel);
      screen = "write";
      await pick(rel);
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }
  // The open note is a record of a database: its fields (in Properties).
  let noteDb = $state<{ table: string; schema: DBSchema } | null>(null);
  function dbOption(key: string, value: string, color: string) {
    const db = noteDb;
    const col = db?.schema.columns?.find((c) => c.key === key);
    if (!db || !col) return;
    col.options = [...(col.options ?? []), { value, color }];
    SaveDatabaseSchema(db.table, $state.snapshot(db.schema) as DBSchema).catch((e) => (error = String(e)));
  }

  async function quickBoard(kind: "canvas" | "drawing", where?: string) {
    try {
      const rel = await makeBoard(kind, where);
      reveal(rel);
      screen = "write";
      await pick(rel);
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }

  // Renaming in the tree itself: the item's name becomes a field.
  let renaming = $state("");
  async function rename(n: Node, name: string | null) {
    renaming = "";
    name = name?.trim() ?? "";
    if (!name || name === n.name) return;
    try {
      if (n.path === current) await save(); // what was typed goes with it
      const to = await RenameItem(n.path, name);
      await menuChanged({ from: n.path, to });
      toast(t.toast.noteRenamed.replace("{name}", fileTitle(to)));
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }

  // Dragging in the tree: a note or a folder into another folder (or to
  // the top, the room under the tree); links follow, like Move to. Only in
  // a vault of notes for now (a site's folders are its sections).
  async function moveTo(path: string, f: string) {
    const parent = path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
    if (!path || parent === f || f === path || f.startsWith(path + "/")) return;
    try {
      const to = await MoveItem(path, f);
      await menuChanged({ from: path, to });
      if (f) openMap[f] = true;
      toast(t.toast.noteMoved.replace("{folder}", f || t.menu.top));
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }
  function overTop(e: DragEvent) {
    if (notesVault && e.dataTransfer?.types.includes("application/x-blup-item")) { e.preventDefault(); e.dataTransfer.dropEffect = "move"; }
  }
  function dropTop(e: DragEvent) {
    const path = e.dataTransfer?.getData("application/x-blup-item");
    if (!notesVault || !path) return;
    e.preventDefault();
    moveTo(path, "");
  }

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
  async function menuNew(kind: NewKind, f: string) {
    openMap[f] = true;
    if (kind === "canvas" || kind === "drawing") { quickBoard(kind, f); return; }
    folder = f;
    startCreate(kind);
  }
  async function finishCreate(typed: string | null) {
    const kind = creating, name = typed?.trim() ?? "";
    creating = null;
    if (!kind || !name) return;
    try {
      const rel = kind === "note" ? await CreateNote(folder, name) : await CreateFolder(folder, name);
      tree = (await LoadTree()).children ?? [];
      if (kind !== "folder") { reveal(rel); screen = "write"; await pick(rel); }
      else { reveal(rel + "/x"); folder = rel; }
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }
  async function loadSites() { others = ((await Sites().catch(() => [])) ?? []).filter((o) => o.dir !== site.dir); }

  let tree = $state<Node[]>([]);
  let current = $state(""); // path of the open note (or canvas, or drawing)
  const board = $derived(boardKind(current));
  // Full screen: a canvas or a drawing alone in the window (F, or Esc to
  // leave); only while one is open.
  let full = $state(false);
  let text = $state("");
  let noteProps = $state<Record<string, any>>({});
  let kind = $state("");
  let edits: Record<string, any> = {}; // properties changed since the last save
  let status = $state<"saved" | "saving" | "unsaved">("saved");
  let error = $state("");
  let previewURL = $state("");
  let previewSrc = $state(""); // the page shown: the open note's, when known
  // Workspace is made again for each site (keyed by its folder).
  let previewOn = $state(!look.previewOff && untrack(() => site.kind) !== "notes");

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
  // The Studio's shortcuts (Settings > Shortcuts); some are for vaults.
  const VAULT_ONLY: Action[] = ["dailyNote", "graph", "newCanvas", "newDrawing", "newDatabase"];
  function shortcut(e: KeyboardEvent) {
    const combo = comboOf(e);
    if (!combo) return;
    const a = ACTIONS.find((x) => keysOf(look.keys, x.key) === combo)?.key as Action | undefined;
    if (!a || (!notesVault && VAULT_ONLY.includes(a))) return; // a site's keys stay the editor's
    e.preventDefault();
    if (a === "terminal") termOpen = !termOpen;
    else if (a === "preview") togglePreview();
    else if (a === "sidebar") sideOpen = !sideOpen;
    else if (a === "settings") prefsOpen = true;
    else if (a === "search") { sideOpen = true; sideTab = "search"; }
    else if (a === "newNote") { sideOpen = true; sideTab = "files"; screen = "write"; startCreate("note"); }
    else if (a === "fullScreen") { if (board) full = !full; }
    else if (a === "dailyNote") dailyNote();
    else if (a === "graph") { save(); screen = screen === "graph" ? "write" : "graph"; }
    else if (a === "newCanvas") quickBoard("canvas");
    else if (a === "newDrawing") quickBoard("drawing");
    else if (a === "newDatabase") dbNew = true;
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
  // The invitation to Blup Cloud: once, to whoever has a site and no account.
  let signedIn = $state(true);
  BlupAccount().then((a) => (signedIn = a.signedIn)).catch(() => {});
  const cloudTip = $derived(!look.cloudTip && !signedIn);
  async function cloudYes() {
    setPref("cloudTip", true);
    await SetSyncRemote("blup").catch(() => {});
    go("publish", "sync");
  }
  function chooseTip() { setPref("shortcutTip", true); prefsTab = "keys"; prefsOpen = true; }
  let previewBusy = $state(false);

  // The screen on the right of the menu: writing or a settings panel.
  let screen = $state("write");
  const fullOn = $derived(full && !!board && screen === "write");
  let settings = $state<Settings | null>(null);
  const reload = () => LoadSettings().then((s) => (settings = s)).catch((e) => (error = String(e)));
  // A vault of notes: the editor, attachments, sync and trash; no site.
  const notesVault = $derived(site.kind === "notes");
  const screens = $derived(notesVault ? [
    { key: "write", label: t.nav.write, Icon: PenLine },
    { key: "graph", label: t.graph.title, Icon: Waypoints },
    { key: "attach", label: t.nav.attach, Icon: Paperclip },
    { key: "publish", label: t.sync, Icon: RefreshCw },
    { key: "trash", label: t.nav.trash, Icon: Trash2 },
  ] : [
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
    // A canvas or a drawing looks after itself.
    if (!current || board || !list.some((p) => p.toLowerCase() === current.toLowerCase())) return;
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

  // Live sync through Blup Cloud (paid plans): its state on the Sync button.
  let cloud = $state<{ state: string; error?: string } | null>(null);
  const offCloud = Events.On("cloudSync", (ev: any) => (cloud = ev.data));
  onDestroy(() => offCloud());
  // Earlier versions on Blup Cloud: checked when the site opens and after
  // each live sync (signing in or changing plan turns it on).
  let cloudHistory = $state(false);
  let historyFor = $state("");
  const checkHistory = () => CloudHistory().then((on) => (cloudHistory = on)).catch(() => {});
  checkHistory();
  $effect(() => { if (cloud?.state === "synced" && !cloudHistory) checkHistory(); });
  const cloudTitle = $derived(!cloud ? undefined : cloud.state === "error" ? t.cloudError.replace("{error}", cloud.error ?? "") : cloud.state === "syncing" ? t.cloudSyncing : t.cloudSynced);

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

  // A note's; a canvas or a drawing saves itself. In a vault, a note keeps
  // when it was made and when it changed last (Properties).
  const today = () => { const d = new Date(), p = (n: number) => String(n).padStart(2, "0"); return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`; };
  async function save() {
    if (status === "saved" || !current || board) return;
    clearTimeout(timer);
    status = "saving";
    if (notesVault) {
      const d = today();
      if (!noteProps.created) { noteProps.created = d; edits.created = d; }
      if (pending !== written && !("updated" in edits) && String(noteProps.updated ?? "").slice(0, 10) !== d) { noteProps.updated = d; edits.updated = d; }
    }
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

  async function pick(path: string, page = "") {
    if (path === current) return;
    await save();
    // A canvas or a drawing: its own screen reads it.
    if (boardKind(path)) {
      current = path;
      text = ""; pending = ""; written = ""; noteProps = {}; edits = {};
      status = "saved";
      error = "";
      if (look.lastNotes?.[site.dir] !== path) setPref("lastNotes", { ...(look.lastNotes ?? {}), [site.dir]: path });
      reveal(path);
      return;
    }
    try {
      const note = await ReadNote(path);
      text = note.body;
      noteProps = note.props ?? {};
      kind = note.kind;
      noteDb = note.database && note.table ? { table: note.table, schema: note.database } : null;
      edits = {};
      pending = text;
      written = text;
      current = path;
      if (look.lastNotes?.[site.dir] !== path) setPref("lastNotes", { ...(look.lastNotes ?? {}), [site.dir]: path });
      reveal(path);
      // Picked in the preview: it is on that page already.
      if (page) previewSrc = page;
      else follow(path);
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

  // A link followed in the preview opens its note in the editor.
  async function navigated(page: string) {
    const note = await NoteAt(page).catch(() => "");
    if (!note || note === current) return;
    await pick(note, new URL(page, previewURL).href);
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

<!-- A vault note's top, scrolling with its text: its name (typing a new
     one renames the file) and its properties. -->
{#snippet noteHead()}
  <input class="inline-title" value={fileTitle(current)} aria-label={t.menu.rename} spellcheck="false"
    onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); } else if (e.key === "Escape") { e.currentTarget.value = fileTitle(current); e.currentTarget.blur(); } }}
    onblur={(e) => rename({ path: current, name: fileTitle(current), dir: false } as Node, e.currentTarget.value)} />
  <NoteProps props={noteProps} database={noteDb?.schema} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onOpenNote={openNote} onNewOption={dbOption} onEdit={(k, v) => { noteProps[k] = v; editProp(k, v); }} />
{/snippet}

<svelte:window onresize={fit} onkeydown={shortcut} onclick={() => (switching = false)} />

<div class="ws" class:no-preview={!previewOn} class:no-side={!sideOpen} class:resizing style:--side="{sideW}px" style:--prev={prevW ? `${prevW}px` : "minmax(0, 1fr)"}>
  <!-- The ribbon, like Obsidian's: icons only, the name on hover. -->
  <nav class="ribbon">
    <button class="rib" title={t.side.toggle} onclick={() => (sideOpen = !sideOpen)}><PanelLeft size={19} strokeWidth={1.75} /></button>
    <span class="sep"></span>
    {#each screens as { key, label, Icon } (key)}
      <button class="rib" class:active={screen === key} title={label} onclick={async () => { await save(); autostart = ""; screen = key; error = ""; }}><Icon size={19} strokeWidth={1.75} /></button>
    {/each}
    {#if notesVault}
      <!-- Making things: a canvas, a drawing (opened right away). -->
      <span class="sep"></span>
      <button class="rib" title={t.boards.newCanvas} onclick={() => quickBoard("canvas")}><LayoutDashboard size={19} strokeWidth={1.75} /></button>
      <button class="rib" title={t.boards.newDrawing} onclick={() => quickBoard("drawing")}><PenTool size={19} strokeWidth={1.75} /></button>
      <button class="rib" title={t.db.newDb} onclick={() => (dbNew = true)}><Database size={19} strokeWidth={1.75} /></button>
      <button class="rib" title={t.daily.button} onclick={dailyNote}><CalendarDays size={19} strokeWidth={1.75} /></button>
    {/if}
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
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="files" ondragover={overTop} ondrop={dropTop}>
          <Tree nodes={tree} {current} {folder} open={openMap} {renaming} onRename={rename}
            creating={creating ? { folder, kind: creating, placeholder: creating === "folder" ? t.side.folderPh : t.side.notePh } : null} onCreate={finishCreate} onMove={notesVault ? moveTo : undefined} onPick={(p) => { screen = "write"; pick(p); }} onFolder={(p) => (folder = p)} onMenu={openMenu} />
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
      <span class="crumbs muted">{current.split("/").slice(0, -1).join(" / ")}{current.includes("/") ? " / " : ""}</span><strong class="title">{fileTitle(current)}</strong>
      <span class="status muted">{status === "saved" ? t.saved : status === "saving" ? t.saving : t.unsaved}</span>
    {:else}
      <strong class="title">{site.title}</strong>
    {/if}
    <span class="spacer"></span>
    {#if !notesVault}<button class="btn ghost" onclick={togglePreview}>{previewOn ? "✓ " : ""}{t.preview}</button>{/if}
    <button class="btn ghost" title={cloudTitle} onclick={() => go("publish", "sync")}>
      {#if cloud?.state === "syncing"}<span class="spin"><RefreshCw size={14} /></span>{:else if cloud?.state === "error"}<CloudAlert size={14} class="bad" />{:else if cloud}<CloudCheck size={14} />{/if}{t.sync}
    </button>
    {#if !notesVault}<button class="btn" onclick={() => go("publish", "publish")}>{t.publish}</button>{/if}
  </header>

  <section class="main" class:scroll={screen !== "write"} class:full={fullOn}>
    {#if error}<p class="error">{t.error}: {error}</p>{/if}
    {#if outside}<div class="outside"><span>{t.term.outside}</span><button class="btn ghost" onclick={reloadOutside}>{t.term.reload}</button><button class="btn ghost" onclick={() => (outside = false)}>{t.term.keep}</button></div>{/if}
    {#if screen === "write"}
      {#if current}
        {#key current + ":" + reloadN}
          {#if board === "canvas"}
            <CanvasView path={current} full={fullOn} onFull={() => (full = !full)} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onStatus={(st) => (status = st)} onError={(e) => (error = e)} onOpenNote={openNote} onOutside={() => reloadN++} />
          {:else if board === "drawing"}
            <DrawingView path={current} full={fullOn} onFull={() => (full = !full)} onStatus={(st) => (status = st)} onError={(e) => (error = e)} onOutside={() => reloadN++} />
          {:else if board === "database"}
            <DatabaseView path={current} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onOpenNote={openNote} onStatus={(st) => (status = st)} onError={(e) => (error = e)} onOutside={() => reloadN++} />
          {:else}
          <div class="note">
            {#if !notesVault}<Props props={noteProps} {kind} note={current} isIndex={current.split("/").pop() === "_index.md"} onEdit={editProp} onSettings={(tab) => { prefsTab = tab; prefsOpen = true; }} />{/if}
            <div class="body"><Editor {text} note={current} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onChange={changed} onSave={save} onError={(e) => (error = e)} onOpenNote={openNote} onNewBoard={notesVault ? (kind) => makeBoard(kind) : undefined} header={notesVault ? noteHead : undefined} {atEnd} onReady={() => (atEnd = false)} /></div>
            {#if notesVault}<Backlinks note={current} onOpenNote={openNote} />{/if}
          </div>
          {/if}
        {/key}
      {:else}
        <p class="hint muted">{t.pickNote}</p>
      {/if}
    {:else if screen === "graph"}
      <GraphView {current} onOpenNote={openNote} />
    {:else if settings}
      {#key screen}
        {#if screen === "look"}<LookPanel {settings} {reload} />
        {:else if screen === "sections"}<SectionsPanel {settings} {reload} onTree={(n) => (tree = n.children ?? [])} />
        {:else if screen === "comments"}<CommentsPanel {settings} {reload} />
        {:else if screen === "cookies"}<CookiesPanel {settings} {reload} onOpenNote={openNote} />
        {:else if screen === "password"}<PasswordPanel {settings} {reload} />
        {:else if screen === "publish"}<PublishPanel autostart={autostart} notes={notesVault} />
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
        <Preview src={previewSrc || previewURL} title={t.preview} onNavigate={navigated} />
      {:else}
        <p class="hint muted">{previewBusy ? t.previewStarting : t.previewOff}</p>
      {/if}
    </section>
  {/if}
  {#if prefsOpen}
    <SettingsModal start={prefsTab} siteLang={settings?.lang} onClose={() => { prefsOpen = false; prefsTab = "general"; }} onPublish={() => { prefsOpen = false; screen = "publish"; }} />
  {/if}
  {#if deletingSite}<DeleteSite dir={deletingSite} onClose={() => (deletingSite = "")} onDone={(dir) => { const name = others.find((o) => o.dir === dir)?.title ?? ""; deletingSite = ""; loadSites(); toast(t.toast.siteDeleted.replace("{name}", name)); }} />{/if}
  {#if cloudTip && !showTip}
    <!-- Once there is a site: keeping the notes in the cloud (optional). -->
    <aside class="cloud-tip" aria-label={t.hello.tipTitle}>
      <img src="/scenes/boas-vindas-encontrados.svg" alt="" />
      <div>
        <b>{t.hello.tipTitle}</b>
        <p>{t.hello.tipLead}</p>
        <div class="tip-actions start">
          <button class="btn" onclick={cloudYes}>{t.hello.tipYes}</button>
          <button class="btn ghost" onclick={() => setPref("cloudTip", true)}>{t.hello.tipNo}</button>
        </div>
      </div>
    </aside>
  {/if}
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
  {#if dbNew}<DatabaseNew onCreate={newDatabase} onClose={() => (dbNew = false)} />{/if}
  {#if historyFor}
    {#key historyFor}<HistoryModal path={historyFor} onClose={() => (historyFor = "")} />{/key}
  {/if}
  {#if menu}
    <FileMenu node={menu.node} x={menu.x} y={menu.y} {current} folders={folders(tree)} boards={notesVault} onRename={(n) => (renaming = n.path)} onClose={() => (menu = null)} onChanged={menuChanged} onNew={menuNew} onHistory={cloudHistory ? (p) => (historyFor = p) : undefined} />
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
  .ws { height: 100%; overflow: hidden; display: grid; grid-template-columns: 3rem var(--side) minmax(0, 1fr) var(--prev); grid-template-rows: auto minmax(0, 1fr) auto; }
  .ws.no-preview { grid-template-columns: 3rem var(--side) minmax(0, 1fr); }
  .ws.no-side { grid-template-columns: 3rem minmax(0, 1fr) var(--prev); }
  /* The edges to drag: a thin line that lights up under the mouse. */
  .tip-shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .45); }
  .tip { position: fixed; z-index: 41; inset: 0; margin: auto; height: fit-content; display: grid; justify-items: center; gap: .9rem; width: min(26rem, calc(100% - 2rem)); padding: 1.75rem 1.5rem 1.5rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); text-align: center; }
  .tip img { width: 4rem; height: 4rem; }
  .tip b { font-size: 1.1rem; }
  .tip p { margin-block: .4rem 1rem; font-size: .9rem; color: var(--muted); }
  .tip-actions { display: flex; flex-wrap: wrap; justify-content: center; gap: .5rem; }
  .tip-actions.start { justify-content: flex-start; }
  .cloud-tip { position: fixed; z-index: 30; inset-inline-end: 1.25rem; inset-block-end: 1.25rem; display: grid; grid-template-columns: 6rem 1fr; gap: 1rem; align-items: center; width: min(27rem, calc(100% - 2.5rem)); padding: 1rem 1.1rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 16px 40px rgb(0 0 0 / .2); }
  .cloud-tip img { width: 6rem; }
  .cloud-tip b { font-size: 1rem; }
  .cloud-tip p { margin-block: .25rem .75rem; font-size: .87rem; color: var(--muted); }
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
  /* The files panel is as tall as the window, never taller: its list
     scrolls by itself (and doesn't pass the scroll on). */
  .side { position: relative; grid-row: 1 / -1; grid-column: 2; display: flex; flex-direction: column; min-height: 0; max-height: 100vh; overflow: hidden; border-inline-end: 1px solid var(--line); background: var(--panel); }
  .side-tabs { display: flex; gap: .2rem; padding: .55rem .6rem; border-block-end: 1px solid var(--line); }
  .tab { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .tab:hover, .tab.on { background: var(--soft); color: var(--text); }
  .side-tools { display: flex; justify-content: center; gap: .3rem; padding: .45rem; }
  .side-tools button { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .side-tools button:hover { background: var(--soft); color: var(--text); }
  .files { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding: 0 .4rem .75rem; }
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
  .main.full { position: fixed; inset: 0; z-index: 35; }
  .preview { position: relative; grid-row: 2; overflow: hidden; border-inline-start: 1px solid var(--line); background: var(--bg); }
  .term { grid-row: 3; grid-column: 3 / -1; position: relative; min-height: 0; }
  .ws.no-side .term { grid-column: 2 / -1; }
  .term.min { display: none; }
  .grip { position: absolute; z-index: 3; inset-inline: 0; inset-block-start: -3px; height: 6px; cursor: row-resize; }
  .outside { position: absolute; z-index: 2; inset-inline: 1rem; inset-block-start: .75rem; display: flex; gap: .5rem; align-items: center; padding: .5rem .75rem; border-radius: .5rem; background: var(--soft); font-size: .85rem; }
  .outside span { flex: 1; }
  .note { height: 100%; display: flex; flex-direction: column; }
  .inline-title { width: 100%; margin-block-end: 1rem; padding: 0; border: 0; outline: none; background: transparent; color: var(--text); font: inherit; font-size: 2.1rem; font-weight: 800; line-height: 1.2; }
  .body { flex: 1; min-height: 0; }
  .hint { padding: 3rem 2rem; text-align: center; }
  .error { position: absolute; inset-inline: 1rem; inset-block-start: .75rem; z-index: 2; padding: .6rem .9rem; border-radius: .5rem; background: var(--soft); color: var(--red); font-size: .85rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
