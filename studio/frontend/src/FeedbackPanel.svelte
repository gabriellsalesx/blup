<script lang="ts">
  import { FeedbackPosts, SetFeedbackStatus, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { FeedbackPost } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { toast, toastError } from "./toast.svelte";
  import { t } from "./i18n";

  // The requests of the site's feedback board (Bento), read from GitHub:
  // the site's owner sets each one's status here (a label on GitHub; done
  // and declined close it), and the site shows it within a minute.
  const f = t.fbb;
  const STATUSES = ["new", "planned", "progress", "done", "declined"];
  const KINDS = ["feature", "bug", "improvement", "other"];

  let posts = $state<FeedbackPost[] | null>(null);
  let error = $state("");
  let kind = $state("");
  let saving = $state(0);

  async function load() {
    error = "";
    try { posts = (await FeedbackPosts()) ?? []; } catch (e) { error = String(e).replace(/^(Runtime)?Error: /, ""); posts = []; }
  }
  load();

  const shown = $derived((posts ?? []).filter((p) => !kind || p.kind === kind).sort((a, b) => b.votes - a.votes || b.number - a.number));

  async function setStatus(p: FeedbackPost, status: string) {
    const before = p.status;
    p.status = status;
    saving = p.number;
    try { await SetFeedbackStatus(p.number, status); toast(f.saved); } catch (e) { p.status = before; toastError(e); }
    saving = 0;
  }
</script>

<div class="panel">
  <h1>{f.title}</h1>
  <p class="lead">{f.lead}</p>
  <div class="fbp-bar">
    <div class="fbp-kinds" role="tablist">
      <button role="tab" aria-selected={kind === ""} class:on={kind === ""} onclick={() => (kind = "")}>{f.all} <b>{posts?.length ?? 0}</b></button>
      {#each KINDS as k (k)}
        <button role="tab" aria-selected={kind === k} class:on={kind === k} onclick={() => (kind = k)}><span class="fbp-dot k-{k}"></span>{f.kinds[k]} <b>{(posts ?? []).filter((p) => p.kind === k).length}</b></button>
      {/each}
    </div>
    <button class="btn ghost" onclick={load}>{f.refresh}</button>
  </div>
  {#if error}<p class="bad">{error}</p>{/if}
  {#if posts === null}
    <p class="muted">…</p>
  {:else if !shown.length}
    <p class="muted fbp-empty">{f.empty}</p>
  {:else}
    <ul class="fbp-list">
      {#each shown as p (p.number)}
        <li class="card fbp-item">
          <div class="fbp-votes"><b>{p.votes}</b><small>▲</small></div>
          <div class="fbp-main">
            <button class="fbp-title" onclick={() => OpenURL(p.url)} title={f.open}>{p.title}</button>
            <small class="muted"><span class="fbp-dot k-{p.kind}"></span>{f.kinds[p.kind] ?? p.kind} · {p.author} · {f.comments.replace("{n}", String(p.comments))}{p.lang ? ` · ${p.lang}` : ""}</small>
          </div>
          <select aria-label={f.status} value={p.status} disabled={saving === p.number} onchange={(e) => setStatus(p, e.currentTarget.value)}>
            {#each STATUSES as s (s)}<option value={s}>{f.statuses[s]}</option>{/each}
          </select>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .fbp-bar { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: .75rem; margin-block: 1rem; }
  .fbp-kinds { display: flex; flex-wrap: wrap; gap: .35rem; }
  .fbp-kinds button { display: inline-flex; align-items: center; gap: .35rem; padding: .3rem .7rem; border: 1px solid var(--line); border-radius: 999px; font-size: .85rem; color: var(--muted); }
  .fbp-kinds button.on { border-color: var(--blue); color: var(--text); background: var(--soft); }
  .fbp-kinds b { font-size: .75rem; }
  .fbp-dot { display: inline-block; width: .5rem; height: .5rem; margin-inline-end: .3rem; border-radius: 50%; background: #64748b; }
  .fbp-dot.k-feature { background: #22c55e; } .fbp-dot.k-bug { background: #ef4444; } .fbp-dot.k-improvement { background: #3b82f6; }
  .fbp-list { display: grid; gap: .5rem; margin: 0; padding: 0; list-style: none; }
  .fbp-item { display: flex; align-items: center; gap: 1rem; padding: .75rem 1rem; }
  .fbp-votes { display: grid; justify-items: center; min-width: 2.5rem; }
  .fbp-votes small { color: var(--muted); font-size: .65rem; }
  .fbp-main { display: grid; gap: .2rem; flex: 1; min-width: 0; }
  .fbp-title { text-align: start; font-weight: 600; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .fbp-title:hover { color: var(--blue); }
  .fbp-empty { padding-block: 2rem; text-align: center; }
  select { width: auto; }
</style>
