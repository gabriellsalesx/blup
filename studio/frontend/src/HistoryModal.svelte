<script lang="ts">
  import { NoteHistory, NoteVersionText, RestoreNoteVersion } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { NoteVersion } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";

  // The earlier versions of a note kept on Blup Cloud: pick one to read it,
  // and bring it back (it becomes the newest version; the one replaced
  // stays in the history).
  let { path, onClose }: { path: string; onClose: () => void } = $props();
  const h = t.history;
  const name = $derived(path.split("/").pop()?.replace(/\.md$/, "") ?? path);

  let versions = $state<NoteVersion[] | null>(null);
  let picked = $state<NoteVersion | null>(null);
  let text = $state<{ text: string; isText: boolean } | null>(null);
  let error = $state("");
  let busy = $state(false);

  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  const when = (d: string) => new Date(d).toLocaleString(t.lang === "pt" ? "pt-BR" : "en", { dateStyle: "medium", timeStyle: "short" });

  // One note per modal: the parent keys it by path.
  async function load() {
    try {
      versions = (await NoteHistory(path)) ?? [];
      if (versions.length) pick(versions[0]);
    } catch (e) { error = msg(e); }
  }
  load();

  async function pick(v: NoteVersion) {
    picked = v;
    text = null;
    try { text = await NoteVersionText(path, v.n); } catch (e) { error = msg(e); }
  }

  async function restore() {
    if (!picked) return;
    busy = true; error = "";
    try {
      await RestoreNoteVersion(path, picked.n);
      toast(h.restored);
      onClose();
    } catch (e) { error = msg(e); } finally { busy = false; }
  }
</script>

<div class="shade" role="presentation" onclick={onClose}></div>
<div class="dialog" role="dialog" aria-modal="true" aria-label={h.title.replace("{name}", name)}>
  <h2>{h.title.replace("{name}", name)}</h2>
  <p class="muted">{h.lead}</p>
  {#if versions === null && !error}
    <p class="muted">{h.loading}</p>
  {:else if versions && versions.length === 0}
    <p class="muted">{h.none}</p>
  {:else if versions}
    <div class="split">
      <ul>
        {#each versions as v (v.n)}
          <li><button class={{ on: picked?.n === v.n }} onclick={() => pick(v)}>{when(v.at)}{#if v.current} <span class="tag">{h.current}</span>{/if}</button></li>
        {/each}
      </ul>
      <div class="view">
        {#if text === null}<p class="muted">{h.loading}</p>
        {:else if text.isText}<pre>{text.text}</pre>
        {:else}<p class="muted">{h.notText}</p>{/if}
      </div>
    </div>
  {/if}
  {#if error}<p class="bad">{error}</p>{/if}
  <div class="actions">
    <button class="btn ghost" onclick={onClose}>{h.close}</button>
    {#if picked && !picked.current}<button class="btn" disabled={busy} onclick={restore}>{busy ? h.restoring : h.restore}</button>{/if}
  </div>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .35); }
  .dialog { position: fixed; z-index: 41; inset-block-start: 8%; inset-inline: 0; margin-inline: auto; width: min(52rem, calc(100% - 2rem)); display: grid; gap: .75rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); box-shadow: 0 20px 50px rgb(0 0 0 / .3); }
  h2 { font-size: 1rem; font-weight: 700; }
  p { font-size: .85rem; }
  .split { display: grid; grid-template-columns: 14rem 1fr; gap: .75rem; min-height: 0; }
  ul { list-style: none; padding: 0; max-height: 26rem; overflow: auto; }
  li button { display: flex; align-items: center; gap: .4rem; width: 100%; padding: .4rem .5rem; border-radius: .35rem; font-size: .85rem; text-align: start; }
  li button:hover { background: var(--soft); }
  li button.on { background: var(--soft); box-shadow: inset 2px 0 0 var(--blue); }
  .tag { font-size: .7rem; font-weight: 600; color: var(--blue); }
  .view { max-height: 26rem; overflow: auto; border: 1px solid var(--line); border-radius: .5rem; padding: .75rem; background: var(--bg); }
  pre { margin: 0; white-space: pre-wrap; word-break: break-word; font-size: .85rem; line-height: 1.5; }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; }
  @media (max-width: 640px) { .split { grid-template-columns: 1fr; } }
</style>
