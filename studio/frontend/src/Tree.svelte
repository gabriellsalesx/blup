<script lang="ts">
  import Tree from "./Tree.svelte";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

  // The notes, like Obsidian's file explorer: folders open and close (open
  // is shared, so "collapse all" works), a vertical guide line inside each
  // open folder; clicking a folder also makes it the place for new notes.
  // Right-click opens the item's menu (onMenu). Canvases and drawings have
  // an icon; notes can be dragged onto a canvas.
  let { nodes, current, folder, open, onPick, onFolder, onMenu, depth = 0 }: {
    nodes: Node[]; current: string; folder: string; open: Record<string, boolean>;
    onPick: (p: string) => void; onFolder: (p: string) => void; onMenu: (n: Node, e: MouseEvent) => void; depth?: number;
  } = $props();
</script>

<ul class:nested={depth > 0}>
  {#each nodes as n (n.path)}
    <li>
      {#if n.dir}
        <button class="row" class:selected={n.path === folder} onclick={() => { open[n.path] = !open[n.path]; onFolder(n.path); }} oncontextmenu={(e) => { e.preventDefault(); onMenu(n, e); }}>
          <span class="chev" class:open={open[n.path]}><ChevronRight size={14} /></span><span class="name">{n.name}</span>
        </button>
        {#if open[n.path] && n.children}
          <Tree nodes={n.children} {current} {folder} {open} {onPick} {onFolder} {onMenu} depth={depth + 1} />
        {/if}
      {:else}
        <button class="row note" class:active={n.path === current} draggable="true" ondragstart={(e) => e.dataTransfer?.setData("application/x-blup-note", n.path)} onclick={() => onPick(n.path)} oncontextmenu={(e) => { e.preventDefault(); onMenu(n, e); }}>
          {#if n.kind === "drawing"}<span class="kind"><PenTool size={13} /></span>{:else if n.kind === "canvas"}<span class="kind"><LayoutDashboard size={13} /></span>{/if}<span class="name">{n.name}</span>
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
  .row.note { padding-inline-start: 1.55rem; color: var(--muted); opacity: 1; }
  .row.note:hover { color: var(--text); }
  .row.note.active, .row.selected { background: var(--soft); color: var(--text); opacity: 1; }
  .name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .kind { display: inline-grid; flex: none; margin-inline-start: -1.15rem; width: .85rem; color: var(--muted); }
  .chev { display: inline-grid; place-items: center; width: 1rem; flex: none; color: var(--muted); transition: transform .15s; }
  .chev.open { transform: rotate(90deg); }
</style>
