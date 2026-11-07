<script module lang="ts">
  export const DOCS = "https://getblup.pages.dev/bento/docs/";
</script>

<script lang="ts">
  import { Page } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { HyprlandSetup, AutostartEnabled, SetAutostart, SetTray, SetToggleHotkey, Version, SetLanguage, Language, Connections, Disconnect, Agents, OpenURL, AIConnected, ConnectAI, OpenAIKeyPage, Voices, VoiceModels, PreviewVoice, LocalVoice, InstallLocalVoice, RemoveLocalVoice, GeneralPassword, SetGeneralPassword, BlupAccount, BlupAIUsage, OpenBlupCloud, CheckUpdate, PickBackupDir, BackupNow, LocalTranslate, InstallLocalTranslate, RemoveLocalTranslate, SetNotifications, ReminderEmails, AIConsent, SetAIConsent } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Connections as Conn, BlupAccount as Account, AIQuota, AIConsent as Consent, UpdateInfo } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Agent } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import type { ReminderUse } from "../bindings/github.com/gabriellsalesx/blup/internal/blupcloud/models";
  import { untrack } from "svelte";
  import Settings from "@lucide/svelte/icons/settings";
  import Palette from "@lucide/svelte/icons/palette";
  import PenLine from "@lucide/svelte/icons/pen-line";
  import KeyRound from "@lucide/svelte/icons/key-round";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import Sparkles from "@lucide/svelte/icons/sparkles";
  import Play from "@lucide/svelte/icons/play";
  import AudioLines from "@lucide/svelte/icons/audio-lines";
  import Pause from "@lucide/svelte/icons/pause";
  import Cloud from "@lucide/svelte/icons/cloud";
  import Info from "@lucide/svelte/icons/info";
  import CloudAccount from "./CloudAccount.svelte";
  import HistoryPurge from "./HistoryPurge.svelte";
  import { soundOut } from "./audio";
  import { Events } from "@wailsio/runtime";
  import X from "@lucide/svelte/icons/x";
  import Check from "@lucide/svelte/icons/check";
  import Copy from "@lucide/svelte/icons/copy";
  import ThemePicker from "./ThemePicker.svelte";
  import KeyRecorder from "./KeyRecorder.svelte";
  import { Clipboard } from "@wailsio/runtime";
  import { toast, toastError } from "./toast.svelte";
  import Keyboard from "@lucide/svelte/icons/keyboard";
  import LockKeyhole from "@lucide/svelte/icons/lock-keyhole";
  import { ACTIONS, keysOf, type Action } from "./keys";
  import Switch from "./Switch.svelte";
  import { look, setPref, spellLang, spell } from "./theme.svelte";
  import { t, setLanguage } from "./i18n";

  // The Studio's settings, like Obsidian's: a window with sections on the
  // left. They are the Studio's (and Blup's), not the open site's.
  // siteLang is the open site's language: the AI tab shows only its voices.
  let { onClose, onPublish, start = "general", siteLang = "" }: { onClose: () => void; onPublish: () => void; start?: string; siteLang?: string } = $props();
  const s = t.prefs;
  // The calendar's reminders: on unless turned off; in Max and Ultra also
  // by e-mail, with the month's use.
  let mailUse = $state<ReminderUse | null>(null);
  ReminderEmails().then((u) => (mailUse = u)).catch(() => {});
  async function setNotify(on: boolean) {
    look.notifications = on;
    try { await SetNotifications(on); } catch (e) { look.notifications = !on; }
  }

  // The sections, in groups: the Studio itself, what publishes the sites,
  // AI and voice. Blup Cloud (the account) sits above them, About below.
  const groups = $derived([
    { label: s.groupStudio, tabs: [
      { key: "general", label: s.general, Icon: Settings },
      { key: "look", label: s.look, Icon: Palette },
      { key: "editor", label: s.editor, Icon: PenLine },
      { key: "keys", label: t.keys.title, Icon: Keyboard },
      { key: "terminal", label: s.terminal, Icon: SquareTerminal },
    ] },
    { label: s.groupSites, tabs: [
      { key: "accounts", label: s.accounts, Icon: KeyRound },
      { key: "password", label: t.pw.general, Icon: LockKeyhole },
    ] },
    { label: s.groupAI, tabs: [
      { key: "ai", label: t.ai.title, Icon: Sparkles },
      { key: "narration", label: t.ai.narration, Icon: AudioLines },
    ] },
  ]);
  const tabs = $derived([{ key: "cloud", label: s.cloud, Icon: Cloud }, ...groups.flatMap((g) => g.tabs), { key: "about", label: s.about, Icon: Info }]);
  const k = t.keys;

  // The desktop: opening with the computer, the tray, the global shortcut.
  let autostart = $state(false);
  AutostartEnabled().then((v) => (autostart = v));
  let deskError = $state("");
  const deskRun = async (fn: () => Promise<void>) => { deskError = ""; try { await fn(); } catch (e) { deskError = String(e).replace(/^(Runtime)?Error: /, ""); } };
  const setAutostart = (v: boolean) => deskRun(async () => { await SetAutostart(v); autostart = v; toast(v ? t.toast.autostartOn : t.toast.autostartOff); });
  const setTray = (v: boolean) => deskRun(async () => { await SetTray(v); look.tray = v; toast(v ? t.toast.trayOn : t.toast.trayOff); });
  const setToggle = (keys: string) => deskRun(async () => { await SetToggleHotkey(keys); look.toggleKey = keys; hypr = await HyprlandSetup(keys); toast(keys ? t.toast.shortcutSet.replace("{keys}", keys) : t.toast.shortcutOff); });
  // On Hyprland (Omarchy) the shortcut is a binding in Hyprland's config,
  // which the Studio writes.
  let hypr = $state<{ on: boolean; file: string; line: string; auto: boolean } | null>(null);
  let lineCopied = $state(false);
  // The account's id, told when asking for help.
  let idCopied = $state(false);
  async function copyId(id: string) {
    try {
      await Clipboard.SetText(id);
      idCopied = true;
      setTimeout(() => (idCopied = false), 1500);
    } catch (e) { toastError(e); }
  }
  HyprlandSetup(look.toggleKey ?? "").then((h) => (hypr = h));
  // The Studio's own shortcuts.
  function setKey(a: Action, keys: string) {
    const all: Record<string, string> = {};
    for (const [x, v] of Object.entries(look.keys ?? {})) all[x] = v ?? "";
    // The same keys can't do two things: the other action loses them.
    for (const x of ACTIONS) if (x.key !== a && keysOf(all, x.key) === keys) all[x.key] = "";
    all[a] = keys;
    setPref("keys", all);
    toast(keys ? t.toast.shortcutSet.replace("{keys}", keys) : t.toast.shortcutOff);
  }
  function resetKeys() { setPref("keys", {}); }
  // The tab it opened on; the window changes it afterwards.
  let tab = $state(untrack(() => start));
  // Each section opens from its top: the list scrolled in a long one
  // (Shortcuts, Narration) left a short one looking empty.
  let bodyEl = $state<HTMLDivElement>();
  $effect(() => {
    void tab;
    if (bodyEl) bodyEl.scrollTop = 0;
  });
  let version = $state("");
  let lang = $state("");
  let conn = $state<Conn | null>(null);
  let agents = $state<Agent[]>([]);
  let error = $state("");
  Version().then((v) => (version = v));
  // A newer Studio: checked once a day (from the day's answer otherwise),
  // or now with the button.
  let update = $state<UpdateInfo | null>(null);
  let updateState = $state<"" | "checking" | "fail">("");
  CheckUpdate(false).then((u) => (update = u)).catch(() => {});
  // Who brings new versions when Blup doesn't (see app.UpdatesBy).
  const updatesBy = (by: string) => ({ msstore: s.updatesStore, flatpak: s.updatesFlatpak }[by] ?? s.updatesPackage);
  async function checkUpdate() {
    updateState = "checking";
    try { update = await CheckUpdate(true); updateState = ""; } catch { updateState = "fail"; }
  }
  Language().then((l) => (lang = l));
  Agents().then((a) => (agents = a ?? []));
  // Blup Cloud: the account (for every vault and site) and how much of the
  // plan is used.
  let account = $state<Account | null>(null);
  let quota = $state<AIQuota | null>(null);
  BlupAccount().then((a) => (account = a)).catch(() => {});
  $effect(() => {
    if (account?.signedIn && account.ai) BlupAIUsage().then((q) => (quota = q)).catch(() => (quota = null));
    else quota = null;
  });
  const size = (n: number) => (n < 1 << 30 ? `${Math.max(0, n / (1 << 20)).toFixed(n < 10 << 20 ? 1 : 0)} MB` : `${(n / (1 << 30)).toFixed(1)} GB`);
  const pct = (used: number, max: number) => (max > 0 ? Math.min(100, Math.round((used / max) * 100)) : 0);
  const day = (iso: string) => new Date(iso).toLocaleDateString(t.locale, { day: "numeric", month: "long" });
  const count = (n: number, max: number) => (max > 0 ? `${n} / ${max}` : `${n} · ${s.unlimited}`);
  // "1.4.0" reads v1.4.0; a build without a version ("dev") as it is.
  const shownVersion = $derived(/^\d/.test(version) ? `v${version}` : version);
  const loadConn = () => Connections().then((c) => (conn = c)).catch((e) => (error = String(e)));
  loadConn();

  // The general password of protected posts, for every site.
  let generalPw = $state(false);
  let newPw = $state("");
  let showPw = $state(false);
  GeneralPassword().then((v) => (generalPw = v)).catch(() => {});
  async function savePw(v: string) {
    try {
      await SetGeneralPassword(v);
      generalPw = !!v;
      newPw = "";
      toast(v ? t.pw.generalSaved : t.pw.generalRemoved);
    } catch (e) { toastError(e); }
  }
  // The daily copy of the notes: a folder, or off.
  let backingUp = $state(false);
  async function pickBackup() {
    const dir = await PickBackupDir().catch(() => "");
    if (dir) setPref("backupDir", dir);
  }
  async function backupNow() {
    backingUp = true;
    try { toast(s.backupDone.replace("{path}", await BackupNow())); } catch (e) { toastError(e); }
    backingUp = false;
  }
  // The new language shows at once, here and on every screen, without
  // leaving this one (the Go side changes the tray's menu too).
  async function changeLang(l: string) {
    try {
      await SetLanguage(l);
      setLanguage(l);
      lang = l;
      spell();
    } catch (e) { toastError(e); }
  }
  async function disconnect(account: string) {
    await Disconnect(account);
    loadConn();
    if (account === "openrouter") { aiOn = false; aiChanged(); toast(t.ai.disconnected); }
  }

  // Tells the rest of the window (the note's narration) that the voices
  // that can narrate changed.
  const aiChanged = () => window.dispatchEvent(new Event("blup:ai"));

  // The AI (OpenRouter): one key for every site, used by the narration.
  const ai = t.ai;
  let aiOn = $state<boolean | null>(null);
  let aiKey = $state("");
  let aiBusy = $state(false);
  let aiError = $state("");
  // The voices for each site language: the local ones (Kokoro), then
  // OpenRouter's. The chosen one is look.voices[lang]; without a choice
  // (or one that can't narrate now), the first that can.
  // OpenRouter voices are "<model>|<voice>"; older choices are a MAI-Voice
  // name alone. The user picks the model too (OpenRouter's list changes).
  type Voice = { id: string; name: string; engine: string; female: boolean; ready: boolean; sample: boolean };
  type Model = { id: string; name: string; perPost: number; recommended: boolean; voices: number };
  const DEFAULT_MODEL = "microsoft/mai-voice-2.1";
  // Blup Cloud's are "blup:<model>|<voice>" (older: "blup:<voice>", of the
  // recommended model).
  const norm = (id = "") => {
    if (id.startsWith("blup:")) return id.includes("|") ? id : `blup:${DEFAULT_MODEL}|${id.slice(5)}`;
    return id && !id.includes("|") && !id.startsWith("kokoro:") ? `${DEFAULT_MODEL}|${id}` : id;
  };
  // A default voice per language, for every site: a post is narrated in
  // the language picked then (Props), with that language's voice.
  const LANGS = [["pt", "Português"], ["en", "English"], ["es", "Español"]] as const;
  // It opens on the open site's language (the voices a post of it gets).
  const firstLang = () => (LANGS.some(([l]) => l === siteLang) ? siteLang : t.lang);
  let voiceLang = $state<string>(firstLang());
  let voices = $state<Record<string, Voice[]>>({ pt: [], en: [], es: [] });
  // The voice models the staff offers (Blup Cloud), in its order: with
  // one's own OpenRouter key and through the plan.
  let models = $state<Record<string, { own: Model[]; plan: Model[] }>>({ pt: { own: [], plan: [] }, en: { own: [], plan: [] }, es: { own: [], plan: [] } });
  // The model chosen for OpenRouter and for Blup Cloud, each from the
  // voice chosen (else the recommended one).
  const modelOf = (l: string, blup: boolean) => { const c = norm(look.voices?.[l]); return c.startsWith("blup:") === blup && c.includes("|") ? c.replace(/^blup:/, "").split("|")[0] : DEFAULT_MODEL; };
  let model = $state<Record<string, string>>({ pt: modelOf("pt", false), en: modelOf("en", false), es: modelOf("es", false) });
  let blupModel = $state<Record<string, string>>({ pt: modelOf("pt", true), en: modelOf("en", true), es: modelOf("es", true) });
  const reloadVoices = (l: string) => Voices(l, model[l], blupModel[l]).then((v) => (voices[l] = v ?? [])).catch(() => {});
  const loadVoices = () => {
    for (const [l] of LANGS) {
      reloadVoices(l);
      VoiceModels(l).then((m) => {
        models[l] = { own: m?.own ?? [], plan: m?.plan ?? [] };
        // A model the staff no longer offers: the recommended one, else the
        // list's first (as the Studio's voices already do).
        const fallback = (list: Model[]) => (list.find((x) => x.recommended) ?? list[0]).id;
        if (models[l].own.length && !models[l].own.some((x) => x.id === model[l])) model[l] = fallback(models[l].own);
        if (models[l].plan.length && !models[l].plan.some((x) => x.id === blupModel[l])) blupModel[l] = fallback(models[l].plan);
      }).catch(() => {});
    }
  };
  loadVoices();
  const pickModel = (l: string, id: string, blup = false) => { if (blup) blupModel[l] = id; else model[l] = id; reloadVoices(l); };
  const price = (m: Model) => (m.perPost > 0 ? `US$ ${m.perPost.toFixed(2).replace(".", t.lang === "pt" ? "," : ".")}/post` : lv.otherPrice);
  AIConnected().then((v) => (aiOn = v)).catch(() => (aiOn = false));
  const chosen = (l: string) => {
    const list = voices[l];
    const pick = list.find((v) => v.id === norm(look.voices?.[l]));
    // Without a choice, the local voice: never one in the cloud by itself.
    return pick?.ready ? pick.id : list.find((v) => v.engine === "local" && v.ready)?.id ?? "";
  };
  const setVoice = (lang: string, id: string) => { setPref("voices", { ...(look.voices ?? {}), [lang]: id }); toast(ai.voiceSet); };

  // The translation on the computer (Firefox's): downloaded on demand.
  const lt = t.ltr;
  let ltState = $state<{ installed: boolean; bytes: number } | null>(null);
  let ltBusy = $state<{ done: number; total: number } | null>(null);
  let ltError = $state("");
  LocalTranslate().then((v) => (ltState = v)).catch(() => {});
  const offLt = Events.On("translateInstall", (ev) => { if (ltBusy) ltBusy = ev.data; });
  $effect(() => () => offLt());
  async function installLt() {
    ltError = "";
    ltBusy = { done: 0, total: ltState?.bytes ?? 0 };
    try { await InstallLocalTranslate(); ltState = await LocalTranslate(); toast(lt.ready); }
    catch (e) { ltError = String(e).replace(/^(Runtime)?Error: /, ""); }
    ltBusy = null;
  }
  async function removeLt() {
    await RemoveLocalTranslate().catch((e) => (ltError = String(e)));
    ltState = await LocalTranslate();
    toast(lt.removed);
  }
  // Who translates: the person's choice (on the computer, OpenRouter or
  // Blup Cloud); nothing chosen is the computer when installed, else
  // nobody (the Studio never picks the cloud by itself).
  const ltWho = $derived(["local", "openrouter", "blup"].includes(look.translateAI) ? look.translateAI : ltState?.installed && look.translateAI === "" ? "local" : "");
  // What the person accepted sending to each AI in the cloud.
  const aip = t.aip;
  let consent = $state<Consent | null>(null);
  AIConsent().then((c) => (consent = c)).catch(() => {});
  async function revoke(dest: "openrouter" | "blup") {
    try { await SetAIConsent(dest, false); consent = await AIConsent(); toast(aip.revoked); } catch (e) { toastError(e); }
  }

  // The local voice: downloaded on demand, then ready to use.
  const lv = t.localVoice;
  let local = $state<{ supported: boolean; installed: boolean; bytes: number } | null>(null);
  let installing = $state<{ done: number; total: number; step: string } | null>(null);
  let localError = $state("");
  LocalVoice().then((v) => (local = v));
  const offInstall = Events.On("voiceInstall", (ev) => { if (installing) installing = ev.data; });
  $effect(() => () => offInstall());
  const mb = (b: number) => Math.round(b / 1e6);
  async function installLocal() {
    localError = "";
    installing = { done: 0, total: local?.bytes ?? 0, step: "runtime" };
    try {
      await InstallLocalVoice();
      local = await LocalVoice();
      loadVoices();
      aiChanged();
      toast(lv.ready);
    } catch (e) { localError = String(e).replace(/^(Runtime)?Error: /, ""); }
    installing = null;
  }
  async function removeLocal() {
    await RemoveLocalVoice().catch((e) => (localError = String(e)));
    local = await LocalVoice();
    loadVoices();
    aiChanged();
    toast(lv.removed);
  }

  // Samples bundled with the Studio (public/voices): one sentence each.
  const sample = (id: string) => `/voices/${id.startsWith("kokoro:") ? id.replace(":", "-") : (id.split("|")[1] ?? id.replace(/^blup:/, "")).split(":")[0].toLowerCase()}.mp3`;
  // Voices of other models have no recorded sample: OpenRouter makes one
  // (a fraction of a cent, once; the Studio keeps it).
  async function sampleBytes(v: Voice, lang: string): Promise<ArrayBuffer> {
    if (v.sample) {
      const res = await fetch(sample(v.id));
      if (res.ok) return res.arrayBuffer();
      // No recorded file after all: an OpenRouter voice gets one made.
      if (v.engine !== "openrouter") throw new Error(res.statusText || String(res.status));
    }
    const b64 = await PreviewVoice(lang, v.id);
    const bin = atob(b64 ?? "");
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out.buffer;
  }
  // The file is read whole, decoded, and only then played (Web Audio): a
  // media element decodes as it plays, and on Linux (WebKitGTK) its first
  // moments stuttered; it also can't stream the app's wails:// addresses.
  // The start waits a moment for the sound output to wake up.
  let playing = $state("");
  let source: AudioBufferSourceNode | null = null;
  let startAt = 0;
  let length = 0;
  // How much of the sample has played (0 to 1), for the ring around ▶.
  let played = $state(0);
  // Until it starts (a sample OpenRouter makes takes a few seconds) the
  // ring turns, so the ▶ doesn't look stuck.
  let sounding = $state(false);
  let frame = 0;
  const tick = () => {
    if (source && length > 0) played = Math.min(1, Math.max(0, (soundOut().currentTime - startAt) / length));
    frame = requestAnimationFrame(tick);
  };
  const stopSample = () => {
    if (source) {
      source.onended = null;
      try { source.stop(); } catch {}
      source.disconnect();
    }
    source = null;
    sounding = false;
    cancelAnimationFrame(frame);
    played = 0;
  };
  async function listen(v: Voice, lang: string) {
    const id = v.id;
    const again = playing === id;
    stopSample();
    playing = "";
    if (again) return;
    playing = id;
    try {
      const audio = soundOut();
      const [bytes] = await Promise.all([sampleBytes(v, lang), audio.resume()]);
      const buffer = await audio.decodeAudioData(bytes);
      if (playing !== id) return; // another one was tapped meanwhile
      const src = audio.createBufferSource();
      src.buffer = buffer;
      src.connect(audio.destination);
      src.onended = () => { if (playing === id) { played = 1; setTimeout(() => { if (playing === id) { playing = ""; played = 0; } }, 250); } };
      startAt = audio.currentTime + 0.08;
      length = buffer.duration;
      source = src;
      src.start(startAt);
      sounding = true;
      frame = requestAnimationFrame(tick);
    } catch (e) {
      if (playing === id) playing = "";
      toast(`${lv.noSample} ${String(e).replace(/^(Runtime)?Error: /, "")}`, { kind: "error" });
    }
  }
  $effect(() => () => stopSample());
  async function connectAI() {
    aiBusy = true; aiError = "";
    try { await ConnectAI(aiKey); aiOn = true; aiKey = ""; toast(ai.connected); loadVoices(); aiChanged(); }
    catch (e) { aiError = String(e).includes("invalid key") ? ai.badKey : String(e).replace(/^(Runtime)?Error: /, ""); }
    aiBusy = false;
  }
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onClose()} />

