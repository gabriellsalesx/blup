<script lang="ts">
  import { Page } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { onDestroy, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { Connections, OpenKeyPage, OpenURL, OpenBlupCloud, ConnectGitHub, ConnectCloudflare, Disconnect, CloudflareAccounts, CloudflareSubdomain, SetCloudflareSubdomain, Target, SetTarget, Publish, CheckSiteSlug, SyncRemote, SetSyncRemote, OpenNewGitHubRepo } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Connections as Conn, Target as Dest, SyncInfo, BlupAccount } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Account } from "../bindings/github.com/gabriellsalesx/blup/internal/deploy/cfworkers/models";
  import type { SlugCheck } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import Check from "@lucide/svelte/icons/check";
  import Alert from "@lucide/svelte/icons/circle-alert";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import External from "@lucide/svelte/icons/external-link";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";
  import CloudAccount from "./CloudAccount.svelte";
  import BlupDomain from "./BlupDomain.svelte";
  import SyncFresh from "./SyncFresh.svelte";
  import { syncJob, startSync, onSyncDone, syncStepText, live as liveSync, onLive } from "./syncjob.svelte";
  import Choice from "./ui/Choice.svelte";

  // Two screens in one: publishing the site (where it goes; its own domain
  // only on Blup Cloud, once it is online) and syncing the notes. notes: a vault of notes
  // has no site, so only syncing shows; a site's Sync button opens the
  // sync part (autostart "sync").
  let { autostart = "", notes = false }: { autostart?: string; notes?: boolean } = $props();
  const syncOnly = $derived(notes || autostart === "sync");
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
  // Syncing is a job of the whole window (syncjob): it goes on when the
  // screen closes, and the check by the note's title shows it too.
  const syncing = $derived(syncJob.running);
  let syncError = $state("");
  // The sync stopped for safety (the cloud shows a version that does not
  // continue this device's): "Compare from scratch" instead of the error.
  let syncStuck = $state(false);
  const isStuck = (e: unknown) => {
    const c = (e as { cause?: unknown } | null)?.cause;
    return typeof c === "object" && c !== null && (c as { stuck?: unknown }).stuck === true;
  };
  function failed(e: unknown) { syncStuck = isStuck(e); syncError = syncStuck ? "" : msg(e); }
  // Where the notes go: Blup Cloud (encrypted) or a GitHub repository.
  let remote = $state("");
  let cloud = $state<BlupAccount | null>(null);
  const cl = t.cloud;
  SyncRemote().then((r) => (remote = r));
  async function pickRemote(r: string) {
    await SetSyncRemote(r);
    remote = r; sync = null; syncError = ""; syncStuck = false; checkedCloud = false;
  }
  const canSync = $derived(remote === "blup" ? cloud?.signedIn && cloud.notes === "ready" : !!conn?.github);
  // Blup Cloud ready for the first time: see what a sync would do (once).
  let checkedCloud = false;
  $effect(() => {
    if (remote === "blup" && cloud?.notes === "ready" && !checkedCloud) {
      checkedCloud = true;
      // The last live result, for its warnings (conflicts, files too large).
      if (liveSync.info) sync = liveSync.info;
    }
  });

  // Cloudflare: the account's workers.dev subdomain, in every site's
  // address (<site>.<subdomain>.workers.dev). A new account has none: it
  // is chosen here once. null while it is looked up.
  let cfSub = $state<string | null>(null);
  let subInput = $state("");
  let subBusy = $state(false);
  let subError = $state("");
  const LABEL = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
  const normLabel = (v: string) => v.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/[^a-z0-9-]+/g, "-").replace(/^-+/, "").slice(0, 63);
  async function loadSub() {
    cfSub = null; subError = "";
    const account = target?.account;
    if (target?.provider !== "cloudflare" || !conn?.cloudflare || !account) return;
    try {
      const sub = await CloudflareSubdomain(account);
      if (target?.account !== account) return;
      cfSub = sub;
      if (!sub) subInput = normLabel(accounts.find((a) => a.id === account)?.name ?? "").replace(/-+$/, "");
    } catch (e) { subError = msg(e); }
  }
  async function createSub() {
    if (!target?.account || !LABEL.test(subInput)) return;
    subBusy = true; subError = "";
    try { cfSub = await SetCloudflareSubdomain(target.account, subInput); } catch (e) { subError = msg(e); } finally { subBusy = false; }
  }
  const cfAddress = $derived(cfSub && target?.project ? `${target.project}.${cfSub}.workers.dev` : "");

  const off = Events.On("progress", (ev) => { log.push(ev.data); });
  onDestroy(() => off());
  // How far a sync is: the step and, sending or receiving, X of Y.
  const stepText = $derived(syncJob.running ? syncStepText(syncJob.phase, syncJob.done, syncJob.total) : "");
  // How a sync ended: while this screen is open, as it ends; opened later,
  // the last end (a stop for safety, an error).
  function ended(end: { info: SyncInfo; error: string; stuck: boolean }) {
    if (end.error) { syncStuck = end.stuck; syncError = end.stuck ? "" : end.error; }
    else { syncStuck = false; syncError = ""; sync = end.info; }
  }
  if (syncJob.last?.error) ended(syncJob.last);
  $effect(() => onSyncDone(ended));
  // The live sync sends each change a moment after it is saved: while it
  // sends, it says so; when it is done, its result replaces what the last
  // check saw (a note being written showed "1 to send" that had gone up).
  $effect(() => onLive((st) => {
    if (st.state === "synced") ended({ info: st.info, error: "", stuck: false });
    else if (st.state === "error") ended({ info: st.info, error: st.error ?? "", stuck: !!st.stuck });
  }));
  const liveSending = $derived(liveSync.state === "syncing" && !syncJob.running);

  async function load() {
    error = "";
    try {
      [conn, target] = await Promise.all([Connections(), Target()]);
      live = target?.url ?? "";
      if (conn?.cloudflare && target?.provider === "cloudflare") { accounts = (await CloudflareAccounts().catch(() => [])) ?? []; loadSub(); }
    } catch (e) { error = msg(e); }
  }
  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  // Blup Cloud: the account signed in and a plan with sites (paid plans).
  const SLUG = /^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])$/;
  const blupOK = $derived(!!cloud?.signedIn && cloud.sitesMax !== 0);
  const slugOK = $derived(!!target && SLUG.test(target.slug) && !target.slug.includes("--"));
  // The address is checked while it is typed: free, taken (with names close
  // to it), kept by Blup.
  let slugCheck = $state<SlugCheck | null>(null);
  let slugChecking = $state(false);
  // What can be typed: lower case, no accents, dashes for the rest.
  const normSlug = (v: string) => v.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/[^a-z0-9-]+/g, "-").replace(/^-+/, "").slice(0, 40);
  let slugTimer: ReturnType<typeof setTimeout> | undefined;
  function checkSlug() {
    clearTimeout(slugTimer);
    slugCheck = null;
    slugChecking = false;
    if (target?.provider !== "blup" || target.slugFixed || !slugOK) return;
    const slug = target.slug;
    slugChecking = true;
    slugTimer = setTimeout(async () => {
      try {
        const r = await CheckSiteSlug(slug);
        if (target?.slug === slug) slugCheck = r;
      } catch { slugCheck = null; } finally { slugChecking = false; }
    }, 400);
  }
  onDestroy(() => clearTimeout(slugTimer));
  const slugFree = $derived(!!target?.slugFixed || slugCheck?.status === "free" || slugCheck?.status === "yours");
  const connected = $derived(target?.provider === "blup" ? blupOK : target?.provider === "cloudflare" ? !!conn?.cloudflare : !!conn?.github);
  const ready = $derived(!!target?.provider && connected && (target.provider === "blup" ? slugOK && slugFree : target.provider === "github" ? !!target.repo : !!target.account && LABEL.test(target.project) && !!cfSub));
  // The steps to connect a Cloudflare or GitHub account show on request.
  let howTo = $state(false);

  async function pick(provider: string) {
    if (!target) return;
    target.provider = provider;
    if (provider === "cloudflare" && conn?.cloudflare && !accounts.length) accounts = (await CloudflareAccounts().catch(() => [])) ?? [];
    if (provider === "cloudflare" && !target.account && accounts.length) target.account = accounts[0].id;
    if (provider === "cloudflare") loadSub();
    howTo = false;
  }

  async function connect() {
    if (!target || !key.trim()) return;
    connecting = true; error = "";
    try {
      if (target.provider === "cloudflare") {
        accounts = (await ConnectCloudflare(key)) ?? [];
        if (accounts.length) target.account = accounts[0].id;
        conn = await Connections();
        await loadSub();
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
      toast(t.toast.published);
    } catch (e) { error = msg(e); toast(t.toast.publishFailed, { kind: "error" }); } finally { publishing = false; }
  }

  // Blup's GitHub key reaches only the repositories picked for it and
  // cannot create them: GitHub's form opens filled in.
  const gr = t.ghrepo;
  async function newRepo(kind: "site" | "notes") {
    try { await OpenNewGitHubRepo(kind, target?.repo ?? ""); } catch (e) { error = msg(e); }
  }

  let ghKey = $state("");
  async function connectGitHubForSync() {
    connecting = true; syncError = "";
    try { await ConnectGitHub(ghKey); ghKey = ""; conn = await Connections(); }
    catch (e) { syncError = msg(e); } finally { connecting = false; }
  }

  async function runSync() {
    syncError = ""; syncStuck = false;
    // The end comes through onSyncDone (also when the screen closes and
    // opens again meanwhile); only a refusal to start lands here.
    startSync().catch(failed);
  }

  load().then(() => {
    if (autostart === "publish" && ready) publish();
  });
</script>

<div class="panel">
  <h1>{syncOnly ? p.syncTitle : p.title}</h1>
  <p class="lead">{notes ? t.vaults.syncLead : syncOnly ? p.syncHelpSite : p.lead}</p>
  {#if error}<p class="bad box">{error}</p>{/if}

  {#if target && conn}
    {#if !syncOnly}
    <section class="card">
      <h2>{p.where}</h2>
      <div class="choices three">
        <Choice title="Blup Cloud · {cl.recommended}" help={p.blupHelp} on={target.provider === "blup"} onclick={() => pick("blup")} />
        <Choice title="Cloudflare Workers" help={p.cfHelp} on={target.provider === "cloudflare"} onclick={() => pick("cloudflare")} />
        <Choice title="GitHub Pages" help={p.ghHelp} on={target.provider === "github"} onclick={() => pick("github")} />
      </div>
    </section>

    {#if target.provider === "blup"}
      <section class="card">
        <h2>Blup Cloud</h2>
        <CloudAccount bind:account={cloud} notes={false} />
        {#if cloud?.signedIn && cloud.sitesMax === 0}
          <p class="box">{p.blupPaid}</p>
          <div class="actions"><button class="btn" onclick={() => OpenBlupCloud(Page.PagePlan)}>{p.blupPlans}</button></div>
        {:else if cloud?.signedIn}
          <label class="field"><span>{p.blupAddress}</span>
            <div class="row addr">
              <input type="text" value={target.slug} oninput={(e) => { target!.slug = normSlug(e.currentTarget.value); e.currentTarget.value = target!.slug; checkSlug(); }} {@attach () => untrack(checkSlug)} disabled={target.slugFixed} spellcheck="false" autocomplete="off" aria-describedby="slug-state" />
              <span class="muted">.getblup.site</span>
            </div>
            <small id="slug-state" class="slug-state" aria-live="polite">
              {#if target.slugFixed}{p.blupFixed}
              {:else if !slugOK}<span class="bad">{p.blupAddressHelp}</span>
              {:else if slugChecking}<span class="muted">{p.slugChecking}</span>
              {:else if slugCheck?.status === "free"}<span class="good"><Check size={13} />{p.slugFree}</span>
              {:else if slugCheck?.status === "yours"}<span class="good"><Check size={13} />{p.slugYours}</span>
              {:else if slugCheck?.status === "reserved"}<span class="bad">{p.slugReserved}</span>
              {:else if slugCheck?.status === "invalid"}<span class="bad">{p.blupAddressHelp}</span>
              {:else if slugCheck?.status === "taken"}
                <span class="bad">{p.slugTaken}</span>
                {#each slugCheck.suggestions ?? [] as s (s)}<button type="button" class="slug-pick" onclick={() => { target!.slug = s; checkSlug(); }}>{s}</button>{/each}
              {:else}{p.blupAddressHelp}{/if}
            </small>
          </label>
          <small class="muted">{p.blupUsage.replace("{n}", String(cloud.sites)).replace("{max}", cloud.sitesMax < 0 ? "∞" : String(cloud.sitesMax))}</small>
        {/if}
      </section>
    {:else if target.provider}
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
                <select bind:value={target.account} onchange={loadSub}>{#each accounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}</select>
              </label>
            {/if}
            {#if cfSub === ""}
              <label class="field"><span>{p.cfSub}</span>
                <div class="row addr">
                  <input type="text" value={subInput} oninput={(e) => { subInput = normLabel(e.currentTarget.value); e.currentTarget.value = subInput; }} spellcheck="false" autocomplete="off" />
                  <span class="muted">.workers.dev</span>
                  <button class="btn" disabled={subBusy || !LABEL.test(subInput)} onclick={createSub}>{subBusy ? p.checking : p.cfSubCreate}</button>
                </div>
                <small>{p.cfSubHelp}</small>
              </label>
            {/if}
            {#if subError}<p class="bad">{subError}</p>{/if}
            <label class="field"><span>{p.project}</span>
              <input type="text" value={target.project} oninput={(e) => { target!.project = normLabel(e.currentTarget.value); e.currentTarget.value = target!.project; }} spellcheck="false" autocomplete="off" />
              <small>{#if !LABEL.test(target.project)}<span class="bad">{p.projectBad}</span>{:else if cfAddress}{p.projectHelp.replace("{address}", cfAddress)}{:else}{p.projectHelpNoSub}{/if}</small>
            </label>
          {:else}
            <label class="field"><span>{p.repo}</span><input type="text" bind:value={target.repo} /><small>{p.repoHelp}</small></label>
            <small class="muted">{gr.siteHelp} <button class="link" onclick={() => newRepo("site")}>{gr.create}</button></small>
          {/if}
          <small class="muted">{p.domainOnCloud}</small>
        {:else if !howTo}
          <p class="muted">{target.provider === "cloudflare" ? p.cfNeeds : p.ghNeeds}</p>
          <div class="actions"><button class="btn" onclick={() => (howTo = true)}>{p.connectAccount}</button></div>
        {:else}
          <ol class="steps">
            {#if target.provider === "github"}<li>{gr.siteHelp} <button class="add" disabled={!target.repo} onclick={() => newRepo("site")}>{gr.create} <External size={14} /></button></li>{/if}
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
    {/if}

    {#if target.provider}
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
            {#each log as l, i (i)}<li class="k-{l.kind}">{#if l.kind === "ok"}<Check size={14} />{:else if l.kind === "warn" || l.kind === "error"}<Alert size={14} />{:else if l.kind === "spin" && i === log.length - 1 && publishing}<span class="spin"><Loader size={14} /></span>{:else}<span class="dot">•</span>{/if}{l.text}</li>{/each}
          </ul>
        {/if}
        {#if live && !publishing && log.length}
          <button class="btn ghost" onclick={() => OpenURL(live)}>{p.openSite} <External size={14} /></button>
        {/if}
      </section>
    {/if}

    <!-- The own domain on Blup Cloud (Max and Ultra), once the site is there. -->
    {#if target.provider === "blup" && cloud?.signedIn && cloud.sitesMax !== 0 && (live || target.slugFixed)}
      <BlupDomain slug={target.slug} />
    {/if}

    {:else}
    <section class="card">
      <h2>{cl.where}</h2>
      <div class="choices">
        <Choice title="Blup Cloud · {cl.recommended}" help={cl.blupHelp} on={remote === "blup"} onclick={() => pickRemote("blup")} />
        <Choice title="GitHub" help={cl.ghHelp} on={remote === "github"} onclick={() => pickRemote("github")} />
      </div>
      {#if remote === "blup"}
        <CloudAccount bind:account={cloud} />
      {:else if remote === "github" && !conn.github}
        <p class="muted">{p.syncNeedsGitHub}</p>
        <ol class="steps">
          <li>{gr.notesHelp} <button class="add" onclick={() => newRepo("notes")}>{gr.create} <External size={14} /></button></li>
          <li>{p.step1} <button class="add" onclick={() => OpenKeyPage("github")}>{p.openForm} <External size={14} /></button></li>
          <li>{p.step2gh}</li>
          <li>{p.step3}</li>
        </ol>
        <div class="row">
          <input type="password" bind:value={ghKey} placeholder={p.pasteKey} autocomplete="off" />
          <button class="btn" disabled={connecting || !ghKey.trim()} onclick={connectGitHubForSync}>{connecting ? p.checking : p.connect}</button>
        </div>
      {:else if remote === "github"}
        <p>{p.syncHelp}</p>
        <small class="muted">{gr.notesHelp} <button class="link" onclick={() => newRepo("notes")}>{gr.create}</button></small>
      {/if}
      {#if canSync && syncStuck}
        <SyncFresh done={(info) => { syncStuck = false; sync = info; }} />
      {:else if canSync}
        {#if sync}
          {#each sync.conflicts ?? [] as c (c)}<p class="bad">{p.conflict} {c}</p>{/each}
          {#if sync.tooLarge?.length}
            <p class="warn">{cl.tooLarge.replace("{mb}", String(cloud?.maxFileMB ?? 100))}</p>
            <ul class="large">{#each sync.tooLarge as f (f)}<li>{f}</li>{/each}</ul>
          {/if}
        {/if}
        <!-- Plans with live sync send by themselves; the others (and
             GitHub) by this button or by the check by the note's title. -->
        {#if !(remote === "blup" && cloud?.live)}
          <div class="actions">
            <button class="btn" disabled={syncing} onclick={runSync}>{syncJob.running ? p.syncing : p.syncNow}</button>
          </div>
        {/if}
        {#if liveSending}<small class="muted">{p.liveSending}</small>{/if}
        {#if syncing && stepText}
          <div class="step">
            {#if syncJob.total}<div class="sbar" role="progressbar" aria-valuemin="0" aria-valuemax={syncJob.total} aria-valuenow={syncJob.done}><span style:width="{Math.max(2, (syncJob.done / syncJob.total) * 100)}%"></span></div>{/if}
            <small class="muted">{stepText}</small>
          </div>
        {/if}
      {/if}
      {#if syncError}<p class="bad">{syncError}</p>{/if}
    </section>
    {/if}
  {/if}
</div>

<style>
  .step { display: grid; gap: .35rem; }
  .sbar { height: 6px; border-radius: 999px; background: var(--soft); overflow: hidden; }
  .sbar span { display: block; height: 100%; border-radius: inherit; background: var(--blue); transition: width .3s ease; }
  .box { padding: .75rem 1rem; border-radius: .5rem; background: var(--soft); }
  .status { align-items: center; }
  .spacer { flex: 1; }
  .steps { display: grid; gap: .5rem; padding-inline-start: 1.25rem; font-size: .9rem; }
  .steps .add { display: inline-flex; margin-inline-start: .35rem; }
  .publish .row { align-items: center; }
  .choices.three { grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr)); }
  .addr { align-items: center; gap: .35rem; }
  .addr input { flex: 1; min-width: 0; }
  .slug-state { display: flex; flex-wrap: wrap; align-items: center; gap: .35rem; }
  .slug-state .good { display: inline-flex; align-items: center; gap: .25rem; }
  .slug-pick { padding: .1rem .55rem; border: 1px solid var(--line); border-radius: 999px; font-size: .8rem; font-weight: 600; }
  .slug-pick:hover { border-color: var(--blue); color: var(--blue); }
  .btn.big { padding: .7rem 1.4rem; font-size: .95rem; }
  .link { color: var(--blue); text-decoration: underline; font-size: inherit; }
  .log { list-style: none; padding: .75rem; display: grid; gap: .35rem; max-height: 16rem; overflow: auto; border-radius: .5rem; background: var(--bg); font-size: .85rem; }
  .log li { display: flex; gap: .45rem; align-items: center; color: var(--muted); }
  .log li.k-ok { color: #16a34a; }
  .log li.k-warn { color: #b45309; }
  .log li.k-error { color: var(--red); }
  .dot { width: 14px; text-align: center; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .sum { color: var(--text); }
  .good { display: inline-flex; align-items: center; gap: .25rem; color: #16a34a; }
  .warn { color: #b45309; }
  .large { margin: 0; padding-inline-start: 1.25rem; font-size: .85rem; color: var(--muted); }
</style>
