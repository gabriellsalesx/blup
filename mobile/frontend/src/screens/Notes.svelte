<script lang="ts">
  import { fade, fly } from "svelte/transition";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import Menu from "@lucide/svelte/icons/menu";
  import Database from "@lucide/svelte/icons/database";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import DatabaseNew from "../desk/DatabaseNew.svelte";
  import NotebookPen from "@lucide/svelte/icons/notebook-pen";
  import Plus from "@lucide/svelte/icons/plus";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import type { TrashItem } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { CalendarService, NotesService , type Node } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message, when } from "../lib/app.svelte";
  import NoteRow from "../ui/NoteRow.svelte";
  import NotesMenu from "../ui/NotesMenu.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import VaultPick from "../ui/VaultPick.svelte";
  import { fmt, plural, t } from "../desk/i18n";
  import { PAGE, nearEnd } from "../lib/paging";

  const inTrash = $derived(app.view.kind === "trash");
  const title = $derived.by(() => {
    const { kind, name } = app.view;
    return kind === "trash" ? t.notes.trash : kind === "untagged" ? t.notes.untagged : kind === "tag" ? `#${name}` : kind === "folder" ? name?.split("/").pop() : t.notes.all;
  });
  // A folder also shows its canvases, drawings and databases; a database's
  // records are in it, so they stay out of the list of the folder.
  const inside = $derived.by(() => {
    const boards: { name: string; path: string; kind: string }[] = [], tables: string[] = [];
    if (app.view.kind !== "folder") return { boards, tables };
    const find = (list: Node[], path: string): Node | undefined => {
      for (const n of list) {
        if (n.path === path) return n;
        const kid = path.startsWith(`${n.path}/`) ? find(n.children ?? [], path) : undefined;
        if (kid) return kid;
      }
    };
    const walk = (n: Node) => {
      for (const kid of n.children ?? []) {
        if (kid.table) { boards.push({ name: kid.name, path: kid.table, kind: "database" }); tables.push(kid.path); }
        else if (kid.dir) walk(kid);
        else if (kid.kind) boards.push({ name: kid.name, path: kid.path, kind: kid.kind });
      }
    };
    const top = find(app.tree, app.view.name ?? "");
    if (top) walk(top);
    return { boards, tables };
  });
  const boardKinds: Record<string, { label: () => string; icon: typeof Database }> = { canvas: { label: () => t.notes.canvas, icon: LayoutDashboard }, drawing: { label: () => t.notes.drawing, icon: PenTool }, database: { label: () => t.notes.database, icon: Database } };
  const shown = $derived.by(() => {
    const { kind, name } = app.view;
    if (kind === "folder") return app.notes.filter((n) => (n.folder === name || n.folder.startsWith(`${name}/`)) && !inside.tables.some((t) => n.folder === t || n.folder.startsWith(`${t}/`)));
    if (kind === "tag") return app.notes.filter((n) => n.tags?.includes(name ?? ""));
    if (kind === "untagged") return app.notes.filter((n) => !n.tags?.length);
    return app.notes;
  });

  // Another list (folder, tag) starts again from the first piece.
  let limit = $derived.by(() => (void app.view, PAGE));
  const rows = $derived(shown.length > limit ? shown.slice(0, limit) : shown);

  // Sliding the list to the right brings the side menu along with the finger.
  let pull = $state(0);
  let pulling = $state(false);
  let closing = $state(false);
  let touch: { x: number; y: number; at: number } | null = null;
  const menuWidth = () => Math.min(innerWidth * 0.85, 352);

  function down(e: PointerEvent) {
    touch = { x: e.clientX, y: e.clientY, at: e.timeStamp };
  }
  function move(e: PointerEvent) {
    if (!touch) return;
    const x = e.clientX - touch.x, y = e.clientY - touch.y;
    if (!pulling) {
      if (Math.abs(x) < 10 && Math.abs(y) < 10) return;
      if (x < 0 || Math.abs(y) > Math.abs(x)) { touch = null; return; }
      pulling = true;
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    }
    pull = Math.min(1, Math.max(0.001, x / menuWidth()));
  }
  function up(e: PointerEvent) {
    const start = touch;
    touch = null;
    if (!pulling || !start) return;
    pulling = false;
    const flicked = (e.clientX - start.x) / (e.timeStamp - start.at) > 0.5;
    if (pull > 0.35 || flicked) app.go({ panel: "menu" });
    else letGo();
    pull = 0;
  }
  function letGo() {
    closing = true;
    setTimeout(() => (closing = false), 200);
  }
  function cancel() {
    touch = null;
    if (!pulling) return;
    pulling = false;
    pull = 0;
    letGo();
  }

  // The + makes a note; held down, it shows the quick note: the Studio's note
  // of the day, with the time at its end and the cursor under it.
  const quickOpen = $derived(app.place.panel === "quick");
  let holdTimer: ReturnType<typeof setTimeout> | undefined;
  let held = false; // the click at the end of a hold is not a tap
  function press() {
    held = false;
    holdTimer = setTimeout(() => {
      held = true;
      if (quickOpen) return;
      NotesService.HapticMedium();
      app.go({ panel: "quick" });
    }, 450);
  }
  const release = () => clearTimeout(holdTimer);
  function tapFab() {
    if (held) return;
    NotesService.Haptic();
    if (quickOpen) app.back();
    else app.create(app.placeFor("note"));
  }
  // The WebView gives a long press a strong vibration of its own: holding the
  // touch back from it leaves only the one above.
  function ownTouch(el: HTMLElement) {
    el.addEventListener("touchstart", (e) => e.preventDefault(), { passive: false });
    el.addEventListener("touchend", (e) => {
      const t = e.changedTouches[0];
      if (document.elementFromPoint(t.clientX, t.clientY)?.closest(".fab")) tapFab();
    });
  }
  async function quickNote() {
    try {
      const id = await CalendarService.DailyNote();
      await app.refresh();
      app.swap({ note: id, edit: true });
    } catch (e) { app.say({ text: message(e) }); }
  }

  let picked = $state<TrashItem | null>(null);
  let confirming = $state(false);
  function pick(item: TrashItem) {
    picked = item;
    confirming = false;
    app.go({ panel: "trash" });
  }
  const from = (item: TrashItem) => item.origin.split("/").slice(0, -1).join("/") || t.notes.all;
  const size = (item: TrashItem) => (item.dir ? plural(item.notes, t.notes.folder1, t.notes.folderN) : t.notes.note);
  const facts = (item: TrashItem) => `${size(item)} · ${fmt(t.notes.wasIn, { folder: from(item) })}`;

  // Runs what changes the trash and closes the sheet it came from.
  async function act(run: () => Promise<unknown>, done?: string) {
    try {
      await run();
      app.back();
      await app.refresh();
      if (done) app.say({ text: done });
    } catch (e) { app.say({ text: message(e) }); }
  }
  const restore = (item: TrashItem) => act(async () => { await CalendarService.Restore(item.path); app.sync(); }, t.notes.restored);
