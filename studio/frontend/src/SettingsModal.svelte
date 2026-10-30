<script module lang="ts">
  export const DOCS = "https://getblup.pages.dev/bento/docs/";
</script>

<script lang="ts">
  import { HyprlandSetup, AutostartEnabled, SetAutostart, SetTray, SetToggleHotkey, Version, SetLanguage, Language, Connections, Disconnect, Agents, OpenURL, AIConnected, ConnectAI, OpenAIKeyPage, Voices } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Connections as Conn } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Agent } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Settings from "@lucide/svelte/icons/settings";
  import Palette from "@lucide/svelte/icons/palette";
  import PenLine from "@lucide/svelte/icons/pen-line";
  import KeyRound from "@lucide/svelte/icons/key-round";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import Sparkles from "@lucide/svelte/icons/sparkles";
  import X from "@lucide/svelte/icons/x";
  import Check from "@lucide/svelte/icons/check";
  import ThemePicker from "./ThemePicker.svelte";
  import KeyRecorder from "./KeyRecorder.svelte";
  import { Clipboard } from "@wailsio/runtime";
  import { toast } from "./toast.svelte";
  import Keyboard from "@lucide/svelte/icons/keyboard";
  import { ACTIONS, keysOf, type Action } from "./keys";
  import Switch from "./Switch.svelte";
  import { look, setPref, spellLang } from "./theme.svelte";
  import { t } from "./i18n";

  // The Studio's settings, like Obsidian's: a window with sections on the
  // left. They are the Studio's (and Blup's), not the open site's.
  let { onClose, onPublish, start = "general" }: { onClose: () => void; onPublish: () => void; start?: string } = $props();
  const s = t.prefs;

  const tabs = [
    { key: "general", label: s.general, Icon: Settings },
    { key: "look", label: s.look, Icon: Palette },
    { key: "editor", label: s.editor, Icon: PenLine },
    { key: "accounts", label: s.accounts, Icon: KeyRound },
    { key: "ai", label: t.ai.title, Icon: Sparkles },
    { key: "terminal", label: s.terminal, Icon: SquareTerminal },
    { key: "keys", label: t.keys.title, Icon: Keyboard },
  ];
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
  let tab = $state(start);
  let version = $state("");
  let lang = $state("");
  let conn = $state<Conn | null>(null);
  let agents = $state<Agent[]>([]);
  let error = $state("");
  Version().then((v) => (version = v));
  Language().then((l) => (lang = l));
  Agents().then((a) => (agents = a ?? []));
  const loadConn = () => Connections().then((c) => (conn = c)).catch((e) => (error = String(e)));
  loadConn();

  async function changeLang(l: string) {
    await SetLanguage(l);
    location.reload();
  }
  async function disconnect(account: string) {
    await Disconnect(account);
    loadConn();
    if (account === "openrouter") { aiOn = false; toast(t.ai.disconnected); }
  }

  // The AI (OpenRouter): one key for every site, used by the narration.
  const ai = t.ai;
  let aiOn = $state<boolean | null>(null);
  let aiKey = $state("");
  let aiBusy = $state(false);
  let aiError = $state("");
  // The voice for each site language (the default first in each list).
  let voices = $state<Record<string, { id: string; name: string }[]>>({ pt: [], en: [] });
  const loadVoices = () => { for (const l of ["pt", "en"]) Voices(l).then((v) => (voices[l] = v ?? [])).catch(() => {}); };
  AIConnected().then((v) => { aiOn = v; if (v) loadVoices(); }).catch(() => (aiOn = false));
  const setVoice = (lang: string, id: string) => { setPref("voices", { ...(look.voices ?? {}), [lang]: id }); toast(ai.voiceSet); };
  async function connectAI() {
    aiBusy = true; aiError = "";
    try { await ConnectAI(aiKey); aiOn = true; aiKey = ""; toast(ai.connected); loadVoices(); }
    catch (e) { aiError = String(e).includes("invalid key") ? ai.badKey : String(e).replace(/^(Runtime)?Error: /, ""); }
    aiBusy = false;
  }
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onClose()} />

