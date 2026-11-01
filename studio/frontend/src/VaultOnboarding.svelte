<script lang="ts">
  import { CreateVault, DefaultVaultFolder, PickFolder, RecentSites, RestoreSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { RecentSite, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import { t } from "./i18n";

  // A new vault of notes, the same idea as a new site but for notes alone:
  // a name and it is made (in Documents, or over a folder picked here).
  // Vaults opened lately are listed to open again.
  let { onDone, onCancel }: { onDone: (v: SiteInfo) => void; onCancel: () => void } = $props();
  const v = t.vaultNew;

  let name = $state("");
  let folder = $state(""); // picked folder; "" = a new one in Documents
  let where = $state("");
  let busy = $state(false);
  let error = $state("");
  let recent = $state<RecentSite[]>([]);
  const clean = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  const home = (p: string) => p.replace(/^\/home\/[^/]+|^\/Users\/[^/]+|^C:\\Users\\[^\\]+/, "~");

  RecentSites().then((list) => (recent = (list ?? []).filter((r) => r.kind === "notes"))).catch(() => {});

  // Where the notes will be, as the name is typed.
  function showWhere() {
    if (!folder) DefaultVaultFolder(name.trim() || v.namePh).then((d) => (where = d)).catch(() => {});
  }
  showWhere();

  async function pick() {
    const f = await PickFolder().catch(() => "");
    if (!f) return;
    folder = f;
    where = f;
    if (!name.trim()) name = f.split(/[\\/]/).pop() ?? "";
  }

  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    busy = true; error = "";
    try { onDone(await CreateVault(name.trim(), folder)); } catch (e) { error = clean(e); } finally { busy = false; }
  }

  async function openRecent(r: RecentSite) {
    try {
      if (!r.listed) await RestoreSite(r.dir);
      onDone(r);
    } catch (e) { error = clean(e); }
  }
</script>

<div class="vault-new">
  <button class="close btn ghost" onclick={onCancel}>{t.ob.cancel}</button>
  <div class="center">
    <img class="bounce" src="/blups/turma-7.svg" alt="" />
    <h1>{v.title}</h1>
    <p class="lead">{v.lead}</p>
    <form onsubmit={create}>
      <input bind:value={name} oninput={showWhere} placeholder={v.namePh} aria-label={v.name} autocomplete="off" />
      <small class="muted where">{v.where} <b>{home(where)}</b> · <button type="button" class="link" onclick={pick}>{folder ? v.otherFolder : v.useFolder}</button></small>
      <button class="btn big" disabled={busy || !name.trim()}>{#if busy}<span class="spin"><Loader size={16} /></span>{/if}{v.create}</button>
    </form>
    {#if error}<p class="bad">{error}</p>{/if}
    {#if recent.length}
      <div class="recent">
        <b>{v.recent}</b>
        <ul>
          {#each recent as r (r.dir)}
            <li><button onclick={() => openRecent(r)}><FolderOpen size={16} /><span class="r-name">{r.title}</span><small class="muted">{home(r.folder)}</small></button></li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>
</div>

<style>
  .vault-new { position: relative; height: 100%; overflow: auto; display: grid; place-items: center; padding: 3rem 1.5rem; }
  .close { position: absolute; inset-block-start: 1rem; inset-inline-end: 1rem; }
  .center { width: min(100%, 30rem); display: grid; justify-items: center; gap: 1rem; text-align: center; }
  .center > img { width: 6rem; }
  h1 { font-size: 1.85rem; font-weight: 800; letter-spacing: -.03em; }
  .lead { color: var(--muted); margin-block-start: -.4rem; }
  form { display: grid; gap: .6rem; width: 100%; }
  form input { font-size: 1.05rem; padding: .75rem .9rem; }
  .where { text-align: start; font-size: .82rem; overflow-wrap: anywhere; }
  .link { color: var(--blue); font-weight: 600; font-size: inherit; }
  .btn.big { justify-content: center; padding: .8rem 1.6rem; font-size: 1rem; }
  .bad { color: var(--red); font-size: .9rem; }
  .recent { display: grid; gap: .4rem; width: 100%; margin-block-start: 1rem; text-align: start; }
  .recent ul { list-style: none; padding: 0; display: grid; gap: .3rem; }
  .recent li button { display: grid; grid-template-columns: auto auto minmax(0, 1fr); align-items: center; gap: .6rem; width: 100%; padding: .55rem .7rem; border: 1px solid var(--line); border-radius: .55rem; background: var(--panel); text-align: start; }
  .recent li button:hover { border-color: var(--muted); }
  .r-name { font-weight: 600; }
  .recent small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: end; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .bounce { animation: bounce 1.8s ease-in-out infinite; }
  @keyframes bounce { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-8px); } }
</style>
