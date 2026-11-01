<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { NoteGraph } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import Search from "@lucide/svelte/icons/search";
  import Scan from "@lucide/svelte/icons/scan";
  import LocateFixed from "@lucide/svelte/icons/locate-fixed";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import { place, radius, step, type GEdge, type GNode } from "./graph";
  import { scheme } from "./theme.svelte";
  import { t } from "./i18n";

  // The map of notes: every note a dot, every link a line; the most linked
  // are bigger. Hovering a note lights it and its neighbors; a click opens
  // it. Filters: a search, a folder, a tag, loose notes only (no link in
  // or out: notes waiting to be connected), and "around this note" (the
  // open note and its neighbors).
  let { current, onOpenNote }: { current: string; onOpenNote: (path: string) => void } = $props();
  const tg = t.graph;

  let host = $state<HTMLDivElement>();
  let canvas = $state<HTMLCanvasElement>();
  let all: GNode[] = [];
  let allEdges: GEdge[] = [];
  let folders = $state<string[]>([]);
  let tagList = $state<string[]>([]);
  let count = $state({ notes: 0, links: 0, loose: 0 });
  let loaded = $state(false);

  let query = $state("");
  let folder = $state("");
  let tag = $state("");
  let loose = $state(false);
  let around = $state(false);

  let nodes: GNode[] = [];
  let edges: GEdge[] = [];
  let pan = { x: 0, y: 0 }, zoom = 1;
  let hover: GNode | null = null;
  let alpha = 1;
  let frame = 0;

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
    folders = [...new Set(all.map((n) => n.folder.split("/")[0]).filter(Boolean))].sort((a, b) => a.localeCompare(b));
    tagList = [...new Set(all.flatMap((n) => n.tags))].sort((a, b) => a.localeCompare(b));
    count = { notes: all.length, links: allEdges.length, loose: all.filter((n) => !n.degree).length };
    loaded = true;
    filter();
  }

  // What is shown, by the filters (the search only lights, it doesn't hide).
  function filter() {
    let keep = all.filter((n) => (!folder || n.folder === folder || n.folder.startsWith(folder + "/")) && (!tag || n.tags.includes(tag)) && (!loose || !n.degree));
    if (around && current) {
      const near = new Set([current]);
      for (const e of allEdges) {
        if (e.a.path === current) near.add(e.b.path);
        if (e.b.path === current) near.add(e.a.path);
      }
      keep = keep.filter((n) => near.has(n.path));
    }
    const shown = new Set(keep);
    nodes = keep;
    edges = allEdges.filter((e) => shown.has(e.a) && shown.has(e.b));
    alpha = 1;
  }
  $effect(() => { folder; tag; loose; around; current; if (loaded) filter(); });

  // Drawing, every frame while it moves (and when something changes).
  let dirty = true;
  function draw() {
    frame = requestAnimationFrame(draw);
    if (alpha > 0.01) { step(nodes, edges, alpha); alpha *= 0.985; dirty = true; }
    if (!dirty || !canvas || !host) return;
    dirty = false;
    const dpr = devicePixelRatio || 1, w = host.clientWidth, h = host.clientHeight;
    if (canvas.width !== w * dpr || canvas.height !== h * dpr) { canvas.width = w * dpr; canvas.height = h * dpr; }
    const g = canvas.getContext("2d")!;
    const css = getComputedStyle(host);
    const col = { line: css.getPropertyValue("--line"), muted: css.getPropertyValue("--muted"), text: css.getPropertyValue("--text"), blue: "#3b82f6", purple: scheme.dark ? "#a78bfa" : "#7048e8", bg: css.getPropertyValue("--bg") };
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    g.clearRect(0, 0, w, h);
    g.translate(w / 2 + pan.x, h / 2 + pan.y);
    g.scale(zoom, zoom);

    const q = query.trim().toLowerCase();
    const near = new Set<GNode>();
    if (hover) { near.add(hover); for (const e of edges) { if (e.a === hover) near.add(e.b); if (e.b === hover) near.add(e.a); } }
    const lit = (n: GNode) => (hover ? near.has(n) : q ? n.name.toLowerCase().includes(q) : true);

    g.lineWidth = 1 / zoom;
    for (const e of edges) {
      const on = hover && (e.a === hover || e.b === hover);
      g.strokeStyle = on ? col.purple : col.line;
      g.globalAlpha = hover && !on ? 0.25 : q ? 0.4 : 0.9;
      g.beginPath(); g.moveTo(e.a.x, e.a.y); g.lineTo(e.b.x, e.b.y); g.stroke();
    }
    for (const n of nodes) {
      const r = radius(n);
      g.globalAlpha = lit(n) ? 1 : 0.2;
      g.fillStyle = n.path === current ? col.purple : n.kind === "canvas" ? col.blue : n.kind === "drawing" ? "#e0b000" : !n.degree ? col.muted : col.text;
      g.beginPath(); g.arc(n.x, n.y, r, 0, Math.PI * 2); g.fill();
      if (n === hover || n.path === current) { g.strokeStyle = col.purple; g.lineWidth = 2 / zoom; g.stroke(); g.lineWidth = 1 / zoom; }
    }
    // Names: when close enough, or for what is lit.
    g.font = `${12 / zoom}px Inter, ui-sans-serif, system-ui, sans-serif`;
    g.textAlign = "center";
    for (const n of nodes) {
      const show = zoom > 0.9 || near.has(n) || n.path === current || (q && lit(n));
      if (!show) continue;
      g.globalAlpha = lit(n) ? 1 : 0.25;
      g.fillStyle = col.text;
      g.fillText(n.name, n.x, n.y + radius(n) + 13 / zoom);
    }
    g.globalAlpha = 1;
  }
  $effect(() => { query; scheme.dark; dirty = true; });

  // Where a point of the screen is on the map; the note there.
  function toMap(e: { clientX: number; clientY: number }) {
    const r = host!.getBoundingClientRect();
    return { x: (e.clientX - r.left - r.width / 2 - pan.x) / zoom, y: (e.clientY - r.top - r.height / 2 - pan.y) / zoom };
  }
  function nodeAt(e: { clientX: number; clientY: number }) {
    const p = toMap(e);
    let best: GNode | null = null, bd = Infinity;
    for (const n of nodes) {
      const d = Math.hypot(n.x - p.x, n.y - p.y);
      if (d < radius(n) + 4 / zoom && d < bd) { best = n; bd = d; }
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
      const p = toMap(e);
      drag.node.x = p.x; drag.node.y = p.y;
      alpha = Math.max(alpha, 0.3);
    } else pan = { x: drag.px + e.clientX - drag.x, y: drag.py + e.clientY - drag.y };
    dirty = true;
  }
  function up() {
    const d = drag;
    drag = null;
    if (!d) return;
    if (d.node) d.node.pinned = false;
    if (d.node && !d.moved) onOpenNote(d.node.path);
  }
  function wheel(e: WheelEvent) {
    e.preventDefault();
    const before = toMap(e);
    zoom = Math.min(Math.max(zoom * Math.exp(-e.deltaY * 0.0015), 0.15), 4);
    const after = toMap(e);
    pan = { x: pan.x + (after.x - before.x) * zoom, y: pan.y + (after.y - before.y) * zoom };
    dirty = true;
  }
  // Everything in view.
  function fit() {
    if (!nodes.length || !host) return;
    const xs = nodes.map((n) => n.x), ys = nodes.map((n) => n.y);
    const w = Math.max(...xs) - Math.min(...xs) + 80, h = Math.max(...ys) - Math.min(...ys) + 80;
    zoom = Math.min(Math.max(Math.min(host.clientWidth / w, host.clientHeight / h), 0.15), 2);
    pan = { x: -((Math.max(...xs) + Math.min(...xs)) / 2) * zoom, y: -((Math.max(...ys) + Math.min(...ys)) / 2) * zoom };
    dirty = true;
  }

  onMount(() => {
    load().then(() => setTimeout(fit, 900));
    frame = requestAnimationFrame(draw);
    const ro = new ResizeObserver(() => (dirty = true));
    ro.observe(host!);
    return () => ro.disconnect();
  });
  onDestroy(() => cancelAnimationFrame(frame));
