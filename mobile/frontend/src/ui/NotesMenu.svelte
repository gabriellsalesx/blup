<script lang="ts">
  import { untrack, type Component } from "svelte";
  import { fade, fly } from "svelte/transition";
  import Check from "@lucide/svelte/icons/check";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import FileText from "@lucide/svelte/icons/file-text";
  import Folder from "@lucide/svelte/icons/folder";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import Hash from "@lucide/svelte/icons/hash";
  import Library from "@lucide/svelte/icons/library";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Plus from "@lucide/svelte/icons/plus";
  import Tag from "@lucide/svelte/icons/tag";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Database from "@lucide/svelte/icons/database";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import { CalendarService, NotesService, SettingsService, type Node } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message, type View } from "../lib/app.svelte";
  import BlupLogo from "./BlupLogo.svelte";
  import Sheet from "./Sheet.svelte";
  import { fmt, plural, t } from "../desk/i18n";

  // The side menu of the notes: what to show in the list. pull is how far it
  // is out, for when a finger brings it: 1 is all the way, as the button does.
  let { pull = 1, moving = false }: { pull?: number; moving?: boolean } = $props();
  const pulled = untrack(() => pull < 1);

  // Sliding it to the left takes it away with the finger, from the menu or
  // from the dark area beside it.
  let menu = $state<HTMLElement>();
  let drag = $state<number | null>(null);
  const shown = $derived(drag ?? pull);
  let touch: { x: number; y: number; at: number } | null = null;
  let dragged = false; // the click after a drag is not a tap

  function down(e: PointerEvent) {
    if ((e.target as HTMLElement).closest("input")) return;
    touch = { x: e.clientX, y: e.clientY, at: e.timeStamp };
    dragged = false;
  }
  function move(e: PointerEvent) {
    if (!touch) return;
    const x = e.clientX - touch.x, y = e.clientY - touch.y;
    if (drag === null) {
      if (Math.abs(x) < 10 && Math.abs(y) < 10) return;
      if (x > 0 || Math.abs(y) > Math.abs(x)) { touch = null; return; }
      dragged = true;
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    }
    drag = Math.min(1, Math.max(0, 1 + x / (menu?.offsetWidth ?? 1)));
  }
  function up(e: PointerEvent) {
    const start = touch;
    touch = null;
    if (drag === null || !start) return;
    const flicked = (start.x - e.clientX) / (e.timeStamp - start.at) > 0.5;
    if (drag < 0.65 || flicked) app.back();
    else drag = null;
  }
  function cancel() {
    touch = null;
    drag = null;
  }
  let showTags = $state(app.view.kind === "tag");

  function count(names: string[]) {
    const seen = new Map<string, number>();
    for (const name of names) seen.set(name, (seen.get(name) ?? 0) + 1);
    return [...seen].sort(([a], [b]) => a.localeCompare(b, t.locale));
  }
  // The vault as a tree, as in the Studio: folders with what is in them,
  // a database as its records' folder. Each folder counts what is in it: a
  // note, a canvas, a drawing or a database is one; a database counts records.
  type Branch = { path: string; name: string; total: number; kids: Branch[]; dir: boolean; kind: string; table: string };
  const branch = (n: Node): Branch => {
    const kids = (n.children ?? []).map(branch);
    const total = kids.reduce((sum, k) => sum + (k.dir && !k.table ? k.total : 1), 0);
    return { path: n.path, name: n.name, total, kids, dir: n.dir, kind: n.kind, table: n.table };
  };
  const folders = $derived(app.tree.map(branch));
  const fileIcons: Record<string, Component<{ size?: number }>> = { canvas: LayoutDashboard, drawing: PenTool, database: Database };
  const tags = $derived.by(() => {
    const used = count(app.notes.flatMap((n) => n.tags ?? []));
    const unused = app.extraTags.filter((t) => !used.some(([name]) => name === t)).map((t) => [t, 0] as [string, number]);
    return [...used, ...unused].sort(([a], [b]) => a.localeCompare(b, t.locale));
  });
  const untagged = $derived(app.notes.filter((n) => !n.tags?.length).length);

  const focus = (el: HTMLInputElement) => {
    setTimeout(() => {
      el.focus();
      el.select();
      el.scrollIntoView({ block: "center" });
    }, 240);
  };

  // Making or renaming happens in the list itself: a line with a field, done
  // by Enter or by leaving it, as in the Studio.
  let adding = $state<{ tag: boolean; parent: string } | null>(null);
  let renaming = $state("");
  let draft = $state("");
  function startNew(parent = "") {
    draft = "";
    adding = { tag: showTags, parent };
    if (parent && !app.openFolders.includes(parent)) app.toggleFolder(parent);
  }
  async function finishNew() {
    const job = adding, name = draft.trim();
    adding = null;
    if (!job || !name) return;
    try {
      if (job.tag) await SettingsService.AddTag(name);
      else await NotesService.CreateFolder(job.parent, name);
      await app.refresh();
    } catch (e) { app.say({ text: message(e) }); }
  }
  const within = (path: string, base: string) => path === base || path.startsWith(`${base}/`);
  async function finishRename() {
    const from = renaming, name = draft.trim();
    renaming = "";
    if (!from || !name || name === from.split("/").pop()) return;
    try {
      const to = await NotesService.Rename(from, name);
      if (app.view.kind === "folder" && app.view.name && within(app.view.name, from)) app.view = { kind: "folder", name: to + app.view.name.slice(from.length) };
      if (app.openFolders.includes(from)) { app.toggleFolder(from); app.toggleFolder(to); }
      await app.refresh();
      app.sync();
    } catch (e) { app.say({ text: message(e) }); }
  }
  const submit = (e: SubmitEvent) => {
    e.preventDefault();
    (e.currentTarget as HTMLFormElement).querySelector("input")?.blur();
  };

  // The three dots of a folder. The calendar's folder (Agenda, or Calendar
  // in English, see internal/app/calendar.go) stays.
  const calendarFolders = ["Agenda", "Calendar"];
  let chosen = $state<Branch | null>(null);
  let confirming = $state(false);
  function more(branch: Branch) {
    chosen = branch;
    confirming = false;
    app.go({ panel: "folder" });
  }
  function subfolder(branch: Branch) {
    app.back();
    startNew(branch.path);
  }
  function rename(branch: Branch) {
    app.back();
    draft = branch.name;
    renaming = branch.path;
  }
  async function remove(branch: Branch) {
    try {
      const where = await CalendarService.Delete(branch.path);
      if (app.view.kind === "folder" && app.view.name && within(app.view.name, branch.path)) app.view = { kind: "all" };
      app.back();
      await app.refresh();
      app.sync();
      app.say({ text: t.menu.folderInTrash, action: { label: t.common.undo, run: async () => { await CalendarService.Restore(where); await app.refresh(); app.sync(); } } });
    } catch (e) { app.say({ text: message(e) }); }
  }

  const on = (v: View) => app.view.kind === v.kind && app.view.name === v.name;
  function choose(v: View) {
    app.view = v;
    app.back();
  }
