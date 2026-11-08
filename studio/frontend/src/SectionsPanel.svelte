<script lang="ts">
  import { AddSection, RemoveSection, RestoreSection, SaveHome } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Check from "@lucide/svelte/icons/check";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";

  let { settings, reload, onTree }: { settings: Settings; reload: () => void; onTree: (n: Node) => void } = $props();
  const s = saver();
  const label = (k: string) => (t.sec as Record<string, string>)[k];

  let asking = $state(""); // the section waiting for "remove?"
  let undo = $state<{ kind: string; trash: string } | null>(null);

  const add = (kind: string) => s.run(async () => { onTree(await AddSection(kind)); undo = null; reload(); }, t.toast.sectionAdded);
  const remove = (kind: string) => s.run(async () => {
    const r = await RemoveSection(kind);
    onTree(r.tree);
    undo = { kind, trash: r.trash };
    asking = "";
    reload();
  }, false); // the notice with Undo above says it
  // Bento: the section the home page leads to ("" shows the sections).
  // Follows the settings: a chosen section removed goes back to "".
  const home = $derived(settings.home ?? "");
  const homes = $derived((settings.sections ?? []).filter((sec) => sec.present && sec.folder));
  const saveHome = (folder: string) => s.run(async () => { await SaveHome(folder); reload(); });

  const restore = () => s.run(async () => {
    if (!undo) return;
    onTree(await RestoreSection(undo.trash));
    undo = null;
    reload();
  }, t.toast.sectionBack);
</script>

<div class="panel">
  <h1>{t.sec.title}</h1>
  <p class="lead">{settings.model === "bento" ? t.sec.lead : t.sec.leadBlog}</p>
  {#if undo}
    <div class="undo"><span>{t.sec.removed.replace("{name}", label(undo.kind))}</span><button class="btn ghost" disabled={s.state.busy} onclick={restore}>{t.sec.undo}</button></div>
  {/if}
  {#if settings.model === "bento" && homes.length}
    <section class="card">
      <label class="field"><span>{t.sec.home}</span>
        <select value={home} onchange={(e) => saveHome(e.currentTarget.value)} disabled={s.state.busy}>
          <option value="">{t.sec.homeSections}</option>
          {#each homes as sec (sec.folder)}<option value={sec.folder}>{sec.folder}</option>{/each}
        </select>
        <small>{t.sec.homeHelp}</small>
      </label>
    </section>
  {/if}
  {#each settings.sections ?? [] as sec (sec.kind)}
    <section class="card section">
      <div class="field"><span>{label(sec.kind)}</span><small>{label(sec.kind + "Help")}</small></div>
      {#if !sec.present}
        <button class="btn ghost" disabled={s.state.busy} onclick={() => add(sec.kind)}>+ {t.sec.add}</button>
      {:else if asking === sec.kind}
        <div class="confirm">
          <span>{t.sec.removeQ.replace("{folder}", sec.folder)}</span>
          <div class="row">
            <button class="btn ghost" onclick={() => (asking = "")}>{t.sec.cancel}</button>
            <button class="btn danger" disabled={s.state.busy} onclick={() => remove(sec.kind)}>{t.remove}</button>
          </div>
        </div>
      {:else}
        <div class="row">
          <span class="badge on"><Check size={12} aria-hidden="true" /> {t.sec.present} · {sec.folder}</span>
          <button class="btn ghost" onclick={() => (asking = sec.kind)}>{t.remove}</button>
        </div>
      {/if}
    </section>
  {/each}
  <div class="actions"><Saved state={s.state} /></div>
</div>

<style>
  .section { grid-template-columns: minmax(0, 1fr) auto; align-items: center; }
  .confirm { display: grid; gap: .5rem; justify-items: end; max-width: 20rem; text-align: end; font-size: .85rem; }
  .btn.danger { background: var(--red); color: #fff; }
  .undo { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .75rem 1rem; border-radius: .6rem; background: var(--soft); font-size: .9rem; }
</style>
