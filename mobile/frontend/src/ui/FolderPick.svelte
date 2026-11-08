<script lang="ts">
  import { untrack } from "svelte";
  import Check from "@lucide/svelte/icons/check";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Folder from "@lucide/svelte/icons/folder";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import Plus from "@lucide/svelte/icons/plus";
  import X from "@lucide/svelte/icons/x";
  import { app, openOne } from "../lib/app.svelte";
  import Sheet from "./Sheet.svelte";
  import { fmt, t } from "../desk/i18n";

  // Picks the note's folder, or makes one (inside another, by its +). The
  // subfolders open by the arrow, as in the side menu.
  let { current, onpick, oncreate }: { current: string; onpick: (folder: string) => void; oncreate: (parent: string, name: string) => void } = $props();

  let query = $state("");
  let parent = $state("");
  let field = $state<HTMLInputElement>();
  // The note's own folder starts in sight.
  const upTo = (f: string) => f.split("/").slice(0, -1).map((_, i, all) => all.slice(0, i + 1).join("/"));
  let open = $state<string[]>(untrack(() => upTo(current)));

  const q = $derived(query.trim().toLowerCase());
  const tree = $derived([...app.folders].sort((a, b) => {
    const x = a.split("/"), y = b.split("/");
    for (let i = 0; i < Math.min(x.length, y.length); i++) {
      const order = x[i].localeCompare(y[i], t.locale);
      if (order) return order;
    }
    return x.length - y.length;
  }));
  const shown = $derived(q ? tree.filter((f) => f.toLowerCase().includes(q)) : tree.filter((f) => upTo(f).every((p) => open.includes(p))));
  const taken = $derived(app.folders.some((f) => f.toLowerCase() === (parent ? `${parent}/${q}` : q).toLowerCase()));
  const hasKids = (f: string) => app.folders.some((x) => x.startsWith(`${f}/`));
  const last = (f: string) => f.split("/").pop();
  const depth = (f: string) => (q ? 0 : f.split("/").length - 1);
  const toggle = (f: string) => (open = openOne(open, f));
  function inside(folder: string) {
    parent = folder;
    query = "";
    field?.focus();
  }
</script>

<Sheet title={t.folderPick.title} tall onclose={() => app.back()}>
  {#snippet top()}
    <input bind:this={field} type="text" bind:value={query} placeholder={t.common.searchOrCreate} aria-label={t.folderPick.search} />
    {#if parent}
      <button type="button" class="inside" aria-label={fmt(t.folderPick.outside, { name: parent })} onclick={() => (parent = "")}>{fmt(t.folderPick.inside, { name: parent })}<X size={15} /></button>
    {/if}
  {/snippet}
  {#if q && !taken}
    <button type="button" class="row" onclick={() => oncreate(parent, query)}><span class="tile"><Plus size={19} /></span><span class="name">{fmt(t.folderPick.newNamed, { name: query.trim() })}</span></button>
  {/if}
  <div class="list">
    {#if !q}
      <div class="line" style:--depth={0}><span class="arrow"></span><button type="button" class="row" onclick={() => onpick("")}><span class="tile"><FolderOpen size={19} /></span><span class="name">{t.folderPick.none}</span>{#if !current}<span class="mark"><Check size={20} /></span>{/if}</button></div>
    {/if}
    {#each shown as folder (folder)}
      <div class="line" style:--depth={depth(folder)}>
        {#if !q && hasKids(folder)}
          <button type="button" class={["arrow", { open: open.includes(folder) }]} aria-expanded={open.includes(folder)} aria-label={fmt(t.folderPick.subfolders, { name: last(folder) ?? "" })} onclick={() => toggle(folder)}><ChevronRight size={19} /></button>
        {:else if !q}<span class="arrow"></span>{/if}
        <button type="button" class="row" onclick={() => onpick(folder)}><span class="tile"><Folder size={19} /></span><span class="name">{q ? folder : last(folder)}</span>{#if folder === current}<span class="mark"><Check size={20} /></span>{/if}</button>
        <button type="button" class="sub" aria-label={fmt(t.folderPick.newSubIn, { name: folder })} onclick={() => inside(folder)}><Plus size={19} /></button>
      </div>
    {/each}
  </div>
</Sheet>

<style>
  .inside { justify-self: start; display: inline-flex; align-items: center; gap: .35rem; min-height: 2rem; padding-inline: .65rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); font-size: .85rem; font-weight: 500; }
  .list { display: grid; }
  .line { display: flex; align-items: center; padding-inline-start: calc(var(--depth) * .85rem); }
  .row { flex: 1; min-width: 0; display: flex; align-items: center; gap: .65rem; min-height: 3.1rem; text-align: start; }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .tile { flex: none; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .arrow { flex: none; display: grid; place-items: center; width: 1.7rem; height: 3rem; color: var(--muted); }
  .arrow :global(svg) { transition: rotate .15s; }
  .arrow.open :global(svg) { rotate: 90deg; }
  .mark { display: grid; color: var(--blue-text); }
  .sub { flex: none; display: grid; place-items: center; width: 2.75rem; height: 2.75rem; color: var(--muted); }
</style>
