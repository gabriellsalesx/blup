<script lang="ts">
  import { ResetSync } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { SyncInfo } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import ShieldAlert from "@lucide/svelte/icons/shield-alert";
  import Compare from "@lucide/svelte/icons/git-compare-arrows";
  import X from "@lucide/svelte/icons/x";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";

  // A sync stopped for safety (Blup Cloud shows a version that does not
  // continue this device's): the notice, and "Compare from scratch". It is
  // only ever run by the person, after this confirmation: comparing from
  // scratch accepts the version the cloud shows now, so it is never done
  // without them knowing. Nothing is deleted on either side.
  let { done }: { done: (info: SyncInfo) => void } = $props();
  const p = t.pubs;

  let asking = $state(false);
  let busy = $state(false);
  let error = $state("");

  async function fresh() {
    busy = true; error = "";
    try {
      const info = await ResetSync();
      asking = false;
      toast(p.freshDone);
      done(info);
    } catch (e) {
      error = String(e).replace(/^(Runtime)?Error: /, "");
    } finally { busy = false; }
  }
</script>

<div class="stuck" role="alert">
  <ShieldAlert size={18} />
  <p>{p.stuck}</p>
</div>
<div class="row">
  <button class="btn" onclick={() => { error = ""; asking = true; }}><Compare size={15} />{p.fresh}</button>
</div>

<svelte:window onkeydown={(e) => asking && e.key === "Escape" && !busy && (asking = false)} />

{#if asking}
  <div class="shade" role="presentation" onclick={() => !busy && (asking = false)}></div>
  <div class="dialog" role="dialog" aria-modal="true" aria-label={p.freshTitle}>
    <header class="head">
      <h2>{p.freshTitle}</h2>
      <button class="close" title={t.attach.close} aria-label={t.attach.close} onclick={() => (asking = false)} disabled={busy}><X size={18} /></button>
    </header>
    <p>{p.freshConfirm}</p>
    {#if error}<p class="bad">{error}</p>{/if}
    <div class="actions">
      <button class="btn ghost" onclick={() => (asking = false)} disabled={busy}>{p.freshCancel}</button>
      <button class="btn" disabled={busy} onclick={fresh}>{#if busy}<span class="spin"><Loader size={15} /></span>{:else}<Compare size={15} />{/if}{p.fresh}</button>
    </div>
  </div>
{/if}

<style>
  .stuck { display: flex; align-items: flex-start; gap: .6rem; padding: .75rem 1rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--soft); color: var(--text); }
  .stuck :global(svg) { flex: none; margin-block-start: .1rem; color: var(--blue); }
  .stuck p { font-size: .9rem; }
  .row { display: flex; justify-content: flex-end; }
  .shade { position: fixed; inset: 0; z-index: 60; background: rgb(0 0 0 / .5); }
  .dialog { position: fixed; z-index: 61; top: 50%; left: 50%; transform: translate(-50%, -50%); width: min(30rem, calc(100% - 2rem)); max-height: calc(100% - 2rem); overflow: auto; display: grid; gap: 1rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  .head { display: flex; align-items: center; gap: .75rem; }
  .head h2 { flex: 1; font-size: 1.05rem; font-weight: 700; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  .dialog p { font-size: .9rem; }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
