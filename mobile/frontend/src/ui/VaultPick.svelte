<script lang="ts">
  import { onMount } from "svelte";
  import Check from "@lucide/svelte/icons/check";
  import Cloud from "@lucide/svelte/icons/cloud";
  import CloudDownload from "@lucide/svelte/icons/cloud-download";
  import CloudUpload from "@lucide/svelte/icons/cloud-upload";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import Library from "@lucide/svelte/icons/library";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Plus from "@lucide/svelte/icons/plus";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import { CloudService, NotesService, SpaceService, type RemoteVault, type Space } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message } from "../lib/app.svelte";
  import Sheet from "./Sheet.svelte";
  import { fmt, t } from "../desk/i18n";

  // The vaults of this phone, one open at a time, and those of the account
  // that aren't here yet. Each has its own notes, calendar and trash.
  const ready = $derived(app.account?.notes === "ready");
  let remote = $state<RemoteVault[]>([]);
  const missing = $derived(remote.filter((v) => !app.spaces.some((s) => s.remote === v.id)));
  let adding = $state(false);
  let draft = $state("");
  let busy = $state(false);

  onMount(async () => {
    if (app.account?.notes !== "ready") return;
    try { remote = (await CloudService.Vaults()) ?? []; } catch {} // offline: the phone's still show
  });

  // Says whether it worked; a failure is already on the screen.
  async function work(run: () => Promise<unknown>) {
    if (busy) return false;
    busy = true;
    try {
      NotesService.Haptic();
      await run();
      app.back();
      await app.entered();
      return true;
    } catch (e) {
      app.say({ text: message(e) });
      return false;
    } finally {
      busy = false;
    }
  }
  const open = (id: string) => (id === app.space?.id ? app.back() : work(() => SpaceService.Open(id)));
  // A vault of the account comes as a new one here, tied to it.
  const bring = (v: RemoteVault) => work(async () => {
    await SpaceService.Create(v.title);
    await CloudService.Link(v.id, v.title);
  });
  async function create(e: SubmitEvent) {
    e.preventDefault();
    const title = draft.trim();
    if (!title) { adding = false; return; }
    await work(async () => {
      await SpaceService.Create(title);
      if (ready) await CloudService.Link("", title);
    });
  }
  // Deleting asks twice, and says what goes: a vault on the Cloud goes from
  // there too, for every device.
  let chosen = $state<Space | null>(null);
  let sure = $state(false);
  function more(s: Space) {
    chosen = s;
    sure = false;
  }
  const remove = (s: Space) => work(async () => {
    await SpaceService.Delete(s.id);
    if (s.remote) {
      remote = remote.filter((v) => v.id !== s.remote);
      app.loadAccount(); // the space it freed
    }
  }).then((done) => done && app.say({ text: s.remote ? t.vaults.deletedBoth : t.vaults.deletedHere }));

  const focus = (el: HTMLInputElement) => { setTimeout(() => el.focus(), 60); };

  // A vault only on this phone (never sent, or let go of after being
  // deleted elsewhere) goes to the Cloud only by this button: a new vault
  // there, with all its notes.
  const send = (s: Space) => work(async () => {
    if (!s.current) await SpaceService.Open(s.id);
    await CloudService.Link("", s.title);
  }).then((done) => done && app.say({ text: t.vaults.sent })); // work() syncs as it opens

  // A new name, here and, for a vault on the Cloud, there too.
  // Both start again when another vault is chosen.
  let newName = $derived(chosen?.title ?? "");
  let naming = $derived.by(() => (void chosen, false)); // the field shows after the Renomear button
  async function rename(e: SubmitEvent, s: Space) {
    e.preventDefault();
    const title = newName.trim();
    if (!title || title === s.title || busy) return;
    busy = true;
    try {
      NotesService.Haptic();
      const cloudOK = await SpaceService.Rename(s.id, title);
      app.spaces = (await SpaceService.List()) ?? [];
      chosen = app.spaces.find((x) => x.id === s.id) ?? null;
      naming = false;
      if (s.remote) remote = (await CloudService.Vaults().catch(() => remote)) ?? remote;
      (document.activeElement as HTMLElement | null)?.blur();
      app.say({ text: cloudOK ? t.vaults.renamed : t.vaults.renamedHere });
    } catch (err) {
      app.say({ text: message(err) });
    } finally {
      busy = false;
    }
  }
</script>

