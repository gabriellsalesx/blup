<script lang="ts">
  import { PurgeHistory, SyncRemote, BlupAccount } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupAccount as Account } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import History from "@lucide/svelte/icons/history";
  import X from "@lucide/svelte/icons/x";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";
  import { bytes, afterPurge } from "./cloudhistory";

  // "Delete old history": the old versions of the open site's notes on
  // Blup Cloud go, the current version stays. Shown only when the notes
  // sync through Blup Cloud. The size shown is the account's (the cloud
  // gives no estimate per vault); what was freed comes from the answer.
  // compact: only the button (the storage banner says the rest).
  let { account = $bindable(), compact = false }: { account: Account | null; compact?: boolean } = $props();
  const h = t.hist;

  let synced = $state(false);
  let asking = $state(false);
  let busy = $state(false);
  let error = $state("");
  SyncRemote().then((r) => (synced = r === "blup")).catch(() => {});

  async function purge() {
    busy = true; error = "";
    try {
      const r = await PurgeHistory();
      if (account) account = afterPurge(account, r);
      asking = false;
      toast(r.freedBytes > 0 ? h.freed.replace("{size}", bytes(r.freedBytes)) : h.nothing);
      BlupAccount().then((a) => (account = a)).catch(() => {});
    } catch (e) {
      error = String(e).replace(/^(Runtime)?Error: /, "");
    } finally { busy = false; }
  }
</script>

{#if synced && account && compact}
  <button class="btn danger" onclick={() => { error = ""; asking = true; }}><History size={15} />{h.purge}</button>
{:else if synced && account}
  <div class="hist">
    <div class="row"><b>{h.title}</b><small class="muted">{account.historyBytes > 0 ? h.used.replace("{size}", bytes(account.historyBytes)) : h.none}</small></div>
    <div class="row">
      <small class="muted">{h.help}</small>
      <button class="btn danger" onclick={() => { error = ""; asking = true; }}><History size={15} />{h.purge}</button>
    </div>
  </div>
{/if}

<svelte:window onkeydown={(e) => asking && e.key === "Escape" && !busy && (asking = false)} />

{#if asking}
  <div class="shade" role="presentation" onclick={() => !busy && (asking = false)}></div>
  <div class="dialog" role="dialog" aria-modal="true" aria-label={h.confirmTitle}>
    <header class="head">
      <h2>{h.confirmTitle}</h2>
      <button class="close" title={t.attach.close} aria-label={t.attach.close} onclick={() => (asking = false)} disabled={busy}><X size={18} /></button>
    </header>
    <p>{h.confirm}</p>
    {#if error}<p class="bad">{error}</p>{/if}
    <div class="actions">
      <button class="btn ghost" onclick={() => (asking = false)} disabled={busy}>{h.cancel}</button>
      <button class="btn danger" disabled={busy} onclick={purge}>{#if busy}<span class="spin"><Loader size={15} /></span>{:else}<History size={15} />{/if}{h.yes}</button>
    </div>
  </div>
{/if}

<style>
  .hist { display: grid; gap: .5rem; padding-block: .85rem; border-block-end: 1px solid var(--line); }
  .row { display: flex; align-items: center; justify-content: space-between; gap: .75rem; flex-wrap: wrap; }
  .shade { position: fixed; inset: 0; z-index: 60; background: rgb(0 0 0 / .5); }
  .dialog { position: fixed; z-index: 61; inset: 0; margin: auto; width: min(30rem, calc(100% - 2rem)); height: fit-content; max-height: calc(100% - 2rem); overflow: auto; display: grid; gap: 1rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  .head { display: flex; align-items: center; gap: .75rem; }
  .head h2 { flex: 1; font-size: 1.05rem; font-weight: 700; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  p { font-size: .9rem; }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
