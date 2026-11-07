<script lang="ts">
  import { untrack } from "svelte";
  import { SiteParts, DeleteSite, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SiteParts as Parts, DeleteProblem } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import X from "@lucide/svelte/icons/x";
  import External from "@lucide/svelte/icons/external-link";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import { t } from "./i18n";

  // Deleting a site: a box for each part. Folders go to the system's trash;
  // what is online is deleted for good, so those boxes start unticked, but
  // the site and the notes on Blup Cloud start ticked (deleting the site
  // takes them down too, unless unticked). The name is typed to confirm.
  let { dir, onClose, onDone }: { dir: string; onClose: () => void; onDone: (dir: string) => void } = $props();
  const d = t.del;
  let parts = $state<Parts | null>(null);
  let pick = $state({ site: true, notes: false, published: false, blupSite: false, syncRepo: false, cloudNotes: false, feedback: false });
  let typed = $state("");
  let busy = $state(false);
  let error = $state("");
  let problems = $state<DeleteProblem[] | null>(null);
  const home = (p: string) => p.replace(/^\/home\/[^/]+|^\/Users\/[^/]+|^C:\\Users\\[^\\]+/, "~");

  // The site whose parts are listed: the one the dialog opened for.
  SiteParts(untrack(() => dir)).then((p) => { parts = p; pick.notes = p.notesOwned; pick.blupSite = !!p.blupSite; pick.cloudNotes = p.cloudNotes; }).catch((e) => (error = String(e)));
  const ready = $derived(!!parts && typed.trim() === parts.title.trim() && Object.values(pick).some(Boolean));

  async function run() {
    busy = true; error = "";
    try {
      problems = (await DeleteSite(dir, pick)) ?? [];
      if (!problems.length) onDone(dir);
    } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); } finally { busy = false; }
  }
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && !busy && onClose()} />
<div class="shade" role="presentation" onclick={() => !busy && onClose()}></div>
<div class="dialog" role="dialog" aria-modal="true" aria-label={d.title}>
  <header class="head">
    <h2>{parts ? d.titleName.replace("{name}", parts.title) : d.title}</h2>
    <button class="close" title={t.attach.close} aria-label={t.attach.close} onclick={onClose} disabled={busy}><X size={18} /></button>
  </header>
  {#if problems}
    <p>{d.doneWithProblems}</p>
    <ul class="problems">
      {#each problems as p (p.part)}
        <li><b>{p.part === "blupSite" ? t.aip.blupSite : d.parts[p.part as keyof typeof d.parts] ?? p.part}</b><small class="muted">{p.err}</small><button class="link" onclick={() => OpenURL(p.link)}>{d.byHand} <External size={13} /></button></li>
      {/each}
    </ul>
    <div class="actions"><button class="btn" onclick={() => onDone(dir)}>{d.ok}</button></div>
  {:else if parts}
    <p class="muted">{d.lead}</p>
    <div class="list">
      <label class="part"><input type="checkbox" bind:checked={pick.site} /><span><b>{d.parts.site}</b><small>{home(parts.siteDir)} · {d.toTrash}</small></span></label>
      <label class="part"><input type="checkbox" bind:checked={pick.notes} /><span><b>{d.parts.notes}</b><small>{home(parts.notesDir)} · {d.toTrash}</small>
        {#if !parts.notesOwned}<small class="warn">{d.notesWarn}</small>{/if}</span></label>
      {#if parts.published}<label class="part"><input type="checkbox" bind:checked={pick.published} /><span><b>{d.parts.published}</b><small>{parts.published} · <em>{d.forGood}</em></small></span></label>{/if}
      {#if parts.blupSite}<label class="part"><input type="checkbox" bind:checked={pick.blupSite} /><span><b>{t.aip.blupSite}</b><small>{parts.blupSite} · <em>{d.forGood}</em></small></span></label>{/if}
      {#if parts.syncRepo}<label class="part"><input type="checkbox" bind:checked={pick.syncRepo} /><span><b>{d.parts.syncRepo}</b><small>github.com/{parts.syncRepo} · <em>{d.forGood}</em></small></span></label>{/if}
      {#if parts.cloudNotes}<label class="part"><input type="checkbox" bind:checked={pick.cloudNotes} /><span><b>{d.parts.cloudNotes}</b><small>{d.cloudNotesHelp} · <em>{d.forGood}</em></small></span></label>{/if}
      {#if parts.feedback}<label class="part"><input type="checkbox" bind:checked={pick.feedback} /><span><b>{d.parts.feedback}</b><small>github.com/{parts.feedback} · <em>{d.forGood}</em></small></span></label>{/if}
    </div>
    <label class="field"><span>{d.confirm.replace("{name}", parts.title)}</span><input type="text" bind:value={typed} placeholder={parts.title} /></label>
    {#if error}<p class="bad">{error}</p>{/if}
    <div class="actions">
      <button class="btn ghost" onclick={onClose} disabled={busy}>{t.sec.cancel}</button>
      <button class="btn danger" disabled={!ready || busy} onclick={run}>{#if busy}<span class="spin"><Loader size={15} /></span>{/if}{d.delete}</button>
    </div>
  {:else if error}<p class="bad">{error}</p>{/if}
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 60; background: rgb(0 0 0 / .5); }
  .dialog { position: fixed; z-index: 61; inset: 0; margin: auto; width: min(32rem, calc(100% - 2rem)); height: fit-content; max-height: calc(100% - 2rem); overflow: auto; display: grid; gap: 1rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  .head { display: flex; align-items: center; gap: .75rem; }
  .head h2 { flex: 1; font-size: 1.05rem; font-weight: 700; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  p { font-size: .9rem; }
  .list { display: grid; border: 1px solid var(--line); border-radius: .6rem; }
  .part { display: flex; gap: .75rem; align-items: flex-start; padding: .7rem .9rem; cursor: pointer; }
  .part + .part { border-block-start: 1px solid var(--line); }
  .part input { margin-block-start: .2rem; accent-color: var(--red); }
  .part span { display: grid; gap: .15rem; min-width: 0; }
  .part small { font-size: .78rem; color: var(--muted); overflow-wrap: anywhere; }
  .part em { color: var(--red); font-style: normal; font-weight: 600; }
  .warn { color: #b45309 !important; }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; }
  .btn.danger { background: var(--red); color: #fff; }
  .problems { list-style: none; padding: 0; display: grid; gap: .6rem; }
  .problems li { display: grid; gap: .15rem; font-size: .9rem; }
  .link { justify-self: start; display: inline-flex; gap: .25rem; align-items: center; color: var(--blue); font-size: .85rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
