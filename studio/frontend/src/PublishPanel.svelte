<script lang="ts">
  import { onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { Connections, OpenKeyPage, OpenURL, ConnectGitHub, ConnectCloudflare, Disconnect, CloudflareAccounts, Target, SetTarget, Publish, SyncStatus, Sync } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Connections as Conn, Target as Dest, SyncInfo } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Account } from "../bindings/github.com/gabriellsalesx/blup/internal/deploy/cfpages/models";
  import Check from "@lucide/svelte/icons/check";
  import Alert from "@lucide/svelte/icons/circle-alert";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import External from "@lucide/svelte/icons/external-link";
  import Refresh from "@lucide/svelte/icons/refresh-cw";
  import { t } from "./i18n";

  let { autostart = "" }: { autostart?: string } = $props();
  const p = t.pubs;

  let conn = $state<Conn | null>(null);
  let target = $state<Dest | null>(null);
  let accounts = $state<Account[]>([]);
  let key = $state(""); // the key being pasted
  let connecting = $state(false);
  let error = $state("");

  type Line = { kind: string; text: string };
  let log = $state<Line[]>([]);
  let publishing = $state(false);
  let live = $state("");

  let sync = $state<SyncInfo | null>(null);
  let syncing = $state(false);
  let syncError = $state("");

  const off = Events.On("progress", (ev: any) => { log.push(ev.data as Line); });
  onDestroy(() => off());

  async function load() {
    error = "";
    try {
      [conn, target] = await Promise.all([Connections(), Target()]);
      live = target?.url ?? "";
      if (conn?.cloudflare && target?.provider === "cloudflare") accounts = (await CloudflareAccounts().catch(() => [])) ?? [];
    } catch (e) { error = msg(e); }
  }
  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  const connected = $derived(target?.provider === "cloudflare" ? !!conn?.cloudflare : !!conn?.github);
  const ready = $derived(!!target?.provider && connected && (target.provider === "github" ? !!target.repo : !!target.account && !!target.project));

  async function pick(provider: string) {
    if (!target) return;
    target.provider = provider;
    if (provider === "cloudflare" && conn?.cloudflare && !accounts.length) accounts = (await CloudflareAccounts().catch(() => [])) ?? [];
    if (provider === "cloudflare" && !target.account && accounts.length) target.account = accounts[0].id;
  }

  async function connect() {
    if (!target || !key.trim()) return;
    connecting = true; error = "";
    try {
      if (target.provider === "cloudflare") {
        accounts = (await ConnectCloudflare(key)) ?? [];
        if (accounts.length) target.account = accounts[0].id;
      } else {
        await ConnectGitHub(key);
      }
      key = "";
      conn = await Connections();
    } catch (e) { error = msg(e); } finally { connecting = false; }
  }

  async function disconnect() {
    if (!target) return;
    await Disconnect(target.provider);
    conn = await Connections();
  }

  async function publish() {
    if (!target) return;
    error = ""; log = []; live = ""; publishing = true;
    try {
      target = await SetTarget(target);
      live = await Publish();
    } catch (e) { error = msg(e); } finally { publishing = false; }
  }

  let ghKey = $state("");
  async function connectGitHubForSync() {
    connecting = true; syncError = "";
    try { await ConnectGitHub(ghKey); ghKey = ""; conn = await Connections(); await checkSync(); }
    catch (e) { syncError = msg(e); } finally { connecting = false; }
  }

  async function checkSync() {
    syncError = ""; syncing = true;
    try { sync = await SyncStatus(); } catch (e) { syncError = msg(e); } finally { syncing = false; }
  }
  async function runSync() {
    syncError = ""; syncing = true;
    try { sync = await Sync(); } catch (e) { syncError = msg(e); } finally { syncing = false; }
  }

  load().then(() => {
    if (autostart === "publish" && ready) publish();
    if (autostart === "sync" && conn?.github) checkSync();
  });
</script>

