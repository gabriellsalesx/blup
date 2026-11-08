import { Events } from "@wailsio/runtime";
import type { DBSchema, Event, TrashItem } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";
import { iso } from "./calendar";
import { warmEditor } from "./screens";
import { t } from "../desk/i18n";
import { BoardService, CalendarService, CloudService, NotesService, SettingsService, SpaceService, type Account, type Node, type Note, type Device, type Space, type SyncResult } from "../../bindings/github.com/gabriellsalesx/blup/mobile";

export type Tab = "notes" | "search" | "calendar" | "account";
type Place = { note?: string; edit?: boolean; fresh?: boolean; board?: string; over?: number; sheet?: boolean; pick?: "folder" | "tags"; event?: Event; isNew?: boolean; panel?: "add" | "day" | "menu" | "month" | "trash" | "empty" | "cloud" | "pin" | "quick" | "folder" | "vaults" | "newdb" | "lang" | "fresh" };
export type View = { kind: "all" | "untagged" | "trash" | "folder" | "tag"; name?: string };
export type Toast = { text: string; action?: { label: string; run: () => void } };

const load = (key: string) => {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
};

// One branch of the folders open at a time: opening one closes the others,
// keeping only the way to it.
export function openOne(open: string[], path: string) {
  if (open.includes(path)) return open.filter((p) => p !== path && !p.startsWith(`${path}/`));
  return path.split("/").map((_, i, all) => all.slice(0, i + 1).join("/"));
}

const AUTO_SYNC_GAP = 30_000; // the least time between two syncs of their own
const LOCK_AFTER = 60_000; // away from the app this long, it asks the PIN again

// The state of the app. Opening a note or a sheet goes through the browser
// history, so the back button of the phone closes them in order.
class App {
  // The lists from Go are only ever replaced whole, never changed in place:
  // $state.raw keeps them as they came, with no proxy over every note.
  notes = $state.raw<Note[]>([]);
  view = $state<View>({ kind: "all" }); // what the notes screen lists
  trash = $state.raw<TrashItem[]>([]);
  // The door: closed from the start, so no note shows before the app knows.
  locked = $state(true);
  lockKnown = $state(false);
  pinOn = $state(false);
  hiddenAt = 0;
  notifications = $state(true);
  showHolidays = $state(load("blup.cal.holidays") !== "0");
  folders = $state.raw<string[]>([]);
  tree = $state.raw<Node[]>([]);
  spaces = $state.raw<Space[]>([]); // the vaults of this phone
  space = $derived(this.spaces.find((s) => s.current));
  extraTags = $state.raw<string[]>([]); // made by hand, maybe on no note yet
  renamed = new Map<string, string>();
  noteKey = $state(0); // changes when another note opens, not when one is renamed
  openFolders = $state<string[]>(openOne([], (load("folders") ?? "").split("\n").filter(Boolean).at(-1) ?? "").filter(Boolean)); // open in the side menu
  tab = $state<Tab>("notes");
  place = $state<Place>({});
  device = $state<Device | null>(null);
  error = $state("");
  keyboard = $state(false);
  events = $state.raw<Event[]>([]);
  eventsLoaded = $state(false);
  day = $state(iso(new Date())); // the calendar's chosen day
  toast = $state<Toast | null>(null);
  toastTimer: ReturnType<typeof setTimeout> | undefined;
  account = $state<Account | null>(null);
  syncing = $state(false);
  synced = $state<SyncResult | null>(null);
  syncError = $state("");
  // The sync stopped for safety (Blup Cloud shows a version that does not
  // continue this phone's): the Cloud card offers "Compare from scratch".
  syncStuck = $state(false);
  lastSync = $state(0); // when the notes last reached the Cloud
  current = $derived(this.notes.find((n) => n.id === this.place.note));

