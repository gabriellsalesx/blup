<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { NoteGraph } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import Settings2 from "@lucide/svelte/icons/settings-2";
  import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
  import X from "@lucide/svelte/icons/x";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import Plus from "@lucide/svelte/icons/plus";
  import Scan from "@lucide/svelte/icons/scan";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Play from "@lucide/svelte/icons/play";
  import { place, radius, step, type GEdge, type GNode } from "./graph";
  import { t } from "./i18n";

  // The map of notes, the whole screen: every note a dot, every link a
  // line; the most linked are bigger. Hovering a note lights it and its
  // neighbors; a click opens it; dragging moves the map (or a note), the
  // wheel zooms.
  //
  // Its choices are in a panel at the top right (240px, like Obsidian's),
  // in four parts that open and close: Filters (search, folder, tag, tags
  // as dots, loose notes, around the open note and how far, canvases and
  // drawings), Groups (colors by folder or tag), Display (arrows, when
  // names show, sizes, and Animate: the notes appear in the order they
  // were made) and Forces (how the notes push and pull). Closed, it is a
  // small button. The choices are kept on this computer.
  let { current, onOpenNote }: { current: string; onOpenNote: (path: string) => void } = $props();
  const tg = t.graph;

  type Group = { match: string; color: string }; // match: "folder:X" or "tag:Y"
  type Prefs = {
    query: string; folder: string; tag: string; tagNodes: boolean; orphans: boolean; around: boolean; depth: number; boards: boolean;
    groups: Group[];
    arrows: boolean; fade: number; size: number; line: number;
    repel: number; link: number; distance: number; center: number;
    open: Record<string, boolean>; panel: boolean;
  };
  const DEFAULTS: Prefs = {
    query: "", folder: "", tag: "", tagNodes: false, orphans: true, around: false, depth: 1, boards: true,
    groups: [],
    arrows: false, fade: 0.9, size: 1, line: 1,
    repel: 1, link: 1, distance: 70, center: 1,
    open: { filters: true }, panel: true,
  };
  const KEY = "blup.graph";
  let saved: Partial<Prefs> = {};
  try { saved = JSON.parse(localStorage.getItem(KEY) ?? "{}"); } catch {}
  let p = $state<Prefs>({ ...structuredClone(DEFAULTS), ...saved });
  $effect(() => { const snap = JSON.stringify(p); try { localStorage.setItem(KEY, snap); } catch {} });
  function reset() {
    const keep = p.open;
    p = { ...structuredClone(DEFAULTS), open: keep, panel: true };
    alpha = 1;
  }

  const COLORS = ["#3b82f6", "#22c55e", "#f59e0b", "#ef4444", "#14b8a6", "#ec4899", "#94a3b8"];

  let host = $state<HTMLDivElement>();
  let canvas = $state<HTMLCanvasElement>();
  let all: GNode[] = [];
  let allEdges: GEdge[] = [];
  let folders = $state<string[]>([]);
  let tagList = $state<string[]>([]);
  let count = $state({ notes: 0, links: 0, loose: 0 });
  let loaded = $state(false);

  let nodes: GNode[] = [];
  let edges: GEdge[] = [];
  let pan = { x: 0, y: 0 }, zoom = 1;
  let hover: GNode | null = null;
  let alpha = 1;
  let frame = 0;
  let dirty = true;

  async function load() {
    const g = await NoteGraph().catch(() => null);
    if (!g) return;
    const byPath = new Map<string, GNode>();
    const old = new Map(all.map((n) => [n.path, n]));
    all = (g.nodes ?? []).map((n) => {
      const was = old.get(n.path);
      const node: GNode = { ...n, tags: n.tags ?? [], x: was?.x ?? 0, y: was?.y ?? 0, vx: 0, vy: 0, degree: 0 };
      byPath.set(n.path, node);
      return node;
    });
    allEdges = [];
    for (const e of g.edges ?? []) {
      const a = byPath.get(e.from), b = byPath.get(e.to);
      if (a && b) { allEdges.push({ a, b }); a.degree++; b.degree++; }
    }
    if (!old.size) place(all);
    folders = [...new Set(all.flatMap((n) => n.folder.split("/").map((_, i, parts) => parts.slice(0, i + 1).join("/"))).filter(Boolean))].sort((a, b) => a.localeCompare(b));
    tagList = [...new Set(all.flatMap((n) => n.tags))].sort((a, b) => a.localeCompare(b));
    count = { notes: all.length, links: allEdges.length, loose: all.filter((n) => !n.degree).length };
    loaded = true;
    filter();
  }

  // Tags as dots (Filters): one per tag, joined to its notes; kept between
  // filterings so they don't jump.
  const tagDots = new Map<string, GNode>();
  function tagDot(tag: string) {
    let d = tagDots.get(tag);
    if (!d) { d = { path: "#" + tag, name: "#" + tag, folder: "", kind: "tag", tags: [], created: "", x: (Math.random() - 0.5) * 300, y: (Math.random() - 0.5) * 300, vx: 0, vy: 0, degree: 0 }; tagDots.set(tag, d); }
    return d;
  }

  // What is shown, by the filters (the search only lights, it doesn't hide).
  function filter() {
    let keep = all.filter((n) =>
      (!p.folder || n.folder === p.folder || n.folder.startsWith(p.folder + "/")) &&
      (!p.tag || n.tags.includes(p.tag)) &&
      (p.orphans || n.degree > 0) &&
      (p.boards || n.kind === "note"));
    if (p.around && current) {
      // The open note and the notes up to `depth` links away.
      const near = new Set([current]);
      for (let i = 0; i < p.depth; i++) {
        for (const e of allEdges) {
          if (near.has(e.a.path)) near.add(e.b.path);
          if (near.has(e.b.path)) near.add(e.a.path);
        }
      }
      keep = keep.filter((n) => near.has(n.path));
    }
    const shown = new Set(keep);
    let list = allEdges.filter((e) => shown.has(e.a) && shown.has(e.b));
    if (p.tagNodes) {
      const dots: GNode[] = [];
      for (const d of tagDots.values()) d.degree = 0;
      for (const n of keep) for (const tag of n.tags) {
        const d = tagDot(tag);
        if (!d.degree) dots.push(d);
        d.degree++;
        list = [...list, { a: n, b: d }];
      }
      keep = [...keep, ...dots];
    }
    nodes = keep;
    edges = list;
    alpha = 1;
  }
  $effect(() => { p.folder; p.tag; p.orphans; p.tagNodes; p.around; p.depth; p.boards; current; if (loaded) filter(); });

  // Animate: the notes appear in the order they were made, the links as
  // both ends are there.
  let anim: { start: number; rank: Map<GNode, number>; length: number } | null = null;
  function animate() {
    const order = [...nodes].sort((a, b) => (a.created || "9").localeCompare(b.created || "9"));
    anim = { start: performance.now(), rank: new Map(order.map((n, i) => [n, i])), length: Math.min(12000, Math.max(3000, order.length * 120)) };
    dirty = true;
  }
  $effect(() => { p.repel; p.link; p.distance; p.center; alpha = Math.max(alpha, 0.6); });
  $effect(() => { JSON.stringify(p.groups); p.query; p.fade; p.size; p.line; p.arrows; dirty = true; });

  // A note's color: its group's (the first that fits), else by kind.
  const groupOf = (n: GNode) => p.groups.find((g) => {
    const at = g.match.indexOf(":"), kind = g.match.slice(0, at), value = g.match.slice(at + 1);
    return kind === "tag" ? n.tags.includes(value) : value !== "" && (n.folder === value || n.folder.startsWith(value + "/"));
  });

  function draw() {
    frame = requestAnimationFrame(draw);
    if (alpha > 0.01) { step(nodes, edges, alpha, p); alpha *= 0.985; dirty = true; }
    if (anim) dirty = true;
    if (!dirty || !canvas || !host) return;
    dirty = false;
    const dpr = devicePixelRatio || 1, w = host.clientWidth, h = host.clientHeight;
    if (canvas.width !== w * dpr || canvas.height !== h * dpr) { canvas.width = w * dpr; canvas.height = h * dpr; }
    const g = canvas.getContext("2d")!;
    const css = getComputedStyle(host);
    const col = { line: css.getPropertyValue("--line"), muted: css.getPropertyValue("--muted"), text: css.getPropertyValue("--text"), accent: "#3b82f6" };
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    g.clearRect(0, 0, w, h);
    g.translate(w / 2 + pan.x, h / 2 + pan.y);
    g.scale(zoom, zoom);

    const q = p.query.trim().toLowerCase();
    const near = new Set<GNode>();
    if (hover) { near.add(hover); for (const e of edges) { if (e.a === hover) near.add(e.b); if (e.b === hover) near.add(e.a); } }
    const lit = (n: GNode) => (hover ? near.has(n) : q ? n.name.toLowerCase().includes(q) : true);
    const r = (n: GNode) => radius(n, p.size);
    // While animating, only what has appeared so far.
    let upTo = Infinity;
    if (anim) {
      const k = (performance.now() - anim.start) / anim.length;
      if (k >= 1) anim = null;
      else upTo = k * anim.rank.size;
    }
    const here = (n: GNode) => !anim || (anim.rank.get(n) ?? 0) <= upTo;

    for (const e of edges) {
      if (!here(e.a) || !here(e.b)) continue;
      const on = hover && (e.a === hover || e.b === hover);
      g.strokeStyle = on ? col.accent : col.line;
      g.globalAlpha = hover && !on ? 0.25 : q ? 0.4 : 0.9;
      g.lineWidth = ((on ? 1.6 : 1) * p.line) / zoom;
      g.beginPath(); g.moveTo(e.a.x, e.a.y); g.lineTo(e.b.x, e.b.y); g.stroke();
      if (p.arrows) {
        // A small head where the line meets the note it points to.
        const a = Math.atan2(e.b.y - e.a.y, e.b.x - e.a.x), tip = r(e.b) + 1, len = 6 * p.line;
        const x = e.b.x - Math.cos(a) * tip, y = e.b.y - Math.sin(a) * tip;
        g.beginPath();
        g.moveTo(x - Math.cos(a - 0.45) * len, y - Math.sin(a - 0.45) * len);
        g.lineTo(x, y);
        g.lineTo(x - Math.cos(a + 0.45) * len, y - Math.sin(a + 0.45) * len);
        g.stroke();
      }
    }
    for (const n of nodes) {
      if (!here(n)) continue;
      g.globalAlpha = lit(n) ? 1 : 0.2;
      g.fillStyle = n.path === current ? col.accent : groupOf(n)?.color ?? (n.kind === "tag" ? "#22c55e" : n.kind === "canvas" ? "#14b8a6" : n.kind === "drawing" ? "#e0b000" : !n.degree ? col.muted : col.text);
      g.beginPath(); g.arc(n.x, n.y, r(n), 0, Math.PI * 2); g.fill();
      if (n === hover || n.path === current) { g.strokeStyle = col.accent; g.lineWidth = 2 / zoom; g.stroke(); }
    }
    // Names: when the map is closer than the text threshold (Display; at
    // 0, always), and for what is lit.
    g.font = `${12 / zoom}px Inter, ui-sans-serif, system-ui, sans-serif`;
    g.textAlign = "center";
    for (const n of nodes) {
      if (!here(n)) continue;
      const show = zoom >= p.fade || near.has(n) || n.path === current || (q && lit(n));
      if (!show) continue;
      g.globalAlpha = lit(n) ? 1 : 0.25;
      g.fillStyle = col.text;
      g.fillText(n.name, n.x, n.y + r(n) + 13 / zoom);
    }
    g.globalAlpha = 1;
  }

  // Where a point of the screen is on the map; the note there.
  function toMap(e: { clientX: number; clientY: number }) {
    const rect = host!.getBoundingClientRect();
    return { x: (e.clientX - rect.left - rect.width / 2 - pan.x) / zoom, y: (e.clientY - rect.top - rect.height / 2 - pan.y) / zoom };
  }
  function nodeAt(e: { clientX: number; clientY: number }) {
    const m = toMap(e);
    let best: GNode | null = null, bd = Infinity;
    for (const n of nodes) {
      const d = Math.hypot(n.x - m.x, n.y - m.y);
      if (d < radius(n, p.size) + 4 / zoom && d < bd) { best = n; bd = d; }
    }
    return best;
  }

  let drag: { node: GNode | null; x: number; y: number; px: number; py: number; moved: boolean } | null = null;
  function down(e: PointerEvent) {
    host!.setPointerCapture(e.pointerId);
    const node = nodeAt(e);
    drag = { node, x: e.clientX, y: e.clientY, px: pan.x, py: pan.y, moved: false };
    if (node) node.pinned = true;
  }
  function move(e: PointerEvent) {
    if (!drag) {
      const h = nodeAt(e);
      if (h !== hover) { hover = h; dirty = true; host!.style.cursor = h ? "pointer" : "grab"; }
      return;
    }
    if (Math.hypot(e.clientX - drag.x, e.clientY - drag.y) > 3) drag.moved = true;
    if (drag.node) {
      const m = toMap(e);
      drag.node.x = m.x; drag.node.y = m.y;
      alpha = Math.max(alpha, 0.3);
    } else pan = { x: drag.px + e.clientX - drag.x, y: drag.py + e.clientY - drag.y };
    dirty = true;
  }
  function up() {
    const d = drag;
    drag = null;
    if (!d) return;
    if (d.node) d.node.pinned = false;
    if (d.node && !d.moved && d.node.kind !== "tag") onOpenNote(d.node.path);
  }
  function wheel(e: WheelEvent) {
    e.preventDefault();
    const before = toMap(e);
    zoom = Math.min(Math.max(zoom * Math.exp(-e.deltaY * 0.0015), 0.15), 4);
    const after = toMap(e);
    pan = { x: pan.x + (after.x - before.x) * zoom, y: pan.y + (after.y - before.y) * zoom };
    dirty = true;
  }
  function fit() {
    if (!nodes.length || !host) return;
    const xs = nodes.map((n) => n.x), ys = nodes.map((n) => n.y);
    const w = Math.max(...xs) - Math.min(...xs) + 80, h = Math.max(...ys) - Math.min(...ys) + 80;
    zoom = Math.min(Math.max(Math.min(host.clientWidth / w, host.clientHeight / h), 0.15), 2);
    pan = { x: -((Math.max(...xs) + Math.min(...xs)) / 2) * zoom, y: -((Math.max(...ys) + Math.min(...ys)) / 2) * zoom };
    dirty = true;
  }

  // Groups: a folder or a tag, and its color.
  const options = $derived([...folders.map((f) => ({ value: "folder:" + f, label: tg.folderOpt.replace("{name}", f) })), ...tagList.map((x) => ({ value: "tag:" + x, label: "#" + x }))]);
  function addGroup() {
    const used = new Set(p.groups.map((g) => g.match));
    const next = options.find((o) => !used.has(o.value));
    p.groups = [...p.groups, { match: next?.value ?? "", color: COLORS[p.groups.length % COLORS.length] }];
  }
  const nextColor = (c: string) => COLORS[(COLORS.indexOf(c) + 1) % COLORS.length];

  onMount(() => {
    load().then(() => setTimeout(fit, 900));
    frame = requestAnimationFrame(draw);
    const ro = new ResizeObserver(() => (dirty = true));
    ro.observe(host!);
    return () => ro.disconnect();
  });
  onDestroy(() => cancelAnimationFrame(frame));
