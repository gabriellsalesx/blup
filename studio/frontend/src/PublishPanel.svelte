<script lang="ts">
  import { Page } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { onDestroy, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { CurrentDomain, SetDomain, CheckDomain, RemoveDomain, Connections, OpenKeyPage, OpenURL, OpenBlupCloud, ConnectGitHub, ConnectCloudflare, Disconnect, CloudflareAccounts, Target, SetTarget, Publish, CheckSiteSlug, SyncStatus, Sync, SyncRemote, SetSyncRemote, OpenNewGitHubRepo } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Connections as Conn, Target as Dest, SyncInfo, DomainStatus, BlupAccount } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Account } from "../bindings/github.com/gabriellsalesx/blup/internal/deploy/cfpages/models";
  import type { SlugCheck } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import Check from "@lucide/svelte/icons/check";
  import Alert from "@lucide/svelte/icons/circle-alert";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import External from "@lucide/svelte/icons/external-link";
  import Refresh from "@lucide/svelte/icons/refresh-cw";
  import Copy from "@lucide/svelte/icons/copy";
  import { Clipboard } from "@wailsio/runtime";
  import { t } from "./i18n";
  import { toast } from "./toast.svelte";
  import CloudAccount from "./CloudAccount.svelte";
  import BlupDomain from "./BlupDomain.svelte";
  import Choice from "./ui/Choice.svelte";

  // Two screens in one: publishing the site (where it goes, then its own
  // domain once it is online) and syncing the notes. notes: a vault of notes
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
  let syncing = $state(false);
  let syncError = $state("");
  // Where the notes go: Blup Cloud (encrypted) or a GitHub repository.
  let remote = $state("");
  let cloud = $state<BlupAccount | null>(null);
  const cl = t.cloud;
  SyncRemote().then((r) => (remote = r));
  async function pickRemote(r: string) {
    await SetSyncRemote(r);
    remote = r; sync = null; syncError = ""; checkedCloud = false;
  }
  const canSync = $derived(remote === "blup" ? cloud?.signedIn && cloud.notes === "ready" : !!conn?.github);
  // Blup Cloud ready for the first time: see what a sync would do (once).
  let checkedCloud = false;
  $effect(() => {
    if (remote === "blup" && cloud?.notes === "ready" && !checkedCloud) { checkedCloud = true; checkSync(); }
  });

  // The custom domain: typed here, connected where the site is published;
  // the DNS records still missing are listed to copy.
  const dm = t.domain;
  let domain = $state("");
  let domainInput = $state("");
  let editingDomain = $state(false);
  let dstatus = $state<DomainStatus | null>(null);
  let dbusy = $state(false);
  let derror = $state("");
  let askRemove = $state(false);
  let republish = $state(false);
  let copiedRec = $state(-1);
  CurrentDomain().then((d) => { domain = d; domainInput = d; });
  async function domainRun(fn: () => Promise<DomainStatus | void>, changed = false) {
    dbusy = true; derror = "";
    try {
      const st = await fn();
      dstatus = st ?? null;
      domain = st ? st.domain : "";
      editingDomain = false; askRemove = false;
      if (changed) republish = true;
    } catch (e) { derror = msg(e); } finally { dbusy = false; }
  }
  const setDomain = () => domainRun(async () => { const st = await SetDomain(domainInput); toast(t.toast.domainSet.replace("{domain}", st.domain)); return st; }, true);
  const checkDomain = () => domainRun(() => CheckDomain());
  const removeDomain = () => domainRun(async () => { await RemoveDomain(); domainInput = ""; toast(t.toast.domainRemoved); }, true);
  async function copyRec(i: number, v: string) { await Clipboard.SetText(v); copiedRec = i; setTimeout(() => (copiedRec = -1), 1500); }

  const off = Events.On("progress", (ev) => { log.push(ev.data); });
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
  const ready = $derived(!!target?.provider && connected && (target.provider === "blup" ? slugOK && slugFree : target.provider === "github" ? !!target.repo : !!target.account && !!target.project));
  // The steps to connect a Cloudflare or GitHub account show on request.
  let howTo = $state(false);

  async function pick(provider: string) {
    if (!target) return;
    target.provider = provider;
    if (provider === "cloudflare" && conn?.cloudflare && !accounts.length) accounts = (await CloudflareAccounts().catch(() => [])) ?? [];
    if (provider === "cloudflare" && !target.account && accounts.length) target.account = accounts[0].id;
    howTo = false;
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
    try { await ConnectGitHub(ghKey); ghKey = ""; conn = await Connections(); await checkSync(); }
    catch (e) { syncError = msg(e); } finally { connecting = false; }
  }

  async function checkSync() {
    syncError = ""; syncing = true;
    try { sync = await SyncStatus(); } catch (e) { syncError = msg(e); } finally { syncing = false; }
  }
  async function runSync() {
    syncError = ""; syncing = true;
    try { sync = await Sync(); toast(t.toast.synced); } catch (e) { syncError = msg(e); } finally { syncing = false; }
  }

  load().then(() => {
    if (autostart === "publish" && ready) publish();
    if (autostart === "sync" && remote === "github" && conn?.github) checkSync();
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
        <Choice title="Cloudflare Pages" help={p.cfHelp} on={target.provider === "cloudflare"} onclick={() => pick("cloudflare")} />
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
                <select bind:value={target.account}>{#each accounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}</select>
              </label>
            {/if}
            <label class="field"><span>{p.project}</span><input type="text" bind:value={target.project} /><small>{p.projectHelp}</small></label>
          {:else}
            <label class="field"><span>{p.repo}</span><input type="text" bind:value={target.repo} /><small>{p.repoHelp}</small></label>
            <small class="muted">{gr.siteHelp} <button class="link" onclick={() => newRepo("site")}>{gr.create}</button></small>
          {/if}
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
            {#each log as l, i (i)}<li class={l.kind}>{#if l.kind === "ok"}<Check size={14} />{:else if l.kind === "warn" || l.kind === "error"}<Alert size={14} />{:else}<span class="dot">•</span>{/if}{l.text}</li>{/each}
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

    <!-- The own domain on GitHub or Cloudflare comes once the site is online. -->
    {#if target.provider && target.provider !== "blup" && (live || domain)}
    <section class="card">
      <h2>{dm.title}</h2>
      <p>{dm.help}</p>
      {#if !domain || editingDomain}
        <div class="row">
          <input type="text" bind:value={domainInput} placeholder="meusite.com.br" onkeydown={(e) => e.key === "Enter" && setDomain()} />
          <button class="btn" disabled={dbusy || !domainInput.trim()} onclick={setDomain}>{dbusy ? dm.connecting : dm.connect}</button>
          {#if editingDomain}<button class="btn ghost" onclick={() => { editingDomain = false; domainInput = domain; }}>{t.sec.cancel}</button>{/if}
        </div>
        <small class="muted">{dm.inputHelp}</small>
      {:else}
        <div class="row status">
          <div class="field"><span>{domain}</span>
            {#if dstatus?.state === "active"}<small class="good"><Check size={13} /> {dm.active}{dstatus.https ? " · HTTPS" : ""}</small>
            {:else if dstatus?.state === "unpublished"}<small>{dm.unpublished}</small>
            {:else if dstatus?.state === "waiting"}<small>{dm.waiting}</small>
            {:else if dstatus?.state === "dns" || dstatus?.state === "apex-cloudflare"}<small class="warn">{dm.missing}</small>
            {:else}<small class="muted">{dm.saved}</small>{/if}
          </div>
          <span class="spacer"></span>
          <button class="btn ghost" disabled={dbusy} onclick={checkDomain}>{#if dbusy}<span class="spin"><Loader size={14} /></span>{:else}<Refresh size={14} />{/if}{dm.check}</button>
          <button class="btn ghost" onclick={() => (editingDomain = true)}>{dm.change}</button>
        </div>
        {#if dstatus?.state === "apex-cloudflare"}
          <p class="box">{dm.apexCf.replace("{domain}", domain)}</p>
          <p>{dm.apexCfWww.replace("{www}", dstatus.www)}</p>
        {:else if dstatus?.state === "dns"}
          <p>{dm.dnsHelp}</p>
        {/if}
        {#if dstatus?.records?.length && (dstatus.state === "dns" || dstatus.state === "apex-cloudflare")}
          <table class="dns">
            <thead><tr><th>{dm.type}</th><th>{dm.name}</th><th>{dm.value}</th><th></th></tr></thead>
            <tbody>
              {#each dstatus.records as r, i (i)}
                <tr><td>{r.type}</td><td><code>{r.name}</code></td><td><code>{r.value}</code></td>
                  <td><button class="icon-btn" title={dm.copy} onclick={() => copyRec(i, r.value)}>{#if copiedRec === i}<Check size={14} />{:else}<Copy size={14} />{/if}</button></td></tr>
              {/each}
            </tbody>
          </table>
          <small class="muted">{dm.dnsTime}</small>
        {/if}
        {#if dstatus?.message}<small class="muted">{dstatus.message}</small>{/if}
        <div class="actions">
          {#if askRemove}
            <span class="muted q">{dm.removeQ.replace("{domain}", domain)}</span>
            <button class="btn ghost" onclick={() => (askRemove = false)}>{t.sec.cancel}</button>
            <button class="btn danger" disabled={dbusy} onclick={removeDomain}>{dm.remove}</button>
          {:else}
            <button class="btn ghost danger-text" onclick={() => (askRemove = true)}>{dm.remove}</button>
          {/if}
        </div>
      {/if}
      {#if derror}<p class="bad">{derror}</p>{/if}
      {#if republish}<p class="box">{dm.republish}</p>{/if}
    </section>
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
      {#if canSync}
        {#if sync}
          <p class="sum">{sync.upToDate && !sync.send && !sync.receive ? p.syncUpToDate : p.syncPlan.replace("{send}", String(sync.send)).replace("{receive}", String(sync.receive))} · <b>{sync.repo}</b></p>
          {#each sync.conflicts ?? [] as c (c)}<p class="bad">{p.conflict} {c}</p>{/each}
          {#if sync.tooLarge?.length}
            <p class="warn">{cl.tooLarge.replace("{mb}", String(cloud?.maxFileMB ?? 100))}</p>
            <ul class="large">{#each sync.tooLarge as f (f)}<li>{f}</li>{/each}</ul>
          {/if}
        {/if}
        <div class="actions">
          <button class="btn ghost" disabled={syncing} onclick={checkSync}><Refresh size={14} />{p.syncCheck}</button>
          <button class="btn" disabled={syncing} onclick={runSync}>{syncing ? p.syncing : p.syncNow}</button>
        </div>
      {/if}
      {#if syncError}<p class="bad">{syncError}</p>{/if}
    </section>
    {/if}
  {/if}
</div>

<style>
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
  .log li.ok { color: #16a34a; }
  .log li.warn { color: #b45309; }
  .log li.error { color: var(--red); }
  .dot { width: 14px; text-align: center; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .sum { color: var(--text); }
  .good { display: inline-flex; align-items: center; gap: .25rem; color: #16a34a; }
  .warn { color: #b45309; }
  .dns { width: 100%; border-collapse: collapse; font-size: .85rem; }
  .dns th { text-align: start; font-size: .75rem; color: var(--muted); font-weight: 600; padding: .3rem .5rem; }
  .dns td { padding: .45rem .5rem; border-block-start: 1px solid var(--line); }
  .dns code { font-size: .85rem; user-select: all; }
  .q { font-size: .85rem; }
  .danger-text { color: var(--red); }
  .large { margin: 0; padding-inline-start: 1.25rem; font-size: .85rem; color: var(--muted); }
  .btn.danger { background: var(--red); color: #fff; }
</style>
