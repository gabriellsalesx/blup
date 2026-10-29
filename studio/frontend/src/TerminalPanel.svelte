<script lang="ts">
  import { onDestroy, tick } from "svelte";
  import { Events, Clipboard } from "@wailsio/runtime";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import "@xterm/xterm/css/xterm.css";
  import { Agents, OpenTerminal, WriteTerminal, ResizeTerminal, CloseTerminal, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Agent } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Sparkles from "@lucide/svelte/icons/sparkles";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
  import Plus from "@lucide/svelte/icons/plus";
  import X from "@lucide/svelte/icons/x";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import { t } from "./i18n";
  import { look } from "./theme.svelte";

  // Terminals in tabs, in the notes folder. The AI agents installed get a
  // button each; the agent starts inside a shell, so the tab stays when it
  // quits.
  // onClose only hides the panel: the terminals keep running.
  let { onClose, visible = true }: { onClose: () => void; visible?: boolean } = $props();
  const tt = t.term;

  type Tab = { uid: number; agent: string; id: string; title: string; ai: boolean; term: Terminal; fit: FitAddon; el: HTMLDivElement | null; ended: boolean };
  let tabs = $state<Tab[]>([]);
  let active = $state("");
  let agents = $state<Agent[]>([]);
  let error = $state("");
  let adding = $state(false); // the AI menu (the sparkles)
  let host: HTMLDivElement;

  Agents().then((a) => (agents = a ?? []));
  const installed = $derived(agents.filter((a) => a.installed));
  const missing = $derived(agents.filter((a) => !a.installed));

  // The theme's colors, resolved: the variables hold light-dark(...), which
  // the terminal can't read, so a probe element turns them into rgb().
  function resolve(value: string) {
    const probe = document.createElement("span");
    probe.style.color = value;
    document.body.append(probe);
    const rgb = getComputedStyle(probe).color;
    probe.remove();
    return rgb;
  }
  function colors() {
    const background = resolve("var(--panel)");
    const foreground = resolve("var(--text)");
    const [r, g, b] = (background.match(/\d+/g) ?? ["255", "255", "255"]).map(Number);
    const dark = r * 0.299 + g * 0.587 + b * 0.114 < 128;
    const palette = dark
      ? { black: "#3f3f46", red: "#f87171", green: "#4ade80", yellow: "#facc15", blue: "#60a5fa", magenta: "#c084fc", cyan: "#22d3ee", white: "#e4e4e7", brightBlack: "#71717a", brightRed: "#fca5a5", brightGreen: "#86efac", brightYellow: "#fde047", brightBlue: "#93c5fd", brightMagenta: "#d8b4fe", brightCyan: "#67e8f9", brightWhite: "#fafafa" }
      : { black: "#18181b", red: "#b91c1c", green: "#15803d", yellow: "#a16207", blue: "#1d4ed8", magenta: "#7e22ce", cyan: "#0e7490", white: "#52525b", brightBlack: "#71717a", brightRed: "#dc2626", brightGreen: "#16a34a", brightYellow: "#ca8a04", brightBlue: "#2563eb", brightMagenta: "#9333ea", brightCyan: "#0891b2", brightWhite: "#27272a" };
    return { background, foreground, cursor: foreground, cursorAccent: background, selectionBackground: "rgba(59,130,246,.4)", selectionInactiveBackground: "rgba(59,130,246,.4)", ...palette };
  }

  let uids = 0;
  // at: where the tab goes (the end, or the place of a restarted one).
  async function open(agent = "", title = tt.shell, at = tabs.length) {
    error = "";
    const term = new Terminal({ fontFamily: "ui-monospace, 'JetBrains Mono', Menlo, monospace", fontSize: look.termFont || 13, cursorBlink: true, theme: colors(), allowProposedApi: true });
    const fit = new FitAddon();
    term.loadAddon(fit);
    const uid = ++uids;
    tabs.splice(at, 0, { uid, agent, id: "", title, ai: !!agent, term, fit, el: null, ended: false });
    const tab = () => tabs.find((x) => x.uid === uid)!;
    await tick();
    const el = document.querySelector<HTMLDivElement>(`[data-term="${uid}"]`)!;
    tab().el = el;
    active = "pending:" + uid;
    term.open(el);
    fit.fit();
    try {
      const id = await OpenTerminal(agent, term.cols, term.rows);
      tab().id = id;
      active = id;
      term.onData((d) => WriteTerminal(id, d).catch(() => {}));
      term.onResize(({ cols, rows }) => ResizeTerminal(id, cols, rows).catch(() => {}));
      // Ctrl+Shift+C and Ctrl+Shift+V copy and paste, like other terminals.
      term.attachCustomKeyEventHandler((e) => {
        if (e.type !== "keydown" || !e.ctrlKey || !e.shiftKey || (e.code !== "KeyC" && e.code !== "KeyV")) return true;
        e.preventDefault();
        if (e.code === "KeyC") copy(tab()); else paste(tab());
        return false;
      });
      term.focus();
    } catch (e) {
      error = String(e).replace(/^(Runtime)?Error: /, "");
      tabs = tabs.filter((x) => x.uid !== uid);
      term.dispose();
    }
  }

  function close(tab: Tab) {
    if (tab.id) CloseTerminal(tab.id);
    tab.term.dispose();
    tabs = tabs.filter((x) => x.uid !== tab.uid);
    if (active === tab.id) active = tabs.at(-1)?.id ?? "";
  }
  const closeMany = (list: Tab[]) => list.forEach(close);

  // The right-click menu, on a tab or inside the terminal.
  let menu = $state<{ uid: number; x: number; y: number; inside: boolean } | null>(null);
  let renaming = $state(0); // uid of the tab being renamed
  const menuTab = $derived(menu ? tabs.find((x) => x.uid === menu!.uid) : undefined);
  const menuAt = $derived(menuTab ? tabs.indexOf(menuTab) : -1);
  function showMenu(e: MouseEvent, tab: Tab, inside: boolean) {
    e.preventDefault();
    e.stopPropagation();
    adding = false;
    if (inside) active = tab.id;
    menu = { uid: tab.uid, x: Math.min(e.clientX, innerWidth - 260), y: e.clientY, inside };
  }
  // Runs a menu item on its tab (taken before the menu closes).
  function act(fn: (tab: Tab, at: number) => void) { const tab = menuTab!, at = menuAt; menu = null; fn(tab, at); }
  async function restart(tab: Tab) {
    const at = tabs.indexOf(tab);
    close(tab);
    await open(tab.agent, tab.title, at);
  }
  async function paste(tab: Tab) { const text = await Clipboard.Text().catch(() => ""); if (text) tab.term.paste(text); tab.term.focus(); }
  function copy(tab: Tab) { const sel = tab.term.getSelection(); if (sel) Clipboard.SetText(sel); tab.term.focus(); }
  // Keep the menu inside the window (it opens upward near the bottom).
  let ctxEl = $state<HTMLDivElement>();
  $effect(() => {
    if (!ctxEl || !menu) return;
    const r = ctxEl.getBoundingClientRect();
    if (r.bottom > innerHeight - 8) ctxEl.style.top = Math.max(8, innerHeight - r.height - 8) + "px";
  });
  const focusSelect = (el: HTMLInputElement) => { el.focus(); el.select(); };

  const decode = (b64: string) => Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
  const off = Events.On("term", (ev: any) => {
    const d = ev.data as { id: string; data: string; exit: boolean };
    const tab = tabs.find((x) => x.id === d.id);
    if (!tab) return;
    if (d.exit) { tab.ended = true; tab.term.write(`\r\n\x1b[2m${tt.ended}\x1b[0m\r\n`); }
    else tab.term.write(decode(d.data));
  });

  // Follow the panel's size.
  const ro = new ResizeObserver(() => tabs.find((x) => x.id === active)?.fit.fit());
  $effect(() => { if (host) ro.observe(host); });
  $effect(() => { const tab = tabs.find((x) => x.id === active); if (tab && visible && !renaming) tick().then(() => { tab.fit.fit(); tab.term.focus(); }); });

  // The text size follows the settings.
  $effect(() => { const size = look.termFont || 13; for (const tab of tabs) { tab.term.options.fontSize = size; tab.fit.fit(); } });

  onDestroy(() => { off(); ro.disconnect(); for (const tab of tabs) close(tab); });

  // The first tab: a plain terminal.
  tick().then(() => { if (!tabs.length) open(); });
