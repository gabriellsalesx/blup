<script lang="ts" module>
  // The item being dragged, shared by every level of the tree.
  let dragged = $state("");
</script>

<script lang="ts">
  import Tree from "./Tree.svelte";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import FileIcon from "@lucide/svelte/icons/file";
  import Folder from "@lucide/svelte/icons/folder";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

  // The notes, like Obsidian's file explorer: folders open and close (open
  // is shared, so "collapse all" works), a vertical guide line inside each
  // open folder; clicking a folder also makes it the place for new notes.
  // Right-click opens the item's menu (onMenu). Each item has its icon (a
  // note, a folder open or closed, a canvas, a drawing); a note's sits
  // where a folder's chevron is. Notes can be dragged onto a canvas.
  //
  // Renaming is right there, like Obsidian: the name becomes a field, all
  // of it selected. Enter or clicking away keeps it, Esc gives up
  // (onRename gets null).
  //
  // Dragging (onMove): a note or a folder onto a folder goes inside it
  // (onto a note, into that note's folder); a folder held over for a
  // moment opens. The top of the vault is the
  // empty room under the tree (Workspace).
  //
  // Making a note or folder (creating): its name is typed in a field at the
  // top of the folder it goes in; onCreate gets it (null: given up).
  type Creating = { folder: string; kind: "note" | "folder"; placeholder: string } | null;
  let { nodes, current, folder, open, renaming = "", creating = null, parent = "", onPick, onFolder, onMenu, onRename, onCreate, onMove, depth = 0 }: {
    nodes: Node[]; current: string; folder: string; open: Record<string, boolean>; renaming?: string;
    creating?: Creating; parent?: string;
    onPick: (p: string) => void; onFolder: (p: string) => void; onMenu: (n: Node, e: MouseEvent) => void;
    onRename: (n: Node, name: string | null) => void;
    onCreate: (name: string | null) => void;
    onMove?: (path: string, folder: string) => void; // when items can be dragged
    depth?: number;
  } = $props();

  let over = $state(""); // the folder under what is dragged
  let opening: ReturnType<typeof setTimeout> | undefined;
  const parentOf = (p: string) => (p.includes("/") ? p.slice(0, p.lastIndexOf("/")) : "");
  // Not into itself, its own folders, or where it already is.
  const fits = (path: string, f: string) => !!path && f !== path && !f.startsWith(path + "/") && parentOf(path) !== f;

  function dragStart(e: DragEvent, n: Node) {
    if (!e.dataTransfer) return;
    if (!n.dir) e.dataTransfer.setData("application/x-blup-note", n.path); // onto a canvas
    if (!onMove) return;
    e.dataTransfer.setData("application/x-blup-item", n.path);
    e.dataTransfer.effectAllowed = "copyMove";
    dragged = n.path;
  }
  const target = (n: Node) => (n.dir ? n.path : parentOf(n.path));
  function dragOver(e: DragEvent, n: Node) {
    if (!onMove || !dragged) return;
    // Over a note: its folder (nothing to do when that is where it is).
    e.preventDefault();
    e.stopPropagation();
    const ok = fits(dragged, target(n));
    e.dataTransfer!.dropEffect = ok ? "move" : "none";
    if (!n.dir || over === n.path) return;
    over = ok ? n.path : "";
    clearTimeout(opening);
    if (ok) opening = setTimeout(() => (open[n.path] = true), 700);
  }
  function dragLeave(n: Node) {
    if (over === n.path) { over = ""; clearTimeout(opening); }
  }
  function drop(e: DragEvent, n: Node) {
    const path = dragged;
    over = ""; clearTimeout(opening);
    if (!onMove || !path) return;
    e.preventDefault();
    e.stopPropagation();
    dragged = "";
    if (fits(path, target(n))) onMove(path, target(n));
  }
  const dragEnd = () => { dragged = ""; over = ""; clearTimeout(opening); };

  // Once per field: Enter and then the blur it causes count as one.
  let done = false;
  const start = (el: HTMLInputElement) => { done = false; el.focus(); el.select(); };
  function finish(n: Node, name: string | null) {
    if (done) return;
    done = true;
    onRename(n, name);
  }
  function made(name: string | null) {
    if (done) return;
    done = true;
    onCreate(name?.trim() ? name : null);
  }
</script>