</script>

{#snippet item(Icon: Component<{ size?: number }>, label: string, total: number, view: View)}
  <button type="button" class="item" aria-current={on(view) ? "true" : undefined} onclick={() => choose(view)}>
    <Icon size={21} /><span>{label}</span><small>{total}</small>
  </button>
{/snippet}

{#snippet entry(Icon: Component<{ size?: number }>, label: string, total: number, done: () => void)}
  <form class="entry" onsubmit={submit}>
    <Icon size={21} /><input type="text" bind:value={draft} {@attach focus} onblur={done} aria-label={label} enterkeyhint="done" autocapitalize={showTags ? "off" : "sentences"} />
    <button type="submit" class="ok" aria-label={t.common.confirm} disabled={!draft.trim()} onpointerdown={(e) => e.preventDefault()} onmousedown={(e) => e.preventDefault()}><Check size={18} /></button>
  </form>
{/snippet}

{#snippet file(leaf: Branch, depth: number)}
  {@const Icon = fileIcons[leaf.kind] ?? FileText}
  <div class="branch" style:--depth={depth}>
    <span class="toggle"></span>
    <button type="button" class="item" onclick={() => app.swap(leaf.kind ? { board: leaf.path } : { note: leaf.path })}><Icon size={21} /><span class="leaf">{leaf.name}</span></button>
  </div>
{/snippet}

{#snippet folder(branch: Branch, depth: number)}
  {@const open = app.openFolders.includes(branch.path)}
  {@const held = !calendarFolders.includes(branch.path) && !branch.table}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="branch" style:--depth={depth} oncontextmenu={(e) => { e.preventDefault(); if (held && renaming !== branch.path) more(branch); }}>
    {#if branch.kids.length}
      <button type="button" class={["toggle", { open }]} aria-expanded={open} aria-label={fmt(t.menu.whatsIn, { name: branch.name })} onclick={() => app.toggleFolder(branch.path)}><ChevronRight size={19} /></button>
    {:else}<span class="toggle"></span>{/if}
    {#if renaming === branch.path}
      {@render entry(Folder, t.menu.folderNewName, branch.total, finishRename)}
    {:else if branch.table}
      <button type="button" class="item" onclick={() => app.swap({ board: branch.table })}><Database size={21} /><span>{branch.name}</span><small>{branch.total}</small></button>
    {:else}
      {@render item(open ? FolderOpen : Folder, branch.name, branch.total, { kind: "folder", name: branch.path })}
      {#if held}<button type="button" class="more" aria-label={fmt(t.common.options, { name: branch.name })} onclick={() => more(branch)}><Ellipsis size={20} /></button>{/if}
    {/if}
  </div>
  {#if adding?.parent === branch.path && !adding.tag}
    <div class="branch" style:--depth={depth + 1}><span class="toggle"></span>{@render entry(Folder, t.menu.subName, 0, finishNew)}</div>
  {/if}
  {#if open}{#each branch.kids as kid (kid.path)}{#if kid.dir}{@render folder(kid, depth + 1)}{:else}{@render file(kid, depth + 1)}{/if}{/each}{/if}
{/snippet}

<button class={["scrim", { passive: pull < 1 }]} aria-label={t.common.close} style:opacity={shown < 1 ? shown : null} onclick={() => { if (!dragged) app.back(); }}
  onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={cancel} in:fade={{ duration: pulled ? 0 : 150 }} out:fade={{ duration: 150 }}></button>
<nav bind:this={menu} class={{ moving: moving || drag !== null }} aria-label={t.menu.filter} style:transform={shown < 1 ? `translateX(${(shown - 1) * 100}%)` : null}
  onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={cancel} onclickcapture={(e) => { if (dragged) { e.stopPropagation(); e.preventDefault(); } }}
  in:fly={{ x: -320, duration: pulled ? 0 : 220, opacity: 1 }} out:fly={{ x: -320, duration: 220, opacity: 1 }}>
  <header><img src="/blups/blup.svg" alt="" width="34" height="34" /><BlupLogo class="wordmark" /></header>
  <div class="fixed">
    <button type="button" class="item" onclick={() => app.swap({ panel: "vaults" })}><Library size={21} /><span>{t.notes.switchVault}</span><small>{app.space?.title}</small></button>
    {@render item(FileText, t.menu.all, app.notes.length, { kind: "all" })}
    {@render item(Tag, t.notes.untagged, untagged, { kind: "untagged" })}
    {@render item(Trash2, t.notes.trash, app.trash.length, { kind: "trash" })}

    <div class="switch" role="group" aria-label={t.menu.groupBy}>
      <button type="button" aria-pressed={!showTags} onclick={() => (showTags = false)}>{t.menu.folders}</button>
      <button type="button" aria-pressed={showTags} onclick={() => (showTags = true)}>{t.menu.tags}</button>
      <button type="button" class="new" aria-label={showTags ? t.menu.newTag : t.menu.newFolder} onclick={() => startNew()}><Plus size={22} /></button>
    </div>
  </div>
  <div class="scroll">
    {#if adding && !adding.parent}
      {#if adding.tag}{@render entry(Hash, t.menu.tagName, 0, finishNew)}
      {:else}<div class="branch" style:--depth={0}><span class="toggle"></span>{@render entry(Folder, t.menu.folderName, 0, finishNew)}</div>{/if}
    {/if}
    {#if showTags}
      {#each tags as [name, total] (name)}{@render item(Hash, name, total, { kind: "tag", name })}
      {:else}{#if !adding}<p>{t.menu.tagsEmpty}</p>{/if}{/each}
    {:else}
      {#each folders as branch (branch.path)}{#if branch.dir}{@render folder(branch, 0)}{:else}{@render file(branch, 0)}{/if}
      {:else}{#if !adding}<p>{t.menu.foldersEmpty}</p>{/if}{/each}
    {/if}
  </div>
</nav>

{#if app.place.panel === "folder" && chosen}
  {@const branch = chosen}
  <Sheet title={branch.name} onclose={() => app.back()}>
    <div class="actions">
      <button type="button" onclick={() => subfolder(branch)}><FolderPlus size={20} />{t.menu.newSub}</button>
      <button type="button" onclick={() => rename(branch)}><Pencil size={20} />{t.common.rename}</button>
      {#if confirming}
        <button type="button" class="danger solid" onclick={() => remove(branch)}><Trash2 size={20} />{branch.total ? plural(branch.total, t.menu.toTrash1, t.menu.toTrashN) : t.common.toTrash}</button>
      {:else}
        <button type="button" class="danger" onclick={() => (confirming = true)}><Trash2 size={20} />{t.menu.deleteFolder}</button>
      {/if}
    </div>
  </Sheet>
{/if}

<style>
  .scrim { position: fixed; inset: 0; z-index: 20; background: rgb(0 0 0 / .5); touch-action: pan-y; transition: opacity .2s; }
  .scrim.passive { pointer-events: none; }
  nav { position: fixed; inset-block: 0; inset-inline-start: 0; z-index: 21; display: flex; flex-direction: column; width: min(85%, 22rem); padding-block: var(--sat) max(var(--sab), var(--kb, 0px)); background: var(--panel); border-radius: 0 1.1rem 1.1rem 0; touch-action: pan-y; transition: transform .2s; }
  nav.moving { transition: none; }
  header { flex: none; display: flex; align-items: center; gap: .6rem; padding: .85rem 1.1rem .75rem; font-size: 1.15rem; }
  header :global(.wordmark) { height: 1.9rem; width: auto; }
  .fixed { flex: none; padding-inline: .6rem; }
  .scroll { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding: 0 .6rem 1rem; touch-action: pan-y; }
  .item { display: flex; align-items: center; gap: .85rem; width: 100%; min-height: 3rem; padding-inline: .6rem; border-radius: .6rem; color: var(--muted); text-align: start; }
  .item span { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 600; color: var(--text); }
  .item span.leaf { font-weight: 500; }
  .item small { flex: none; max-width: 45%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: .9rem; }
  .item:active { background: var(--soft); }
  .item[aria-current] { background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .item[aria-current] span { color: var(--blue-text); }
  .branch { display: flex; align-items: center; padding-inline-start: calc(var(--depth) * .85rem); -webkit-touch-callout: none; user-select: none; }
  .branch .item { flex: 1; min-width: 0; padding-inline-start: .1rem; gap: .6rem; }
  .toggle { flex: none; display: grid; place-items: center; width: 1.9rem; height: 3rem; color: var(--muted); }
  .toggle :global(svg) { transition: rotate .15s; }
  .toggle.open :global(svg) { rotate: 90deg; }
  .switch { display: flex; align-items: center; gap: 1rem; padding: 1.1rem .6rem .4rem; }
  .switch .new { display: grid; place-items: center; width: 2.5rem; margin-inline: auto -.4rem; color: var(--text); }
  .switch button:focus-visible { outline: none; }
  .more { flex: none; display: grid; place-items: center; width: 2.4rem; height: 3rem; color: var(--muted); }
  .entry { flex: 1; min-width: 0; display: flex; align-items: center; gap: .85rem; min-height: 3rem; padding-inline: .6rem; color: var(--muted); }
  .branch .entry { padding-inline-start: .1rem; gap: .6rem; }
  .entry input { flex: 1; min-width: 0; min-height: 0; height: 2.2rem; padding: 0; border: 0; border-radius: 0; background: none; font-weight: 600; }
  .entry input:focus-visible { outline: none; }
  .ok { flex: none; display: grid; place-items: center; width: 2.4rem; height: 2.4rem; border-radius: .6rem; background: var(--blue); color: #fff; }
  .ok:disabled { opacity: .4; }
  .actions { display: grid; }
  .actions button { display: flex; align-items: center; gap: .75rem; min-height: 3.1rem; padding-inline: .5rem; border-radius: .6rem; font-weight: 500; text-align: start; }
  .actions :global(svg) { color: var(--blue-text); }
  .danger, .actions .danger :global(svg) { color: var(--red); }
  .actions .solid, .actions .solid :global(svg) { background: var(--red); color: #fff; }
  .switch button { min-height: 2.5rem; font-size: 1.05rem; font-weight: 700; color: var(--muted); }
  .switch [aria-pressed="true"] { color: var(--text); }
  p { padding: .5rem .6rem; font-size: .9rem; line-height: 1.4; color: var(--muted); }
</style>