</script>

{#snippet section(key: string, title: string)}
  <button class="sec" aria-expanded={!!p.open[key]} onclick={() => (p.open = { ...p.open, [key]: !p.open[key] })}>
    <span class={["chev", { open: p.open[key] }]}><ChevronRight size={15} /></span>{title}
  </button>
{/snippet}

{#snippet slider(label: string, key: "fade" | "depth" | "size" | "line" | "repel" | "link" | "distance" | "center", min: number, max: number, stepBy: number)}
  <label class="slider"><span>{label}</span><input type="range" {min} {max} step={stepBy} bind:value={p[key]} /></label>
{/snippet}

{#snippet toggle(label: string, key: "orphans" | "tagNodes" | "around" | "boards" | "arrows", disabled = false)}
  <label class={["toggle", { disabled }]}><span>{label}</span><input type="checkbox" bind:checked={p[key]} {disabled} /><i aria-hidden="true"></i></label>
{/snippet}

<div class="graph">
  <div class="map" bind:this={host} role="application" aria-label={tg.title} onpointerdown={down} onpointermove={move} onpointerup={up} onpointerleave={() => { if (!drag && hover) { hover = null; dirty = true; } }} onwheel={wheel}>
    <canvas bind:this={canvas}></canvas>
    {#if loaded && !count.notes}<p class="empty muted">{tg.empty}</p>{/if}
  </div>

  {#if p.panel}
    <aside class="panel" aria-label={tg.panel}>
      <header>
        {@render section("filters", tg.filters)}
        <button class="icon" title={tg.reset} aria-label={tg.reset} onclick={reset}><RotateCcw size={16} /></button>
        <button class="icon" title={tg.close} aria-label={tg.close} onclick={() => (p.panel = false)}><X size={16} /></button>
      </header>
      {#if p.open.filters}
        <div class="body">
          <input class="search" bind:value={p.query} placeholder={tg.search} aria-label={tg.search} />
          <select bind:value={p.folder} aria-label={tg.folder}>
            <option value="">{tg.allFolders}</option>
            {#each folders as f (f)}<option value={f}>{f}</option>{/each}
          </select>
          {#if tagList.length}
            <select bind:value={p.tag} aria-label={tg.tag}>
              <option value="">{tg.allTags}</option>
              {#each tagList as x (x)}<option value={x}>#{x}</option>{/each}
            </select>
          {/if}
          {@render toggle(tg.tagNodes, "tagNodes")}
          {@render toggle(`${tg.orphans} (${count.loose})`, "orphans")}
          {@render toggle(tg.boards, "boards")}
          {@render toggle(tg.around, "around", !current)}
          {#if p.around && current}{@render slider(tg.depth, "depth", 1, 4, 1)}{/if}
          <small class="muted">{tg.count.replace("{notes}", String(count.notes)).replace("{links}", String(count.links))}</small>
        </div>
      {/if}

      {@render section("groups", tg.groups)}
      {#if p.open.groups}
        <div class="body">
          {#each p.groups as g, i (i)}
            <div class="group">
              <button class="swatch" style:background={g.color} title={tg.color} aria-label={tg.color} onclick={() => (g.color = nextColor(g.color))}></button>
              <select bind:value={g.match} aria-label={tg.groupWhat}>
                {#each options as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
              </select>
              <button class="icon" title={tg.removeGroup} aria-label={tg.removeGroup} onclick={() => (p.groups = p.groups.filter((_, k) => k !== i))}><X size={14} /></button>
            </div>
          {/each}
          <button class="add" disabled={!options.length} onclick={addGroup}><Plus size={15} />{tg.newGroup}</button>
        </div>
      {/if}

      {@render section("display", tg.display)}
      {#if p.open.display}
        <div class="body">
          {@render toggle(tg.arrows, "arrows")}
          {@render slider(tg.fade, "fade", 0, 3, 0.05)}
          {@render slider(tg.size, "size", 0.5, 2.5, 0.1)}
          {@render slider(tg.line, "line", 0.5, 4, 0.1)}
          <button class="cta" onclick={animate}><Play size={14} />{tg.animate}</button>
        </div>
      {/if}

      {@render section("forces", tg.forces)}
      {#if p.open.forces}
        <div class="body">
          {@render slider(tg.center, "center", 0, 4, 0.1)}
          {@render slider(tg.repel, "repel", 0.1, 4, 0.1)}
          {@render slider(tg.link, "link", 0.1, 4, 0.1)}
          {@render slider(tg.distance, "distance", 20, 250, 5)}
        </div>
      {/if}
    </aside>
  {:else}
    <button class="open" title={tg.panel} aria-label={tg.panel} onclick={() => (p.panel = true)}><Settings2 size={18} /></button>
  {/if}

  <div class="corner">
    <button class="icon" title={tg.fit} aria-label={tg.fit} onclick={fit}><Scan size={16} /></button>
    <button class="icon" title={tg.refresh} aria-label={tg.refresh} onclick={load}><RefreshCw size={16} /></button>
  </div>
</div>

<style>
  .graph { position: absolute; inset: 0; background: var(--bg); }
  .map { position: absolute; inset: 0; overflow: hidden; cursor: grab; touch-action: none; }
  canvas { position: absolute; inset: 0; width: 100%; height: 100%; }
  .empty { position: absolute; inset: 0; display: grid; place-items: center; }

  /* Like Obsidian's: 240px wide, 12px from the corner, 13px text,
     scrolling without a bar when the window is short. */
  .panel { position: absolute; z-index: 3; inset-block-start: 12px; inset-inline-end: 12px; width: 240px; max-height: calc(100% - 16px); overflow-y: auto; scrollbar-width: none; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 8px 24px rgb(0 0 0 / .2); font-size: 13px; }
  .panel::-webkit-scrollbar { display: none; }
  header { display: flex; align-items: center; padding-inline-end: 4px; }
  header .sec { flex: 1; border-block-start: 0; }
  .sec { display: flex; align-items: center; gap: .35rem; width: 100%; padding: 8px 12px; border-block-start: 1px solid var(--line); font-size: 13px; font-weight: 600; color: var(--text); text-align: start; }
  .sec:hover { background: var(--soft); }
  .chev { display: inline-grid; color: var(--muted); transition: transform .15s; }
  .chev.open { transform: rotate(90deg); }
  .body { display: grid; gap: 10px; padding: 2px 12px 14px 16px; }
  .cta { display: flex; align-items: center; justify-content: center; gap: .35rem; width: 100%; margin-top: 2px; padding: 6px; border-radius: .45rem; background: var(--blue); color: #fff; font-size: 13px; font-weight: 600; }
  .cta:hover { filter: brightness(1.08); }
  .icon { display: grid; place-items: center; width: 26px; height: 26px; border-radius: .35rem; color: var(--muted); }
  .icon:hover { background: var(--soft); color: var(--text); }
  .search, select { width: 100%; padding: 5px 8px; border: 1px solid var(--line); border-radius: .4rem; background: var(--panel); color: var(--text); font: inherit; font-size: 13px; }
  .search:focus, select:focus { outline: none; border-color: var(--blue); }
  small { font-size: .76rem; }

  .toggle { display: flex; align-items: center; justify-content: space-between; gap: .6rem; font-size: 13px; color: var(--text); cursor: pointer; }
  .toggle.disabled { opacity: .45; cursor: default; }
  .toggle input { position: absolute; opacity: 0; pointer-events: none; }
  .toggle i { position: relative; flex: none; width: 2.1rem; height: 1.2rem; border-radius: 999px; background: var(--line); transition: background .15s; }
  .toggle i::after { content: ""; position: absolute; inset-block-start: .15rem; inset-inline-start: .15rem; width: .9rem; height: .9rem; border-radius: 50%; background: #fff; transition: translate .15s; }
  .toggle input:checked + i { background: var(--blue); }
  .toggle input:checked + i::after { translate: .9rem 0; }
  .toggle input:focus-visible + i { outline: 2px solid var(--blue); outline-offset: 2px; }

  .slider { display: grid; gap: 6px; font-size: 13px; color: var(--text); }
  .slider input { width: 100%; accent-color: var(--blue); }

  .group { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .4rem; }
  .swatch { width: 1.25rem; height: 1.25rem; border-radius: 50%; border: 2px solid var(--panel); box-shadow: 0 0 0 1px var(--line); }
  .add { display: flex; align-items: center; justify-content: center; gap: .35rem; padding: .4rem; border: 1px dashed var(--line); border-radius: .45rem; font-size: .84rem; color: var(--muted); }
  .add:hover:not(:disabled) { color: var(--text); border-color: var(--muted); }
  .add:disabled { opacity: .4; }

  .open { position: absolute; z-index: 3; inset-block-start: 12px; inset-inline-end: 12px; display: grid; place-items: center; width: 34px; height: 34px; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); color: var(--muted); box-shadow: 0 8px 24px rgb(0 0 0 / .18); }
  .open:hover { color: var(--text); }
  .corner { position: absolute; z-index: 3; inset-block-end: .9rem; inset-inline-start: .9rem; display: flex; gap: .2rem; padding: .25rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
</style>
