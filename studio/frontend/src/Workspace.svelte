<script lang="ts">
  import { onDestroy, tick, untrack } from "svelte";
  import { CalendarEvents, HyprlandSetup, OpenURL, Sites, RecentSites, SearchNotes, CreateNote, CreateFolder, CreateCanvas, CreateDrawing, CreateDatabase, SaveDatabaseSchema, DailyNote, RenameItem, MoveItem, Open, Tree as LoadTree, ReadNote, SaveNote, StartPreview, StopPreview, PreviewPage, NoteAt, Settings as LoadSettings, CloudHistory, BlupAccount, SetSyncRemote, SyncRemote, SyncStatus } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import type { Settings, SearchHit } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Tree from "./Tree.svelte";
  import Editor from "./Editor.svelte";
  import Database from "@lucide/svelte/icons/database";
  import NotebookPen from "@lucide/svelte/icons/notebook-pen";
  import CalendarDays from "@lucide/svelte/icons/calendar-days";
  import type { DBSchema } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Json, JsonProps, Template } from "./db";
  import { boardKind, fileTitle, type NewKind } from "./boards";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import Props from "./Props.svelte";
  import NoteProps from "./NoteProps.svelte";
  import Backlinks from "./Backlinks.svelte";
  import Waypoints from "@lucide/svelte/icons/waypoints";
  import Preview from "./Preview.svelte";
  import FileMenu from "./FileMenu.svelte";
  import ContextMenu, { type MenuEntry } from "./ContextMenu.svelte";
  import { DOCS } from "./docs";
  import Gear from "@lucide/svelte/icons/settings";
  import CircleHelp from "@lucide/svelte/icons/circle-question-mark";
  import { look, setPref } from "./theme.svelte";
  import { ACTIONS, comboOf, keysOf, type Action } from "./keys";
  import { toast, toastError } from "./toast.svelte";
  import { job, LANG_NAMES } from "./translatejob.svelte";
  import { syncJob, syncStepText, startSync, onSyncDone } from "./syncjob.svelte";
  import { publishJob, publishStep } from "./publishjob.svelte";
  import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
  import CircleCheck from "@lucide/svelte/icons/circle-check";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import PanelLeft from "@lucide/svelte/icons/panel-left";
  import Files from "@lucide/svelte/icons/files";
  import Search from "@lucide/svelte/icons/search";
  import SquarePen from "@lucide/svelte/icons/square-pen";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import ChevronsUpDown from "@lucide/svelte/icons/chevrons-up-down";
  import ChevronsDownUp from "@lucide/svelte/icons/chevrons-down-up";
  import { Events } from "@wailsio/runtime";
  import type { BlupAccount as BlupAccountInfo } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import StorageBanner from "./StorageBanner.svelte";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Check from "@lucide/svelte/icons/check";
  import Cloud from "@lucide/svelte/icons/cloud";
  import Monitor from "@lucide/svelte/icons/monitor";
  import Paperclip from "@lucide/svelte/icons/paperclip";
  import Users from "@lucide/svelte/icons/users";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import House from "@lucide/svelte/icons/house";
  import { blupOf, homeView } from "./blups.svelte";
  import PenLine from "@lucide/svelte/icons/pen-line";
  import Palette from "@lucide/svelte/icons/palette";
  import Languages from "@lucide/svelte/icons/languages";
  import LayoutGrid from "@lucide/svelte/icons/layout-grid";
  import MessageCircle from "@lucide/svelte/icons/message-circle";
  import MessageSquare from "@lucide/svelte/icons/message-square";
  import Podcast from "@lucide/svelte/icons/podcast";
  import Cookie from "@lucide/svelte/icons/cookie";
  import Upload from "@lucide/svelte/icons/upload";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import { t, shown } from "./i18n";
  import { byDay, iso, parse, type Event as CalEvent } from "./calendar";
  import { lazy } from "./lazy";
  // The views and panels opened now and then load when first opened: a
  // lighter start, on any computer.
  const CanvasView = lazy(() => import("./CanvasView.svelte"));
  const DrawingView = lazy(() => import("./DrawingView.svelte"));
  const DatabaseView = lazy(() => import("./DatabaseView.svelte"));
  const DatabaseNew = lazy(() => import("./DatabaseNew.svelte"));
  const GraphView = lazy(() => import("./GraphView.svelte"));
  const CalendarView = lazy(() => import("./CalendarView.svelte"));
  const TerminalPanel = lazy(() => import("./TerminalPanel.svelte"));
  const HistoryModal = lazy(() => import("./HistoryModal.svelte"));
  const LookPanel = lazy(() => import("./LookPanel.svelte"));
  const LangsPanel = lazy(() => import("./LangsPanel.svelte"));
  const SectionsPanel = lazy(() => import("./SectionsPanel.svelte"));
  const CommentsPanel = lazy(() => import("./CommentsPanel.svelte"));
  const FeedbackPanel = lazy(() => import("./FeedbackPanel.svelte"));
  const PodcastPanel = lazy(() => import("./PodcastPanel.svelte"));
  const CookiesPanel = lazy(() => import("./CookiesPanel.svelte"));
  const PublishPanel = lazy(() => import("./PublishPanel.svelte"));
  const TrashPanel = lazy(() => import("./TrashPanel.svelte"));
  const AuthorsPanel = lazy(() => import("./AuthorsPanel.svelte"));
  const AttachmentsPanel = lazy(() => import("./AttachmentsPanel.svelte"));
  const SettingsModal = lazy(() => import("./SettingsModal.svelte"));

  let { site, onBack, onSwitch }: { site: SiteInfo; onBack: () => void; onSwitch: (s: SiteInfo) => void } = $props();

  // The files panel: tabs (files, search), tools (new
  // note, new folder, expand, collapse), the tree and the site switcher.
  let sideOpen = $state(true);
  let sideTab = $state<"files" | "search">("files");
  let openMap = $state<Record<string, boolean>>({});
  let folder = $state(""); // where new notes go: the folder clicked last
  let creating = $state<"note" | "folder" | null>(null);
  let query = $state("");
  let switching = $state(false);
  let others = $state<SiteInfo[]>([]);
  // The switcher: sites and vaults apart, the ones opened lately first,
  // five of each; "See all" opens the home screen with that list whole.
  const SWITCH_MAX = 5;
  const otherSites = $derived(others.filter((o) => o.kind !== "notes"));
  const otherVaults = $derived(others.filter((o) => o.kind === "notes"));

  const flat = (nodes: Node[]): Node[] => nodes.flatMap((n) => (n.dir ? flat(n.children ?? []) : [n]));
  const findDir = (nodes: Node[], path: string): Node | undefined => {
    for (const n of nodes) if (n.dir) { if (n.path === path) return n; const f = findDir(n.children ?? [], path); if (f) return f; }
  };
  const folders = (nodes: Node[]): string[] => nodes.flatMap((n) => (n.dir ? [n.path, ...folders(n.children ?? [])] : []));
  // The search: names and texts of the notes, ignoring case and accents
  // (in Go), asked a moment after the typing stops.
  let found = $state<SearchHit[]>([]);
  let searched = $state("");
  $effect(() => {
    const q = query.trim();
    if (q.length < 2) { found = []; searched = q; return; }
    const id = setTimeout(() => SearchNotes(q).then((h) => { if (query.trim() === q) { found = h ?? []; searched = q; } }).catch(() => {}), 180);
    return () => clearTimeout(id);
  });
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
  let dbFolder = ""; // where it goes ("" the databases' folder)
  async function newDatabase(title: string, tpl: Template) {
    dbNew = false;
    try {
      const rel = await CreateDatabase(dbFolder || t.db.folder, title, tpl.schema);
      dbFolder = "";
      tree = (await LoadTree()).children ?? [];
      reveal(rel);
      screen = "write";
      await pick(rel);
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }
  // A right-click on the empty part of the files: what can be made at the
  // top (a site's notes only take notes and folders).
  let sideMenu = $state<{ x: number; y: number } | null>(null);
  function emptyMenu(e: MouseEvent) {
    if (e.defaultPrevented || (e.target as HTMLElement).closest("input")) return;
    e.preventDefault();
    sideMenu = { x: e.clientX, y: e.clientY };
  }
  const topNew = (kind: "note" | "folder") => { folder = ""; startCreate(kind); };
  const sideItems = (): MenuEntry[] => [
    { label: t.side.newNote, Icon: SquarePen, run: () => topNew("note") },
    { label: t.side.newFolder, Icon: FolderPlus, run: () => topNew("folder") },
    ...(notesVault ? [
      "sep" as const,
      { label: t.boards.newCanvas, Icon: LayoutDashboard, run: () => quickBoard("canvas") },
      { label: t.db.newDb, Icon: Database, run: () => { dbFolder = ""; dbNew = true; } },
      { label: t.boards.newDrawing, Icon: PenTool, run: () => quickBoard("drawing") },
    ] : []),
  ];

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
      // A database's item renames its .table (the folder follows).
      const path = n.table || n.path;
      if (path === current) await save(); // what was typed goes with it
      const to = await RenameItem(path, name);
      await menuChanged({ from: path, to });
      toast(t.toast.noteRenamed.replace("{name}", fileTitle(to)));
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); }
  }

  // Dragging in the tree: a note or a folder into another folder (or to
  // the top, the room under the tree); links follow, like Move to. Only in
  // a vault of notes for now (a site's folders are its sections).
  async function moveTo(path: string, f: string) {
    const parent = path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
    if (!path || parent === f || f === path || f.startsWith(path + "/")) return;
    // A database's item moves its .table (the folder follows).
    const what = folders(tree).includes(path) ? findDir(tree, path)?.table || path : path;
    try {
      await save();
      const to = await MoveItem(what, f);
      await menuChanged({ from: what, to });
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
    else if (inside && to) { text = pending || text; current = to + current.slice(from.length); reveal(current); follow(current); }
    if (to && !open) reveal(to);
    if (open) { reveal(open); screen = "write"; await pick(open); }
  }
  async function menuNew(kind: NewKind, f: string) {
    openMap[f] = true;
    if (kind === "canvas" || kind === "drawing") { quickBoard(kind, f); return; }
    if (kind === "database") { dbFolder = f; dbNew = true; return; }
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
  async function loadSites() {
    const [all, recent] = await Promise.all([Sites().catch(() => []), RecentSites().catch(() => [])]);
    const rank = (dir: string) => { const i = (recent ?? []).findIndex((r) => r.dir === dir); return i < 0 ? Infinity : i; };
    others = (all ?? []).filter((o) => o.dir !== site.dir)
      .sort((a, b) => rank(a.dir) - rank(b.dir) || a.title.localeCompare(b.title));
  }
  async function seeAll(which: "sites" | "vaults") {
    homeView.expand = which;
    switching = false;
    await back();
  }

  let tree = $state<Node[]>([]);
  let current = $state(""); // path of the open note (or canvas, or drawing)
  const board = $derived(boardKind(current));
  // The path at the top: the folders, as text; a database (the folder of
  // the open record) shows marked and opens its table.
  const crumbs = $derived(current.split("/").slice(0, -1).map((name, i, parts) => {
    const path = parts.slice(0, i + 1).join("/");
    return { name, path, table: findDir(tree, path)?.table ?? "" };
  }));
  async function goCrumb(c: { table: string }) {
    screen = "write";
    await pick(c.table);
  }
  // Full screen: a canvas or a drawing alone in the window (F, or Esc to
  // leave); only while one is open.
  let full = $state(false);
  let text = $state("");
  let noteProps = $state<JsonProps>({});
  let kind = $state("");
  let edits: JsonProps = {}; // properties changed since the last save
  let status = $state<"saved" | "saving" | "unsaved">("saved");
  let error = $state("");
  let previewURL = $state("");
  let previewSrc = $state(""); // the page shown: the open note's, when known
  // Workspace is made again for each site (keyed by its folder).
  let previewOn = $state(!look.previewOff && untrack(() => site.kind) !== "notes");

  // The files panel and the preview are as wide as the user drags them
  // (their edge), within limits that keep the editor usable;
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
  // The same edges by keyboard (they are buttons): the arrows move them,
  // Home goes back to the default.
  function nudge(e: KeyboardEvent, which: "side" | "preview") {
    const step = e.key === "ArrowRight" ? 16 : e.key === "ArrowLeft" ? -16 : 0;
    if (e.key === "Home") { e.preventDefault(); resetWidth(which); return; }
    if (!step) return;
    e.preventDefault();
    if (which === "side") { sideW = clampSide(sideW + step); setPref("sideWidth", sideW); }
    else { prevW = clampPrev((prevW || previewEl?.getBoundingClientRect().width || PREV_MIN) - step); setPref("previewWidth", prevW); }
  }
  function resetWidth(which: "side" | "preview") {
    if (which === "side") { sideW = SIDE; setPref("sideWidth", 0); }
    else { prevW = 0; setPref("previewWidth", 0); }
  }
  // The Studio's shortcuts (Settings > Shortcuts); some are for vaults.
  const VAULT_ONLY: Action[] = ["dailyNote", "graph", "newCanvas", "newDrawing", "newDatabase"];
  function shortcut(e: KeyboardEvent) {
    if (e.defaultPrevented) return; // the editor (or a board) used them
    const combo = comboOf(e);
    if (!combo) return;
    const a = ACTIONS.find((x) => keysOf(look.keys, x.key) === combo)?.key as Action | undefined;
    if (!a || (!notesVault && VAULT_ONLY.includes(a)) || (notesVault && a === "preview")) return; // each kind's own keys
    e.preventDefault();
    if (a === "terminal") toggleTerm();
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
    else if (a === "newDatabase") { dbFolder = ""; dbNew = true; }
  }

  // A smaller window keeps the editor's room.
  function fit() {
    sideW = clampSide(sideW);
    if (prevW) prevW = clampPrev(prevW);
  }
  let prefsOpen = $state(false);
  let prefsTab = $state("general");

  // Once: the tip about the shortcut (Alt+Space by default) that shows and
  // hides the Studio from anywhere. On a Hyprland that isn't Omarchy, the
  // user pastes a line first: the tip leads there.
  const showTip = $derived(!!look.toggleKey && !look.shortcutTip);
  let manual = $state(false);
  HyprlandSetup(look.toggleKey ?? "").then((h) => (manual = h.on && !h.auto)).catch(() => {});
  function okTip() { setPref("shortcutTip", true); }
  // The invitation to Blup Cloud: once, to whoever has a site and no account.
  let signedIn = $state(true);
  // The account, also for the banner when the storage is above the plan's.
  let account = $state<BlupAccountInfo | null>(null);
  const loadAccount = () => BlupAccount().then((a) => { account = a; signedIn = a.signedIn; }).catch(() => {});
  loadAccount();

  // The check by the note's title: saved here, sent to the cloud, or not
  // yet (plans without live sync, GitHub), or an error. It is also the
  // sync button: a click sends now, or opens where to sync (none chosen
  // yet, or an error to read).
  let syncRemote = $state("");
  // Saved here but not sent yet (plans without live sync): a click sends.
  let unsent = $state(false);
  const liveSync = $derived(syncRemote === "blup" && !!account?.live);
  function loadRemote() {
    SyncRemote().then((r) => {
      const first = r !== syncRemote;
      syncRemote = r;
      // Opening: whether something waits to go (only where a click sends).
      if (first && r && !(r === "blup" && account?.live)) SyncStatus().then((i) => (unsent = !!i.send)).catch(() => {});
    }).catch(() => {});
  }
  loadRemote();
  $effect(() => onSyncDone((end) => { if (!end.error) unsent = false; }));
  const cloudTip = $derived(!look.cloudTip && !signedIn);
  async function cloudYes() {
    setPref("cloudTip", true);
    await SetSyncRemote("blup").catch(() => {});
    if (notesVault) go("publish", "sync");
    else go("sync", "");
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
  // A note of the day (Diário/2026-11-01.md): what the calendar has on
  // its day, shown at its top.
  const dailyDay = $derived(notesVault && current.startsWith(t.daily.folder + "/") ? (current.slice(t.daily.folder.length + 1).match(/^(\d{4}-\d{2}-\d{2})\.md$/)?.[1] ?? "") : "");
  let dayEvents = $state<CalEvent[]>([]);
  $effect(() => {
    const d = dailyDay;
    dayEvents = [];
    if (d) CalendarEvents().then((list) => { if (d === dailyDay) dayEvents = byDay(list ?? [], parse(d), parse(d)).get(d) ?? []; }).catch(() => {});
  });
  // The terminal button's hint, with its shortcut when there is one.
  const termTip = $derived(t.term.button + (keysOf(look.keys, "terminal") ? ` (${keysOf(look.keys, "terminal")})` : ""));
  // The screens, in groups split by a line: what one writes, the site,
  // what the public sees and answers, privacy; publishing and the trash
  // sit at the bottom ("end").
  const screens = $derived(notesVault ? [
    { key: "write", label: t.nav.write, Icon: PenLine, group: "content" },
    { key: "graph", label: t.graph.title, Icon: Waypoints, group: "content" },
    { key: "calendar", label: t.cal.title, Icon: CalendarDays, group: "content" },
    { key: "attach", label: t.nav.attach, Icon: Paperclip, group: "content" },
    { key: "publish", label: t.sync, Icon: RefreshCw, group: "end" },
    { key: "trash", label: t.nav.trash, Icon: Trash2, group: "end" },
  ] : [
    { key: "write", label: t.nav.write, Icon: PenLine, group: "content" },
    { key: "attach", label: t.nav.attach, Icon: Paperclip, group: "content" },
    { key: "authors", label: t.nav.authors, Icon: Users, group: "content" },
    { key: "look", label: t.nav.look, Icon: Palette, group: "site" },
    // Bento's parts; a blog's podcast.
    { key: "sections", label: t.nav.sections, Icon: LayoutGrid, group: "site" },
    { key: "langs", label: t.nav.langs, Icon: Languages, group: "site" },
    { key: "comments", label: t.nav.comments, Icon: MessageCircle, group: "public" },
    // The feedback board's requests, when the Bento site has the section.
    ...(site.model === "bento" && settings?.sections?.some((x) => x.kind === "feedback" && x.present) ? [{ key: "feedback", label: t.nav.feedback, Icon: MessageSquare, group: "public" }] : []),
    // A blog's podcast, when it has the section.
    ...(site.model !== "bento" && settings?.sections?.some((x) => x.kind === "podcast" && x.present) ? [{ key: "podcast", label: t.nav.podcast, Icon: Podcast, group: "public" }] : []),
    { key: "cookies", label: t.nav.cookies, Icon: Cookie, group: "privacy" },
    // Where the notes sync (Blup Cloud or GitHub): the check by the note's
    // title shows and runs the sync itself.
    { key: "sync", label: t.sync, Icon: RefreshCw, group: "end" },
    { key: "publish", label: t.nav.publish, Icon: Upload, group: "end" },
    { key: "trash", label: t.nav.trash, Icon: Trash2, group: "end" },
  ]);
  // Each screen opens from its top: the scroll of a long one (Site,
  // Publish) left a short one (Sections) looking empty.
  let mainEl = $state<HTMLElement>();
  $effect(() => {
    void screen;
    if (mainEl) mainEl.scrollTop = 0;
  });
  // The site's translation in the background: a ring that fills with the
  // percentage, then a check for a moment when it is done.
  const jobPct = $derived(job.total ? Math.floor((job.done / job.total) * 100) : 0);
  const jobTip = $derived(job.running
    ? t.sitelangs.ringWorking.replace("{lang}", LANG_NAMES[job.lang] ?? job.lang).replace("{done}", String(job.done)).replace("{total}", String(job.total))
    : t.sitelangs.ringDone.replace("{lang}", LANG_NAMES[job.finished] ?? job.finished));
  // The sync of the notes in the background, the same way: the ring with
  // the percentage while sending or receiving, turning while it reads,
  // compares or saves, then the check.
  const syncPct = $derived(syncJob.total ? Math.floor((syncJob.done / syncJob.total) * 100) : -1);
  const syncTip = $derived(syncJob.running ? syncStepText(syncJob.phase, syncJob.done, syncJob.total) : t.toast.synced);
  const top = $derived(screens.filter((x) => x.group !== "end"));
  const bottom = $derived(screens.filter((x) => x.group === "end"));
  const openScreen = async (key: string) => { await save(); autostart = ""; screen = key; error = ""; loadRemote(); };

  // The terminal at the bottom, and its height (dragging its top edge).
  let termOpen = $state(false);
  // Once opened, the terminals stay alive: closing the panel only hides it.
  let termStarted = $state(false);
  function toggleTerm() {
    termOpen = !termOpen;
    if (termOpen) termStarted = true;
  }
  let termHeight = $state(300);
  const clampTerm = (h: number) => Math.min(Math.max(140, h), window.innerHeight - 200);
  function nudgeTerm(e: KeyboardEvent) {
    const step = e.key === "ArrowUp" ? 16 : e.key === "ArrowDown" ? -16 : 0;
    if (!step) return;
    e.preventDefault();
    termHeight = clampTerm(termHeight + step);
  }
  function resizeTerm(e: MouseEvent) {
    const start = e.clientY, h = termHeight;
    const move = (m: MouseEvent) => (termHeight = clampTerm(h + start - m.clientY));
    const up = () => { removeEventListener("mousemove", move); removeEventListener("mouseup", up); };
    addEventListener("mousemove", move);
    addEventListener("mouseup", up);
  }

  // Notes changed on disk (an agent in the terminal, another editor): the
  // tree follows; the open note too, unless it has changes of its own.
  let reloadN = $state(0);
  let outside = $state(false);
  const offNotes = Events.On("notesChanged", async (ev) => {
    const list = String(ev.data ?? "").split("\n");
    const isOpen = (p: string) => !!current && p.toLowerCase() === current.toLowerCase();
    // A canvas or a drawing looks after itself.
    const mine = !!current && !board && list.some(isOpen);
    const note = mine ? await ReadNote(current).catch(() => null) : null;
    // Our own saves come back here too: only a text that is neither what
    // we wrote last nor what is being typed came from outside. A save of
    // the open note alone changes no name: the tree isn't read again
    // (a walk of the whole vault on every save).
    const ours = !!note && (same(note.body, pending) || same(note.body, written));
    if (!(ours && list.every(isOpen))) {
      try { tree = (await LoadTree()).children ?? []; } catch {}
    }
    if (!note || ours) return;
    if (status === "saved") applyOutside(note);
    else outside = true;
  });
  function applyOutside(note: { body: string; props: JsonProps | null; kind: string }) {
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
  let cloud = $state<{ state: string; error?: string; stuck?: boolean } | null>(null);
  // After each sync the use may have changed (a refusal for the storage,
  // or space freed): the banner follows.
  const offCloud = Events.On("cloudSync", (ev) => {
    // Deleted from Blup Cloud on another device: said once, and the notes
    // stop syncing until the person clicks Sync.
    if (ev.data?.state === "gone") {
      cloud = null;
      toast(ev.data.error ?? "", { ms: 15000 });
      return;
    }
    cloud = ev.data;
    if (ev.data?.state !== "syncing" && signedIn) loadAccount();
  });
  // The preview started again (a translation brought a new language): its
  // new address, on the open note's page.
  const offPreview = Events.On("previewRestarted", (ev) => {
    if (!previewURL || typeof ev.data !== "string") return;
    previewURL = previewSrc = ev.data;
    follow(current);
  });
  // A rebuild may bring the open note's page (a translation just made):
  // the preview goes there.
  const offRebuilt = Events.On("previewChanged", () => { if (current) follow(current); });
  // A reminder was clicked: its day on the calendar, when its vault is the
  // open one.
  let calStart = $state("");
  const offCal = Events.On("calendarOpen", async (ev) => {
    const d = ev.data;
    if (!notesVault || !d || d.dir !== site.dir) return;
    await save();
    calStart = d.date ?? "";
    screen = "calendar";
  });
  // The Studio published the site again alone (its own domain on Blup
  // Cloud turned active or stopped being): a quiet notice.
  const offRepublish = Events.On("republish", (ev) => {
    const r = ev.data;
    if (!r || r.dir !== site.dir) return;
    if (r.state === "start") toast(t.bdom.republishing, { kind: "info" });
    else if (r.state === "done") toast(t.bdom.republished.replace("{url}", r.url), { kind: "info", ms: 6000 });
    else if (r.state === "error") toast(t.bdom.republishFailed.replace("{error}", r.error), { kind: "error" });
  });
  onDestroy(() => { offCloud(); offPreview(); offRebuilt(); offCal(); offRepublish(); });
  // Earlier versions on Blup Cloud: checked when the site opens and after
  // each live sync (signing in or changing plan turns it on).
  let cloudHistory = $state(false);
  let historyFor = $state("");
  const checkHistory = () => CloudHistory().then((on) => (cloudHistory = on)).catch(() => {});
  checkHistory();
  $effect(() => { if (cloud?.state === "synced" && !cloudHistory) checkHistory(); });
  const cloudTitle = $derived(!cloud ? undefined : cloud.state === "error" && cloud.stuck ? t.pubs.stuck : cloud.state === "error" ? t.cloudError.replace("{error}", cloud.error ?? "") : cloud.state === "syncing" ? t.cloudSyncing : t.cloudSynced);
  const syncScreen = $derived(notesVault ? "publish" : "sync");
  const syncErr = $derived(cloud?.state === "error" ? (cloud.error ?? "") : (syncJob.last?.error ?? ""));
  const mark = $derived(status !== "saved" || syncJob.running || cloud?.state === "syncing" ? "saving" : syncErr ? "error" : syncRemote && !liveSync && unsent ? "pending" : "saved");
  const markCloud = $derived(
    !syncRemote ? t.mark.choose
    : syncJob.running ? syncStepText(syncJob.phase, syncJob.done, syncJob.total)
    : syncErr ? t.cloudError.replace("{error}", syncErr) + " " + t.mark.see
    : !liveSync ? (unsent ? t.mark.pending : t.cloudSynced)
    : cloudTitle ?? t.cloudSynced);
  function markClick() {
    if (!syncRemote || mark === "error") { openScreen(syncScreen); return; }
    if (mark === "saving") return;
    startSync().catch(toastError);
  }

  let timer: ReturnType<typeof setTimeout> | undefined;
  let pending = ""; // text not saved yet
  let written = ""; // the text the editor wrote to disk last
  const same = (a: string, b: string) => a.trimEnd() === b.trimEnd();

  // The site is the one this Workspace opened (a new site is a new one).
  Open(untrack(() => site.dir))
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
  // A canvas, drawing or database gives its own save (boardSave), so a
  // rename, move or delete waits for it like for a note's.
  let boardSave: (() => Promise<void>) | null = null;
  async function save() {
    if (board) { await boardSave?.(); return; }
    if (status === "saved" || !current) return;
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
      // Typed (or changed) while it was saving: that goes in the next save.
      if (pending !== written || Object.keys(edits).length) later();
      else {
        status = "saved";
        if (syncRemote && !liveSync) unsent = true;
      }
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

  function editProp(key: string, value: Json) {
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
  {#if dayEvents.length}
    <button type="button" class="day-events" onclick={() => { calStart = dailyDay; screen = "calendar"; }}>
      <CalendarDays size={15} /><b>{dailyDay === iso(new Date()) ? t.cal.todayHere : t.cal.title}:</b>
      <span>{dayEvents.map((e) => (e.time ? e.time + " " : "") + e.title).join(" · ")}</span>
    </button>
  {/if}
  <input class="inline-title" value={fileTitle(current)} aria-label={t.menu.rename} spellcheck="false"
    onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); } else if (e.key === "Escape") { e.currentTarget.value = fileTitle(current); e.currentTarget.blur(); } }}
    onblur={(e) => rename({ path: current, name: fileTitle(current), dir: false } as Node, e.currentTarget.value)} />
  <NoteProps props={noteProps} database={noteDb?.schema} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onOpenNote={openNote} onNewOption={dbOption} onEdit={(k, v) => { noteProps[k] = v; editProp(k, v); }} />
{/snippet}

<svelte:window onresize={fit} onkeydown={shortcut} onclick={() => (switching = false)} />

<div class="ws" class:names={look.railNames} class:no-preview={!previewOn} class:no-side={!sideOpen} class:resizing style:--side="{sideW}px" style:--prev={prevW ? `${prevW}px` : "minmax(0, 1fr)"}>
  <!-- The ribbon: icons only, the name on hover. -->
  <nav class="ribbon">
    <button class="rib" aria-label={t.side.toggle} data-tip={t.side.toggle} onclick={() => (sideOpen = !sideOpen)}><PanelLeft size={19} strokeWidth={1.75} /><span class="rib-name">{t.side.toggle}</span></button>
    <span class="sep"></span>
    {#each top as { key, label, Icon, group }, i (key)}
      {#if i > 0 && top[i - 1].group !== group}<span class="sep"></span>{/if}
      <button class="rib" class:active={screen === key} aria-label={label} data-tip={label} onclick={() => openScreen(key)}><Icon size={19} strokeWidth={1.75} /><span class="rib-name">{label}</span></button>
    {/each}
    {#if notesVault}
      <!-- Making things: a canvas, a drawing (opened right away). -->
      <span class="sep"></span>
      <button class="rib" aria-label={t.boards.newCanvas} data-tip={t.boards.newCanvas} onclick={() => quickBoard("canvas")}><LayoutDashboard size={19} strokeWidth={1.75} /><span class="rib-name">{t.boards.newCanvas}</span></button>
      <button class="rib" aria-label={t.boards.newDrawing} data-tip={t.boards.newDrawing} onclick={() => quickBoard("drawing")}><PenTool size={19} strokeWidth={1.75} /><span class="rib-name">{t.boards.newDrawing}</span></button>
      <button class="rib" aria-label={t.db.newDb} data-tip={t.db.newDb} onclick={() => { dbFolder = ""; dbNew = true; }}><Database size={19} strokeWidth={1.75} /><span class="rib-name">{t.db.newDb}</span></button>
      <button class="rib" aria-label={t.daily.button} data-tip={t.daily.button} onclick={dailyNote}><NotebookPen size={19} strokeWidth={1.75} /><span class="rib-name">{t.daily.button}</span></button>
    {/if}
    <span class="grow"></span>
    {#if !notesVault && (publishJob.running || publishJob.finished)}
      {@const pubTip = publishJob.running ? publishStep() : t.toast.published}
      <button class="rib job" aria-label={pubTip} data-tip={pubTip} onclick={() => openScreen("publish")}>
        {#if publishJob.running && publishJob.total}<span class="ring" style:--p={Math.floor((publishJob.done / publishJob.total) * 100)} role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.floor((publishJob.done / publishJob.total) * 100)}><span class="pct">{Math.floor((publishJob.done / publishJob.total) * 100)}%</span></span>
        {:else if publishJob.running}<span class="ring spin" style:--p={25} role="progressbar" aria-valuemin="0" aria-valuemax="100"></span>
        {:else}<span class="ok"><CircleCheck size={19} strokeWidth={1.75} /></span>{/if}
        <span class="rib-name">{pubTip}</span>
      </button>
    {/if}
    {#if syncJob.running || syncJob.finished}
      <button class="rib job" aria-label={syncTip} data-tip={syncTip} onclick={() => openScreen(syncScreen)}>
        {#if syncJob.running}<span class="ring" class:spin={syncPct < 0} style:--p={syncPct < 0 ? 25 : syncPct} role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={syncPct < 0 ? undefined : syncPct}>{#if syncPct >= 0}<span class="pct">{syncPct}%</span>{/if}</span>
        {:else}<span class="ok"><CircleCheck size={19} strokeWidth={1.75} /></span>{/if}
        <span class="rib-name">{syncTip}</span>
      </button>
    {/if}
    {#if !notesVault && (job.running || job.finished)}
      <button class="rib job" aria-label={jobTip} data-tip={jobTip} onclick={() => openScreen("langs")}>
        {#if job.running}<span class="ring" style:--p={jobPct} role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={jobPct}><span class="pct">{jobPct}%</span></span>
        {:else}<span class="ok"><CircleCheck size={19} strokeWidth={1.75} /></span>{/if}
        <span class="rib-name">{jobTip}</span>
      </button>
    {/if}
    {#each bottom as { key, label, Icon } (key)}
      <button class="rib" class:active={screen === key} aria-label={label} data-tip={label} onclick={() => openScreen(key)}><Icon size={19} strokeWidth={1.75} /><span class="rib-name">{label}</span></button>
    {/each}
    <span class="sep"></span>
    <button class="rib" class:active={termOpen} aria-label={termTip} data-tip={termTip} onclick={toggleTerm}><SquareTerminal size={19} strokeWidth={1.75} /><span class="rib-name">{t.term.button}</span></button>
  </nav>

  {#if sideOpen}
    <aside class="side">
      <button type="button" class="resizer right" aria-label={t.side.resize} title={t.side.resize} onmousedown={(e) => drag(e, "side")} onkeydown={(e) => nudge(e, "side")} ondblclick={() => resetWidth("side")}></button>
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
        <div class="files" ondragover={overTop} ondrop={dropTop} oncontextmenu={emptyMenu}>
          <Tree nodes={tree} {current} {folder} bind:open={openMap} {renaming} onRename={rename}
            creating={creating ? { folder, kind: creating, placeholder: creating === "folder" ? t.side.folderPh : t.side.notePh } : null} onCreate={finishCreate} onMove={notesVault ? moveTo : undefined} onPick={(p) => { screen = "write"; pick(p); }} onFolder={(p) => (folder = p)} onMenu={openMenu} />
        </div>
      {:else}
        <div class="side-search">
          <!-- svelte-ignore a11y_autofocus -->
          <input type="text" autofocus bind:value={query} placeholder={t.side.searchPh} />
          <ul>
            {#each found as n (n.path)}
              <li><button class="hit" class:active={n.path === current} onclick={() => { screen = "write"; pick(n.path); }}>
                <span>{n.name}</span>
                {#if n.folder}<small class="muted">{n.folder.split("/").join(" / ")}</small>{/if}
                {#each n.lines ?? [] as l, i (i)}<small class="hit-line">{l.before}<mark>{l.link}</mark>{l.after}</small>{/each}
              </button></li>
            {/each}
            {#if searched.length >= 2 && searched === query.trim() && !found.length}<li class="muted none">{t.side.nothing}</li>{/if}
          </ul>
        </div>
      {/if}
      <!-- The site switcher. -->
      <div class="switcher">
        <button class="vault" onclick={(e) => { e.stopPropagation(); switching = !switching; if (switching) loadSites(); }} aria-expanded={switching}>{#if switching}<ChevronsDownUp size={15} />{:else}<ChevronsUpDown size={15} />{/if}<span>{site.title}</span></button>
        <button class="foot" title={t.prefs.helpIcon} onclick={() => OpenURL(DOCS)}><CircleHelp size={17} strokeWidth={1.75} /></button>
        <button class="foot" title={t.prefs.gear} onclick={() => (prefsOpen = true)}><Gear size={17} strokeWidth={1.75} /></button>
        {#if switching}
          <div class="vault-menu" role="menu">
            {#each [{ label: t.side.sitesLabel, list: otherSites, key: "sites" as const }, { label: t.side.vaultsLabel, list: otherVaults, key: "vaults" as const }] as sec (sec.key)}
              {#if sec.list.length}
                <p class="vault-head">{sec.label}</p>
                {#each sec.list.slice(0, SWITCH_MAX) as o (o.dir)}
                  <button class="vault-item" onclick={async () => { await save(); StopPreview(); onSwitch(o); }}><img src="/blups/{blupOf(o)}.svg" alt="" /><span>{o.title}</span></button>
                {/each}
                {#if sec.list.length > SWITCH_MAX}<button class="vault-more" onclick={() => seeAll(sec.key)}>{t.side.seeAll.replace("{n}", String(sec.list.length))}</button>{/if}
              {/if}
            {/each}
            <button class="all" onclick={back}><House size={15} />{t.side.allSites}</button>
          </div>
        {/if}
      </div>
    </aside>
  {/if}

  <header class="bar">
    {#if screen === "write" && current}
      <nav class="crumbs" aria-label={t.side.files}>
        {#each crumbs as c (c.path)}
          {#if c.table}<button class="crumb db" title={t.db.openDb} onclick={() => goCrumb(c)}><Database size={13} />{c.name}</button>
          {:else}<span class="crumb">{c.name}</span>{/if}<span class="slash">/</span>
        {/each}
      </nav>
      <strong class="title">{fileTitle(current)}</strong>
      <button class="status" aria-label={`${status === "saved" ? t.savedHere : t.savingHere}. ${markCloud}`} onclick={markClick}>
        {#if mark === "saving"}<span class="state saving"><RefreshCw size={10} strokeWidth={3} /></span>
        {:else if mark === "error"}<span class="state error"><TriangleAlert size={10} strokeWidth={3} /></span>
        {:else}<span class={["state", mark]}><Check size={11} strokeWidth={3.5} /></span>{/if}
        <span class="status-info" role="tooltip">
          <span><Monitor size={15} />{status === "saved" ? t.savedHere : t.savingHere}</span>
          <span><Cloud size={15} />{markCloud}</span>
        </span>
      </button>
    {:else}
      <strong class="title">{site.title}</strong>
    {/if}
    <span class="spacer"></span>
    {#if !notesVault}<button class="btn ghost" onclick={togglePreview}>{#if previewOn}<Check size={14} aria-hidden="true" />{/if}{t.preview}</button>{/if}
    {#if !notesVault}<button class="btn" onclick={() => go("publish", "publish")}>{t.publish}</button>{/if}
  </header>

  <section class="main" bind:this={mainEl} class:scroll={screen !== "write"} class:full={fullOn}>
    {#if error}<p class="error">{t.error}: {error}</p>{/if}
    <StorageBanner bind:account />
    {#if outside}<div class="outside"><span>{t.term.outside}</span><button class="btn ghost" onclick={reloadOutside}>{t.term.reload}</button><button class="btn ghost" onclick={() => (outside = false)}>{t.term.keep}</button></div>{/if}
    {#if screen === "write"}
      {#if current}
        {#key current + ":" + reloadN}
          {#if board === "canvas"}
            {#await CanvasView() then { default: C }}<C path={current} bindSave={(fn) => (boardSave = fn)} full={fullOn} onFull={() => (full = !full)} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onStatus={(st) => (status = st)} onError={(e) => (error = e)} onOpenNote={openNote} onOutside={() => reloadN++} />{/await}
          {:else if board === "drawing"}
            {#await DrawingView() then { default: C }}<C path={current} bindSave={(fn) => (boardSave = fn)} full={fullOn} onFull={() => (full = !full)} onStatus={(st) => (status = st)} onError={(e) => (error = e)} onOutside={() => reloadN++} />{/await}
          {:else if board === "database"}
            {#await DatabaseView() then { default: C }}<C path={current} bindSave={(fn) => (boardSave = fn)} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onOpenNote={openNote} onStatus={(st) => (status = st)} onError={(e) => (error = e)} />{/await}
          {:else}
          <div class="note">
            {#if !notesVault}<Props props={noteProps} {kind} note={current} isIndex={current.split("/").pop() === "_index.md"} onEdit={editProp} onSettings={(tab) => { prefsTab = tab; prefsOpen = true; }} onOpenNote={openNote} />{/if}
            <div class="body">{#key shown.lang}<Editor {text} note={current} notes={flat(tree).map((n) => ({ name: n.name, path: n.path }))} onChange={changed} onSave={save} onError={(e) => (error = e)} onOpenNote={openNote} onNewBoard={notesVault ? (kind) => makeBoard(kind) : undefined} header={notesVault ? noteHead : undefined} {atEnd} onReady={() => (atEnd = false)} />{/key}</div>
            {#if notesVault}<Backlinks note={current} onOpenNote={openNote} />{/if}
          </div>
          {/if}
        {/key}
      {:else}
        <p class="hint muted">{t.pickNote}</p>
      {/if}
    {:else if screen === "graph"}
      {#await GraphView() then { default: C }}<C {current} onOpenNote={openNote} />{/await}
    {:else if screen === "calendar"}
      {#key calStart}
        {#await CalendarView() then { default: C }}<C start={calStart} hasNote={(rel) => flat(tree).some((n) => n.path === rel)} onOpenNote={(rel) => { screen = "write"; openNote(rel); }} onDailyNote={dailyNote} onChanged={async () => (tree = (await LoadTree()).children ?? [])} />{/await}
      {/key}
    {:else if settings}
      {#key screen}
        {#if screen === "look"}{#await LookPanel() then { default: C }}<C {settings} {reload} />{/await}
        {:else if screen === "langs"}{#await LangsPanel() then { default: C }}<C {settings} onTree={async () => (tree = (await LoadTree()).children ?? [])} />{/await}
        {:else if screen === "sections"}{#await SectionsPanel() then { default: C }}<C {settings} {reload} onTree={(n) => (tree = n.children ?? [])} />{/await}
        {:else if screen === "comments"}{#await CommentsPanel() then { default: C }}<C {settings} {reload} />{/await}
        {:else if screen === "feedback"}{#await FeedbackPanel() then { default: C }}<C />{/await}
        {:else if screen === "podcast"}{#await PodcastPanel() then { default: C }}<C />{/await}
        {:else if screen === "cookies"}{#await CookiesPanel() then { default: C }}<C {settings} {reload} onOpenNote={openNote} />{/await}
        {:else if screen === "publish"}{#await PublishPanel() then { default: C }}<C autostart={autostart} notes={notesVault} />{/await}
        {:else if screen === "sync"}{#await PublishPanel() then { default: C }}<C autostart="sync" />{/await}
        {:else if screen === "authors"}{#await AuthorsPanel() then { default: C }}<C onTree={async () => (tree = (await LoadTree()).children ?? [])} />{/await}
        {:else if screen === "attach"}{#await AttachmentsPanel() then { default: C }}<C onOpenNote={openNote} />{/await}
        {:else if screen === "trash"}{#await TrashPanel() then { default: C }}<C {reload} onTree={(n) => (tree = n.children ?? [])} />{/await}
        {/if}
      {/key}
    {/if}
  </section>

  {#if previewOn}
    <section class="preview" bind:this={previewEl}>
      <button type="button" class="resizer left" aria-label={t.side.resize} title={t.side.resize} onmousedown={(e) => drag(e, "preview")} onkeydown={(e) => nudge(e, "preview")} ondblclick={() => resetWidth("preview")}></button>
      {#if previewURL}
        <Preview src={previewSrc || previewURL} title={t.preview} onNavigate={navigated} />
      {:else}
        <p class="hint muted">{previewBusy ? t.previewStarting : t.previewOff}</p>
      {/if}
    </section>
  {/if}
  {#if prefsOpen}
    {#await SettingsModal() then { default: C }}<C start={prefsTab} siteLang={settings?.lang} onClose={() => { prefsOpen = false; prefsTab = "general"; }} onPublish={() => { prefsOpen = false; screen = "publish"; }} />{/await}
  {/if}
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
  {#if dbNew}{#await DatabaseNew() then { default: C }}<C onCreate={newDatabase} onClose={() => { dbNew = false; dbFolder = ""; }} />{/await}{/if}
  {#if historyFor}
    {#key historyFor}{#await HistoryModal() then { default: C }}<C path={historyFor} onClose={() => (historyFor = "")} />{/await}{/key}
  {/if}
  {#if sideMenu}<ContextMenu x={sideMenu.x} y={sideMenu.y} items={sideItems()} onClose={() => (sideMenu = null)} />{/if}
  {#if menu}
    <FileMenu node={menu.node} x={menu.x} y={menu.y} {current} folders={folders(tree)} boards={notesVault} onRename={(n) => (renaming = n.path)} onClose={() => (menu = null)} onChanged={menuChanged} onNew={menuNew} onHistory={cloudHistory ? (p) => (historyFor = p) : undefined} />
  {/if}
  {#if termStarted}
    <div class="term" class:min={!termOpen} style:height="{termHeight}px">
      <button type="button" class="grip" aria-label={t.side.resizeTerm} title={t.side.resizeTerm} onmousedown={resizeTerm} onkeydown={nudgeTerm}></button>
      {#await TerminalPanel() then { default: C }}<C visible={termOpen} onClose={() => (termOpen = false)} />{/await}
    </div>
  {/if}
</div>

<style>
  /* Columns: ribbon, files, editor, preview. The top bar starts after the
     files; the terminal sits under the editor and the preview. */
  .ws { --rail: 3rem; height: 100%; overflow: hidden; display: grid; grid-template-columns: var(--rail) var(--side) minmax(0, 1fr) var(--prev); grid-template-rows: auto minmax(0, 1fr) auto; }
  .ws.no-preview { grid-template-columns: var(--rail) var(--side) minmax(0, 1fr); }
  .ws.no-side { grid-template-columns: var(--rail) minmax(0, 1fr) var(--prev); }
  /* The edges to drag: a thin line that lights up under the mouse. */
  .tip-shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .45); }
  .tip { position: fixed; z-index: 41; top: 50%; left: 50%; transform: translate(-50%, -50%); display: grid; justify-items: center; gap: .9rem; width: min(26rem, calc(100% - 2rem)); padding: 1.75rem 1.5rem 1.5rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); text-align: center; }
  .tip img { width: 4rem; height: 4rem; }
  .tip b { font-size: 1.1rem; }
  .tip p { margin-block: .4rem 1rem; font-size: .9rem; color: var(--muted); }
  .tip-actions { display: flex; flex-wrap: wrap; justify-content: center; gap: .5rem; }
  .tip-actions.start { justify-content: flex-start; }
  .cloud-tip { position: fixed; z-index: 30; inset-inline-end: 1.25rem; inset-block-end: 1.25rem; display: grid; grid-template-columns: 6rem 1fr; gap: 1rem; align-items: center; width: min(27rem, calc(100% - 2.5rem)); padding: 1rem 1.1rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 16px 40px rgb(0 0 0 / .2); }
  .cloud-tip img { width: 6rem; }
  .cloud-tip b { font-size: 1rem; }
  .cloud-tip p { margin-block: .25rem .75rem; font-size: .87rem; color: var(--muted); }
  .resizer { position: absolute; z-index: 5; inset-block: 0; width: 7px; padding: 0; cursor: col-resize; }
  .resizer:focus-visible, .grip:focus-visible { outline: none; }
  /* On the panel's border, half outside it, above the files' scrollbar:
     hovering the border lights it blue, as in every view. */
  .resizer.right { inset-inline-end: -4px; }
  .resizer.left { inset-inline-start: 0; } /* the preview cuts what sticks out */
  .resizer.left::after { inset-inline-start: 0; }
  .resizer::after { content: ""; position: absolute; inset-block: 0; inset-inline-start: 3px; width: 2px; background: transparent; transition: background .1s; }
  .resizer:hover::after, .resizer:focus-visible::after, .ws.resizing .resizer:active::after { background: var(--blue); }
  .ws.resizing { cursor: col-resize; -webkit-user-select: none; user-select: none; }
  .ws.resizing :global(iframe) { pointer-events: none; }
  .ws.no-side.no-preview { grid-template-columns: var(--rail) minmax(0, 1fr); }
  .ribbon { position: relative; z-index: 20; grid-row: 1 / -1; grid-column: 1; display: flex; flex-direction: column; align-items: center; gap: .2rem; padding-block: .6rem; border-inline-end: 1px solid var(--line); background: var(--panel); }
  .rib { position: relative; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .45rem; color: var(--muted); }
  .rib-name { display: none; }
  .ring { position: relative; display: grid; place-items: center; flex: none; width: 1.65rem; height: 1.65rem; }
  .ring::before { content: ""; position: absolute; inset: 0; border-radius: 50%; background: conic-gradient(var(--blue) calc(var(--p) * 1%), var(--line) 0); mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 2.5px)); }
  .ring.spin::before { animation: ring-spin 1s linear infinite; }
  @keyframes ring-spin { to { transform: rotate(1turn); } }
  @media (prefers-reduced-motion: reduce) { .ring.spin::before { animation: none; } }
  .pct { font-size: .5rem; font-weight: 700; color: var(--text); font-variant-numeric: tabular-nums; letter-spacing: -.02em; }
  .ok { display: grid; color: light-dark(#16a34a, #4ade80); }
  /* The screen's name at once on hover (the system's tip waits a second). */
  .rib[data-tip]:hover::after, .rib[data-tip]:focus-visible::after { content: attr(data-tip); position: absolute; z-index: 50; inset-inline-start: calc(100% + .5rem); padding: .3rem .55rem; border-radius: .4rem; background: var(--text); color: var(--bg); font-size: .78rem; font-weight: 500; white-space: nowrap; pointer-events: none; }
  /* Names next to the icons (Settings > Look). */
  .ws.names { --rail: 11.5rem; }
  .ws.names .ribbon { align-items: stretch; padding-inline: .5rem; }
  .ws.names .rib { display: flex; justify-content: flex-start; gap: .65rem; width: 100%; padding-inline: .55rem; }
  .ws.names .rib-name { display: inline; overflow: hidden; font-size: .85rem; text-overflow: ellipsis; white-space: nowrap; }
  .ws.names .rib[data-tip]:hover::after { content: none; }
  .ws.names .sep { width: auto; margin-inline: .3rem; }
  .rib:hover { background: var(--soft); color: var(--text); }
  .rib.active { background: var(--soft); color: var(--text); }
  /* The translation's ring is a sign, not a screen: no highlight. */
  .rib.job, .rib.job:hover { background: none; }
  .sep { width: 1.4rem; height: 1px; margin-block: .35rem; background: var(--line); }
  .grow { flex: 1; }
  /* The files panel is as tall as the window, never taller: its list
     scrolls by itself (and doesn't pass the scroll on). */
  .side { position: relative; grid-row: 1 / -1; grid-column: 2; display: flex; flex-direction: column; min-height: 0; max-height: 100vh; border-inline-end: 1px solid var(--line); background: var(--panel); }
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
  .hit-line { display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; margin-block-start: .15rem; color: var(--muted); font-size: .75rem; line-height: 1.35; }
  .hit-line mark { background: none; color: var(--blue); font-weight: 600; }
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
  .vault-head { padding: .45rem .6rem .2rem; font-size: .7rem; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; color: var(--muted); }
  .vault-head:not(:first-child) { margin-block-start: .2rem; border-block-start: 1px solid var(--line); padding-block-start: .6rem; }
  .vault-menu .vault-item { display: flex; align-items: center; gap: .55rem; min-width: 0; }
  .vault-item img { width: 1.15rem; flex: none; }
  .vault-item span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vault-menu .vault-more { color: var(--blue); font-size: .8rem; font-weight: 600; }
  .vault-menu .all { display: flex; align-items: center; gap: .55rem; border-block-start: 1px solid var(--line); margin-block-start: .2rem; padding-block-start: .55rem; color: var(--muted); }
  .bar { grid-row: 1; grid-column: 3 / -1; display: flex; align-items: center; gap: .6rem; padding: .55rem 1rem; border-block-end: 1px solid var(--line); background: var(--panel); min-width: 0; }
  .ws.no-side .bar { grid-column: 2 / -1; }
  .crumbs { display: flex; align-items: center; min-width: 0; max-width: 45%; overflow: hidden; font-size: .85rem; white-space: nowrap; }
  .crumb { flex: none; display: inline-flex; align-items: center; gap: .3rem; padding: .15rem .35rem; border-radius: .35rem; color: var(--muted); }

  /* The database a record belongs to, marked. */
  .crumb.db { background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue); font-weight: 600; }
  .crumb.db:hover { background: color-mix(in srgb, var(--blue) 22%, transparent); }
  .slash { flex: none; padding-inline: .15rem; color: var(--muted); }
  .title { font-size: .95rem; white-space: nowrap; }
  /* Saved or saving, as on the phone; what it means, on hover. */
  .status { position: relative; flex: none; display: grid; place-items: center; padding: .2rem; border-radius: 50%; }
  .state { display: grid; place-items: center; width: 1.1rem; height: 1.1rem; border-radius: 50%; }
  .state.saved { background: #16a34a; color: #fff; }
  .state.saving { background: var(--soft); color: var(--text); }
  .state.pending { background: var(--soft); color: var(--muted); }
  .state.error { background: var(--red); color: #fff; }
  .state.saving :global(svg) { animation: spin 1s linear infinite; }
  .status-info { position: absolute; z-index: 40; inset-block-start: calc(100% + .45rem); inset-inline-start: 50%; translate: -50% 0; display: none; gap: .4rem; padding: .55rem .75rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 8px 20px rgb(0 0 0 / .18); font-size: .8rem; font-weight: 500; white-space: nowrap; }
  .status-info span { display: flex; align-items: center; gap: .45rem; }
  .status-info :global(svg) { color: var(--blue); }
  .status:hover .status-info { display: grid; }
  .spacer { flex: 1; }
  .main { grid-row: 2; position: relative; overflow: hidden; background: var(--panel); }
  .main.scroll { overflow: auto; background: var(--bg); }
  .main.full { position: fixed; inset: 0; z-index: 35; }
  .preview { position: relative; grid-row: 2; overflow: hidden; border-inline-start: 1px solid var(--line); background: var(--bg); }
  .term { grid-row: 3; grid-column: 3 / -1; position: relative; min-height: 0; }
  .ws.no-side .term { grid-column: 2 / -1; }
  .term.min { display: none; }
  .grip { position: absolute; z-index: 3; inset-inline: 0; inset-block-start: -3px; height: 6px; padding: 0; cursor: row-resize; }
  .grip:focus-visible { box-shadow: inset 0 2px 0 var(--blue); }
  .outside { position: absolute; z-index: 2; inset-inline: 1rem; inset-block-start: .75rem; display: flex; gap: .5rem; align-items: center; padding: .5rem .75rem; border-radius: .5rem; background: var(--soft); font-size: .85rem; }
  .outside span { flex: 1; }
  .note { height: 100%; display: flex; flex-direction: column; }
  .inline-title { width: 100%; margin-block-end: 1rem; padding: 0; border: 0; outline: none; background: transparent; color: var(--text); font: inherit; font-size: 2.1rem; font-weight: 800; line-height: 1.2; }
  .day-events { display: flex; align-items: center; gap: .45rem; width: 100%; margin-block-end: 1rem; padding: .5rem .75rem; border: 1px solid color-mix(in srgb, var(--blue) 30%, var(--line)); border-radius: .6rem; background: color-mix(in srgb, var(--blue) 7%, transparent); color: var(--blue); font-size: .85rem; text-align: start; }
  .day-events b { flex-shrink: 0; }
  .day-events span { overflow: hidden; color: var(--text); white-space: nowrap; text-overflow: ellipsis; }
  .day-events:hover { background: color-mix(in srgb, var(--blue) 12%, transparent); }
  .body { flex: 1; min-height: 0; }
  .hint { padding: 3rem 2rem; text-align: center; }
  .error { position: absolute; inset-inline: 1rem; inset-block-start: .75rem; z-index: 2; padding: .6rem .9rem; border-radius: .5rem; background: var(--soft); color: var(--red); font-size: .85rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
