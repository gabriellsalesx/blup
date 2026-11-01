<script lang="ts">
  import { NoteLangs, TranslateNote } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { NoteLang } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { Events } from "@wailsio/runtime";
  import { toast } from "./toast.svelte";
  import { t } from "./i18n";
  import { useLocal, translateNoteLocal } from "./localtranslate";

  // The note's languages: the original and each translation (a folder en,
  // es or pt next to the note), to open, translate with AI, or translate
  // again when the original changed.
  let { note, onOpen, onSettings }: { note: string; onOpen: (path: string) => void; onSettings: (tab: string) => void } = $props();
  const tr = t.langs;
  const NAMES: Record<string, string> = { pt: "Português", en: "English", es: "Español" };

  let langs = $state<NoteLang[]>([]);
  let busy = $state("");
  let part = $state<{ done: number; total: number } | null>(null);
  let needKey = $state(false);

  const load = (rel: string) => NoteLangs(rel).then((l) => { if (rel === note) langs = l ?? []; }).catch(() => (langs = []));
  $effect(() => { load(note); });

  const off = Events.On("translate", (ev: any) => { if (busy && ev.data?.lang === busy) part = { done: ev.data.done, total: ev.data.total }; });
  $effect(() => () => off());

  async function translate(lang: string) {
    busy = lang;
    part = null;
    needKey = false;
    try {
      // On the computer when it is installed (and chosen), else by the AI
      // in the cloud.
      const original = langs.find((l) => l.original);
      if (original && (await useLocal())) {
        await translateNoteLocal(original.path, original.lang, lang, (done, total) => (part = { done, total }));
      } else await TranslateNote(note, lang);
      toast(tr.done.replace("{lang}", NAMES[lang]));
      await load(note);
    } catch (e) {
      const msg = String(e).replace(/^(Runtime)?Error: /, "");
      if (msg.includes("no OpenRouter key")) needKey = true;
      else toast(msg, { kind: "error" });
    }
    busy = "";
    part = null;
  }
  const label = (l: NoteLang) => (l.original ? tr.original : !l.exists ? tr.none : l.stale ? tr.stale : tr.ready);
</script>

{#if langs.length}
  <div class="field wide"><span>{tr.title}</span>
    <div class="note-langs">
      {#each langs as l (l.lang)}
        <div class="lang-row" class:here={l.path === note}>
          <div class="lang-text">
            <b>{NAMES[l.lang] ?? l.lang}</b>
            <small class:warn={l.stale} class:muted={!l.stale}>{busy === l.lang ? (part && part.total > 1 ? tr.part.replace("{done}", String(part.done + 1)).replace("{total}", String(part.total)) : tr.working) : label(l)}</small>
          </div>
          <span class="lang-acts">
            {#if l.exists && l.path !== note}<button class="btn ghost" onclick={() => onOpen(l.path)}>{tr.open}</button>{/if}
            {#if !l.original}
              <button class="btn ghost" disabled={!!busy} onclick={() => translate(l.lang)}>{l.exists ? tr.again : tr.translate}</button>
            {/if}
          </span>
        </div>
      {/each}
      {#if needKey}
        <p class="muted">{tr.needKey}</p>
        <button class="btn ghost" onclick={() => onSettings("ai")}>{tr.openSettings}</button>
      {:else}
        <small class="muted">{tr.help.replace("{folder}", langs.filter((l) => !l.original).map((l) => l.lang).join(", "))}</small>
      {/if}
    </div>
  </div>
{/if}

<style>
  .note-langs { display: grid; gap: .35rem; padding: .6rem .75rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
  .note-langs p { margin: 0; font-size: .85rem; }
  .note-langs > small { font-size: .78rem; line-height: 1.4; }
  .lang-row { display: flex; align-items: center; justify-content: space-between; gap: .75rem; padding: .3rem .4rem; border-radius: .45rem; }
  .lang-row.here { background: var(--soft); }
  .lang-text { display: grid; min-width: 0; font-size: .875rem; }
  .lang-text small { font-size: .78rem; }
  .lang-text .warn { color: var(--warn, #d97706); }
  .lang-acts { display: inline-flex; gap: .3rem; flex: none; }
  .lang-acts .btn { padding: .3rem .65rem; font-size: .8rem; }
</style>
