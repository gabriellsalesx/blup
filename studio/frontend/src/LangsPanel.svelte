<script lang="ts">
  import { SiteLangs, TranslateSite, TranslationTodo, FollowTranslations } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SiteLangState, Settings } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { Events } from "@wailsio/runtime";
  import { toast, toastError } from "./toast.svelte";
  import { t } from "./i18n";
  import { useLocal, translateNoteLocal } from "./localtranslate";

  // The site in other languages: how much of it is translated into each,
  // and translating what is missing (or out of date) with AI.
  let { settings, onTree }: { settings: Settings; onTree: () => void } = $props();
  const sl = t.sitelangs;
  const NAMES: Record<string, string> = { pt: "Português", en: "English", es: "Español" };

  let langs = $state<SiteLangState[]>([]);
  let busy = $state("");
  let progress = $state<{ done: number; total: number; note: string } | null>(null);
  const load = () => SiteLangs().then((l) => (langs = l ?? [])).catch(toastError);
  load();

  const off = Events.On("translateSite", (ev: any) => { if (busy && ev.data?.lang === busy) progress = ev.data; });
  $effect(() => () => off());

  async function translate(lang: string) {
    busy = lang;
    progress = null;
    try {
      let n: number;
      if (await useLocal()) {
        // On the computer: note by note in the window.
        const todo = (await TranslationTodo(lang)) ?? [];
        for (const [i, rel] of todo.entries()) {
          progress = { done: i, total: todo.length, note: rel };
          await translateNoteLocal(rel, settings.lang, lang, undefined, false);
        }
        n = todo.length;
        await FollowTranslations();
      } else n = await TranslateSite(lang);
      toast(sl.finished.replace("{n}", String(n)).replace("{lang}", NAMES[lang]));
      onTree();
    } catch (e) { toastError(e); }
    busy = "";
    progress = null;
    load();
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
      {#if busy === l.lang && progress}
        <div class="lbar" role="progressbar" aria-valuemin="0" aria-valuemax={progress.total} aria-valuenow={progress.done}>
          <span style:width="{progress.total ? Math.max(3, (progress.done / progress.total) * 100) : 3}%"></span>
        </div>
        <small class="muted">{sl.working.replace("{done}", String(progress.done + 1)).replace("{total}", String(progress.total)).replace("{note}", noteName(progress.note))}</small>
      {:else if l.missing || l.stale}
        <div><button class="btn" disabled={!!busy} onclick={() => translate(l.lang)}>{l.missing ? sl.translate : sl.update}</button></div>
      {:else}
        <small class="muted">{sl.none}</small>
      {/if}
    </section>
  {/each}
  <small class="muted">{sl.help}</small>
</div>

<style>
  .lang-card { display: grid; gap: .6rem; }
  .counts { display: flex; flex-wrap: wrap; gap: .4rem 1rem; margin: 0; font-size: .875rem; }
  .counts .warn { color: var(--warn, #d97706); }
  .lbar { height: 6px; border-radius: 999px; background: var(--soft); overflow: hidden; }
  .lbar span { display: block; height: 100%; border-radius: inherit; background: var(--blue); transition: width .3s ease; }
</style>
