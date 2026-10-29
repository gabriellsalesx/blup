<script lang="ts">
  import { onDestroy, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import "@xterm/xterm/css/xterm.css";
  import { Agents, OpenTerminal, WriteTerminal, ResizeTerminal, CloseTerminal, OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Agent } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import Sparkles from "@lucide/svelte/icons/sparkles";
  import SquareTerminal from "@lucide/svelte/icons/square-terminal";
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

  type Tab = { id: string; title: string; term: Terminal; fit: FitAddon; el: HTMLDivElement | null; ended: boolean };
  let tabs = $state<Tab[]>([]);
  let active = $state("");
  let agents = $state<Agent[]>([]);
  let error = $state("");
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
    return { background, foreground, cursor: foreground, cursorAccent: background, selectionBackground: "rgba(59,130,246,.35)", ...palette };
  }

  async function open(agent = "", title = tt.shell) {
    error = "";
    const term = new Terminal({ fontFamily: "ui-monospace, 'JetBrains Mono', Menlo, monospace", fontSize: look.termFont || 13, cursorBlink: true, theme: colors(), allowProposedApi: true });
    const fit = new FitAddon();
    term.loadAddon(fit);
    const tab: Tab = { id: "", title, term, fit, el: null, ended: false };
    tabs.push(tab);
    const idx = tabs.length - 1;
    await tick();
    const el = document.querySelector<HTMLDivElement>(`[data-term="${idx}"]`)!;
    tabs[idx].el = el;
    active = "pending";
    term.open(el);
    fit.fit();
    try {
      const id = await OpenTerminal(agent, term.cols, term.rows);
      tabs[idx].id = id;
      active = id;
      term.onData((d) => WriteTerminal(id, d).catch(() => {}));
      term.onResize(({ cols, rows }) => ResizeTerminal(id, cols, rows).catch(() => {}));
      term.focus();
    } catch (e) {
      error = String(e).replace(/^(Runtime)?Error: /, "");
      tabs.splice(idx, 1);
      term.dispose();
    }
  }

  function close(tab: Tab) {
    if (tab.id) CloseTerminal(tab.id);
    tab.term.dispose();
    tabs = tabs.filter((x) => x !== tab);
    if (active === tab.id) active = tabs.at(-1)?.id ?? "";
  }

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
  $effect(() => { const tab = tabs.find((x) => x.id === active); if (tab && visible) tick().then(() => { tab.fit.fit(); tab.term.focus(); }); });

  // The text size follows the settings.
  $effect(() => { const size = look.termFont || 13; for (const tab of tabs) { tab.term.options.fontSize = size; tab.fit.fit(); } });

  onDestroy(() => { off(); ro.disconnect(); for (const tab of tabs) close(tab); });

  // The first tab: a plain terminal.
  tick().then(() => { if (!tabs.length) open(); });
</script>

<section class="term-panel">
  <header>
    <div class="tabs">
      {#each tabs as tab, i (i)}
        <div class="tab" class:on={tab.id === active}>
          <button onclick={() => (active = tab.id)}>{tab.title}{tab.ended ? " ·" : ""}</button>
          <button class="x" title={tt.closeTab} onclick={() => close(tab)}><X size={12} /></button>
        </div>
      {/each}
      <button class="add" title={tt.newShell} onclick={() => open()}><SquareTerminal size={15} /></button>
    </div>
    <div class="agents">
      {#each installed as a (a.key)}
        <button class="agent" onclick={() => open(a.key, a.name)}><Sparkles size={13} />{a.name}</button>
      {/each}
      {#if !installed.length && agents.length}
        <span class="muted none">{tt.noAgents}</span>
        {#each missing.slice(0, 3) as a (a.key)}<button class="link" onclick={() => OpenURL(a.site)}>{a.name}</button>{/each}
      {/if}
    </div>
    <button class="close" title={tt.close} onclick={onClose}><ChevronDown size={18} /></button>
  </header>
  {#if error}<p class="bad err">{error}</p>{/if}
  <div class="host" bind:this={host}>
    {#each tabs as tab, i (i)}
      <div class="xterm-box" data-term={i} class:hidden={tab.id !== active && !(active === "pending" && i === tabs.length - 1)}></div>
    {/each}
  </div>
</section>

<style>
  /* A column: the terminal's area takes all the room left (with or without
     an error line above it). */
  .term-panel { display: flex; flex-direction: column; height: 100%; background: var(--panel); border-block-start: 1px solid var(--line); }
  header { display: flex; align-items: center; gap: .75rem; padding: .3rem .6rem; border-block-end: 1px solid var(--line); }
  .tabs { display: flex; align-items: center; gap: .2rem; min-width: 0; overflow: auto; }
  .tab { display: flex; align-items: center; border-radius: .4rem; font-size: .8rem; color: var(--muted); }
  .tab > button:first-child { padding: .3rem .1rem .3rem .6rem; white-space: nowrap; }
  .tab.on { background: var(--soft); color: var(--text); font-weight: 600; }
  .x, .add, .close { display: inline-grid; place-items: center; padding: .3rem; border-radius: .35rem; color: var(--muted); }
  .x:hover, .add:hover, .close:hover { color: var(--text); background: var(--soft); }
  .agents { display: flex; align-items: center; gap: .35rem; margin-inline-start: auto; flex-wrap: wrap; }
  .agent { display: inline-flex; align-items: center; gap: .35rem; padding: .3rem .6rem; border: 1px solid var(--line); border-radius: 999px; font-size: .78rem; font-weight: 600; }
  .agent:hover { border-color: var(--blue); color: var(--blue); }
  .none { font-size: .78rem; }
  .link { font-size: .78rem; color: var(--blue); text-decoration: underline; }
  .err { padding: .4rem .8rem; font-size: .8rem; }
  .host { position: relative; flex: 1; min-height: 0; }
  .xterm-box { position: absolute; inset: .4rem .2rem .2rem .6rem; }
  .xterm-box.hidden { visibility: hidden; }
  .host :global(.xterm-viewport) { background: transparent !important; }
</style>
