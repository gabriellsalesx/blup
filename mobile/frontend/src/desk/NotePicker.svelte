<script lang="ts">
  import FileText from "@lucide/svelte/icons/file-text";
  import BackSheet from "./BackSheet.svelte";
  import { t } from "./i18n";

  // Pick a note: a search over every note of the vault, in the app's sheet.
  let { notes, title, onPick, onClose }: { notes: { name: string; path: string }[]; title?: string; onPick: (n: { name: string; path: string }) => void; onClose: () => void } = $props();
  const te = t.editor;
  let query = $state("");
  const found = $derived(notes.filter((n) => n.name !== "_index" && (n.name + " " + n.path).toLowerCase().includes(query.trim().toLowerCase())));
  const folder = (path: string) => path.split("/").slice(0, -1).join("/");
</script>

<BackSheet title={title ?? te.linkNote} tall onclose={onClose}>
  {#snippet top()}<input type="text" bind:value={query} placeholder={te.findNote} aria-label={te.findNote} />{/snippet}
  <div class="list">
    {#each found as n (n.path)}
      <button type="button" class="row" onclick={() => onPick(n)}>
        <span class="tile"><FileText size={20} /></span><span class="name">{n.name}</span>{#if folder(n.path)}<small>{folder(n.path)}</small>{/if}
      </button>
    {/each}
    {#if !found.length}<p>{t.side.nothing}</p>{/if}
  </div>
</BackSheet>

<style>
  .list { display: grid; }
  .row { display: flex; align-items: center; gap: .65rem; min-height: 3.1rem; text-align: start; }
  .tile { flex: none; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  small { flex: none; max-width: 40%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: .85rem; color: var(--muted); }
  p { padding-block: .5rem; color: var(--muted); }
</style>