</script>

<svelte:window onclick={() => { adding = false; menu = null; }} onkeydowncapture={(e) => { if (e.key === "Escape" && menu) { e.stopPropagation(); menu = null; } }} />

<section class="term-panel">
  <!-- Tabs on the left; on the right, a new terminal (+), the AIs
       (sparkles) and minimize. -->
  <header>
    <div class="tabs">
      {#each tabs as tab (tab.uid)}
        <div class="tab" class:on={tab.id === active} role="tab" tabindex="-1" aria-selected={tab.id === active} oncontextmenu={(e) => showMenu(e, tab, false)}>
          {#if renaming === tab.uid}
            <input class="rename" type="text" value={tab.title} use:focusSelect
              onkeydown={(e) => { if (e.key === "Enter") { tab.title = e.currentTarget.value.trim() || tab.title; renaming = 0; tab.term.focus(); } if (e.key === "Escape") renaming = 0; }}
              onblur={(e) => { tab.title = e.currentTarget.value.trim() || tab.title; renaming = 0; }} />
          {:else}
            <button class="pick" onclick={() => (active = tab.id)} ondblclick={() => (renaming = tab.uid)}>
              <span class="ico">{#if tab.ai}<Sparkles size={12} />{:else}<SquareTerminal size={12} />{/if}</span>{tab.title}{tab.ended ? " ·" : ""}
            </button>
          {/if}
          <button class="x" title={tt.closeTab} onclick={() => close(tab)}><X size={12} /></button>
        </div>
      {/each}
    </div>
    <div class="tools">
      <button class="tool" title={tt.newShell} onclick={() => open()}><Plus size={17} /></button>
      <div class="ai">
        <button class="tool" class:on={adding} title={tt.ai} onclick={(e) => { e.stopPropagation(); adding = !adding; }}><Sparkles size={16} /></button>
        {#if adding}
          <div class="add-menu" role="menu">
            {#each installed as a (a.key)}
              <button onclick={() => { adding = false; open(a.key, a.name); }}><Sparkles size={15} />{a.name}</button>
            {/each}
            {#if !installed.length}
              <small class="muted">{tt.noAgents}</small>
              {#each missing.slice(0, 3) as a (a.key)}<button class="link" onclick={() => OpenURL(a.site)}>{a.name}</button>{/each}
            {/if}
          </div>
        {/if}
      </div>
      <button class="tool" title={tt.close} onclick={onClose}><ChevronDown size={17} /></button>
    </div>
  </header>
  {#if error}<p class="bad err">{error}</p>{/if}
  <div class="host" bind:this={host}>
    {#each tabs as tab (tab.uid)}
      <div class="xterm-box" data-term={tab.uid} role="presentation" class:hidden={tab.id !== active && active !== "pending:" + tab.uid} oncontextmenu={(e) => showMenu(e, tab, true)}></div>
    {/each}
  </div>
  {#if menu && menuTab}
    {@const tab = menuTab}
    <div class="ctx" bind:this={ctxEl} role="menu" tabindex="-1" style:left="{menu.x}px" style:top="{menu.y}px" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
      <div class="head"><span class="ico">{#if tab.ai}<Sparkles size={12} />{:else}<SquareTerminal size={12} />{/if}</span><b>{tab.title}</b><small class="muted">{tab.ended ? tt.stateEnded : tt.stateRunning}</small></div>
      {#if menu.inside}
        <button disabled={!tab.term.hasSelection()} onclick={() => act((tab, at) => copy(tab))}>{tt.copy}<kbd>Ctrl+Shift+C</kbd></button>
        <button onclick={() => act((tab, at) => paste(tab))}>{tt.paste}<kbd>Ctrl+Shift+V</kbd></button>
        <button onclick={() => act((tab, at) => { tab.term.selectAll(); tab.term.focus(); })}>{tt.selectAll}</button>
        <button onclick={() => act((tab, at) => { tab.term.clear(); tab.term.focus(); })}>{tt.clear}</button>
        <hr />
      {/if}
      <button onclick={() => act((tab, at) => (renaming = tab.uid))}>{tt.rename}</button>
      <button onclick={() => act((tab, at) => open(tab.agent, tab.title))}>{tt.duplicate}</button>
      <button onclick={() => act((tab, at) => restart(tab))}>{tt.restart}</button>
      <hr />
      <button onclick={() => act((tab, at) => close(tab))}>{tt.closeTab}</button>
      <button disabled={tabs.length < 2} onclick={() => act((tab, at) => closeMany(tabs.filter((x) => x !== tab)))}>{tt.closeOthers}</button>
      <button disabled={menuAt < 1} onclick={() => act((tab, at) => closeMany(tabs.slice(0, at)))}>{tt.closeLeft}</button>
      <button disabled={menuAt >= tabs.length - 1} onclick={() => act((tab, at) => closeMany(tabs.slice(at + 1)))}>{tt.closeRight}</button>
      <button onclick={() => act((tab, at) => closeMany([...tabs]))}>{tt.closeAll}</button>
    </div>
  {/if}
</section>

<style>
  /* A column: the terminal's area takes all the room left (with or without
     an error line above it). */
  .term-panel { display: flex; flex-direction: column; height: 100%; background: var(--panel); border-block-start: 1px solid var(--line); }
  header { display: flex; align-items: stretch; min-height: 2.4rem; border-block-end: 1px solid var(--line); }
  .tabs { display: flex; flex-wrap: wrap; min-width: 0; flex: 1; }
  .tab { display: flex; align-items: center; gap: .2rem; padding-inline: .7rem .4rem; border-inline-end: 1px solid var(--line); font-size: .82rem; color: var(--muted); }
  .tab.on { background: var(--soft); color: var(--text); }
  .pick { display: flex; align-items: center; gap: .5rem; padding-block: .45rem; white-space: nowrap; }
  .ico { display: inline-grid; place-items: center; width: 1.15rem; height: 1.15rem; border-radius: .25rem; background: var(--line); color: var(--text); }
  .x { display: inline-grid; place-items: center; padding: .25rem; border-radius: .3rem; color: var(--muted); opacity: 0; }
  .tab:hover .x, .tab.on .x { opacity: 1; }
  .x:hover { color: var(--text); background: var(--line); }
  .tools { display: flex; align-items: center; gap: .1rem; padding-inline: .5rem; border-inline-start: 1px solid var(--line); }
  .tool { display: grid; place-items: center; width: 1.9rem; height: 1.9rem; border-radius: .35rem; color: var(--muted); }
  .tool:hover, .tool.on { color: var(--text); background: var(--soft); }
  .ai { position: relative; }
  .add-menu { position: absolute; z-index: 10; inset-block-start: calc(100% + .3rem); inset-inline-end: 0; display: grid; min-width: 12rem; padding: .3rem; border: 1px solid var(--line); border-radius: .55rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .22); }
  .add-menu button { display: flex; align-items: center; gap: .55rem; padding: .42rem .6rem; border-radius: .35rem; font-size: .85rem; text-align: start; white-space: nowrap; }
  .add-menu button :global(svg) { color: var(--muted); }
  .add-menu button:hover { background: var(--soft); }
  .add-menu small { padding: .3rem .6rem; font-size: .75rem; }
  .link { color: var(--blue); }
  .rename { width: 9rem; padding: .2rem .4rem; font-size: .82rem; color: var(--text); }
  .ctx { position: fixed; z-index: 50; display: grid; min-width: 15rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .28); }
  .ctx .head { display: grid; grid-template-columns: auto 1fr; align-items: center; gap: .15rem .5rem; margin: .1rem .1rem .3rem; padding: .5rem .6rem; border: 1px solid var(--line); border-radius: .45rem; font-size: .85rem; }
  .ctx .head small { grid-column: 2; font-size: .75rem; }
  .ctx button { display: flex; align-items: center; justify-content: space-between; gap: 1.5rem; padding: .4rem .6rem; border-radius: .35rem; font-size: .85rem; text-align: start; }
  .ctx button:hover:not(:disabled) { background: var(--soft); }
  .ctx button:disabled { opacity: .4; cursor: default; }
  .ctx kbd { font-family: inherit; font-size: .75rem; color: var(--muted); }
  .ctx hr { border: 0; border-block-start: 1px solid var(--line); margin: .25rem .2rem; }
  .err { padding: .4rem .8rem; font-size: .8rem; }
  .host { position: relative; flex: 1; min-height: 0; }
  .xterm-box { position: absolute; inset: .4rem .2rem .2rem .6rem; }
  .xterm-box.hidden { visibility: hidden; }
  .host :global(.xterm-viewport) { background: transparent !important; }
</style>
