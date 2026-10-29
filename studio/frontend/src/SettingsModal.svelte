<script module lang="ts">
  export const DOCS = "https://getblup.pages.dev/bento/docs/";
</script>

<script lang="ts">
  import { Version, SetLanguage, Language, Connections, Disconnect, Agents, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Connections as Conn } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import type { Agent } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Settings from "@lucide/svelte/icons/settings";
  import Palette from "@lucide/svelte/icons/palette";
  import PenLine from "@lucide/svelte/icons/pen-line";
  import KeyRound from "@lucide/svelte/icons/key-round";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import X from "@lucide/svelte/icons/x";
  import Check from "@lucide/svelte/icons/check";
  import ThemePicker from "./ThemePicker.svelte";
  import Switch from "./Switch.svelte";
  import { look, setPref } from "./theme.svelte";
  import { t } from "./i18n";

  // The Studio's settings, like Obsidian's: a window with sections on the
  // left. They are the Studio's (and Blup's), not the open site's.
  let { onClose, onPublish }: { onClose: () => void; onPublish: () => void } = $props();
  const s = t.prefs;

  const tabs = [
    { key: "general", label: s.general, Icon: Settings },
    { key: "look", label: s.look, Icon: Palette },
    { key: "editor", label: s.editor, Icon: PenLine },
    { key: "accounts", label: s.accounts, Icon: KeyRound },
    { key: "terminal", label: s.terminal, Icon: SquareTerminal },
  ];
  let tab = $state("general");
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
  <div class="body">
    <button class="close" title={t.sec.cancel} onclick={onClose}><X size={18} /></button>
    {#if tab === "general"}
      <h2>{s.general}</h2>
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
      <h2>{s.look}</h2>
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
      <h2>{s.editor}</h2>
      <div class="item"><Switch label={s.spell} help={s.spellHelp} checked={look.spellcheck} onchange={(v) => setPref("spellcheck", v)} /></div>
      <div class="item"><Switch label={s.preview} help={s.previewHelp} checked={!look.previewOff} onchange={(v) => setPref("previewOff", !v)} /></div>
    {:else if tab === "accounts"}
      <h2>{s.accounts}</h2>
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
    {:else}
      <h2>{s.terminal}</h2>
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

<style>
  .shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .45); }
  .modal { position: fixed; z-index: 41; inset: 0; margin: auto; width: min(52rem, calc(100% - 2rem)); height: min(36rem, calc(100% - 2rem)); display: grid; grid-template-columns: 13rem minmax(0, 1fr); border: 1px solid var(--line); border-radius: .85rem; overflow: hidden; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  nav { display: flex; flex-direction: column; gap: .15rem; padding: 1rem .6rem; border-inline-end: 1px solid var(--line); background: var(--bg); }
  nav small { padding: .2rem .6rem .5rem; font-size: .72rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
  nav button { display: flex; align-items: center; gap: .55rem; padding: .45rem .6rem; border-radius: .4rem; font-size: .875rem; color: var(--muted); text-align: start; }
  nav button:hover { background: var(--soft); color: var(--text); }
  nav button.on { background: var(--soft); color: var(--text); font-weight: 600; }
  .body { position: relative; overflow: auto; padding: 1.5rem 2rem 2rem; }
  .close { position: absolute; inset-block-start: .8rem; inset-inline-end: .8rem; display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  h2 { font-size: 1.15rem; font-weight: 700; margin-block-end: .75rem; }
  h3 { font-size: .95rem; font-weight: 700; margin-block: 1.25rem .25rem; }
  .lead { font-size: .85rem; margin-block-end: .5rem; }
  .item { display: flex; align-items: center; justify-content: space-between; gap: 1.5rem; padding-block: .85rem; border-block-end: 1px solid var(--line); }
  .item > :global(.switch) { flex: 1; }
  .text { display: grid; gap: .15rem; font-size: .9rem; }
  .text small { color: var(--muted); font-size: .8rem; }
  select { width: auto; }
  input[type="range"] { width: 11rem; accent-color: var(--blue); }
  .seg { display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--line); border-radius: .6rem; }
  .seg button { padding: .35rem .7rem; border-radius: .45rem; font-size: .85rem; font-weight: 600; color: var(--muted); }
  .seg button.on { background: var(--soft); color: var(--text); }
  .ok { color: var(--green, #16a34a); display: inline-grid; }
</style>