<div class="panel">
  <h1>{p.title}</h1>
  <p class="lead">{p.lead}</p>
  {#if error}<p class="bad box">{error}</p>{/if}

  {#if target && conn}
    <section class="card">
      <h2>{p.where}</h2>
      <div class="choices">
        <button class="choice" class:on={target.provider === "cloudflare"} onclick={() => pick("cloudflare")}>
          <b>Cloudflare Pages</b><span>{p.cfHelp}</span>
        </button>
        <button class="choice" class:on={target.provider === "github"} onclick={() => pick("github")}>
          <b>GitHub Pages</b><span>{p.ghHelp}</span>
        </button>
      </div>
    </section>

    {#if target.provider}
      <section class="card">
        <h2>{target.provider === "cloudflare" ? p.cfAccount : p.ghAccount}</h2>
        {#if connected}
          <div class="row status">
            <span class="badge on"><Check size={13} />{target.provider === "cloudflare" ? p.connected : p.connectedAs + " " + conn.github}</span>
            <span class="spacer"></span>
            <button class="btn ghost" onclick={disconnect}>{p.disconnect}</button>
          </div>
          {#if conn.githubError && target.provider === "github"}<p class="bad">{conn.githubError}</p>{/if}
          {#if target.provider === "cloudflare"}
            {#if accounts.length > 1}
              <label class="field"><span>{p.account}</span>
                <select bind:value={target.account}>{#each accounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}</select>
              </label>
            {/if}
            <label class="field"><span>{p.project}</span><input type="text" bind:value={target.project} /><small>{p.projectHelp}</small></label>
          {:else}
            <label class="field"><span>{p.repo}</span><input type="text" bind:value={target.repo} /><small>{p.repoHelp}</small></label>
          {/if}
        {:else}
          <ol class="steps">
            <li>{p.step1} <button class="add" onclick={() => OpenKeyPage(target!.provider)}>{p.openForm} <External size={14} /></button></li>
            <li>{target.provider === "cloudflare" ? p.step2cf : p.step2gh}</li>
            <li>{p.step3}</li>
          </ol>
          <div class="row">
            <input type="password" bind:value={key} placeholder={p.pasteKey} autocomplete="off" />
            <button class="btn" disabled={connecting || !key.trim()} onclick={connect}>{connecting ? p.checking : p.connect}</button>
          </div>
        {/if}
      </section>

      <section class="card publish">
        <div class="row">
          <div class="field"><span>{p.now}</span>{#if live}<small>{p.liveAt} <button class="link" onclick={() => OpenURL(live)}>{live}</button></small>{/if}</div>
          <span class="spacer"></span>
          <button class="btn big" disabled={!ready || publishing} onclick={publish}>
            {#if publishing}<span class="spin"><Loader size={16} /></span>{p.publishing}{:else}{p.publish}{/if}
          </button>
        </div>
        {#if log.length}
          <ul class="log">
            {#each log as l, i (i)}<li class={l.kind}>{#if l.kind === "ok"}<Check size={14} />{:else if l.kind === "warn" || l.kind === "error"}<Alert size={14} />{:else}<span class="dot">•</span>{/if}{l.text}</li>{/each}
          </ul>
        {/if}
        {#if live && !publishing && log.length}
          <button class="btn ghost" onclick={() => OpenURL(live)}>{p.openSite} <External size={14} /></button>
        {/if}
      </section>
    {/if}

    <section class="card">
      <h2>{p.syncTitle}</h2>
      <p>{p.syncHelp}</p>
      {#if !conn.github}
        <p class="muted">{p.syncNeedsGitHub}</p>
        <ol class="steps">
          <li>{p.step1} <button class="add" onclick={() => OpenKeyPage("github")}>{p.openForm} <External size={14} /></button></li>
          <li>{p.step2gh}</li>
          <li>{p.step3}</li>
        </ol>
        <div class="row">
          <input type="password" bind:value={ghKey} placeholder={p.pasteKey} autocomplete="off" />
          <button class="btn" disabled={connecting || !ghKey.trim()} onclick={connectGitHubForSync}>{connecting ? p.checking : p.connect}</button>
        </div>
        {#if syncError}<p class="bad">{syncError}</p>{/if}
      {:else}
        {#if sync}
          <p class="sum">{sync.upToDate && !sync.send && !sync.receive ? p.syncUpToDate : p.syncPlan.replace("{send}", String(sync.send)).replace("{receive}", String(sync.receive))} · <b>{sync.repo}</b></p>
          {#each sync.conflicts ?? [] as c (c)}<p class="bad">{p.conflict} {c}</p>{/each}
        {/if}
        {#if syncError}<p class="bad">{syncError}</p>{/if}
        <div class="actions">
          <button class="btn ghost" disabled={syncing} onclick={checkSync}><Refresh size={14} />{p.syncCheck}</button>
          <button class="btn" disabled={syncing} onclick={runSync}>{syncing ? p.syncing : p.syncNow}</button>
        </div>
      {/if}
    </section>
  {/if}
</div>

<style>
  .box { padding: .75rem 1rem; border-radius: .5rem; background: var(--soft); }
  .choices { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; }
  .choice { display: grid; gap: .3rem; text-align: start; padding: 1rem; border: 1px solid var(--line); border-radius: .6rem; }
  .choice span { font-size: .8rem; color: var(--muted); }
  .choice.on { border-color: var(--blue); box-shadow: inset 0 0 0 1px var(--blue); }
  .status { align-items: center; }
  .spacer { flex: 1; }
  .steps { display: grid; gap: .5rem; padding-inline-start: 1.25rem; font-size: .9rem; }
  .steps .add { display: inline-flex; margin-inline-start: .35rem; }
  .publish .row { align-items: center; }
  .btn.big { padding: .7rem 1.4rem; font-size: .95rem; }
  .link { color: var(--blue); text-decoration: underline; font-size: inherit; }
  .log { list-style: none; padding: .75rem; display: grid; gap: .35rem; max-height: 16rem; overflow: auto; border-radius: .5rem; background: var(--bg); font-size: .85rem; }
  .log li { display: flex; gap: .45rem; align-items: center; color: var(--muted); }
  .log li.ok { color: #16a34a; }
  .log li.warn { color: #b45309; }
  .log li.error { color: var(--red); }
  .dot { width: 14px; text-align: center; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .sum { color: var(--text); }
</style>
