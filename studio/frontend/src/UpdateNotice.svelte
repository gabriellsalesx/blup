<script lang="ts">
  import { onMount } from "svelte";
  import { fly } from "svelte/transition";
  import X from "@lucide/svelte/icons/x";
  import { CheckUpdate, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { UpdateInfo } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { t } from "./i18n";
  import { updateJob, downloadUpdate, installUpdate, updatePercent, downloadPage, CHANGELOG } from "./updatejob.svelte";
  import { readNotes, type Span } from "./releasenotes";

  // A newer Blup Studio, told when the Studio opens: a small card in the
  // corner, never in the way. GitHub is asked at most once a day (Go keeps
  // the day's answer), a few seconds after opening so the opening is not
  // slowed. "Later" closes it until the next time the Studio opens: it comes
  // back every time until the update is done. Where a store or the
  // system's packages update the Studio (Microsoft Store, Flatpak, AUR),
  // Go answers with updatesBy and nothing shows. What changed comes inside
  // it: every version after this one up to the latest, in the Studio's
  // language (Go picks the section), read as plain blocks (releasenotes.ts).
  const s = t.prefs;
  const v = (x: string) => (/^\d/.test(x) ? `v${x}` : x);
  let info = $state<UpdateInfo | null>(null);
  let later = $state(false);
  const shown = $derived(!!info?.newer && !info.updatesBy && !later);
  const changes = $derived((info?.changes ?? []).map((c) => ({ version: v(c.version), blocks: readNotes(c.notes) })).filter((c) => c.blocks.length));

  onMount(() => {
    const wait = setTimeout(() => {
      CheckUpdate(false).then((u) => (info = u)).catch(() => {});
    }, 3000);
    return () => clearTimeout(wait);
  });

  function update() {
    if (!info) return;
    if (!info.selfUpdate) OpenURL(downloadPage());
    else if (updateJob.phase === "ready") installUpdate();
    else downloadUpdate();
  }
</script>

{#snippet text(spans: Span[])}{#each spans as sp, i (i)}{#if sp.code}<code>{sp.text}</code>{:else if sp.bold}<b>{sp.text}</b>{:else}{sp.text}{/if}{/each}{/snippet}

{#if shown && info}
  <div class="notice" role="dialog" aria-labelledby="update-title" transition:fly={{ y: 16, duration: 180 }}>
    <button class="close" aria-label={s.noticeLater} title={s.noticeLater} onclick={() => (later = true)}><X size={16} /></button>
    <img src="/scenes/atualizacao-disponivel.svg" alt="" />
    <h2 id="update-title">{s.noticeTitle}</h2>
    <p>{s.noticeLead.replace("{v}", v(info.latest)).replace("{current}", v(info.current))}</p>
    {#if changes.length}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex: a scrolling region takes the keyboard (WCAG 2.1.1) -->
      <section class="changes" aria-label={s.noticeChanges} tabindex="0">
        {#each changes as c (c.version)}
          {#if changes.length > 1}<h3>{c.version}</h3>{/if}
          {#each c.blocks as b, i (i)}
            {#if b.kind === "heading"}<h4>{@render text(b.spans)}</h4>
            {:else if b.kind === "item"}<p class="item">{@render text(b.spans)}</p>
            {:else}<p>{@render text(b.spans)}</p>{/if}
          {/each}
        {/each}
      </section>
    {/if}
    {#if updateJob.phase === "downloading"}
      <div class="dl" role="progressbar" aria-label={s.updateDownloading} aria-valuemin="0" aria-valuemax="100" aria-valuenow={updatePercent()}><span style:width={`${Math.max(2, updatePercent())}%`}></span></div>
      <small>{s.updateDownloading}</small>
    {:else if updateJob.phase === "ready"}
      <small>{s.updateReadyHelp}</small>
    {:else if updateJob.phase === "failed"}
      <small class="bad">{s.updateDlFail}</small>
    {/if}
    <div class="actions">
      {#if updateJob.phase === "failed"}
        <button class="btn" onclick={() => OpenURL(downloadPage())}>{s.updateGet}</button>
      {:else}
        <button class="btn" disabled={updateJob.phase === "downloading" || updateJob.phase === "installing"} onclick={update}>
          {updateJob.phase === "ready" ? s.updateRestart : updateJob.phase === "installing" ? s.updateInstalling : s.updateNow}
        </button>
      {/if}
      <button class="btn ghost" onclick={() => (later = true)}>{s.noticeLater}</button>
    </div>
    <button class="all" onclick={() => OpenURL(CHANGELOG)}>{s.noticeAll}</button>
  </div>
{/if}

<style>
  .notice { position: fixed; inset-inline-end: 1rem; inset-block-end: 1rem; z-index: 80; display: grid; gap: .5rem; width: min(24rem, calc(100vw - 2rem)); padding: 1rem 1.1rem 1.1rem; border: 1px solid var(--line); border-radius: .9rem; background: var(--panel); box-shadow: 0 14px 40px rgb(0 0 0 / .22); }
  .close { position: absolute; inset-block-start: .5rem; inset-inline-end: .5rem; display: grid; place-items: center; width: 1.8rem; height: 1.8rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  img { justify-self: center; width: 9rem; height: auto; }
  h2 { margin: 0; font-size: 1rem; font-weight: 700; }
  p { margin: 0; font-size: .875rem; color: var(--muted); }
  small { font-size: .78rem; color: var(--muted); }
  small.bad { color: var(--red); }
  .actions { display: flex; flex-wrap: wrap; gap: .5rem; margin-block-start: .35rem; }
  /* What changed: its own scroll, so a long list never pushes the buttons
     off the window. */
  .changes { max-height: min(15rem, 40vh); overflow-y: auto; overscroll-behavior: contain; padding: .5rem .75rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--soft); font-size: .84rem; line-height: 1.45; }
  .changes h3 { margin: .4rem 0 .15rem; font-size: .8rem; font-weight: 700; color: var(--muted); }
  .changes h3:first-child, .changes h4:first-child { margin-block-start: 0; }
  .changes h4 { margin: .5rem 0 .2rem; font-size: .86rem; font-weight: 700; }
  .changes p { margin: .2rem 0; color: var(--text); }
  .changes p.item { position: relative; padding-inline-start: .9rem; }
  .changes p.item::before { content: ""; position: absolute; inset-inline-start: .2rem; inset-block-start: .6em; width: .3rem; height: .3rem; border-radius: 50%; background: var(--muted); }
  .changes code { padding: 0 .25rem; border-radius: .25rem; background: var(--panel); font-size: .92em; }
  .all { justify-self: start; padding: 0; font-size: .78rem; color: var(--muted); text-decoration: underline; text-underline-offset: 2px; }
  .all:hover { color: var(--text); }
  .dl { height: .35rem; border-radius: 1rem; background: var(--soft); overflow: hidden; }
  .dl span { display: block; height: 100%; background: var(--accent); }
</style>
