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
  let { nodes, current, folder, open, renaming = "", onPick, onFolder, onMenu, onRename, depth = 0 }: {
    nodes: Node[]; current: string; folder: string; open: Record<string, boolean>; renaming?: string;
    onPick: (p: string) => void; onFolder: (p: string) => void; onMenu: (n: Node, e: MouseEvent) => void;
    onRename: (n: Node, name: string | null) => void; depth?: number;
  } = $props();

  // Once per field: Enter and then the blur it causes count as one.
  let done = false;
  const start = (el: HTMLInputElement) => { done = false; el.focus(); el.select(); };
  function finish(n: Node, name: string | null) {
    if (done) return;
    done = true;
    onRename(n, name);
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
          <Tree nodes={n.children} {current} {folder} {open} {renaming} {onPick} {onFolder} {onMenu} {onRename} depth={depth + 1} />
        {/if}
      {:else if n.dir}
        <button class="row" class:selected={n.path === folder} onclick={() => { open[n.path] = !open[n.path]; onFolder(n.path); }} oncontextmenu={(e) => { e.preventDefault(); onMenu(n, e); }}>
          <span class="chev" class:open={open[n.path]}><ChevronRight size={14} /></span>{@render icon(n)}<span class="name">{n.name}</span>
        </button>
        {#if open[n.path] && n.children}
          <Tree nodes={n.children} {current} {folder} {open} {renaming} {onPick} {onFolder} {onMenu} {onRename} depth={depth + 1} />
        {/if}
      {:else}
        <button class="row note" class:active={n.path === current} draggable="true" ondragstart={(e) => e.dataTransfer?.setData("application/x-blup-note", n.path)} onclick={() => onPick(n.path)} oncontextmenu={(e) => { e.preventDefault(); onMenu(n, e); }}>
          {@render icon(n)}<span class="name">{n.name}</span>
        </button>
      {/if}
    </li>
  {/each}
</ul>

<style>
  ul { list-style: none; padding: 0; }
  /* The guide line of an open folder, under its chevron. */
  ul.nested { margin-inline-start: .95rem; padding-inline-start: .35rem; border-inline-start: 1px solid var(--line); }
  .row { display: flex; align-items: center; gap: .3rem; width: 100%; text-align: start; padding: .28rem .5rem .28rem .3rem; border-radius: .35rem; font-size: .875rem; color: var(--text); opacity: .85; }
  .row:hover { background: var(--soft); opacity: 1; }
  .row.note { color: var(--muted); opacity: 1; }
  .row.note:hover { color: var(--text); }
  .row.note.active, .row.selected { background: var(--soft); color: var(--text); opacity: 1; }
  .name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ico { display: inline-grid; place-items: center; width: 1rem; flex: none; color: var(--muted); }
  .rename { flex: 1; min-width: 0; padding: .1rem .35rem; margin-block: -.1rem; border: 1px solid var(--blue); border-radius: .3rem; outline: none; background: var(--panel); color: var(--text); font: inherit; }
  .chev { display: inline-grid; place-items: center; width: 1rem; flex: none; color: var(--muted); transition: transform .15s; }
  .chev.open { transform: rotate(90deg); }
</style>