{#snippet icon(n: Node)}
  <span class="ico">
    {#if n.dir}{#if open[n.path]}<FolderOpen size={15} strokeWidth={1.75} />{:else}<Folder size={15} strokeWidth={1.75} />{/if}
    {:else if n.kind === "drawing"}<PenTool size={15} strokeWidth={1.75} />
    {:else if n.kind === "canvas"}<LayoutDashboard size={15} strokeWidth={1.75} />
    {:else}<FileIcon size={15} strokeWidth={1.75} />{/if}
  </span>
{/snippet}

<ul class:nested={depth > 0}>
  {#if creating && creating.folder === parent}
    <li>
      <div class="row" class:note={creating.kind === "note"}>
        {#if creating.kind === "folder"}<span class="chev"><ChevronRight size={14} /></span>{/if}
        <span class="ico">{#if creating.kind === "folder"}<Folder size={15} strokeWidth={1.75} />{:else}<FileIcon size={15} strokeWidth={1.75} />{/if}</span>
        <input class="rename" use:start placeholder={creating.placeholder} aria-label={creating.placeholder}
          onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); made(e.currentTarget.value); } else if (e.key === "Escape") { e.preventDefault(); made(null); } }}
          onblur={(e) => made(e.currentTarget.value)} />
      </div>
    </li>
  {/if}
  {#each nodes as n (n.path)}
    <li>
      {#if n.path === renaming}
        <div class="row" class:note={!n.dir}>
          {#if n.dir}<span class="chev" class:open={open[n.path]}><ChevronRight size={14} /></span>{/if}
          {@render icon(n)}
          <input class="rename" value={n.name} use:start aria-label={n.name}
            onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); finish(n, e.currentTarget.value); } else if (e.key === "Escape") { e.preventDefault(); finish(n, null); } }}
            onblur={(e) => finish(n, e.currentTarget.value)} />
        </div>
        {#if n.dir && open[n.path] && n.children}
          <Tree nodes={n.children} {current} {folder} {open} {renaming} {creating} parent={n.path} {onPick} {onFolder} {onMenu} {onRename} {onCreate} {onMove} depth={depth + 1} />
        {/if}
      {:else if n.dir}
        <button class={["row", { selected: n.path === folder, target: over === n.path }]} draggable={!!onMove} ondragstart={(e) => dragStart(e, n)} ondragend={dragEnd}
          ondragover={(e) => dragOver(e, n)} ondragleave={() => dragLeave(n)} ondrop={(e) => drop(e, n)} onclick={() => { open[n.path] = !open[n.path]; onFolder(n.path); }} oncontextmenu={(e) => { e.preventDefault(); onMenu(n, e); }}>
          <span class="chev" class:open={open[n.path]}><ChevronRight size={14} /></span>{@render icon(n)}<span class="name">{n.name}</span>
        </button>
        {#if open[n.path] && n.children}
          <Tree nodes={n.children} {current} {folder} {open} {renaming} {creating} parent={n.path} {onPick} {onFolder} {onMenu} {onRename} {onCreate} {onMove} depth={depth + 1} />
        {/if}
      {:else}
        <button class="row note" class:active={n.path === current} draggable="true" ondragstart={(e) => dragStart(e, n)} ondragend={dragEnd} ondragover={(e) => dragOver(e, n)} ondrop={(e) => drop(e, n)} onclick={() => onPick(n.path)} oncontextmenu={(e) => { e.preventDefault(); onMenu(n, e); }}>
          {@render icon(n)}<span class="name">{n.name}</span>
        </button>
      {/if}
    </li>
  {/each}
</ul>

<style>
  /* A little room between the items, so their highlights never touch. */
  ul { list-style: none; padding: 0; display: grid; gap: 3px; }
  /* The guide line of an open folder, under its chevron. */
  ul.nested { margin-block-start: 3px; margin-inline-start: .95rem; padding-inline-start: .35rem; border-inline-start: 1px solid var(--line); }
  .row { display: flex; align-items: center; gap: .35rem; width: 100%; min-height: 1.9rem; text-align: start; padding: .3rem .5rem .3rem .3rem; border-radius: .35rem; font-size: .875rem; color: var(--text); opacity: .85; }
  .row:hover { background: var(--soft); opacity: 1; }
  .row.note { color: var(--muted); opacity: 1; }
  .row.note:hover { color: var(--text); }
  .row.note.active, .row.selected { background: var(--soft); color: var(--text); opacity: 1; }
  .name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ico { display: inline-grid; place-items: center; width: 1rem; flex: none; color: var(--muted); }
  .rename { flex: 1; min-width: 0; padding: .1rem .35rem; margin-block: -.1rem; border: 1px solid var(--blue); border-radius: .3rem; outline: none; background: var(--panel); color: var(--text); font: inherit; }
  .row.target { background: color-mix(in srgb, var(--blue) 18%, transparent); outline: 1px solid var(--blue); opacity: 1; }
  .chev { display: inline-grid; place-items: center; width: 1rem; flex: none; color: var(--muted); transition: transform .15s; }
  .chev.open { transform: rotate(90deg); }
</style>
