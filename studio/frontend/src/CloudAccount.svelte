<script lang="ts">
  import { Page } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { BlupAccount, BlupLogin, BlupLoginWait, BlupLoginCancel, BlupSignOut, CreateNotesPassword, UnlockNotes, OpenBlupCloud, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { BlupAccount as Account } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { Clipboard } from "@wailsio/runtime";
  import Lock from "@lucide/svelte/icons/lock";
  import Copy from "@lucide/svelte/icons/copy";
  import Check from "@lucide/svelte/icons/check";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";
  import TextField from "./ui/TextField.svelte";

  // Blup Cloud on this computer, step by step: signing in through the
  // browser, then the notes password (created here, or typed when it was
  // created on another computer). `account` is ready to sync when its
  // notes are "ready".
  // bare: the screen around already says what signing in is for (the
  // welcome), so only the button shows.
  // brief: the storage is shown around (the settings' usage), not here.
  // notes: false when only the account matters (publishing a site), not
  // the notes password.
  let { account = $bindable(null), bare = false, brief = false, notes = true }: { account?: Account | null; bare?: boolean; brief?: boolean; notes?: boolean } = $props();
  const c = t.cloud;

  let error = $state("");
  let busy = $state(false);
  let login = $state<{ code: string; url: string } | null>(null);
  let pw = $state("");
  let pw2 = $state("");
  let recovery = $state("");
  let kept = $state(false);
  let copied = $state(false);
  let secret = $state("");

  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  const size = (n: number) => (n < 1 << 30 ? `${Math.max(0, n / (1 << 20)).toFixed(n < 10 << 20 ? 1 : 0)} MB` : `${(n / (1 << 30)).toFixed(1)} GB`);

  async function load() {
    error = "";
    try { account = await BlupAccount(); } catch (e) { error = msg(e); }
  }
  load();

  async function signIn() {
    error = "";
    try {
      login = await BlupLogin();
      account = await BlupLoginWait();
    } catch (e) {
      if (!String(e).includes("canceled")) error = msg(e);
    } finally { login = null; }
  }
  function cancel() { BlupLoginCancel(); login = null; }
  // Offline, the reminder e-mails are cancelled when the Studio connects again.
  async function signOut() {
    if (await BlupSignOut()) toast(t.aip.signOutPending);
    account = await BlupAccount();
  }

  async function create(e: SubmitEvent) {
    e.preventDefault();
    error = "";
    if ([...pw].length < 8) return (error = c.short); // characters, as the Go side counts
    if (pw !== pw2) return (error = c.differ);
    busy = true;
    try { recovery = await CreateNotesPassword(pw); pw = pw2 = ""; } catch (e) {
      // Another computer may have created it first: the account is read
      // again (it then asks for that password), the message stays.
      const m = msg(e);
      await load();
      error = m;
    } finally { busy = false; }
  }
  async function copyRecovery() { await Clipboard.SetText(recovery); copied = true; setTimeout(() => (copied = false), 1500); }
  async function finish() { recovery = ""; kept = false; await load(); }

  async function unlock(e: SubmitEvent) {
    e.preventDefault();
    error = ""; busy = true;
    try { await UnlockNotes(secret); secret = ""; await load(); } catch (e) { error = msg(e); } finally { busy = false; }
  }
</script>

{#if account && !account.signedIn}
  {#if !bare}
    <h3>{c.signInTitle}</h3>
    <p>{c.signInHelp}</p>
  {/if}
  {#if login}
    <p class="muted">{c.waiting}</p>
    <p class="code">{login.code}</p>
    <div class="actions">
      <button class="btn ghost" onclick={() => OpenURL(login!.url)}>{c.reopen}</button>
      <button class="btn ghost" onclick={cancel}>{c.cancel}</button>
    </div>
  {:else}
    <div class="actions" class:wide={bare}><button class="btn" onclick={signIn}>{c.signIn}</button></div>
  {/if}
{:else if account}
  <div class="row who">
    <span>{c.signedAs} <b>{account.email || "Blup Cloud"}</b> · {c.plan} {account.label}</span>
    <span class="spacer"></span>
    <button class="btn ghost" onclick={signOut}>{c.signOut}</button>
  </div>
  {#if notes && recovery}
    <h3>{c.recoveryTitle}</h3>
    <p>{c.recoveryHelp}</p>
    <div class="row">
      <code class="recovery">{recovery}</code>
      <button class="btn ghost" onclick={copyRecovery}>{#if copied}<Check size={14} />{c.copied}{:else}<Copy size={14} />{c.copy}{/if}</button>
    </div>
    <label class="check"><input type="checkbox" bind:checked={kept} /> {c.kept}</label>
    <div class="actions"><button class="btn" disabled={!kept} onclick={finish}>{c.done}</button></div>
  {:else if notes && account.notes === "new"}
    <h3><Lock size={16} /> {c.newTitle}</h3>
    <p>{c.newHelp}</p>
    <form class="grid" onsubmit={create}>
      <TextField type="password" label={c.password} bind:value={pw} autocomplete="new-password" help={c.short} />
      <TextField type="password" label={c.confirm} bind:value={pw2} autocomplete="new-password" />
      <div class="actions"><button class="btn" disabled={busy || !pw || !pw2}>{c.create}</button></div>
    </form>
  {:else if notes && account.notes === "locked"}
    <h3><Lock size={16} /> {c.lockedTitle}</h3>
    <p>{c.lockedHelp}</p>
    <form class="grid" onsubmit={unlock}>
      <TextField type="password" label={c.secret} bind:value={secret} autocomplete="current-password" />
      <div class="actions"><button class="btn" disabled={busy || !secret.trim()}>{c.unlock}</button></div>
    </form>
  {:else if notes}
    <p class="muted">
      {account.live ? c.live : c.manual}
      {#if !brief}{c.storage.replace("{used}", size(account.storageBytes)).replace("{total}", size(account.storageMB * 1048576))}.{/if}
      {#if !account.live}<button class="link" onclick={() => OpenBlupCloud(Page.PagePlan)}>{c.upgrade}</button>{/if}
    </p>
  {/if}
{/if}
{#if error}<p class="bad">{error}</p>{/if}

<style>
  h3 { display: flex; align-items: center; gap: .4rem; font-size: 1rem; margin: .5rem 0 0; }
  .actions.wide { display: grid; justify-content: stretch; width: 100%; }
  .actions.wide .btn { width: 100%; justify-content: center; padding: .8rem 1.6rem; font-size: 1rem; }
  .code { font-family: ui-monospace, monospace; font-size: 1.8rem; font-weight: 700; letter-spacing: .2em; }
  .who { align-items: center; font-size: .9rem; }
  .spacer { flex: 1; }
  .grid { display: grid; gap: 1rem; max-width: 26rem; }
  .recovery { font-size: .95rem; user-select: all; padding: .5rem .75rem; border-radius: .5rem; background: var(--soft); word-break: break-all; }
  .check { display: flex; align-items: center; gap: .5rem; font-size: .9rem; }
  .link { color: var(--blue); text-decoration: underline; font-size: inherit; }
</style>