<div class="shade" role="presentation" onclick={onClose}></div>
<div class="modal" role="dialog" aria-modal="true" aria-label={s.title}>
  <nav>
    <small class="muted">{s.title}</small>
    {#each tabs as { key, label, Icon } (key)}
      <button class:on={tab === key} onclick={() => (tab = key)}><Icon size={16} />{label}</button>
    {/each}
  </nav>
  <div class="side">
    <!-- The title and the X stay put; only what's below scrolls. -->
    <header class="head">
      <h2>{tabs.find((x) => x.key === tab)?.label}</h2>
      <button class="close" title={t.attach.close} aria-label={t.attach.close} onclick={onClose}><X size={18} /></button>
    </header>
    <div class="body">
    {#if tab === "general"}
      <div class="item"><Switch label={k.autostart} help={k.autostartHelp} checked={autostart} onchange={setAutostart} /></div>
      <div class="item"><Switch label={k.tray} help={k.trayHelp} checked={!!look.tray} onchange={setTray} /></div>
      {#if deskError}<p class="bad">{deskError}</p>{/if}
      <div class="item">
        <div class="text"><b>{s.version}</b><small>Blup Studio {version}</small></div>
        <button class="btn ghost" onclick={() => OpenURL("https://getblup.pages.dev/bento/changelog/")}>{s.news}</button>
      </div>
      <div class="item">
        <div class="text"><b>{s.lang}</b><small>{s.langHelp}</small></div>
        <select value={lang} onchange={(e) => changeLang(e.currentTarget.value)}>
          <option value="pt">Português</option>
          <option value="en">English</option>
        </select>
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
      <div class="item">
        <div class="text"><b>OpenRouter</b><small>{aiOn ? ai.on : ai.off}</small></div>
        {#if aiOn}<button class="btn ghost" onclick={() => disconnect("openrouter")}>{s.disconnect}</button>{/if}
      </div>
      {#if aiOn === false}
        <div class="item column">
          <div class="text"><b>{ai.step1}</b><small>{ai.step1Help}</small></div>
          <button class="btn ghost start" onclick={() => OpenAIKeyPage()}>{ai.openPage}</button>
          <div class="text"><b>{ai.step2}</b></div>
          <form class="keyform" onsubmit={(e) => { e.preventDefault(); if (aiKey.trim()) connectAI(); }}>
            <input type="password" bind:value={aiKey} placeholder="sk-or-v1-…" autocomplete="off" spellcheck="false" aria-label={ai.step2} />
            <button class="btn" disabled={aiBusy || !aiKey.trim()}>{aiBusy ? ai.checking : s.connect}</button>
          </form>
          {#if aiError}<p class="bad">{aiError}</p>{/if}
          <small class="muted">{ai.keyNote}</small>
        </div>
      {/if}
      <h3>{ai.narration}</h3>
      <p class="muted lead">{ai.narrationHelp}</p>
      {#each [["pt", ai.voicePt], ["en", ai.voiceEn]] as [l, label] (l)}
        <div class="item">
          <div class="text"><b>{label}</b><small>{ai.voiceHelp}</small></div>
          <select value={look.voices?.[l] || voices[l][0]?.id || ""} disabled={!voices[l].length} onchange={(e) => setVoice(l, e.currentTarget.value)}>
            {#each voices[l] as v (v.id)}<option value={v.id}>{v.name}</option>{/each}
          </select>
        </div>
      {/each}
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
  nav { display: flex; flex-direction: column; gap: .15rem; padding: 1rem .6rem; border-inline-end: 1px solid var(--line); background: var(--bg); }
  nav small { padding: .2rem .6rem .5rem; font-size: .72rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
  nav button { display: flex; align-items: center; gap: .55rem; padding: .45rem .6rem; border-radius: .4rem; font-size: .875rem; color: var(--muted); text-align: start; }
  nav button:hover { background: var(--soft); color: var(--text); }
  nav button.on { background: var(--soft); color: var(--text); font-weight: 600; }
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
  .item > :global(.switch) { flex: 1; }
  .text { display: grid; gap: .15rem; font-size: .9rem; }
  .text small { color: var(--muted); font-size: .8rem; }
  select { width: auto; }
  .start { align-self: flex-start; }
  .keyform { display: flex; gap: .5rem; }
  .keyform input { flex: 1; font-family: ui-monospace, monospace; }
  input[type="range"] { width: 11rem; accent-color: var(--blue); }
  .seg { display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--line); border-radius: .6rem; }
  .seg button { padding: .35rem .7rem; border-radius: .45rem; font-size: .85rem; font-weight: 600; color: var(--muted); }
  .seg button.on { background: var(--soft); color: var(--text); }
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