<div class="shade" role="presentation" onclick={onClose}></div>
<div class="modal" role="dialog" aria-modal="true" aria-label={s.title}>
  <nav>
    <!-- The Blup Cloud account, for every vault and site. -->
    <button class="whoami" class:on={tab === "cloud"} onclick={() => (tab = "cloud")}>
      <span class="avatar" aria-hidden="true">{#if account?.signedIn}{(account.email || "B")[0].toUpperCase()}{:else}<Cloud size={15} />{/if}</span>
      <span class="who-text">
        {#if account?.signedIn}<b>{account.email || s.cloud}</b><small>{s.cloud} · {account.label}</small>
        {:else}<b>{s.signInNav}</b><small>{s.signInNavHelp}</small>{/if}
      </span>
    </button>
    {#each groups as g (g.label)}
      <small class="muted grouphead">{g.label}</small>
      {#each g.tabs as { key, label, Icon } (key)}
        <button class:on={tab === key} onclick={() => (tab = key)}><Icon size={16} />{label}</button>
      {/each}
    {/each}
    <span class="navfill"></span>
    <button class="appver" class:on={tab === "about"} onclick={() => (tab = "about")}><Info size={14} />Blup Studio {shownVersion}{#if update?.newer}<span class="dot" title={s.updateNew.replace("{v}", update.latest)}></span>{/if}</button>
  </nav>
  <div class="side">
    <!-- The title and the X stay put; only what's below scrolls. -->
    <header class="head">
      <h2>{tabs.find((x) => x.key === tab)?.label}</h2>
      <button class="close" title={t.attach.close} aria-label={t.attach.close} onclick={onClose}><X size={18} /></button>
    </header>
    <div class="body" bind:this={bodyEl}>
    {#if tab === "general"}
      <div class="item"><Switch label={k.autostart} help={k.autostartHelp} checked={autostart} onchange={setAutostart} /></div>
      <div class="item"><Switch label={k.tray} help={k.trayHelp} checked={!!look.tray} onchange={setTray} /></div>
      <div class="item"><Switch label={s.notify} help={s.notifyHelp} checked={look.notifications} onchange={setNotify} /></div>
      {#if mailUse && mailUse.limit > 0 && look.notifications}
        <!-- A passkey only account has no e-mail: the reminders can't go by e-mail. -->
        {#if mailUse.emailMissing}<p class="bad note">{s.notifyNoEmail}</p>
        {:else}<p class="muted note">{s.notifyMail.replace("{used}", String(mailUse.used)).replace("{limit}", String(mailUse.limit))}</p>{/if}
      {/if}
      {#if deskError}<p class="bad">{deskError}</p>{/if}
      <div class="item">
        <div class="text"><b>{s.backup}</b><small>{look.backupDir || s.backupOff}</small><small>{s.backupHelp}</small></div>
        <span class="keyrow">
          {#if look.backupDir}
            <button class="btn ghost" disabled={backingUp} onclick={backupNow}>{s.backupNow}</button>
            <button class="btn ghost" onclick={() => setPref("backupDir", "")}>{s.backupStop}</button>
          {:else}<button class="btn ghost" onclick={pickBackup}>{s.backupPick}</button>{/if}
        </span>
      </div>
      <div class="item">
        <div class="text"><b>{s.lang}</b><small>{s.langHelp}</small></div>
        <select value={lang} onchange={(e) => changeLang(e.currentTarget.value)}>
          <option value="pt">Português</option>
          <option value="en">English</option>
          <option value="es">Español</option>
        </select>
      </div>
    {:else if tab === "cloud"}
      <p class="muted lead">{s.cloudLead}</p>
      <div class="cloudbox"><CloudAccount bind:account brief /></div>
      {#if account?.signedIn && account.id}
        <div class="item">
          <div class="text"><b>{s.accountId}</b><small>{s.accountIdHelp}</small><code class="acct-id">{account.id}</code></div>
          <button class="btn ghost copy" onclick={() => account && copyId(account.id)}>{#if idCopied}<Check size={14} />{s.idCopied}{:else}<Copy size={14} />{s.copyId}{/if}</button>
        </div>
      {/if}
      {#if account?.signedIn}
        <h3>{s.usage}</h3>
        <div class="item column">
          <div class="row"><b>{s.storageUse}</b><small class="muted">{t.cloud.storage.replace("{used}", size(account.storageBytes)).replace("{total}", size(account.storageMB * 1048576))}</small></div>
          <div class="dl" role="progressbar" aria-label={s.storageUse} aria-valuemin="0" aria-valuemax="100" aria-valuenow={pct(account.storageBytes, account.storageMB * 1048576)}>
            <span class:full={pct(account.storageBytes, account.storageMB * 1048576) >= 90} style:width="{Math.max(1, pct(account.storageBytes, account.storageMB * 1048576))}%"></span>
          </div>
        </div>
        <HistoryPurge bind:account />
        <div class="item column">
          <div class="row"><b>{s.aiUse}</b>{#if quota}<small class="muted">{s.aiUsed.replace("{n}", String(quota.percent))}</small>{/if}</div>
          {#if account.ai && quota}
            <div class="dl" role="progressbar" aria-label={s.aiUse} aria-valuemin="0" aria-valuemax="100" aria-valuenow={quota.percent}>
              <span class:full={quota.percent >= 90} style:width="{Math.max(1, quota.percent)}%"></span>
            </div>
            <small class="muted">{s.aiResets.replace("{date}", day(quota.resetsAt))}{#if quota.left < 100 - quota.percent}. {s.aiWeek.replace("{n}", String(quota.left))}{/if}</small>
          {:else if !account.ai}
            <small class="muted">{s.aiNone}</small>
          {/if}
        </div>
        <div class="item"><div class="text"><b>{s.sitesUse}</b></div><span class="muted num">{count(account.sites, account.sitesMax)}</span></div>
        <div class="item"><div class="text"><b>{s.devicesUse}</b></div><span class="muted num">{count(account.devices, account.devicesMax)}</span></div>
        <div class="item">
          <span></span>
          <span class="keyrow">
            <button class="btn ghost" onclick={() => OpenBlupCloud(Page.PagePlan)}>{s.plans}</button>
            <button class="btn ghost" onclick={() => OpenBlupCloud(Page.PageAccount)}>{s.manage}</button>
          </span>
        </div>
      {/if}
    {:else if tab === "about"}
      <p class="muted lead">{s.aboutLead}</p>
      <div class="item">
        <div class="text"><b>{s.version}</b><small>Blup Studio {shownVersion}</small></div>
        <button class="btn ghost" onclick={() => OpenURL("https://getblup.pages.dev/bento/changelog/")}>{s.news}</button>
      </div>
      <div class="item">
        <div class="text"><b>{s.update}</b>
          <small class:warn={update?.newer}>{version === "dev" ? s.updateDev : update?.updatesBy ? updatesBy(update.updatesBy) : updateState === "checking" ? s.updateChecking : updateState === "fail" ? s.updateFail : update?.newer ? s.updateNew.replace("{v}", update.latest) : update?.latest ? s.updateCurrent : ""}</small>
        </div>
        <!-- A store or the system's packages update it: nothing to look for here. -->
        {#if update?.newer && !update.updatesBy}<button class="btn" onclick={() => OpenURL(update!.url)}>{s.updateGet}</button>
        {:else if version !== "dev" && !update?.updatesBy}<button class="btn ghost" disabled={updateState === "checking"} onclick={checkUpdate}>{s.updateCheck}</button>{/if}
      </div>
      <div class="item">
        <div class="text"><b>{s.help}</b><small>{s.helpHelp}</small></div>
        <button class="btn ghost" onclick={() => OpenURL(DOCS)}>{s.openDocs}</button>
      </div>
      <div class="item">
        <div class="text"><b>{s.code}</b><small>{s.codeHelp}</small></div>
        <button class="btn ghost" onclick={() => OpenURL("https://github.com/gabriellsalesx/blup")}>GitHub</button>
      </div>
    {:else if tab === "look"}
      <div class="item">
        <div class="text"><b>{s.theme}</b><small>{s.themeHelp}</small></div>
        <ThemePicker />
      </div>
      <div class="item"><Switch label={s.railNames} help={s.railNamesHelp} checked={look.railNames} onchange={(v) => setPref("railNames", v)} /></div>
      <div class="item">
        <div class="text"><b>{s.font}</b><small>{look.editorFont}px</small></div>
        <input type="range" min="13" max="22" value={look.editorFont} oninput={(e) => setPref("editorFont", +e.currentTarget.value)} />
      </div>
      <div class="item">
        <div class="text"><b>{s.width}</b><small>{s.widthHelp}</small></div>
        <div class="seg">
          <button class:on={look.editorWidth !== "wide"} onclick={() => setPref("editorWidth", "narrow")}>{s.narrow}</button>
          <button class:on={look.editorWidth === "wide"} onclick={() => setPref("editorWidth", "wide")}>{s.wide}</button>
        </div>
      </div>
    {:else if tab === "editor"}
      <div class="item"><Switch label={s.spell} help={s.spellHelp} checked={look.spellcheck} onchange={(v) => setPref("spellcheck", v)} /></div>
      <div class="item">
        <div class="text"><b>{s.spellLang}</b><small>{s.spellLangHelp}</small></div>
        <select value={spellLang()} disabled={!look.spellcheck} onchange={(e) => setPref("spellLang", e.currentTarget.value)}>
          <option value="pt">Português (Brasil)</option>
          <option value="en">English (US)</option>
          <option value="es">Español</option>
        </select>
      </div>
      <div class="item column">
        <div class="text"><b>{s.myWords}</b><small>{s.myWordsHelp}</small></div>
        <div class="chips">
          {#each look.words ?? [] as w (w)}<span class="chip">{w}<button title={t.sec.cancel} onclick={() => { setPref("words", (look.words ?? []).filter((x) => x !== w)); }}><X size={12} /></button></span>{/each}
          {#if !(look.words ?? []).length}<small class="muted">{s.noWords}</small>{/if}
        </div>
      </div>
      <div class="item"><Switch label={s.preview} help={s.previewHelp} checked={!look.previewOff} onchange={(v) => setPref("previewOff", !v)} /></div>
    {:else if tab === "accounts"}
      <p class="muted lead">{s.accountsHelp}</p>
      {#if error}<p class="bad">{error}</p>{/if}
      {#if conn}
        <div class="item">
          <div class="text"><b>GitHub</b><small>{conn.github ? s.connectedAs.replace("{user}", conn.github) : s.notConnected}</small></div>
          {#if conn.github}<button class="btn ghost" onclick={() => disconnect("github")}>{s.disconnect}</button>
          {:else}<button class="btn ghost" onclick={onPublish}>{s.connect}</button>{/if}
        </div>
        <div class="item">
          <div class="text"><b>Cloudflare</b><small>{conn.cloudflare ? s.connected : s.notConnected}</small></div>
          {#if conn.cloudflare}<button class="btn ghost" onclick={() => disconnect("cloudflare")}>{s.disconnect}</button>
          {:else}<button class="btn ghost" onclick={onPublish}>{s.connect}</button>{/if}
        </div>
      {/if}
    {:else if tab === "ai"}
      <p class="muted lead">{ai.lead}</p>
      <h3>{s.aiCloud}</h3>
      <div class="item">
        <div class="text"><small>{account?.ai && quota ? s.aiCloudOn.replace("{n}", String(quota.percent)) : s.aiCloudOff}</small></div>
        <button class="btn ghost" onclick={() => (tab = "cloud")}>{s.aiCloudSee}</button>
      </div>
      <h3>{lv.title}</h3>
      <p class="muted lead">{lv.lead.replace("{mb}", String(mb(local?.bytes || 110e6)))}</p>
      <div class="item column">
        <div class="row">
          <div class="text"><b>Kokoro</b><small>{!local ? "" : !local.supported ? lv.unsupported : local.installed ? lv.installed : lv.notInstalled}</small></div>
          {#if local?.supported && !installing}
            {#if local.installed}<button class="btn ghost" onclick={removeLocal}>{lv.remove}</button>
            {:else}<button class="btn" onclick={installLocal}>{lv.install.replace("{mb}", String(mb(local.bytes)))}</button>{/if}
          {/if}
        </div>
        {#if installing}
          <div class="dl" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={installing.total ? Math.round((installing.done / installing.total) * 100) : 0}>
            <span style:width={`${installing.total ? Math.max(2, (installing.done / installing.total) * 100) : 2}%`}></span>
          </div>
          <small class="muted">{lv.steps[installing.step as keyof typeof lv.steps] ?? lv.steps.runtime} {installing.total ? `${mb(installing.done)}/${mb(installing.total)} MB` : ""}</small>
        {/if}
        {#if localError}<p class="bad">{localError}</p>{/if}
      </div>
      <h3>OpenRouter</h3>
      {#if aiOn}
        <div class="item">
          <div class="text"><small>{ai.on}</small></div>
          <button class="btn ghost" onclick={() => disconnect("openrouter")}>{s.disconnect}</button>
        </div>
      {:else if aiOn === false}
        <div class="item column">
          <small class="muted">{ai.openrouterLead}</small>
          <form class="keyform" onsubmit={(e) => { e.preventDefault(); if (aiKey.trim()) connectAI(); }}>
            <input type="password" bind:value={aiKey} placeholder={ai.keyPh} autocomplete="off" spellcheck="false" aria-label={ai.keyPh} />
            <button class="btn" disabled={aiBusy || !aiKey.trim()}>{aiBusy ? ai.checking : s.connect}</button>
          </form>
          {#if aiError}<p class="bad">{aiError}</p>{/if}
          <small class="muted">{ai.noKey} <button class="link" onclick={() => OpenAIKeyPage()}>{ai.makeKey}</button></small>
        </div>
      {/if}
      <h3>{lt.title}</h3>
      <p class="muted lead">{lt.lead}</p>
      <div class="item column">
        <div class="row">
          <div class="text"><b>{lt.localName}</b><small>{!ltState ? "" : ltState.installed ? lt.installed : lt.notInstalled}</small></div>
          {#if ltState && !ltBusy}
            {#if ltState.installed}<button class="btn ghost" onclick={removeLt}>{lt.remove}</button>
            {:else}<button class="btn" onclick={installLt}>{lt.install.replace("{mb}", String(mb(ltState.bytes)))}</button>{/if}
          {/if}
        </div>
        <small class="muted">{lt.localHelp.replace("{mb}", String(mb(ltState?.bytes || 100e6)))}</small>
        {#if ltBusy}
          <div class="dl" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={ltBusy.total ? Math.round((ltBusy.done / ltBusy.total) * 100) : 0}>
            <span style:width={`${ltBusy.total ? Math.max(2, (ltBusy.done / ltBusy.total) * 100) : 2}%`}></span>
          </div>
          <small class="muted">{lt.downloading} {ltBusy.total ? `${mb(ltBusy.done)}/${mb(ltBusy.total)} MB` : ""}</small>
        {/if}
        {#if ltError}<p class="bad">{ltError}</p>{/if}
      </div>
      <div class="item who">
        <div class="text"><b>{lt.who}</b><small>{ltWho ? lt.whoHelp : aip.whoNone}</small></div>
        <div class="seg">
          <button class:on={ltWho === "local"} disabled={!ltState?.installed} onclick={() => setPref("translateAI", "local")}>{lt.whoLocal}</button>
          <button class:on={ltWho === "openrouter"} disabled={!aiOn} onclick={() => setPref("translateAI", "openrouter")}>{aip.whoOpenRouter}</button>
          <button class:on={ltWho === "blup"} disabled={!account?.ai} onclick={() => setPref("translateAI", "blup")}>{aip.whoBlup}</button>
        </div>
      </div>
      <h3>{aip.consentTitle}</h3>
      <p class="muted lead">{aip.consentHelp}</p>
      {#each [["openrouter", aip.whoOpenRouter], ["blup", aip.whoBlup]] as const as [dest, name] (dest)}
        <div class="item">
          <div class="text"><b>{name}</b><small>{consent?.[dest] ? aip.accepted : aip.notAsked}</small></div>
          {#if consent?.[dest]}<button class="btn ghost" onclick={() => revoke(dest)}>{aip.revoke}</button>{/if}
        </div>
      {/each}
    {:else if tab === "narration"}
      <p class="muted lead">{ai.narrationHelp}</p>
      {#if local && !local.installed && aiOn === false}
        <p class="muted lead">{ai.narrationNeed} <button class="link" onclick={() => (tab = "ai")}>{ai.openAI}</button></p>
      {/if}
      <div class="seg langs" role="tablist" aria-label={ai.voiceLangs}>
        {#each LANGS as [l, name] (l)}<button role="tab" aria-selected={voiceLang === l} class:on={voiceLang === l} onclick={() => (voiceLang = l)}>{name}</button>{/each}
      </div>
      {#each LANGS.filter(([l]) => l === voiceLang) as [l, name] (l)}
        {@const label = ai.voiceFor.replace("{lang}", name)}
        <div class="voices" role="radiogroup" aria-label={label}>
          <b>{label}</b>
          {#each [["local", lv.groupLocal], ["blup", lv.groupBlup], ["openrouter", lv.groupOpenRouter]] as [engine, group] (engine)}
            <!-- The computer's voices always; Blup Cloud's and OpenRouter's once
                 connected (signed in on a plan with AI, or the key saved). -->
            {#if engine === "local" ? voices[l].some((v) => v.engine === "local") : voices[l].some((v) => v.engine === engine && v.ready)}
              <small class="muted group">{group}</small>
              {#if engine === "openrouter" && models[l].own.length}
                <label class="model">
                  <span>{lv.model}</span>
                  <select value={model[l]} onchange={(e) => pickModel(l, e.currentTarget.value)}>
                    {#each models[l].own as m (m.id)}<option value={m.id}>{m.name.replace(/^[^:]+:\s*/, "")}{m.recommended ? ` (${lv.recommended})` : ""} · {price(m)}</option>{/each}
                  </select>
                </label>
              {:else if engine === "blup" && models[l].plan.length}
                <!-- Without the price: the plan pays (a dearer one uses the
                     month's AI allowance faster). -->
                <label class="model">
                  <span>{lv.model}</span>
                  <select value={blupModel[l]} onchange={(e) => pickModel(l, e.currentTarget.value, true)}>
                    {#each models[l].plan as m (m.id)}<option value={m.id}>{m.name.replace(/^[^:]+:\s*/, "")}{m.recommended ? ` (${lv.recommended})` : ""}</option>{/each}
                  </select>
                </label>
                <small class="muted model-help">{lv.blupModelHelp}</small>
              {/if}
              {#each voices[l].filter((v) => v.engine === engine) as v (v.id)}
                <div class="voice" class:off={!v.ready} class:on={chosen(l) === v.id}>
                  <label>
                    <input type="radio" name="voice-{l}" checked={chosen(l) === v.id} disabled={!v.ready} onchange={() => setVoice(l, v.id)} />
                    <span>{v.name}</span>
                    {#if engine === "local"}<small class="muted">{v.female ? lv.female : lv.male}</small>{/if}
                    {#if !v.ready}<small class="muted">{engine === "local" ? lv.needInstall : engine === "blup" ? lv.needPlan : lv.needKey}</small>{/if}
                  </label>
                  {#if v.ready}<button class="icon-btn listen" class:playing={playing === v.id} class:loading={playing === v.id && !sounding} title={playing === v.id ? lv.stop : v.sample ? lv.listen : lv.listenMake} aria-label="{playing === v.id ? lv.stop : lv.listen}: {v.name}" onclick={() => listen(v, l)}>
                    <!-- The ring fills as the sample plays. -->
                    <svg class="ring" viewBox="0 0 36 36" aria-hidden="true">
                      <circle class="track" cx="18" cy="18" r="16" />
                      {#if playing === v.id}<circle class="fill" cx="18" cy="18" r="16" pathLength="100" stroke-dasharray="{played * 100} 100" />{/if}
                    </svg>
                    {#if playing === v.id}<Pause size={13} />{:else}<Play size={13} />{/if}
                  </button>{/if}
                </div>
              {/each}
            {/if}
          {/each}
          {#if !voices[l].some((v) => v.engine !== "local" && v.ready)}
            <p class="muted more">{lv.moreVoices} <button class="link" onclick={() => (tab = "ai")}>{ai.openAI}</button></p>
          {/if}
        </div>
      {/each}
    {:else if tab === "password"}
      <p class="muted lead">{t.pw.generalHelp}</p>
      <div class="item column">
        <div class="row">
          <div class="text"><b>{t.pw.general}</b><small>{generalPw ? t.pw.generalSet : t.pw.notSet}</small></div>
          {#if generalPw}<button class="btn ghost" onclick={() => savePw("")}>{t.remove}</button>{/if}
        </div>
        <div class="row">
          <input type={showPw ? "text" : "password"} bind:value={newPw} aria-label={t.pw.general} autocomplete="new-password" />
          <button class="btn ghost" onclick={() => (showPw = !showPw)}>{showPw ? t.wel.hide : t.wel.show}</button>
          <button class="btn" disabled={!newPw} onclick={() => savePw(newPw)}>{t.save}</button>
        </div>
      </div>
    {:else if tab === "keys"}
      <h3>{k.globalTitle}</h3>
      <div class="item">
        <div class="text"><b>{k.toggle}</b><small>{look.tray ? k.toggleHelpTray : k.toggleHelp}</small></div>
        <span class="keyrow">
          <KeyRecorder value={look.toggleKey ?? ""} onChange={setToggle} />
          {#if look.toggleKey}<button class="icon-btn" title={k.clear} onclick={() => setToggle("")}><X size={15} /></button>{/if}
        </span>
      </div>
      {#if hypr?.on && hypr.auto}
        <p class="muted lead">{k.hyprland.replace("{file}", hypr.file.replace(/^.*\/\.config/, "~/.config"))}</p>
        {#if hypr.line}<code class="line">{hypr.line}</code>{/if}
      {:else if hypr?.on}
        <!-- Hyprland that isn't Omarchy: its config is the user's own. -->
        <p class="muted lead">{k.hyprlandManual}</p>
        {#if hypr.line}
          <code class="line">{hypr.line}</code>
          <button class="btn ghost" onclick={async () => { await Clipboard.SetText(hypr!.line); lineCopied = true; }}>{lineCopied ? k.copied : k.copyLine}</button>
        {/if}
      {:else}
        <p class="muted lead">{k.wayland}</p>
      {/if}
      {#if deskError}<p class="bad">{deskError}</p>{/if}
      <h3>{k.appTitle}</h3>
      {#each ACTIONS as a (a.key)}
        <div class="item">
          <div class="text"><b>{k.actions[a.key]}</b>{#if keysOf(look.keys, a.key) !== a.keys}<small>{k.default}: {a.keys}</small>{/if}</div>
          <span class="keyrow">
            <KeyRecorder value={keysOf(look.keys, a.key)} onChange={(v) => setKey(a.key, v)} />
            {#if keysOf(look.keys, a.key)}<button class="icon-btn" title={k.clear} onclick={() => setKey(a.key, "")}><X size={15} /></button>{/if}
          </span>
        </div>
      {/each}
      <div class="item"><span></span><button class="btn ghost" onclick={resetKeys}>{k.reset}</button></div>
      <h3>{k.editorTitle}</h3>
      <p class="muted lead">{k.editorHelp}</p>
      {#each k.editorKeys as [label, keys] (label)}
        <div class="item fixed"><div class="text"><b>{label}</b></div><span class="fixed-keys">{#each keys.split("+") as part, i (i)}<kbd>{part}</kbd>{/each}</span></div>
      {/each}
    {:else}
      <div class="item">
        <div class="text"><b>{s.font}</b><small>{look.termFont}px</small></div>
        <input type="range" min="11" max="20" value={look.termFont} oninput={(e) => setPref("termFont", +e.currentTarget.value)} />
      </div>
      <h3>{s.agents}</h3>
      <p class="muted lead">{s.agentsHelp}</p>
      {#each agents as a (a.key)}
        <div class="item">
          <div class="text"><b>{a.name}</b><small>{a.installed ? s.installed : s.missing}</small></div>
          {#if a.installed}<span class="ok"><Check size={18} /></span>
          {:else}<button class="btn ghost" onclick={() => OpenURL(a.site)}>{s.install}</button>{/if}
        </div>
      {/each}
    {/if}
    </div>
  </div>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .45); }
  .modal { position: fixed; z-index: 41; inset: 0; margin: auto; width: min(52rem, calc(100% - 2rem)); height: min(36rem, calc(100% - 2rem)); display: grid; grid-template-columns: 13rem minmax(0, 1fr); border: 1px solid var(--line); border-radius: .85rem; overflow: hidden; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  nav { display: flex; flex-direction: column; gap: .15rem; padding: 1rem .6rem; overflow-y: auto; border-inline-end: 1px solid var(--line); background: var(--bg); }
  nav small { padding: .2rem .6rem .5rem; font-size: .72rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
  nav button { display: flex; align-items: center; gap: .55rem; padding: .45rem .6rem; border-radius: .4rem; font-size: .875rem; color: var(--muted); text-align: start; }
  nav button:hover { background: var(--soft); color: var(--text); }
  nav button.on { background: var(--soft); color: var(--text); font-weight: 600; }
  nav .grouphead { padding: .9rem .6rem .3rem; }
  .whoami { gap: .6rem; padding: .5rem .55rem; margin-block-end: .1rem; border: 1px solid var(--line); border-radius: .6rem; }
  .avatar { flex: none; display: grid; place-items: center; width: 1.9rem; height: 1.9rem; border-radius: 50%; background: var(--blue); color: #fff; font-size: .85rem; font-weight: 700; }
  .who-text { display: grid; min-width: 0; }
  .who-text b { color: var(--text); font-size: .82rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .who-text small { padding: 0; font-size: .72rem; font-weight: 500; text-transform: none; letter-spacing: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .navfill { flex: 1; }
  nav .appver { gap: .4rem; font-size: .75rem; }
  .appver .dot { width: .45rem; height: .45rem; border-radius: 50%; background: var(--blue); }
  .text small.warn { color: var(--blue); font-weight: 600; }
  .acct-id { font-family: ui-monospace, monospace; font-size: .8rem; user-select: all; overflow-wrap: anywhere; }
  .copy { display: inline-flex; align-items: center; gap: .35rem; flex-shrink: 0; }
  .cloudbox { display: grid; gap: .75rem; padding-block: .5rem 1rem; border-block-end: 1px solid var(--line); font-size: .9rem; }
  .dl span.full { background: var(--red); }
  .num { font-size: .85rem; font-variant-numeric: tabular-nums; }
  .side { display: grid; grid-template-rows: auto minmax(0, 1fr); min-height: 0; }
  .head { display: flex; align-items: center; gap: 1rem; padding: 1.1rem 1rem .6rem 2rem; border-block-end: 1px solid var(--line); }
  .head h2 { flex: 1; margin: 0; }
  .body { overflow: auto; padding: 1rem 2rem 2rem; }
  .close { flex: none; display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  h2 { font-size: 1.15rem; font-weight: 700; margin-block-end: .75rem; }
  h3 { font-size: .95rem; font-weight: 700; margin-block: 1.25rem .25rem; }
  .lead { font-size: .85rem; margin-block-end: .5rem; }
  .item { display: flex; align-items: center; justify-content: space-between; gap: 1.5rem; padding-block: .85rem; border-block-end: 1px solid var(--line); }
  .note { padding-block: .6rem 0; font-size: .8rem; }
  .item > :global(.switch) { flex: 1; }
  /* The buttons keep their text on one line; the text beside them wraps. */
  .item > :global(.btn), .item > .keyrow, .keyrow > :global(.btn) { flex-shrink: 0; white-space: nowrap; }
  .item > .text { flex: 1; min-width: 0; }
  .text { display: grid; gap: .15rem; font-size: .9rem; }
  .text small { color: var(--muted); font-size: .8rem; }
  select { width: auto; }
  .keyform { display: flex; gap: .5rem; }
  .item.column .row { display: flex; align-items: center; justify-content: space-between; gap: 1.5rem; }
  .dl { height: 6px; border-radius: 999px; background: var(--soft); overflow: hidden; }
  .dl span { display: block; height: 100%; border-radius: inherit; background: var(--blue); transition: width .3s ease; }
  .voices { display: grid; gap: .15rem; padding-block: .85rem; border-block-end: 1px solid var(--line); }
  .voices > b { margin-block-end: .25rem; }
  .voices .group { margin-block: .5rem .1rem; font-size: .72rem; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; }
  .voice { display: flex; align-items: center; justify-content: space-between; gap: .75rem; padding: .3rem .5rem; border-radius: .45rem; }
  .voice.on { background: var(--soft); }
  .link { color: var(--blue); font-weight: 600; }
  .link:hover { text-decoration: underline; }
  .model { display: flex; align-items: center; gap: .6rem; margin-block: .2rem .4rem; padding-inline: .5rem; font-size: .85rem; }
  .model select { flex: 1; min-width: 0; }
  .voice.off label { opacity: .55; }
  .voice label { display: flex; align-items: center; gap: .6rem; flex: 1; min-width: 0; cursor: pointer; }
  .voice.off label { cursor: default; }
  .voice label small { font-size: .78rem; }
  .listen { position: relative; width: 2rem; height: 2rem; display: grid; place-items: center; border-radius: 50%; }
  .listen .ring { position: absolute; inset: 0; width: 100%; height: 100%; transform: rotate(-90deg); pointer-events: none; }
  .listen .ring circle { fill: none; stroke-width: 2.5; }
  .listen .ring .track { stroke: var(--line); }
  .listen .ring .fill { stroke: var(--blue); stroke-linecap: round; }
  .listen.playing { color: var(--blue); }
  .more { margin-top: .5rem; font-size: .85rem; }
  .model-help { font-size: .78rem; }
  .listen.loading .ring { animation: ring-turn .9s linear infinite; }
  .listen.loading .ring .fill { stroke-dasharray: 22 78; }
  @keyframes ring-turn { from { transform: rotate(-90deg); } to { transform: rotate(270deg); } }
  .keyform input { flex: 1; font-family: ui-monospace, monospace; }
  input[type="range"] { width: 11rem; accent-color: var(--blue); }
  .seg { display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--line); border-radius: .6rem; }
  .seg button { padding: .35rem .7rem; border-radius: .45rem; font-size: .85rem; font-weight: 600; color: var(--muted); white-space: nowrap; }
  .seg button.on { background: var(--soft); color: var(--text); }
  .seg.langs { margin-block: .25rem .5rem; }
  /* Who translates: the choice keeps each option on one line, beside the
     help while it fits, below it otherwise. */
  .item.who { flex-wrap: wrap; gap: .75rem 1.5rem; }
  .item.who .text { flex: 1 1 16rem; }
  .item.who .seg { flex: none; }
  .item.column { flex-direction: column; align-items: stretch; gap: .6rem; }
  .chips { display: flex; flex-wrap: wrap; gap: .35rem; }
  .chip { display: inline-flex; align-items: center; gap: .25rem; padding: .2rem .3rem .2rem .6rem; border: 1px solid var(--line); border-radius: 999px; font-size: .8rem; }
  .chip button { display: inline-grid; place-items: center; padding: .15rem; border-radius: 999px; color: var(--muted); }
  .chip button:hover { background: var(--soft); color: var(--text); }
  .line { display: block; padding: .5rem .7rem; border-radius: .4rem; background: var(--soft); font-size: .75rem; word-break: break-all; }
  .keyrow { display: inline-flex; align-items: center; gap: .3rem; }
  .item.fixed { padding-block: .55rem; }
  .fixed-keys { display: inline-flex; gap: .25rem; }
  .fixed-keys kbd { padding: .1rem .4rem; border: 1px solid var(--line); border-block-end-width: 2px; border-radius: .3rem; background: var(--soft); font-family: inherit; font-size: .78rem; font-weight: 600; }
  .ok { color: var(--green, #16a34a); display: inline-grid; }
</style>