  async start() {
    addEventListener("popstate", (e) => {
      this.leaving = false;
      this.arrive((e.state as Place) ?? {});
    });
    matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => this.applyTheme());
    this.applyTheme();
    // The window does not shrink under the keyboard, so the app does it.
    Events.On("common:keyboard", (e) => {
      const k = keyboardOf(e.data);
      if (!k) return;
      const { visible, height } = k;
      const root = document.documentElement.style;
      this.keyboard = visible;
      root.setProperty("--kb", visible ? `${height / devicePixelRatio}px` : "0px");
      root.setProperty("--kb-pad", visible ? "0px" : "var(--sab)");
    });
    try {
      // Asked together: each call crosses to Go and back.
      const [pinOn, notifications, d] = await Promise.all([SettingsService.Locked(), SettingsService.Notifications(), NotesService.Device(Intl.DateTimeFormat().resolvedOptions().timeZone)]);
      this.pinOn = pinOn;
      this.notifications = notifications;
      this.locked = this.pinOn;
      this.lockKnown = true;
      this.device = d;
      const root = document.documentElement.style;
      root.setProperty("--sat", `${d.top / devicePixelRatio}px`);
      root.setProperty("--sab", `${d.bottom / devicePixelRatio}px`);
      await this.refresh();
      performance.mark("blup:ready"); // the notes are on screen: what measures the opening
      warmEditor();
    } catch (e) {
      this.error = String(e);
      this.locked = false;
    }
    await this.loadAccount();
    this.autoSync();
    document.addEventListener("visibilitychange", () => {
      if (document.hidden) {
        this.hiddenAt = Date.now();
        return;
      }
      if (this.pinOn && Date.now() - this.hiddenAt > LOCK_AFTER) this.locked = true;
      this.autoSync();
    });
  }

  async loadAccount() {
    try {
      this.account = await CloudService.Account();
    } catch (e) {
      this.syncError = message(e);
    }
  }

  // Sync happens on opening the app and on coming back to it. An open note
  // waits: what is on screen must not change under the person's fingers.
  // Back to the app a moment after the last one (a photo picked, a glance
  // at another app), it waits: each sync goes to the network.
  autoSync() {
    if (!this.place.note && Date.now() - this.lastSync > AUTO_SYNC_GAP) this.sync();
  }

  // fresh: "Compare from scratch", only after the person confirmed it in
  // its sheet (the phone forgets the last sync and compares file by file;
  // nothing is deleted).
  async sync(fresh = false) {
    if (this.syncing || !this.account?.vault || this.account.notes !== "ready") return;
    this.syncing = true;
    this.syncError = "";
    try {
      const result = await (fresh ? CloudService.SyncFresh() : CloudService.Sync());
      this.syncStuck = false;
      if (fresh) this.say({ text: t.cloud.freshDone });
      if (result.gone) {
        // Deleted on another device: said once; the vault is only here now.
        this.say({ text: t.cloud.gone });
        this.spaces = (await SpaceService.List()) ?? [];
        await this.loadAccount();
        return;
      }
      this.synced = result;
      this.lastSync = Date.now();
      await this.refresh();
      await this.loadAccount(); // the space in use changed
    } catch (e) {
      this.syncError = message(e);
      this.syncStuck = stuck(e);
    } finally {
      this.syncing = false;
    }
  }

  // Everything the screens show, asked together and put on screen at once.
  async refresh() {
    const [notes, events, trash, folders, tree, spaces, extraTags] = await Promise.all([
      NotesService.List(), CalendarService.Events(), NotesService.Trash(), NotesService.Folders(), BoardService.Tree(), SpaceService.List(), SettingsService.Tags(),
    ]);
    this.notes = notes ?? [];
    this.events = events ?? [];
    this.eventsLoaded = true;
    this.trash = trash ?? [];
    this.folders = folders ?? [];
    this.tree = tree ?? [];
    this.spaces = spaces ?? [];
    this.extraTags = extraTags ?? [];
  }

  // Only the notes: what a save in the editor changes (its text, tags, links
  // and date). The rest stays, and typing costs the phone less.
  async refreshNotes() {
    this.notes = (await NotesService.List()) ?? [];
  }

  // After another vault opens: what was on screen belongs to the last one.
  async entered() {
    this.view = { kind: "all" };
    this.synced = null;
    this.syncError = "";
    this.syncStuck = false;
    this.lastSync = 0;
    this.openFolders = [];
    this.renamed.clear();
    this.toast = null;
    this.device = await NotesService.Device(Intl.DateTimeFormat().resolvedOptions().timeZone);
    await this.refresh();
    await this.loadAccount();
    this.sync();
  }

  say(toast: Toast) {
    clearTimeout(this.toastTimer);
    this.toast = toast;
    this.toastTimer = setTimeout(() => (this.toast = null), toast.text.length > 90 ? 12000 : 6000); // a long one needs time to be read
  }

  setHolidays(on: boolean) {
    this.showHolidays = on;
    try {
      localStorage.setItem("blup.cal.holidays", on ? "1" : "0");
    } catch {}
  }

  toggleFolder(path: string) {
    this.openFolders = openOne(this.openFolders, path);
    try {
      localStorage.setItem("folders", this.openFolders.join("\n"));
    } catch {}
  }

  arrive(place: Place) {
    // The history may still hold the old ID of a note renamed or moved.
    const now = place.note && this.renamed.get(place.note);
    if (now) {
      place = { ...place, note: now };
      history.replaceState(place, "");
    }
    if (place.note !== this.place.note) this.noteKey++;
    this.place = place;
  }

  go(place: Place) {
    history.pushState(place, "");
    this.arrive(place);
  }

  // Goes to a place in the current one's stead, so back skips what it replaced.
  swap(place: Place) {
    history.replaceState(place, "");
    this.arrive(place);
  }

  // The open note got a new ID (renamed or moved): it stays open as it is.
  async moved(id: string) {
    const notes = (await NotesService.List()) ?? [];
    this.notes = notes;
    const old = this.place.note;
    if (old) {
      for (const [from, to] of this.renamed) if (to === old) this.renamed.set(from, id);
      this.renamed.set(old, id);
    }
    this.place = { ...this.place, note: id };
    history.replaceState($state.snapshot(this.place), "");
    await this.refresh();
  }

  // One step at a time: history.back() takes a moment, and a second gesture
  // in that moment would go a screen too far.
  private leaving = false;
  back() {
    if (this.leaving) return;
    this.leaving = true;
    setTimeout(() => (this.leaving = false), 400);
    history.back();
  }

  // Notes pressed while already there: back to all of them, from a folder.
  // Coming from another tab, the list stays as it was.
  select(tab: Tab) {
    NotesService.Haptic();
    if (tab === "notes" && this.tab === "notes" && this.view.kind !== "all") this.view = { kind: "all" };
    this.tab = tab;
  }

  open(id: string, edit = false) {
    this.go({ note: id, edit });
  }

  // A new canvas or drawing, with no name to ask, as in the Studio.
  // Where something new goes: the open folder, unless it is the folder of
  // another kind (Canvas, Desenhos, Bancos de dados) or inside one; then the
  // kind's own folder. A note there goes to the vault's top.
  placeFor(kind: "note" | "canvas" | "drawing" | "database") {
    const homes = { note: "", canvas: t.boards.canvasFolder, drawing: t.boards.drawingFolder, database: t.db.folder };
    const here = this.view.kind === "folder" ? this.view.name ?? "" : "";
    const top = here.split("/")[0];
    const other = Object.entries(homes).some(([k, f]) => k !== kind && f && top.toLowerCase() === f.toLowerCase());
    return !here || other ? homes[kind] : here;
  }
  async newBoard(kind: "canvas" | "drawing") {
    try {
      const b = t.boards, p = (n: number) => String(n).padStart(2, "0"), d = new Date();
      const stamp = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}.${p(d.getMinutes())}.${p(d.getSeconds())}`;
      const rel = kind === "canvas" ? await BoardService.CreateCanvas(this.placeFor(kind), `${b.canvasName} ${stamp}`) : await BoardService.CreateDrawing(this.placeFor(kind), `${b.drawingName} ${stamp}`);
      await this.refresh();
      this.swap({ board: rel });
    } catch (e) { this.say({ text: message(e) }); }
  }
  async newDatabase(title: string, schema: DBSchema) {
    try {
      const rel = await BoardService.CreateDatabase(this.placeFor("database"), title, schema);
      await this.refresh();
      this.swap({ board: rel });
    } catch (e) { this.say({ text: message(e) }); }
  }

  openTitle(title: string) {
    const note = this.notes.find((n) => n.title.toLowerCase() === title.toLowerCase());
    if (note) this.open(note.id);
  }

  // Sends a note to the trash, with Undo on the toast.
  async trashNote(id: string) {
    const where = await CalendarService.Delete(id);
    await this.refresh();
    this.sync();
    this.say({ text: t.notes.noteInTrash, action: { label: t.common.undo, run: async () => { await CalendarService.Restore(where); await this.refresh(); this.sync(); } } });
  }

  async create(folder = "") {
    const id = await NotesService.Create(folder);
    await this.refresh();
    this.go({ note: id, edit: true, fresh: true });
  }

  // The app always follows the phone's light or dark.
  applyTheme() {
    const dark = matchMedia("(prefers-color-scheme: dark)").matches;
    document.documentElement.dataset.theme = dark ? "dark" : "light";
    NotesService.StatusBar(!dark);
  }
}

export const app = new App();

// stuck tells a sync stopped for safety: the Go side sends {stuck: true}
// as the error's cause.
export function stuck(e: unknown) {
  const c = isObject(e) ? e.cause : null;
  return isObject(c) && c.stuck === true;
}

export function message(e: unknown) {
  const text = (e instanceof Error ? e.message : String(e)).replace(/^.*returned an error: /, "");
  try {
    const body: unknown = JSON.parse(text);
    return isObject(body) && typeof body.message === "string" ? body.message : text;
  } catch {
    return text;
  }
}

// Dates and numbers go by the app's language (t.locale).

// A size, as the Cloud's space shows it.
export function bytes(n: number) {
  const text = (v: number, digits: number) => v.toLocaleString(t.locale, { maximumFractionDigits: digits });
  if (n < 1 << 20) return `${text(Math.max(0, Math.ceil(n / 1024)), 0)} KB`;
  if (n < 1 << 30) return `${text(n / (1 << 20), n < 10 << 20 ? 1 : 0)} MB`;
  return `${text(n / (1 << 30), 1)} GB`;
}

export function when(ms: number) {
  const date = new Date(ms);
  if (date.toDateString() === new Date().toDateString()) return date.toLocaleTimeString(t.locale, { hour: "2-digit", minute: "2-digit" });
  return date.toLocaleDateString(t.locale, { day: "numeric", month: "short" }).replace(" de ", " ").replace(".", "");
}

const isObject = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);

// The keyboard event from Go: { visible, height }, as JSON text or an object.
function keyboardOf(data: unknown): { visible: boolean; height: number } | null {
  let v: unknown = data;
  if (typeof v === "string") {
    try { v = JSON.parse(v); } catch { return null; }
  }
  if (!isObject(v) || typeof v.visible !== "boolean") return null;
  return { visible: v.visible, height: typeof v.height === "number" && Number.isFinite(v.height) ? v.height : 0 };
}
