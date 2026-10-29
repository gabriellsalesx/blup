<script lang="ts">
  import { Trash, RestoreSection, DeleteFromTrash, EmptyTrash } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { TrashItem } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Folder from "@lucide/svelte/icons/folder";
  import FileText from "@lucide/svelte/icons/file-text";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";

  let { reload, onTree }: { reload: () => void; onTree: (n: Node) => void } = $props();
  const tr = t.trash;
  let items = $state<TrashItem[]>([]);
  let asking = $state(""); // path waiting for "delete for good?"
  let askingAll = $state(false);
  const s = saver();

  const load = async () => (items = (await Trash()) ?? []);
  load();
  const when = (d: string) => new Date(d).toLocaleString(t.lang === "pt" ? "pt-BR" : "en", { dateStyle: "medium", timeStyle: "short" });

  const restore = (p: string) => s.run(async () => { onTree(await RestoreSection(p)); await load(); reload(); }, t.toast.restored);
  const drop = (p: string) => s.run(async () => { await DeleteFromTrash(p); asking = ""; await load(); }, t.toast.deletedForGood);
  const empty = () => s.run(async () => { await EmptyTrash(); askingAll = false; await load(); }, t.toast.trashEmptied);
</script>

<div class="panel">
  <h1>{tr.title}</h1>
  <p class="lead">{tr.lead}</p>
  {#if items.length === 0}
    <p class="empty muted">{tr.empty}</p>
  {:else}
    {#each items as it (it.path)}
      <section class="card item">
        <span class="ico">{#if it.dir}<Folder size={20} />{:else}<FileText size={20} />{/if}</span>
        <div class="field"><span>{it.name}</span><small>{tr.removedAt} {when(it.removed)}{it.dir ? " · " + tr.notes.replace("{n}", String(it.notes)) : ""}</small><small>{tr.from} {it.origin.includes("/") ? it.origin.slice(0, it.origin.lastIndexOf("/")).replaceAll("/", " / ") : t.menu.top}</small></div>
        {#if asking === it.path}
          <div class="row">
            <span class="q">{tr.deleteQ}</span>
            <button class="btn ghost" onclick={() => (asking = "")}>{t.sec.cancel}</button>
            <button class="btn danger" disabled={s.state.busy} onclick={() => drop(it.path)}>{tr.delete}</button>
          </div>
        {:else}
          <div class="row">
            <button class="btn ghost" disabled={s.state.busy} onclick={() => restore(it.path)}>{tr.restore}</button>
            <button class="btn ghost" onclick={() => (asking = it.path)}>{tr.delete}</button>
          </div>
        {/if}
      </section>
    {/each}
    <div class="actions">
      <Saved state={s.state} />
      {#if askingAll}
        <span class="q">{tr.emptyQ}</span>
        <button class="btn ghost" onclick={() => (askingAll = false)}>{t.sec.cancel}</button>
        <button class="btn danger" disabled={s.state.busy} onclick={empty}>{tr.emptyAll}</button>
      {:else}
        <button class="btn ghost" onclick={() => (askingAll = true)}>{tr.emptyAll}</button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .item { grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; }
  .ico { color: var(--muted); display: inline-grid; }
  .empty { padding: 2rem; border: 1px dashed var(--line); border-radius: .75rem; text-align: center; }
  .q { font-size: .85rem; color: var(--muted); }
  .btn.danger { background: var(--red); color: #fff; }
</style>
