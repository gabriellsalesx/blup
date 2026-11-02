<script lang="ts" module>
  import type { Component } from "svelte";
  // An item of the menu: an action, a submenu (sub) or a line ("sep").
  export type MenuEntry = { label: string; Icon?: Component<{ size?: number | string }>; run?: () => void; sub?: MenuEntry[]; disabled?: boolean; hint?: string } | "sep";
</script>

<script lang="ts">
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Self from "./ContextMenu.svelte";

  // A right-click menu, like Obsidian's: icons, lines between groups and
  // submenus that open to the side on hover. It stays inside the window.
  let { x, y, items, onClose, nested = false }: { x: number; y: number; items: MenuEntry[]; onClose: () => void; nested?: boolean } = $props();

  let el = $state<HTMLDivElement>();
  let pos = $state({ x: 0, y: 0 });
  let openSub = $state(-1);
  let subAt = $state({ x: 0, y: 0 });
  $effect(() => {
    if (!el) return;
    const r = el.getBoundingClientRect();
    pos = { x: x + r.width > innerWidth - 8 ? (nested ? x - r.width - (el.parentElement?.getBoundingClientRect().width ?? 0) : innerWidth - r.width - 8) : x, y: Math.max(8, Math.min(y, innerHeight - r.height - 8)) };
  });
  function hover(i: number, e: MouseEvent) {
    const it = items[i];
    if (it === "sep" || !it.sub) { openSub = -1; return; }
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    subAt = { x: r.right + 2, y: r.top - 5 };
    openSub = i;
  }
</script>

<svelte:window onkeydown={(e) => !nested && e.key === "Escape" && onClose()} />
{#if !nested}<div class="catch" role="presentation" onmousedown={onClose} oncontextmenu={(e) => { e.preventDefault(); onClose(); }}></div>{/if}

<div class="ctx" bind:this={el} role="menu" tabindex="-1" style:left="{pos.x}px" style:top="{pos.y}px" onmousedown={(e) => e.preventDefault()}>
  {#each items as it, i (i)}
    {#if it === "sep"}<hr />
    {:else}
      <button role="menuitem" disabled={it.disabled} class:open={openSub === i} onmouseenter={(e) => hover(i, e)}
        onclick={() => { if (it.sub) return; it.run?.(); onClose(); }}>
        <span class="ico">{#if it.Icon}<it.Icon size={16} />{/if}</span>
        <span class="label">{it.label}</span>
        {#if it.hint}<kbd>{it.hint}</kbd>{/if}
        {#if it.sub}<ChevronRight size={15} />{/if}
      </button>
      {#if it.sub && openSub === i}
        <Self x={subAt.x} y={subAt.y} items={it.sub} {onClose} nested />
      {/if}
    {/if}
  {/each}
</div>

<style>
  .catch { position: fixed; inset: 0; z-index: 69; }
  .ctx { position: fixed; z-index: 70; display: grid; min-width: 14rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .3); }
  button { display: grid; grid-template-columns: 1.25rem 1fr auto auto; align-items: center; gap: .6rem; padding: .42rem .6rem; border-radius: .35rem; font-size: .875rem; text-align: start; white-space: nowrap; }
  button > :global(*) { pointer-events: none; }
  button:hover:not(:disabled), button.open { background: var(--soft); }
  button:disabled { opacity: .4; cursor: default; }
  .ico { display: inline-grid; place-items: center; color: var(--muted); }
  kbd { font-family: inherit; font-size: .75rem; color: var(--muted); }
  button :global(.lucide-chevron-right) { color: var(--muted); }
  hr { border: 0; border-block-start: 1px solid var(--line); margin: .25rem .2rem; }
</style>