<Sheet title={chosen ? chosen.title : t.vaults.title} onclose={() => app.back()}>
  {#if chosen}
    {@const s = chosen}
    {#if naming}
      <form class="rename" onsubmit={(e) => rename(e, s)}>
        <label for="vault-name">{t.vaults.newName}</label>
        <input id="vault-name" type="text" bind:value={newName} {@attach focus} maxlength="120" enterkeyhint="done" autocapitalize="sentences" />
        <div>
          <button type="button" class="plain" onclick={() => { naming = false; newName = s.title; }}>{t.common.cancel}</button>
          <button type="submit" class="save" disabled={busy || !newName.trim() || newName.trim() === s.title}>{t.common.save}</button>
        </div>
      </form>
    {:else}
      <button type="button" class="plain" onclick={() => (naming = true)}><Pencil size={19} />{t.common.rename}</button>
    {/if}
    {#if ready && !s.remote}
      <button type="button" class="plain" disabled={busy || app.syncing} onclick={() => send(s)}><CloudUpload size={19} />{t.vaults.send}</button>
    {/if}
    {#if app.spaces.length < 2}
      <p>{t.vaults.needOne}</p>
    {:else if s.remote}
      <p class="warn">{fmt(s.notes === 1 ? t.vaults.remote1 : s.notes ? t.vaults.remoteN : t.vaults.remote0, { n: s.notes })}</p>
    {:else}
      {#if s.notes}
        <p class="warn">{fmt(s.notes === 1 ? t.vaults.local1 : t.vaults.localN, { n: s.notes })}</p>
      {:else}
        <p>{t.vaults.empty}</p>
      {/if}
    {/if}
    {#if app.spaces.length > 1}
      {#if sure}
        <button type="button" class="danger solid" disabled={busy || app.syncing} onclick={() => remove(s)}><Trash2 size={19} />{s.remote ? t.vaults.confirmBoth : t.vaults.confirmHere}</button>
      {:else}
        <button type="button" class="danger" onclick={() => (sure = true)}><Trash2 size={19} />{t.vaults.remove}</button>
      {/if}
    {/if}
    <button type="button" class="plain" onclick={() => (chosen = null)}>{t.vaults.backToList}</button>
  {:else}
  <div class="list">
    {#each app.spaces as s (s.id)}
      <div class="line">
        <button type="button" class="row" disabled={app.syncing} onclick={() => open(s.id)}>
          <span class="tile"><Library size={19} /></span><span class="name">{s.title}</span>
          {#if s.remote}<span class="cloud" aria-label={t.vaults.onCloud}><Cloud size={17} /></span>{/if}
          {#if s.current}<span class="mark"><Check size={20} /></span>{/if}
        </button>
        <button type="button" class="more" aria-label={fmt(t.common.options, { name: s.title })} onclick={() => more(s)}><Ellipsis size={20} /></button>
      </div>
    {/each}
    {#each missing as v (v.id)}
      <button type="button" class="row" disabled={app.syncing} onclick={() => bring(v)}>
        <span class="tile"><CloudDownload size={19} /></span><span class="name">{v.title}</span><small>{t.vaults.download}</small>
      </button>
    {/each}
    {#if adding}
      <form class="row" onsubmit={create}>
        <span class="tile"><Library size={19} /></span>
        <input type="text" bind:value={draft} {@attach focus} placeholder={t.vaults.name} aria-label={t.vaults.name} enterkeyhint="done" autocapitalize="sentences" />
        <button type="submit" class="ok" aria-label={t.vaults.create} disabled={!draft.trim() || busy} onpointerdown={(e) => e.preventDefault()} onmousedown={(e) => e.preventDefault()}><Check size={20} /></button>
      </form>
    {:else}
      <button type="button" class="row" disabled={app.syncing} onclick={() => (adding = true)}><span class="tile"><Plus size={19} /></span><span class="name">{t.vaults.newVault}</span></button>
    {/if}
  </div>
  {#if app.syncing}<p>{t.vaults.syncWait}</p>{/if}
  {/if}
</Sheet>

<style>
  .list { display: grid; }
  .rename { display: grid; gap: .35rem; }
  .rename label { font-size: .875rem; font-weight: 600; }
  .rename div { display: grid; grid-template-columns: 1fr 1fr; gap: .5rem; margin-block-start: .25rem; }
  .rename input { font-size: 1rem; }
  .save { min-height: 2.9rem; border-radius: .6rem; background: var(--blue); color: #fff; font-weight: 600; }
  .save:disabled { opacity: .45; }
  .row { display: flex; align-items: center; gap: .75rem; width: 100%; min-height: 3.1rem; text-align: start; }
  .row:disabled { opacity: .5; }
  .line { display: flex; align-items: center; }
  .more { flex: none; display: grid; place-items: center; width: 2.6rem; height: 3rem; color: var(--muted); }
  .warn { color: var(--red); font-weight: 500; }
  .danger, .plain { display: flex; align-items: center; justify-content: center; gap: .5rem; min-height: 2.9rem; border-radius: .6rem; font-weight: 600; }
  .danger { border: 1px solid var(--line); color: var(--red); }
  .danger.solid { border-color: var(--red); background: var(--red); color: #fff; }
  .plain { border: 1px solid var(--line); }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .tile { flex: none; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .cloud { display: grid; color: var(--muted); }
  .mark { display: grid; color: var(--blue-text); }
  small { font-size: .85rem; color: var(--blue-text); font-weight: 600; }
  input { flex: 1; min-width: 0; }
  .ok { flex: none; display: grid; place-items: center; width: 2.75rem; height: 2.75rem; border-radius: .6rem; background: var(--blue); color: #fff; }
  .ok:disabled { opacity: .4; }
  p { font-size: .9rem; line-height: 1.4; color: var(--muted); }
</style>
