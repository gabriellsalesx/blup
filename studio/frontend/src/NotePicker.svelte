<script lang="ts">
  import X from "@lucide/svelte/icons/x";
  import { t } from "./i18n";

  // Pick a note to link to: a search over every note of the site.
  let { notes, title, onPick, onClose }: { notes: { name: string; path: string }[]; title?: string; onPick: (n: { name: string; path: string }) => void; onClose: () => void } = $props();
  const te = t.editor;
  let query = $state("");
  let pick = $state(0);
  const found = $derived(notes.filter((n) => n.name !== "_index" && (n.name + " " + n.path).toLowerCase().includes(query.trim().toLowerCase())).slice(0, 50));
  const focus = (el: HTMLInputElement) => el.focus();
  function key(e: KeyboardEvent) {
    if (e.key === "ArrowDown") { e.preventDefault(); pick = Math.min(pick + 1, found.length - 1); }
    else if (e.key === "ArrowUp") { e.preventDefault(); pick = Math.max(pick - 1, 0); }
    else if (e.key === "Enter" && found[pick]) { e.preventDefault(); onPick(found[pick]); }
    else if (e.key === "Escape") onClose();
  }
</script>

<div class="shade" role="presentation" onclick={onClose}></div>
<div class="picker" role="dialog" aria-modal="true" aria-label={title ?? te.linkNote}>
  <header><h2>{title ?? te.linkNote}</h2><button class="close" title={t.attach.close} onclick={onClose}><X size={18} /></button></header>
  <input type="text" bind:value={query} oninput={() => (pick = 0)} onkeydown={key} placeholder={te.findNote} use:focus />
  <ul>
    {#each found as n, i (n.path)}
      <li><button class:on={i === pick} onmouseenter={() => (pick = i)} onclick={() => onPick(n)}><b>{n.name}</b><small class="muted">{n.path.includes("/") ? n.path.slice(0, n.path.lastIndexOf("/")) : ""}</small></button></li>
    {/each}
    {#if !found.length}<li class="muted none">{t.side.nothing}</li>{/if}
  </ul>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 60; background: rgb(0 0 0 / .45); }
  .picker { position: fixed; z-index: 61; top: 50%; left: 50%; transform: translate(-50%, -50%); width: min(32rem, calc(100% - 2rem)); max-height: 70%; display: grid; grid-template-rows: auto auto minmax(0, 1fr); gap: .75rem; padding: 1.1rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  header { display: flex; align-items: center; }
  h2 { flex: 1; font-size: 1rem; font-weight: 700; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  ul { list-style: none; padding: 0; overflow: auto; display: grid; gap: .1rem; }
  li button { display: flex; align-items: baseline; gap: .6rem; width: 100%; padding: .45rem .6rem; border-radius: .4rem; text-align: start; font-size: .9rem; }
  li button.on { background: var(--soft); }
  small { font-size: .75rem; }
  .none { padding: .5rem; font-size: .85rem; }
</style>
