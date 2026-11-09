<script lang="ts">
  import { CreateVault, DefaultVaultFolder, PickFolder, RecentSites, RestoreSite } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { RecentSite, SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Loader from "@lucide/svelte/icons/loader-circle";
  import StepScreen from "./ui/StepScreen.svelte";
  import TextField from "./ui/TextField.svelte";
  import CloudBring from "./CloudBring.svelte";
  import { t } from "./i18n";

  // A new vault of notes, the same idea as a new site but for notes alone:
  // a name and it is made (in Documents, or over a folder picked here).
  // Vaults opened lately are listed to open again.
  let { onDone, onCancel }: { onDone: (v: SiteInfo) => void; onCancel: () => void } = $props();
  const v = t.vaultNew;

  let step = $state<"hello" | "name">("hello");
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

  async function create(e?: SubmitEvent) {
    e?.preventDefault();
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

{#if step === "hello"}
  <!-- First, as with a new site: hello, Começar, and what can be opened or
       brought instead (Blup Cloud, the vaults opened lately). -->
  <StepScreen image="/blups/turma-7.svg" title={v.helloTitle} lead={v.helloLead} closeLabel={t.ob.cancel} onClose={onCancel}>
    <button class="btn big" onclick={() => (step = "name")}>{t.ob.start}</button>
    <CloudBring title={v.fromCloud} onBrought={onDone} />
    {#if recent.length}
      <div class="recent">
        <b>{v.recent}</b>
        <ul>
          {#each recent as r (r.dir)}
            <li><button onclick={() => openRecent(r)}><span class="r-name">{r.title}</span><small class="muted">{home(r.folder)}</small></button></li>
          {/each}
        </ul>
      </div>
    {/if}
    {#if error}<p class="bad">{error}</p>{/if}
  </StepScreen>
{:else}
  <StepScreen image="/blups/turma-7.svg" title={v.title} lead={v.lead} closeLabel={t.ob.cancel} onClose={onCancel}>
    <form class="card" onsubmit={create}>
      <TextField large label={v.name} bind:value={name} oninput={showWhere} placeholder={v.namePh} autocomplete="off" error={error} />
      <div class="where">
        <small class="muted">{v.where} <b>{home(where)}</b></small>
        <button type="button" class="link" onclick={pick}>{folder ? v.otherFolder : v.useFolder}</button>
      </div>
    </form>
    {#snippet actions()}
      <button class="btn ghost" onclick={() => (step = "hello")}>{t.ob.back}</button>
      <button class="btn big" disabled={busy || !name.trim()} onclick={() => create()}>{#if busy}<span class="spin"><Loader size={16} /></span>{/if}{v.create}</button>
    {/snippet}
  </StepScreen>
{/if}

<style>
  .where { display: flex; flex-wrap: wrap; justify-content: space-between; gap: .25rem .75rem; font-size: .82rem; overflow-wrap: anywhere; }
  .recent { display: grid; gap: .6rem; width: min(28rem, 100%); margin-block-start: 1rem; padding: 1rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); text-align: start; }
  .recent ul { list-style: none; padding: 0; display: grid; gap: .2rem; max-height: 16rem; overflow: auto; }
  .recent li button { display: grid; gap: .1rem; width: 100%; padding: .5rem .65rem; border-radius: .45rem; text-align: start; }
  .recent li button:hover { background: var(--soft); }
  .r-name { font-weight: 600; font-size: .9rem; }
  .recent small { font-size: .75rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .spin { display: inline-grid; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