</script>

<div class="body" role="presentation" onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={cancel}>
<header>
  <button class="icon" aria-label={t.notes.menu} onclick={() => app.go({ panel: "menu" })}><Menu size={24} /></button>
  {#if app.view.kind === "all"}
    <button class="vault" aria-label={t.notes.switchVault} onclick={() => app.go({ panel: "vaults" })}><h1>{app.space?.title ?? t.notes.all}</h1><ChevronDown size={20} /></button>
  {:else}
    <h1>{title}</h1>
  {/if}
  {#if inTrash}
    {#if app.trash.length}<button class="clear" onclick={() => app.go({ panel: "empty" })}>{t.notes.empty}</button>{/if}
  {:else}
    <span>{shown.length + inside.boards.length}</span>
    {#if app.account?.vault}
      <button class={["icon sync", { spin: app.syncing }]} aria-label={t.common.sync} disabled={app.syncing} onclick={() => app.sync()}><RefreshCw size={21} /></button>
    {/if}
  {/if}
</header>
<div class="list">
  {#if inTrash}
    {#each app.trash as item (item.path)}
      <button class="gone" onclick={() => pick(item)}>
        <span class="top"><b>{item.name}</b><time>{when(new Date(item.removed).getTime())}</time></span>
        <small>{facts(item)}</small>
      </button>
    {:else}<p>{t.notes.trashEmpty}</p>{/each}
    {#if app.trash.length}<p class="note">{t.notes.trashLocal}</p>{/if}
  {:else}
    {#each inside.boards as item (item.path)}
      {@const kind = boardKinds[item.kind]}
      <button class="board" onclick={() => app.go({ board: item.path })}>
        <span class="tile"><kind.icon size={20} /></span>
        <span class="text"><b>{item.name}</b><small>{kind.label()}</small></span>
      </button>
    {/each}
    {#each rows as note (note.id)}<NoteRow {note} />{:else}{#if !inside.boards.length}<p>{app.view.kind === "folder" ? t.notes.nothingFolder : t.notes.nothingNotes}</p>{/if}{/each}
    {#if rows.length < shown.length}{#key limit}<div class="more" {@attach nearEnd(() => (limit += PAGE * 2))}></div>{/key}{/if}
  {/if}
</div>
</div>
{#if !inTrash}
  {#if quickOpen}
    <button class="scrim" aria-label={t.common.close} onclick={() => app.back()} transition:fade={{ duration: 150 }}></button>
    <div class="quicks" transition:fly={{ y: 24, duration: 160 }}>
      <button class="quick" onclick={() => app.swap({ panel: "newdb" })}><span>{t.notes.database}</span><i><Database size={23} /></i></button>
      <button class="quick" onclick={() => app.newBoard("drawing")}><span>{t.notes.drawing}</span><i><PenTool size={23} /></i></button>
      <button class="quick" onclick={() => app.newBoard("canvas")}><span>{t.notes.canvas}</span><i><LayoutDashboard size={23} /></i></button>
      <button class="quick" onclick={quickNote}><span>{t.notes.daily}</span><i><NotebookPen size={23} /></i></button>
    </div>
  {/if}
  <button {@attach ownTouch} class={["fab", { open: quickOpen }]} aria-label={quickOpen ? t.common.close : t.notes.newNote} onpointerdown={press} onpointerup={release} onpointerleave={release} onpointercancel={release} oncontextmenu={(e) => e.preventDefault()} onclick={tapFab}><Plus size={26} /></button>
{/if}

{#if app.place.panel === "newdb"}
  <DatabaseNew onCreate={(title, tpl) => app.newDatabase(title, tpl.schema)} onClose={() => app.back()} />
{/if}
{#if app.place.panel === "menu" || app.place.panel === "folder" || pulling || closing}
  <NotesMenu pull={app.place.panel === "menu" || app.place.panel === "folder" ? 1 : pull} moving={pulling} />
{/if}

{#if app.place.panel === "vaults"}<VaultPick />{/if}

{#if app.place.panel === "trash" && picked}
  {@const item = picked}
  <Sheet title={item.name} onclose={() => app.back()}>
    <p class="facts">{facts(item)}</p>
    <button class="main" onclick={() => restore(item)}>{t.notes.restore}</button>
    {#if confirming}
      <button class="danger solid" onclick={() => act(() => NotesService.TrashDelete(item.path), t.notes.deletedForGood)}>{t.notes.confirmDelete}</button>
    {:else}
      <button class="danger" onclick={() => (confirming = true)}>{t.notes.deleteForGood}</button>
    {/if}
  </Sheet>
{:else if app.place.panel === "empty"}
  <Sheet title={t.notes.emptyTitle} onclose={() => app.back()}>
    <p class="facts">{t.notes.emptyLead}</p>
    <button class="danger solid" onclick={() => act(() => NotesService.TrashEmpty(), t.notes.emptied)}>{t.notes.empty}</button>
  </Sheet>
{/if}

<style>
  .body { flex: 1; min-height: 0; display: flex; flex-direction: column; touch-action: pan-y; }
  header { flex: none; display: flex; align-items: center; gap: .25rem; padding: calc(.5rem + var(--sat)) .75rem .5rem .35rem; }
  h1 { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 1.6rem; font-weight: 700; letter-spacing: -.02em; }
  header span { margin-inline-start: auto; color: var(--muted); font-size: .9rem; }
  .vault { min-width: 0; display: flex; align-items: center; gap: .25rem; color: var(--muted); }
  .vault h1 { color: var(--text); }
  .icon { flex: none; display: grid; place-items: center; width: 2.75rem; height: 2.75rem; }
  .sync { color: var(--muted); }
  .clear { margin-inline-start: auto; min-height: 2.5rem; padding-inline: .7rem; font-weight: 600; color: var(--red); }
  .spin :global(svg) { animation: turn 1s linear infinite; }
  @keyframes turn { to { transform: rotate(1turn); } }
  .list { flex: 1; min-height: 0; overflow-y: auto; touch-action: pan-y; padding-block-end: 5.5rem; border-block-start: 1px solid var(--line); }
  .more { height: 1px; }
  p { padding: 3rem 1rem; text-align: center; color: var(--muted); }
  .note { padding: 1.25rem 1rem; font-size: .85rem; }
  .gone { display: grid; gap: .3rem; width: 100%; padding: .9rem 1rem; text-align: start; border-block-end: 1px solid var(--line); }
  .gone:active { background: var(--soft); }
  .board { display: flex; align-items: center; gap: .75rem; width: 100%; padding: .8rem 1rem; text-align: start; border-block-end: 1px solid var(--line); }
  .board:active { background: var(--soft); }
  .tile { flex: none; display: grid; place-items: center; width: 2.4rem; height: 2.4rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .text { min-width: 0; display: grid; gap: .1rem; }
  .text b { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .text small { font-size: .85rem; color: var(--muted); }
  .top { display: flex; align-items: baseline; justify-content: space-between; gap: .75rem; }
  .top b { font-weight: 600; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  time { flex: none; font-size: .8rem; color: var(--muted); }
  .gone small { font-size: .85rem; color: var(--muted); }
  .facts { padding: 0; text-align: start; font-size: .9rem; line-height: 1.4; }
  .main, .danger { min-height: 2.9rem; border-radius: .6rem; font-weight: 600; }
  .main { background: var(--blue); color: #fff; }
  .danger { border: 1px solid var(--line); color: var(--red); }
  .danger.solid { border-color: var(--red); background: var(--red); color: #fff; }
  .fab { position: absolute; inset-inline-end: 1rem; inset-block-end: 1rem; display: grid; place-items: center; width: 3.5rem; height: 3.5rem; border-radius: 50%; background: var(--blue); color: #fff; box-shadow: 0 6px 16px rgb(0 0 0 / .3); touch-action: none; -webkit-touch-callout: none; user-select: none; }
  .fab:active { transform: scale(.95); }
  .fab :global(svg) { transition: rotate .15s; }
  .fab.open { z-index: 21; background: var(--panel); color: var(--text); }
  .fab.open :global(svg) { rotate: 45deg; }
  .scrim { position: fixed; inset: 0; z-index: 20; background: rgb(0 0 0 / .6); }
  .quicks { position: absolute; inset-inline-end: 1rem; inset-block-end: 5.25rem; z-index: 21; display: grid; justify-items: end; gap: .7rem; }
  .quick { display: flex; align-items: center; gap: .75rem; }
  .quick span { padding: .4rem .75rem; border-radius: .6rem; background: var(--panel); font-size: .9rem; font-weight: 600; }
  .quick i { display: grid; place-items: center; width: 3.5rem; height: 3.5rem; border-radius: 50%; background: var(--blue); color: #fff; box-shadow: 0 6px 16px rgb(0 0 0 / .3); }
</style>
