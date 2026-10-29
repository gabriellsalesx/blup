<script lang="ts">
  import Tree from "./Tree.svelte";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

  let { nodes, current, onPick, depth = 0 }: { nodes: Node[]; current: string; onPick: (p: string) => void; depth?: number } = $props();
  // Folders start open at the top level.
  let open = $state<Record<string, boolean>>({});
  const isOpen = (n: Node) => open[n.path] ?? depth === 0;
</script>

<ul>
  {#each nodes as n (n.path)}
    <li>
      {#if n.dir}
        <button class="row folder" style:padding-inline-start="{0.5 + depth * 0.9}rem" onclick={() => (open[n.path] = !isOpen(n))}>
          <span class="chev" class:open={isOpen(n)}><ChevronRight size={14} /></span>{n.name}
        </button>
        {#if isOpen(n) && n.children}
          <Tree nodes={n.children} {current} {onPick} depth={depth + 1} />
        {/if}
      {:else}
        <button class="row note" class:active={n.path === current} style:padding-inline-start="{1.4 + depth * 0.9}rem" onclick={() => onPick(n.path)}>
          {n.name}
        </button>
      {/if}
    </li>
  {/each}
</ul>

<style>
  ul { list-style: none; padding: 0; }
  .row { display: flex; align-items: center; gap: .35rem; width: 100%; text-align: start; padding-block: .3rem; padding-inline-end: .5rem; border-radius: .375rem; font-size: .875rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .row:hover { background: var(--soft); }
  .folder { font-weight: 600; }
  .note { color: var(--muted); }
  .note.active { background: var(--soft); color: var(--text); font-weight: 600; }
  .chev { display: inline-grid; place-items: center; width: .9rem; color: var(--muted); transition: transform .15s; }
  .chev.open { transform: rotate(90deg); }
</style>
