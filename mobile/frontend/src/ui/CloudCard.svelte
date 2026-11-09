<script lang="ts">
  import { untrack } from "svelte";
  import ShieldAlert from "@lucide/svelte/icons/shield-alert";
  import Compare from "@lucide/svelte/icons/git-compare-arrows";
  import { CloudService, type RemoteVault } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, bytes as size, message, when } from "../lib/app.svelte";
  import { fmt, plural, t } from "../desk/i18n";

  // The code being approved and the recovery key are the app's (app.login,
  // app.recovery): they stay when this sheet closes and opens again.
  let password = $state("");
  let repeat = $state("");
  let vaults = $state<RemoteVault[] | null>(null);
  let busy = $state(false);
  let error = $state("");
  const account = $derived(app.account);
  // Opened with no internet: the session is there (signed out is no
  // error), only the account could not be read. Not the same as signed out.
  const offline = $derived(!account && app.accountFailed);
  let retrying = $state(false);
  async function retry() {
    retrying = true;
    try { await app.retry(); } finally { retrying = false; }
  }

  async function run(work: () => Promise<unknown>) {
    busy = true;
    error = "";
    try {
      await work();
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }

  // Asked by more than one step at once (the password typed, the account
  // read again): one request answers them all.
  let loading: Promise<void> | null = null;
  const loadVaults = () => (loading ??= run(async () => (vaults = (await CloudService.Vaults()) ?? [])).finally(() => (loading = null)));

  async function login() {
    error = "";
    try {
      await app.startLogin();
    } catch (e) {
      console.error("blup cloud login", e);
      error = t.cloud.unreachable;
      return;
    }
    try {
      await app.waitLogin();
    } catch (e) {
      error = message(e);
    }
  }

  const create = () =>
    run(async () => {
      if ([...password].length < 8) throw new Error(t.cloud.short); // characters, as the Go side counts
      if (password !== repeat) throw new Error(t.cloud.differ);
      app.recovery = await CloudService.CreatePassword(password);
      password = repeat = "";
    });

  const unlock = () =>
    run(async () => {
      await CloudService.Unlock(password);
      password = "";
      await app.loadAccount();
      if (app.account?.vault) await app.sync();
      else await loadVaults();
    });

  async function saw() {
    app.recovery = "";
    await app.loadAccount();
    await loadVaults();
  }

  const link = (id: string, title: string) =>
    run(async () => {
      await CloudService.Link(id, title);
      await app.refresh();
      await app.loadAccount();
      await app.sync();
    });

  const signOut = () =>
    run(async () => {
      await CloudService.SignOut();
      await app.loadAccount();
    });

  const used = $derived(account?.storageLimit ? Math.min(100, (account.storageBytes / account.storageLimit) * 100) : 0);

  // Ready with no vault linked (now, or once the sign in or the password is
  // through, even when it ended with the sheet closed): the account's
  // vaults to pick from, asked once; asked again on coming back to the app
  // (or the network back) when the last try failed.
  let asked = -1; // the app.resumed it was asked at
  $effect(() => {
    const at = app.resumed;
    if (account?.notes !== "ready" || account.vault || app.recovery || asked === at || untrack(() => vaults) !== null) return;
    asked = at;
    untrack(loadVaults);
  });
</script>

<section>
  {#if !account?.signedIn}
  <div class="who">
    <img src="/blups/blup.svg" alt="" width="52" height="52" />
    <div>
      <b>Blup Cloud</b>
      <span>{offline ? t.cloud.offline : t.account.localOnly}</span>
    </div>
  </div>
  {/if}

  {#if app.recovery}
    <h3>{t.cloud.recoveryTitle}</h3>
    <p>{t.cloud.recoveryHelp}</p>
    <code>{app.recovery}</code>
    <button class="main" onclick={saw}>{t.cloud.kept}</button>
  {:else if app.login}
    <p>{t.cloud.checkCode}</p>
    <code>{app.login.code}</code>
    <p class="muted">{t.cloud.waiting}</p>
    <button class="plain" onclick={() => app.cancelLogin()}>{t.common.cancel}</button>
  {:else if offline}
    <p>{t.cloud.offlineLead}</p>
    <button class="main" disabled={retrying || app.syncing} onclick={retry}>{retrying || app.syncing ? t.common.syncing : t.common.syncNow}</button>
  {:else if !account?.signedIn}
    <p>{t.cloud.connectLead}</p>
    <button class="main" onclick={login}>{t.cloud.connect}</button>
  {:else if account.notes === "new"}
    <h3>{t.account.createPassword}</h3>
    <p>{t.cloud.newHelp}</p>
    <input type="password" aria-label={t.cloud.password} placeholder={t.cloud.password} autocomplete="new-password" bind:value={password} />
    <input type="password" aria-label={t.cloud.repeat} placeholder={t.cloud.repeat} autocomplete="new-password" bind:value={repeat} />
    <button class="main" disabled={busy} onclick={create}>{t.cloud.create}</button>
  {:else if account.notes === "locked"}
    <h3>{t.cloud.lockedTitle}</h3>
    <p>{t.cloud.lockedHelp}</p>
    <input type="password" aria-label={t.cloud.password} placeholder={t.cloud.secret} autocomplete="current-password" bind:value={password} />
    <button class="main" disabled={busy || !password} onclick={unlock}>{t.cloud.unlock}</button>
  {:else if !account.vault}
    <h3>{t.cloud.linkTitle}</h3>
    <p>{t.cloud.linkHelp}</p>
    {#each (vaults ?? []).filter((v) => !app.spaces.some((s) => s.remote === v.id)) as vault (vault.id)}
      <button class="plain" disabled={busy} onclick={() => link(vault.id, vault.title)}>{vault.title}</button>
    {/each}
    <button class="main" disabled={busy} onclick={() => link("", t.cloud.phoneVault)}>{t.cloud.newVault}</button>
  {:else}
    {#if account.storageLimit}
      <div class="space">
        <p><span>{t.cloud.space}</span><b>{fmt(t.common.ofTotal, { used: size(account.storageBytes), total: size(account.storageLimit) })}</b></p>
        <div class={["bar", { full: used >= 90 }]} role="progressbar" aria-label={t.cloud.spaceUsed} aria-valuenow={Math.round(used)} aria-valuemin={0} aria-valuemax={100}>
          <i style:width={`${Math.max(used, used > 0 ? 1.5 : 0)}%`}></i>
        </div>
      </div>
    {/if}
    <dl>
      {#if account.devicesMax}<dt>{t.cloud.devices}</dt><dd>{fmt(t.common.ofTotal, { used: account.devices, total: account.devicesMax })}</dd>{/if}
      <dt>{t.cloud.vault}</dt><dd>{account.title}</dd>
      <dt>{t.cloud.last}</dt><dd>{app.synced ? when(app.synced.at) : t.cloud.never}</dd>
      {#if app.synced}
        <dt>{t.cloud.sent}</dt><dd>{app.synced.sent}</dd>
        <dt>{t.cloud.received}</dt><dd>{app.synced.received}</dd>
      {/if}
    </dl>
    {#if app.synced?.conflicts?.length}
      <p class="bad">{plural(app.synced.conflicts.length, t.cloud.conflicts1, t.cloud.conflictsN)}</p>
    {/if}
    {#if app.syncStuck && !app.syncing}
      <div class="stuck" role="alert"><ShieldAlert size={20} /><p>{app.syncError}</p></div>
      <button class="plain icon" onclick={() => app.go({ panel: "fresh" })}><Compare size={18} />{t.cloud.fresh}</button>
    {/if}
    <button class="main" disabled={app.syncing} onclick={() => app.sync()}>{app.syncing ? t.common.syncing : t.common.syncNow}</button>
  {/if}

  {#if offline && !error}<p class="bad">{t.cloud.unreachable}</p>
  {:else if error || (app.syncError && !app.syncStuck)}<p class="bad">{error || app.syncError}</p>{/if}
  {#if account?.signedIn && !app.recovery}
    <button class="plain" disabled={busy} onclick={signOut}>{t.cloud.signOut}</button>
  {/if}
</section>

<style>
  section { display: grid; gap: .75rem; }
  .who { display: grid; grid-template-columns: auto 1fr; align-items: center; gap: .85rem; }
  .who div { display: grid; gap: .1rem; min-width: 0; }
  .who span { font-size: .85rem; color: var(--muted); overflow-wrap: anywhere; }
  h3 { font-size: .95rem; font-weight: 700; }
  p { font-size: .9rem; line-height: 1.4; }
  .muted { color: var(--muted); }
  .bad { color: var(--red); }
  code { padding: .7rem; border-radius: .5rem; background: var(--soft); font-family: ui-monospace, monospace; font-size: 1rem; font-weight: 600; text-align: center; overflow-wrap: anywhere; user-select: text; }
  .main, .plain { min-height: 3rem; padding-inline: 1rem; border-radius: .6rem; font-weight: 600; }
  .main { background: var(--blue); color: #fff; }
  .plain { border: 1px solid var(--line); }
  button:disabled { opacity: .5; }
  .icon { display: flex; align-items: center; justify-content: center; gap: .5rem; }
  .stuck { display: flex; gap: .65rem; align-items: flex-start; padding: .75rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--soft); }
  .stuck :global(svg) { flex-shrink: 0; color: var(--blue); }
  .space { display: grid; gap: .4rem; }
  .space p { display: flex; justify-content: space-between; gap: 1rem; font-size: .9rem; }
  .space span { color: var(--muted); }
  .space b { font-weight: 400; }
  .bar { height: .5rem; border-radius: 999px; overflow: hidden; background: var(--soft); }
  .bar i { display: block; height: 100%; border-radius: 999px; background: var(--blue); }
  .bar.full i { background: var(--red); }
  dl { display: grid; grid-template-columns: auto 1fr; gap: .4rem 1rem; font-size: .9rem; }
  dt { color: var(--muted); }
  dd { margin: 0; text-align: end; }
</style>