</script>

<div class="graph">
  <div class="bar" role="toolbar" aria-label={tg.filters}>
    <label class="search"><Search size={15} /><input bind:value={query} placeholder={tg.search} aria-label={tg.search} /></label>
    <select bind:value={folder} aria-label={tg.folder}>
      <option value="">{tg.allFolders}</option>
      {#each folders as f (f)}<option value={f}>{f}</option>{/each}
    </select>
    {#if tagList.length}
      <select bind:value={tag} aria-label={tg.tag}>
        <option value="">{tg.allTags}</option>
        {#each tagList as x (x)}<option value={x}>#{x}</option>{/each}
      </select>
    {/if}
    <button class={["chip", { on: loose }]} onclick={() => (loose = !loose)} title={tg.looseHelp}>{tg.loose} ({count.loose})</button>
    <button class={["chip", { on: around }]} disabled={!current} onclick={() => (around = !around)} title={tg.aroundHelp}><LocateFixed size={14} />{tg.around}</button>
    <span class="grow"></span>
    <span class="muted count">{tg.count.replace("{notes}", String(count.notes)).replace("{links}", String(count.links))}</span>
    <button class="icon" title={tg.refresh} aria-label={tg.refresh} onclick={load}><RefreshCw size={15} /></button>
    <button class="icon" title={tg.fit} aria-label={tg.fit} onclick={fit}><Scan size={15} /></button>
  </div>
  <div class="map" bind:this={host} role="application" aria-label={tg.title} onpointerdown={down} onpointermove={move} onpointerup={up} onpointerleave={() => { if (!drag && hover) { hover = null; dirty = true; } }} onwheel={wheel}>
    <canvas bind:this={canvas}></canvas>
    {#if loaded && !count.notes}<p class="empty muted">{tg.empty}</p>{/if}
  </div>
  <p class="legend muted"><span class="dot note"></span>{tg.note}<span class="dot loose"></span>{tg.looseOne}<span class="dot canvas"></span>{tg.canvas}<span class="dot drawing"></span>{tg.drawing}</p>
</div>

<style>
  .graph { position: absolute; inset: 0; display: grid; grid-template-rows: auto minmax(0, 1fr) auto; background: var(--bg); }
  .bar { display: flex; flex-wrap: wrap; align-items: center; gap: .45rem; padding: .6rem .9rem; border-block-end: 1px solid var(--line); background: var(--panel); }
  .search { display: flex; align-items: center; gap: .35rem; padding: .3rem .55rem; border: 1px solid var(--line); border-radius: .45rem; color: var(--muted); }
  .search input { width: 12rem; border: 0; outline: none; background: transparent; color: var(--text); font: inherit; font-size: .88rem; }
  select { padding: .32rem .5rem; border: 1px solid var(--line); border-radius: .45rem; background: var(--panel); color: var(--text); font: inherit; font-size: .85rem; }
  .chip { display: flex; align-items: center; gap: .3rem; padding: .32rem .65rem; border: 1px solid var(--line); border-radius: 999px; font-size: .82rem; color: var(--muted); }
  .chip:hover:not(:disabled) { color: var(--text); }
  .chip.on { border-color: var(--blue); color: var(--text); background: color-mix(in srgb, var(--blue) 12%, transparent); }
  .chip:disabled { opacity: .4; }
  .grow { flex: 1; }
  .count { font-size: .8rem; }
  .icon { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .icon:hover { background: var(--soft); color: var(--text); }
  .map { position: relative; overflow: hidden; cursor: grab; touch-action: none; }
  canvas { position: absolute; inset: 0; width: 100%; height: 100%; }
  .empty { position: absolute; inset: 0; display: grid; place-items: center; }
  .legend { display: flex; align-items: center; gap: .4rem; padding: .4rem .9rem; font-size: .75rem; }
  .dot { width: .6rem; height: .6rem; border-radius: 50%; margin-inline-start: .5rem; }
  .dot.note { background: var(--text); } .dot.loose { background: var(--muted); } .dot.canvas { background: #3b82f6; } .dot.drawing { background: #e0b000; }
</style>
