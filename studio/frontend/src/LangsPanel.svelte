<script lang="ts">
  import { SiteLangs, RemoveTranslation, RestoreTranslation } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SiteLangState, Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { toast, toastError } from "./toast.svelte";
  import { t } from "./i18n";
  import { job, onJob, translateSite, LANG_NAMES as NAMES } from "./translatejob.svelte";
  import { onNotesChanged } from "./changes";

  // The site in other languages: how much of it is translated into each,
  // and translating what is missing (or out of date). The translation is
  // a job of the whole window (translatejob): it goes on when this screen
  // closes, and shows here again when it opens.
  let { settings, onTree }: { settings: Settings; onTree: () => void } = $props();
  const sl = t.sitelangs;

  let langs = $state<SiteLangState[]>([]);
  const load = () => SiteLangs().then((l) => (langs = l ?? [])).catch(toastError);
  load();
  // Notes or translations changed on disk (the sync, an agent, the
  // editor): the counts follow, quietly.
  $effect(() => onNotesChanged(() => { SiteLangs().then((l) => (langs = l ?? [])).catch(() => {}); }, 800));
  // While this screen is open, the counts follow the job note by note and
  // the file tree its end (opened later, it loads them anyway).
  $effect(() => onJob((what) => {
    load();
    if (what === "ended") onTree();
  }));
  // Taking a language off: asked first, here on its card; its folders go
  // to the Blup trash, and the notice brings them back.
  let asking = $state("");
  let removing = $state(false);
  async function remove(lang: string) {
    removing = true;
    try {
      const r = await RemoveTranslation(lang);
      asking = "";
      onTree();
      load();
      const name = NAMES[lang] ?? lang;
      toast(sl.removed.replace("{lang}", name), { action: { label: t.toast.undo, run: () => RestoreTranslation(lang, r.trash ?? []).then(() => { onTree(); load(); }).catch(toastError) } });
    } catch (e) {
      toastError(e);
    }
    removing = false;
  }
  const noteName = (p: string) => p.split("/").pop()?.replace(/\.md$/i, "") ?? "";
</script>

<div class="panel">
  <p class="muted">{sl.lead}</p>
  <p><b>{sl.main.replace("{lang}", NAMES[settings.lang] ?? settings.lang)}</b></p>
  {#each langs as l (l.lang)}
    <section class="card lang-card">
      <h2>{NAMES[l.lang] ?? l.lang}</h2>
      <p class="counts">
        <span>{sl.done.replace("{n}", String(l.done))}</span>
        {#if l.stale}<span class="warn">{sl.stale.replace("{n}", String(l.stale))}</span>{/if}
        {#if l.missing}<span class="muted">{sl.missing.replace("{n}", String(l.missing))}</span>{/if}
      </p>
      {#if job.running && job.lang === l.lang}
        <div class="lbar" role="progressbar" aria-valuemin="0" aria-valuemax={job.total} aria-valuenow={job.done}>
          <span style:width="{job.total ? Math.max(3, (job.done / job.total) * 100) : 3}%"></span>
        </div>
        {#if job.total}<small class="muted">{sl.working.replace("{done}", String(job.done + 1)).replace("{total}", String(job.total)).replace("{note}", noteName(job.note))}</small>{/if}
      {:else if l.missing || l.stale}
        <div><button class="btn" disabled={job.running} onclick={() => translateSite(l.lang, settings.lang)}>{l.missing ? sl.translate : sl.update}</button></div>
      {:else}
        <small class="muted">{sl.none}</small>
      {/if}
      {#if (l.done || l.stale || l.on) && !(job.running && job.lang === l.lang)}
        {#if asking === l.lang}
          <div class="confirm">
            <span>{sl.removeQ.replaceAll("{lang}", NAMES[l.lang] ?? l.lang).replace("{folder}", l.lang)}</span>
            <div class="row">
              <button class="btn ghost" onclick={() => (asking = "")}>{sl.removeCancel}</button>
              <button class="btn danger" disabled={removing} onclick={() => remove(l.lang)}>{sl.removeOk}</button>
            </div>
          </div>
        {:else}
          <div><button class="btn ghost" disabled={job.running} onclick={() => (asking = l.lang)}>{sl.removeBtn}</button></div>
        {/if}
      {/if}
    </section>
  {/each}
  <small class="muted">{sl.help}</small>
</div>

<style>
  .lang-card { display: grid; gap: .6rem; }
  .counts { display: flex; flex-wrap: wrap; gap: .4rem 1rem; margin: 0; font-size: .875rem; }
  .counts .warn { color: var(--warn, #d97706); }
  .confirm { display: grid; gap: .5rem; font-size: .85rem; }
  .row { display: flex; gap: .5rem; justify-content: flex-end; flex-wrap: wrap; }
  .btn.danger { background: var(--red); color: #fff; }
  .lbar { height: 6px; border-radius: 999px; background: var(--soft); overflow: hidden; }
  .lbar span { display: block; height: 100%; border-radius: inherit; background: var(--blue); transition: width .3s ease; }
</style>
