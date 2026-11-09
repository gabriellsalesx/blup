<script lang="ts">
  import Check from "@lucide/svelte/icons/check";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Cloud from "@lucide/svelte/icons/cloud";
  import FileText from "@lucide/svelte/icons/file-text";
  import Languages from "@lucide/svelte/icons/languages";
  import Library from "@lucide/svelte/icons/library";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import { app, bytes as size, message, when } from "../lib/app.svelte";
  import CloudCard from "../ui/CloudCard.svelte";
  import Settings from "../ui/Settings.svelte";
  import Sheet from "../ui/Sheet.svelte";
  import SyncFresh from "../ui/SyncFresh.svelte";
  import VaultPick from "../ui/VaultPick.svelte";
  import { SettingsService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { LANGS, fmt, plural, setLanguage, shown, t, type Lang } from "../desk/i18n";

  // The account: the blup as its picture, over a cover of the blups' things,
  // and everything that can be set, as a list.
  const account = $derived(app.account);
  const ready = $derived(!!account?.vault && account.notes === "ready");
  // Opened with no internet: signed in, the account just not read yet.
  const offline = $derived(!account && app.accountFailed);
  const name = $derived(account?.signedIn && account.email ? account.email.split("@")[0] : "Blup");
  const cloud = $derived.by(() => {
    const a = t.account;
    if (offline) return t.cloud.offline;
    if (!account?.signedIn) return a.connectHint;
    if (account.notes === "new") return a.createPassword;
    if (account.notes === "locked") return a.typePassword;
    if (!account.vault) return a.pickVault;
    const parts = [];
    if (account.storageLimit) parts.push(fmt(t.common.ofTotal, { used: size(account.storageBytes), total: size(account.storageLimit) }));
    if (account.devicesMax) parts.push(fmt(account.devicesMax === 1 ? a.devices1 : a.devicesN, { n: account.devices, max: account.devicesMax }));
    return parts.join(" · ") || a.connected;
  });
  const vaults = $derived(plural(app.spaces.length, t.account.vaults1, t.account.vaultsN));
  const notes = $derived(plural(app.notes.length, t.account.notesIn1, t.account.notesInN, { vault: app.space?.title ?? t.account.yourVault }));

  // The language, from a sheet: the whole app changes in place, and the
  // pick is kept for the next time (settings.go).
  const langName = $derived(LANGS.find((l) => l.id === shown.lang)?.name ?? "");
  async function pickLang(lang: Lang) {
    if (lang !== shown.lang) {
      setLanguage(lang);
      try { await SettingsService.SetLanguage(lang); } catch (e) { app.say({ text: message(e) }); }
    }
    app.back();
  }
</script>

{#snippet row(Icon: typeof Cloud, label: string, hint: string, run?: () => void, bad = false, next = true)}
  {#if run}
    <button type="button" class="row" onclick={run}>
      <span class="ico"><Icon size={22} /></span>
      <span class="text"><b>{label}</b><small class={{ bad }}>{hint}</small></span>
      {#if next}<ChevronRight size={20} />{/if}
    </button>
  {:else}
    <div class="row">
      <span class="ico"><Icon size={22} /></span>
      <span class="text"><b>{label}</b><small>{hint}</small></span>
    </div>
  {/if}
{/snippet}

<div class="page">
  <div class="cover"><i></i></div>
  <div class="who">
    <span class="photo"><img src="/blups/blup.svg" alt="" /></span>
    <h1>{name}</h1>
    <p>{account?.signedIn ? [account.email, account.plan].filter(Boolean).join(" · ") : offline ? t.cloud.offline : t.account.localOnly}</p>
  </div>

  {@render row(Cloud, "Blup Cloud", offline ? cloud : app.syncError || cloud, () => app.go({ panel: "cloud" }), !!app.syncError)}
  {#if offline}{@render row(RefreshCw, app.syncing ? t.common.syncing : t.common.syncNow, t.cloud.unreachable, () => app.retry(), false, false)}
  {:else if ready}{@render row(RefreshCw, app.syncing ? t.common.syncing : t.common.syncNow, app.lastSync ? fmt(t.account.lastTime, { when: when(app.lastSync) }) : t.account.sendReceive, () => app.sync(), false, false)}{/if}
  {@render row(Library, t.account.vaults, vaults, () => app.go({ panel: "vaults" }))}

  <h2>{t.account.prefs}</h2>
  {@render row(Languages, t.account.language, langName, () => app.go({ panel: "lang" }))}
  <Settings />

  <h2>{t.account.onDevice}</h2>
  {@render row(FileText, t.account.notes, notes)}
</div>

{#if app.place.panel === "cloud"}<Sheet title="Blup Cloud" onclose={() => app.back()}><CloudCard /></Sheet>{/if}
{#if app.place.panel === "vaults"}<VaultPick />{/if}
{#if app.place.panel === "fresh"}<SyncFresh />{/if}
{#if app.place.panel === "lang"}
  <Sheet title={t.account.language} onclose={() => app.back()}>
    <p class="help">{t.account.languageHelp}</p>
    <div class="langs" role="radiogroup" aria-label={t.account.language}>
      {#each LANGS as l (l.id)}
        <button type="button" class="lang" role="radio" aria-checked={l.id === shown.lang} lang={l.id} onclick={() => pickLang(l.id)}>
          <span>{l.name}</span>{#if l.id === shown.lang}<span class="mark"><Check size={20} /></span>{/if}
        </button>
      {/each}
    </div>
  </Sheet>
{/if}

<style>
  .page { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; padding-block-end: 1.5rem; touch-action: pan-y; }
  .cover { position: relative; height: calc(11rem + var(--sat)); overflow: hidden; background: color-mix(in srgb, var(--blue) 12%, var(--bg)); }
  .cover i { position: absolute; inset: var(--sat) 0 0; background: var(--blue-text); opacity: .3; -webkit-mask: url("/blups/capa-conta.svg") center top / cover no-repeat; mask: url("/blups/capa-conta.svg") center top / cover no-repeat; }
  .who { display: grid; justify-items: center; gap: .15rem; margin-block-start: -4.75rem; padding: 0 1.25rem 1rem; text-align: center; }
  .photo { position: relative; display: grid; place-items: center; width: 9rem; height: 9rem; margin-block-end: .6rem; border: 4px solid var(--bg); border-radius: 50%; background: color-mix(in srgb, var(--blue) 16%, var(--panel)); }
  .photo img { width: 72%; height: 72%; object-fit: contain; }
  h1 { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 1.7rem; font-weight: 700; letter-spacing: -.02em; }
  .who p { max-width: 100%; overflow-wrap: anywhere; color: var(--muted); }
  h2 { padding: 1.1rem 1.25rem .35rem; font-size: .85rem; font-weight: 600; color: var(--blue-text); }
  .row { display: flex; align-items: center; gap: 1.1rem; width: 100%; min-height: 4rem; padding: .5rem 1.25rem; text-align: start; color: var(--muted); }
  button.row:active { background: var(--soft); }
  .ico { flex: none; display: grid; place-items: center; width: 1.6rem; }
  .text { flex: 1; min-width: 0; display: grid; gap: .1rem; }
  .text b { font-weight: 500; color: var(--text); }
  .text small { font-size: .85rem; line-height: 1.3; }
  .text small.bad { color: var(--red); }
  .help { margin-block-end: .5rem; font-size: .9rem; color: var(--muted); }
  .langs { display: grid; }
  .lang { display: flex; align-items: center; gap: .65rem; min-height: 3.1rem; text-align: start; font-weight: 500; color: var(--text); }
  .lang span:first-child { flex: 1; }
  .mark { display: grid; color: var(--blue-text); }
</style>
