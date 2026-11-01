<script lang="ts">
  import { onDestroy, tick, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { OpenURL, ReadBoard, SaveBoard } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import StickyNote from "@lucide/svelte/icons/sticky-note";
  import FileText from "@lucide/svelte/icons/file-text";
  import LinkIcon from "@lucide/svelte/icons/link";
  import Group from "@lucide/svelte/icons/group";
  import ZoomIn from "@lucide/svelte/icons/zoom-in";
  import ZoomOut from "@lucide/svelte/icons/zoom-out";
  import Scan from "@lucide/svelte/icons/scan";
  import Undo2 from "@lucide/svelte/icons/undo-2";
  import Redo2 from "@lucide/svelte/icons/redo-2";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Pencil from "@lucide/svelte/icons/pencil";
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import CanvasFile from "./CanvasFile.svelte";
  import NotePicker from "./NotePicker.svelte";
  import { bounds, cardHTML, colorOf, COLORS, curve, edgePath, inside, newId, parseCanvas, sidePoint, sideToward, writeCanvas, type CEdge, type CNode, type Canvas, type Side } from "./canvas";
  import { resolveHref, resolveWiki, type NoteRef } from "./notelinks";
  import { t } from "./i18n";

  // A canvas of the vault, like Obsidian's: an endless board of cards
  // (text, notes of the vault, web addresses, groups) joined by arrows.
  // Saved by itself a moment after each change, as JSON Canvas.
  //
  // Moving around: drag the empty board (or the wheel) to move, Ctrl and
  // the wheel (or a pinch) to zoom. Two clicks on the board make a card;
  // dragging from the dot on a card's side makes an arrow. Notes dragged
  // from the files panel land as cards.
  let { path, notes, onStatus, onError, onOpenNote, onOutside }: {
    path: string; notes: NoteRef[];
    onStatus: (s: "saved" | "saving" | "unsaved") => void;
    onError: (e: string) => void;
    onOpenNote: (path: string) => void;
    onOutside: () => void;
  } = $props();
  const tb = t.boards;

  let nodes = $state<CNode[]>([]);
  let edges = $state<CEdge[]>([]);
  let rest: Canvas["rest"] = {};
  let loaded = $state(false);
  let board = $state<HTMLDivElement>();
  let pan = $state({ x: 0, y: 0 });
  let zoom = $state(1);
  let selected = $state<Set<string>>(new Set()); // cards
  let edgeSel = $state(""); // an arrow
  let editing = $state(""); // the card (or arrow label) being written
  let picking = $state(false); // the note picker, to add a note
  let fileVersion = $state(0); // notes on cards read again when it changes
  const clean = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  const byId = $derived(new Map(nodes.map((n) => [n.id, n])));
  const groups = $derived(nodes.filter((n) => n.type === "group"));
  const cards = $derived(nodes.filter((n) => n.type !== "group"));

  // Loading and saving.
  let written = "";
  let dirty = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  // The view is made again for each canvas (keyed by its path).
  ReadBoard(untrack(() => path)).then(async (text) => {
    written = text;
    const c = parseCanvas(text);
    nodes = c.nodes;
    edges = c.edges;
    rest = c.rest;
    loaded = true;
    await tick();
    fit();
  }).catch((e) => onError(clean(e)));

  function changed() {
    dirty = true;
    onStatus("unsaved");
    clearTimeout(timer);
    timer = setTimeout(save, 400);
  }
  async function save() {
    clearTimeout(timer);
    if (!dirty) return;
    dirty = false;
    onStatus("saving");
    try {
      written = writeCanvas({ nodes: $state.snapshot(nodes) as CNode[], edges: $state.snapshot(edges) as CEdge[], rest });
      await SaveBoard(path, written);
      onStatus(dirty ? "unsaved" : "saved");
    } catch (e) {
      dirty = true;
      onStatus("unsaved");
      onError(clean(e));
    }
  }

  // Undo and redo: the whole canvas before each change.
  let undoStack: string[] = [], redoStack: string[] = [];
  let canUndo = $state(false), canRedo = $state(false);
  const snap = () => JSON.stringify({ nodes: $state.snapshot(nodes), edges: $state.snapshot(edges) });
  function remember() {
    undoStack.push(snap());
    if (undoStack.length > 100) undoStack.shift();
    redoStack = [];
    canUndo = true; canRedo = false;
  }
  function restore(from: string[], to: string[]) {
    const s = from.pop();
    if (!s) return;
    to.push(snap());
    const c = JSON.parse(s);
    nodes = c.nodes; edges = c.edges;
    selected = new Set(); edgeSel = ""; editing = "";
    canUndo = undoStack.length > 0; canRedo = redoStack.length > 0;
    changed();
  }
  const undo = () => restore(undoStack, redoStack);
  const redo = () => restore(redoStack, undoStack);

  // Where things are: the board's pixels and the canvas's own units.
  function toWorld(cx: number, cy: number) {
    const r = board!.getBoundingClientRect();
    return { x: (cx - r.left - pan.x) / zoom, y: (cy - r.top - pan.y) / zoom };
  }
  function zoomAt(cx: number, cy: number, z: number) {
    const r = board!.getBoundingClientRect();
    const nz = Math.min(Math.max(z, 0.1), 3);
    const wx = (cx - r.left - pan.x) / zoom, wy = (cy - r.top - pan.y) / zoom;
    pan = { x: cx - r.left - wx * nz, y: cy - r.top - wy * nz };
    zoom = nz;
  }
  function zoomBy(f: number) {
    const r = board!.getBoundingClientRect();
    zoomAt(r.left + r.width / 2, r.top + r.height / 2, zoom * f);
  }
  // Everything in view (or the start, when empty).
  function fit() {
    if (!board) return;
    const r = board.getBoundingClientRect();
    if (!nodes.length) { zoom = 1; pan = { x: r.width / 2, y: r.height / 2 }; return; }
    const b = bounds(nodes);
    const z = Math.min(Math.max(Math.min((r.width - 120) / Math.max(b.width, 1), (r.height - 160) / Math.max(b.height, 1)), 0.1), 1);
    zoom = z;
    pan = { x: r.width / 2 - (b.x + b.width / 2) * z, y: r.height / 2 - (b.y + b.height / 2) * z };
  }
  function wheel(e: WheelEvent) {
    e.preventDefault();
    if (e.ctrlKey || e.metaKey) zoomAt(e.clientX, e.clientY, zoom * Math.exp(-e.deltaY * 0.01));
    else pan = { x: pan.x - e.deltaX, y: pan.y - e.deltaY };
  }

  // Dragging: the board (to move around), cards (to move them), a card's
  // corner (to resize it) or a side's dot (to make an arrow).
  type Drag =
    | { kind: "pan"; x: number; y: number; px: number; py: number }
    | { kind: "move"; x: number; y: number; start: Map<string, { x: number; y: number }>; moved: boolean }
    | { kind: "resize"; id: string; x: number; y: number; w: number; h: number; moved: boolean }
    | { kind: "arrow"; from: string; side: Side; to: { x: number; y: number } }
    | { kind: "box"; a: { x: number; y: number }; b: { x: number; y: number } };
  let drag = $state<Drag | null>(null);

  function downBoard(e: PointerEvent) {
    if (e.button !== 0 && e.button !== 1) return;
    finishEdit();
    edgeSel = "";
    if (e.shiftKey && e.button === 0) {
      const p = toWorld(e.clientX, e.clientY);
      drag = { kind: "box", a: p, b: p };
    } else {
      selected = new Set();
      drag = { kind: "pan", x: e.clientX, y: e.clientY, px: pan.x, py: pan.y };
    }
    capture(e);
  }
  function downCard(e: PointerEvent, n: CNode) {
    if (e.button !== 0 || editing === n.id) return;
    e.stopPropagation();
    finishEdit();
    edgeSel = "";
    if (e.shiftKey) {
      const s = new Set(selected);
      if (s.has(n.id)) s.delete(n.id); else s.add(n.id);
      selected = s;
    } else if (!selected.has(n.id)) selected = new Set([n.id]);
    // The cards selected, and what is inside the groups among them.
    const moving = new Set(selected);
    for (const id of selected) {
      const g = byId.get(id);
      if (g?.type === "group") for (const m of nodes) if (inside(m, g)) moving.add(m.id);
    }
    const start = new Map([...moving].map((id) => [id, { x: byId.get(id)!.x, y: byId.get(id)!.y }]));
    drag = { kind: "move", x: e.clientX, y: e.clientY, start, moved: false };
    capture(e);
  }
  function downCorner(e: PointerEvent, n: CNode) {
    e.stopPropagation();
    drag = { kind: "resize", id: n.id, x: e.clientX, y: e.clientY, w: n.width, h: n.height, moved: false };
    capture(e);
  }
  function downSide(e: PointerEvent, n: CNode, side: Side) {
    e.stopPropagation();
    drag = { kind: "arrow", from: n.id, side, to: toWorld(e.clientX, e.clientY) };
    capture(e);
  }
  function capture(e: PointerEvent) { (e.currentTarget as Element).setPointerCapture?.(e.pointerId); }

  function move(e: PointerEvent) {
    const d = drag;
    if (!d) return;
    if (d.kind === "pan") pan = { x: d.px + e.clientX - d.x, y: d.py + e.clientY - d.y };
    else if (d.kind === "move") {
      const dx = (e.clientX - d.x) / zoom, dy = (e.clientY - d.y) / zoom;
      if (!d.moved && Math.hypot(dx, dy) * zoom < 3) return;
      if (!d.moved) { remember(); d.moved = true; }
      for (const [id, p] of d.start) { const n = byId.get(id); if (n) { n.x = p.x + dx; n.y = p.y + dy; } }
    } else if (d.kind === "resize") {
      const n = byId.get(d.id);
      if (!n) return;
      if (!d.moved) { remember(); d.moved = true; }
      n.width = Math.max(80, d.w + (e.clientX - d.x) / zoom);
      n.height = Math.max(40, d.h + (e.clientY - d.y) / zoom);
    } else if (d.kind === "arrow") d.to = toWorld(e.clientX, e.clientY);
    else if (d.kind === "box") d.b = toWorld(e.clientX, e.clientY);
  }
  function up(e: PointerEvent) {
    const d = drag;
    drag = null;
    if (!d) return;
    if ((d.kind === "move" || d.kind === "resize") && d.moved) changed();
    else if (d.kind === "box") {
      const x = Math.min(d.a.x, d.b.x), y = Math.min(d.a.y, d.b.y), w = Math.abs(d.a.x - d.b.x), h = Math.abs(d.a.y - d.b.y);
      selected = new Set(nodes.filter((n) => n.x < x + w && n.x + n.width > x && n.y < y + h && n.y + n.height > y).map((n) => n.id));
    } else if (d.kind === "arrow") {
      // Dropped on a card: joined to it. On the board: a new card there.
      const el = document.elementFromPoint(e.clientX, e.clientY)?.closest("[data-node]") as HTMLElement | null;
      const target = el ? byId.get(el.dataset.node!) : undefined;
      if (target?.id === d.from) return;
      const from = byId.get(d.from);
      if (!from) return;
      if (!target && Math.hypot(d.to.x - sidePoint(from, d.side).x, d.to.y - sidePoint(from, d.side).y) < 30) return;
      remember();
      let to = target;
      if (!to) {
        to = { id: newId(), type: "text", text: "", x: d.to.x - 125, y: d.to.y - 30, width: 250, height: 60 };
        nodes.push(to);
        editing = to.id;
      }
      edges.push({ id: newId(), fromNode: from.id, fromSide: d.side, toNode: to.id, toSide: sideToward(to, sidePoint(from, d.side)) });
      changed();
    }
  }

  // Making cards.
  function center() {
    const r = board!.getBoundingClientRect();
    return toWorld(r.left + r.width / 2, r.top + r.height / 2);
  }
  function add(n: Omit<CNode, "id">, edit = false) {
    remember();
    const card = { id: newId(), ...n } as CNode;
    nodes.push(card);
    selected = new Set([card.id]);
    if (edit) editing = card.id;
    changed();
    return card;
  }
  function addText(at = center()) { add({ type: "text", text: "", x: at.x - 125, y: at.y - 30, width: 250, height: 60 }, true); }
  function addLink(at = center()) { add({ type: "link", url: "", x: at.x - 175, y: at.y - 40, width: 350, height: 80 }, true); }
  function addFile(file: string, at = center()) {
    const image = /\.(png|jpe?g|webp|gif|svg|avif)$/i.test(file);
    add({ type: "file", file, x: at.x - 200, y: at.y - 150, width: image ? 320 : 400, height: image ? 240 : 300 });
  }
  // A group around the cards selected (or an empty one in the middle).
  function addGroup() {
    const sel = nodes.filter((n) => selected.has(n.id));
    const b = sel.length ? bounds(sel) : { ...center(), width: 400, height: 300 };
    if (!sel.length) { b.x -= 200; b.y -= 150; }
    remember();
    const g: CNode = { id: newId(), type: "group", label: "", x: b.x - 40, y: b.y - 40, width: b.width + 80, height: b.height + 80 };
    nodes.unshift(g); // groups go under the cards
    selected = new Set([g.id]);
    editing = g.id;
    changed();
  }
  function dblBoard(e: MouseEvent) {
    if (e.target !== e.currentTarget) return;
    addText(toWorld(e.clientX, e.clientY));
  }

  // Notes dragged from the files panel.
  function over(e: DragEvent) {
    if (e.dataTransfer?.types.includes("application/x-blup-note")) { e.preventDefault(); e.dataTransfer.dropEffect = "copy"; }
  }
  function drop(e: DragEvent) {
    const file = e.dataTransfer?.getData("application/x-blup-note");
    if (!file) return;
    e.preventDefault();
    addFile(file, toWorld(e.clientX, e.clientY));
  }

  // Pasted text: a card (a web address becomes a link card).
  function paste(e: ClipboardEvent) {
    if (editing || (e.target as HTMLElement)?.closest?.("input, textarea")) return;
    const text = e.clipboardData?.getData("text/plain")?.trim();
    if (!text) return;
    e.preventDefault();
    const at = center();
    if (/^https?:\/\/\S+$/.test(text)) add({ type: "link", url: text, x: at.x - 175, y: at.y - 40, width: 350, height: 80 });
    else add({ type: "text", text, x: at.x - 150, y: at.y - 60, width: 300, height: Math.min(400, 40 + 24 * text.split("\n").length) });
  }

  // Writing on a card (two clicks): text, a web address or a group's name.
  function startEdit(n: CNode) {
    if (n.type === "file") { openFile(n.file!); return; }
    remember();
    editing = n.id;
  }
  function finishEdit() {
    if (!editing) return;
    const n = byId.get(editing);
    editing = "";
    // An empty text card left behind goes away.
    if (n?.type === "text" && !n.text?.trim()) remove(new Set([n.id]), false);
    changed();
  }
  function grow(e: Event, n: CNode) {
    const el = e.currentTarget as HTMLTextAreaElement;
    if (el.scrollHeight + 2 > n.height * 1) n.height = Math.ceil(el.scrollHeight + 2);
    changed();
  }
  const focus = (el: HTMLElement) => { el.focus(); if (el instanceof HTMLTextAreaElement) el.setSelectionRange(el.value.length, el.value.length); };

  function remove(ids: Set<string>, keep = true) {
    if (!ids.size) return;
    if (keep) remember();
    nodes = nodes.filter((n) => !ids.has(n.id));
    edges = edges.filter((e) => !ids.has(e.fromNode) && !ids.has(e.toNode));
    selected = new Set();
    changed();
  }
  function removeEdge(id: string) {
    remember();
    edges = edges.filter((e) => e.id !== id);
    edgeSel = "";
    changed();
  }
  function paint(color: string | undefined) {
    remember();
    if (edgeSel) { const e = edges.find((x) => x.id === edgeSel); if (e) e.color = color; }
    for (const id of selected) { const n = byId.get(id); if (n) n.color = color; }
    for (const x of [...nodes, ...edges]) if (!x.color) delete x.color;
    changed();
  }

  // Following: a note or another board opens in the Studio; a web address
  // in the browser.
  function openFile(file: string) {
    if (/\.(md|canvas|draw)$/i.test(file)) onOpenNote(file);
  }
  function openLink(href: string, from: string) {
    const wiki = /^\[\[([^\]|#]+)/.exec(href);
    const to = wiki ? resolveWiki(wiki[1], from, notes) : resolveHref(href, from, notes);
    if (to && "note" in to) onOpenNote(to.note);
    else if (to) OpenURL(to.url);
  }
  function clickLink(e: MouseEvent) {
    const a = (e.target as HTMLElement).closest?.(".lnk") as HTMLElement | null;
    if (!a) return;
    e.stopPropagation();
    openLink(a.dataset.href ?? "", (a.closest("[data-from]") as HTMLElement | null)?.dataset.from ?? path);
  }

  function key(e: KeyboardEvent) {
    if ((e.target as HTMLElement)?.closest?.("input, textarea, [contenteditable]")) {
      if (e.key === "Escape") { e.preventDefault(); finishEdit(); }
      return;
    }
    if (!board?.isConnected) return;
    const mod = e.ctrlKey || e.metaKey;
    if (mod && e.key.toLowerCase() === "z") { e.preventDefault(); if (e.shiftKey) redo(); else undo(); }
    else if (mod && e.key.toLowerCase() === "y") { e.preventDefault(); redo(); }
    else if (mod && e.key.toLowerCase() === "a") { e.preventDefault(); selected = new Set(nodes.map((n) => n.id)); }
    else if (e.key === "Delete" || e.key === "Backspace") {
      if (edgeSel) { e.preventDefault(); removeEdge(edgeSel); }
      else if (selected.size) { e.preventDefault(); remove(selected); }
    } else if (e.key === "Escape") { selected = new Set(); edgeSel = ""; }
    else if (e.key === "Enter" && selected.size === 1) { e.preventDefault(); startEdit(byId.get([...selected][0])!); }
  }

  // The menu over what is selected: colors, write, open, delete.
  const one = $derived(selected.size === 1 ? byId.get([...selected][0]) : undefined);
  const menuAt = $derived.by(() => {
    if (drag) return null;
    if (edgeSel) {
      const e = edges.find((x) => x.id === edgeSel);
      const a = e && byId.get(e.fromNode), b = e && byId.get(e.toNode);
      if (!a || !b) return null;
      const m = edgePath(a, b, e).mid;
      return { x: m.x * zoom + pan.x, y: m.y * zoom + pan.y - 24 };
    }
    const sel = nodes.filter((n) => selected.has(n.id));
    if (!sel.length || editing) return null;
    const b = bounds(sel);
    return { x: (b.x + b.width / 2) * zoom + pan.x, y: b.y * zoom + pan.y - (sel.some((n) => n.type === "group") ? 34 : 12) };
  });

  // An arrow's color, and its label being written.
  function editLabel(id: string) {
    remember();
    editing = "edge:" + id;
  }

  // Changed on disk by something else (Obsidian, an AI): opened again,
  // unless there are changes of its own; notes on cards are read again.
  const off = Events.On("notesChanged", async (ev: any) => {
    const list = String(ev.data ?? "").split("\n").map((p) => p.toLowerCase());
    if (nodes.some((n) => n.file && list.includes(n.file.toLowerCase()))) fileVersion++;
    if (!list.includes(path.toLowerCase())) return;
    const now = await ReadBoard(path).catch(() => written);
    if (now.trimEnd() !== written.trimEnd() && !dirty) onOutside();
  });
  onDestroy(() => { off(); save(); });

  const arrowTo = $derived(drag?.kind === "arrow" ? drag : null);
  const box = $derived(drag?.kind === "box" ? drag : null);
  const tint = (c?: string) => colorOf(c) ?? "";
</script>

<svelte:window onkeydown={key} onpaste={paste} />

<div class="canvas" class:panning={drag?.kind === "pan"}>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="board" bind:this={board} style:--z={zoom} style:background-position="{pan.x}px {pan.y}px" style:background-size="{20 * zoom}px {20 * zoom}px"
    onpointerdown={downBoard} onpointermove={move} onpointerup={up} onwheel={wheel} ondblclick={dblBoard} ondragover={over} ondrop={drop}>
    <div class="world" style:transform="translate({pan.x}px, {pan.y}px) scale({zoom})">
      <!-- Groups, under everything. -->
      {#each groups as g (g.id)}
        <div class="group" class:sel={selected.has(g.id)} data-node={g.id} style:left="{g.x}px" style:top="{g.y}px" style:width="{g.width}px" style:height="{g.height}px" style:--c={tint(g.color)}
          role="group" onpointerdown={(e) => downCard(e, g)} ondblclick={(e) => { e.stopPropagation(); startEdit(g); }}>
          {#if editing === g.id}
            <input class="glabel edit" bind:value={g.label} placeholder={tb.groupPh} use:focus onkeydown={(e) => e.key === "Enter" && finishEdit()} onblur={finishEdit} onpointerdown={(e) => e.stopPropagation()} />
          {:else if g.label}<span class="glabel">{g.label}</span>{/if}
          <span class="corner" role="presentation" onpointerdown={(e) => downCorner(e, g)}></span>
        </div>
      {/each}

      <!-- The arrows. -->
      <svg class="edges" aria-hidden="true">
        <defs>
          <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="context-stroke" /></marker>
        </defs>
        {#each edges as e (e.id)}
          {@const a = byId.get(e.fromNode)}
          {@const b = byId.get(e.toNode)}
          {#if a && b}
            {@const p = edgePath(a, b, e)}
            <g class="edge" class:sel={edgeSel === e.id} style:--c={tint(e.color)}>
              <path class="hit" d={p.d} role="presentation" onpointerdown={(ev) => { ev.stopPropagation(); selected = new Set(); finishEdit(); edgeSel = e.id; }} ondblclick={(ev) => { ev.stopPropagation(); editLabel(e.id); }} />
              <path class="line" d={p.d} marker-end={(e.toEnd ?? "arrow") === "arrow" ? "url(#arrow)" : undefined} marker-start={e.fromEnd === "arrow" ? "url(#arrow)" : undefined} />
            </g>
          {/if}
        {/each}
        {#if arrowTo}
          {@const a = byId.get(arrowTo.from)}
          {#if a}<path class="line drawing" d={curve(sidePoint(a, arrowTo.side), arrowTo.side, arrowTo.to, null).d} marker-end="url(#arrow)" />{/if}
        {/if}
      </svg>
      <!-- Arrow labels, over the lines. -->
      {#each edges as e (e.id)}
        {@const a = byId.get(e.fromNode)}
        {@const b = byId.get(e.toNode)}
        {#if a && b && (e.label || editing === "edge:" + e.id)}
          {@const m = edgePath(a, b, e).mid}
          <div class="elabel" style:left="{m.x}px" style:top="{m.y}px">
            {#if editing === "edge:" + e.id}
              <input bind:value={e.label} placeholder={tb.labelPh} use:focus onkeydown={(ev) => ev.key === "Enter" && finishEdit()} onblur={finishEdit} onpointerdown={(ev) => ev.stopPropagation()} />
            {:else}
              <span role="presentation" ondblclick={(ev) => { ev.stopPropagation(); editLabel(e.id); }} onpointerdown={(ev) => { ev.stopPropagation(); edgeSel = e.id; selected = new Set(); }}>{e.label}</span>
            {/if}
          </div>
        {/if}
      {/each}

      <!-- The cards (the keyboard works on the selection: Enter, Delete). -->
      {#each cards as n (n.id)}
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
        <div class="card {n.type}" class:sel={selected.has(n.id)} class:editing={editing === n.id} data-node={n.id}
          style:left="{n.x}px" style:top="{n.y}px" style:width="{n.width}px" style:height="{n.height}px" style:--c={tint(n.color)}
          role="group" onpointerdown={(e) => downCard(e, n)} ondblclick={(e) => { e.stopPropagation(); startEdit(n); }} onclick={clickLink}>
          <div class="content">
            {#if n.type === "text"}
              {#if editing === n.id}
                <textarea bind:value={n.text} placeholder={tb.textPh} use:focus oninput={(e) => grow(e, n)} onblur={finishEdit} onpointerdown={(e) => e.stopPropagation()}></textarea>
              {:else}
                <div class="md">{@html cardHTML(n.text ?? "")}</div>
              {/if}
            {:else if n.type === "file"}
              <CanvasFile file={n.file ?? ""} version={fileVersion} />
            {:else if n.type === "link"}
              {#if editing === n.id}
                <input class="url" bind:value={n.url} placeholder="https://" use:focus onkeydown={(e) => e.key === "Enter" && finishEdit()} onblur={finishEdit} onpointerdown={(e) => e.stopPropagation()} />
              {:else}
                <button class="site" onclick={(e) => { e.stopPropagation(); if (n.url) OpenURL(n.url); }} onpointerdown={(e) => e.stopPropagation()}>
                  <LinkIcon size={15} /><span><b>{(() => { try { return new URL(n.url ?? "").host; } catch { return n.url; } })()}</b><small>{n.url}</small></span>
                </button>
              {/if}
            {/if}
          </div>
          {#each ["top", "right", "bottom", "left"] as const as side (side)}
            <span class="dot {side}" role="presentation" title={tb.connect} onpointerdown={(e) => downSide(e, n, side)}></span>
          {/each}
          <span class="corner" role="presentation" onpointerdown={(e) => downCorner(e, n)}></span>
        </div>
      {/each}
    </div>

    {#if box}
      {@const x = Math.min(box.a.x, box.b.x) * zoom + pan.x}
      {@const y = Math.min(box.a.y, box.b.y) * zoom + pan.y}
      <div class="box" style:left="{x}px" style:top="{y}px" style:width="{Math.abs(box.a.x - box.b.x) * zoom}px" style:height="{Math.abs(box.a.y - box.b.y) * zoom}px"></div>
    {/if}

    {#if loaded && !nodes.length}
      <div class="empty muted"><b>{tb.emptyTitle}</b><p>{tb.emptyLead}</p></div>
    {/if}
  </div>

  {#if menuAt}
    <div class="menu" style:left="{menuAt.x}px" style:top="{menuAt.y}px" role="toolbar" aria-label={tb.menu}>
      <button class="swatch none" title={tb.noColor} aria-label={tb.noColor} onclick={() => paint(undefined)}></button>
      {#each Object.entries(COLORS) as [k, c] (k)}
        <button class="swatch" style:background={c} title={tb.color} aria-label={tb.color} onclick={() => paint(k)}></button>
      {/each}
      <span class="msep"></span>
      {#if edgeSel}
        <button title={tb.label} onclick={() => editLabel(edgeSel)}><Pencil size={15} /></button>
        <button title={tb.remove} onclick={() => removeEdge(edgeSel)}><Trash2 size={15} /></button>
      {:else}
        {#if one && one.type !== "file"}<button title={tb.edit} onclick={() => startEdit(one)}><Pencil size={15} /></button>{/if}
        {#if one?.type === "file"}<button title={tb.open} onclick={() => openFile(one.file ?? "")}><ExternalLink size={15} /></button>{/if}
        <button title={tb.remove} onclick={() => remove(selected)}><Trash2 size={15} /></button>
      {/if}
    </div>
  {/if}

  <!-- Adding: a card, a note of the vault, a web address, a group. -->
  <div class="add" role="toolbar" aria-label={tb.add}>
    <button title={tb.addText} onclick={() => addText()}><StickyNote size={18} strokeWidth={1.75} /><span>{tb.text}</span></button>
    <button title={tb.addNote} onclick={() => (picking = true)}><FileText size={18} strokeWidth={1.75} /><span>{tb.note}</span></button>
    <button title={tb.addLink} onclick={() => addLink()}><LinkIcon size={18} strokeWidth={1.75} /><span>{tb.link}</span></button>
    <button title={tb.addGroup} onclick={addGroup}><Group size={18} strokeWidth={1.75} /><span>{tb.group}</span></button>
  </div>

  <div class="controls" role="toolbar" aria-label={tb.view}>
    <button title={tb.zoomIn} onclick={() => zoomBy(1.2)}><ZoomIn size={17} strokeWidth={1.75} /></button>
    <button title={tb.zoomOut} onclick={() => zoomBy(1 / 1.2)}><ZoomOut size={17} strokeWidth={1.75} /></button>
    <button title={tb.fit} onclick={fit}><Scan size={17} strokeWidth={1.75} /></button>
    <span class="csep"></span>
    <button title={tb.undo} disabled={!canUndo} onclick={undo}><Undo2 size={17} strokeWidth={1.75} /></button>
    <button title={tb.redo} disabled={!canRedo} onclick={redo}><Redo2 size={17} strokeWidth={1.75} /></button>
  </div>
</div>

{#if picking}
  <NotePicker title={tb.addNote} notes={notes.filter((n) => n.path !== path)} onClose={() => (picking = false)} onPick={(n) => { picking = false; addFile(n.path); }} />
{/if}

<style>
  .canvas { position: relative; height: 100%; overflow: hidden; background: var(--bg); user-select: none; }
  .board { position: absolute; inset: 0; cursor: default; touch-action: none;
    background-image: radial-gradient(circle, color-mix(in srgb, var(--muted) 45%, transparent) 1px, transparent 1.2px); }
  .panning .board { cursor: grabbing; }
  .world { position: absolute; inset-inline-start: 0; inset-block-start: 0; transform-origin: 0 0; }
  .edges { position: absolute; inset-inline-start: 0; inset-block-start: 0; width: 1px; height: 1px; overflow: visible; }

  /* Cards, like Obsidian's: a border in the card's color, a hint of it
     inside. */
  .card { position: absolute; display: flex; border: 2px solid var(--c, var(--line)); border-radius: .6rem; background: color-mix(in srgb, var(--c, transparent) 10%, var(--panel)); box-shadow: 0 1px 3px rgb(0 0 0 / .08); }
  .card.sel { outline: 2px solid var(--blue); outline-offset: 2px; }
  .content { flex: 1; min-width: 0; overflow: auto; padding: .65rem .85rem; border-radius: inherit; font-size: .95rem; line-height: 1.5; color: var(--text); }
  .card.editing .content { overflow: hidden; padding: 0; }
  textarea { width: 100%; height: 100%; padding: .65rem .85rem; border: 0; border-radius: inherit; outline: none; resize: none; background: transparent; color: var(--text); font: inherit; line-height: 1.5; user-select: text; }
  .url { width: 100%; height: 100%; padding: 0 .85rem; border: 0; outline: none; background: transparent; color: var(--text); font: inherit; }
  .site { display: flex; gap: .6rem; align-items: center; width: 100%; height: 100%; text-align: start; color: var(--text); }
  .site span { display: grid; min-width: 0; }
  .site small { color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .md :global(h1), .md :global(h2), .md :global(h3) { margin-block: .2rem .4rem; line-height: 1.25; }
  .md :global(h1) { font-size: 1.4rem; } .md :global(h2) { font-size: 1.2rem; } .md :global(h3) { font-size: 1.05rem; }
  .md :global(p), .card :global(.md p) { margin-block: 0 .45rem; }
  .card :global(.md ul), .card :global(.md ol) { margin-block: 0 .45rem; padding-inline-start: 1.25rem; }
  .card :global(.md .task) { list-style: "☐  "; } .card :global(.md .task.done) { list-style: "☑  "; color: var(--muted); }
  .card :global(.md blockquote) { margin: 0 0 .45rem; padding-inline-start: .7rem; border-inline-start: 3px solid var(--line); color: var(--muted); }
  .card :global(.md code) { padding: .05rem .3rem; border-radius: .25rem; background: var(--soft); font-size: .88em; }
  .card :global(.md pre) { padding: .5rem; border-radius: .4rem; background: var(--soft); overflow: auto; }
  .card :global(.md mark) { background: color-mix(in srgb, #e0ac00 35%, transparent); color: inherit; }
  .card :global(.md .lnk) { color: var(--blue); text-decoration: underline; padding: 0; font: inherit; cursor: pointer; }

  /* The dots on the sides, to drag an arrow out of, and the corner to
     resize; shown on hover. */
  .dot { position: absolute; width: 12px; height: 12px; border: 2px solid var(--panel); border-radius: 50%; background: var(--blue); opacity: 0; cursor: crosshair; transition: opacity .1s; }
  .card:hover .dot, .card.sel .dot { opacity: 1; }
  .dot.top { inset-block-start: -7px; inset-inline-start: calc(50% - 6px); }
  .dot.bottom { inset-block-end: -7px; inset-inline-start: calc(50% - 6px); }
  .dot.left { inset-inline-start: -7px; inset-block-start: calc(50% - 6px); }
  .dot.right { inset-inline-end: -7px; inset-block-start: calc(50% - 6px); }
  .corner { position: absolute; inset-inline-end: -4px; inset-block-end: -4px; width: 14px; height: 14px; cursor: nwse-resize; }

  .group { position: absolute; border: 2px solid var(--c, var(--line)); border-radius: .8rem; background: color-mix(in srgb, var(--c, var(--muted)) 7%, transparent); }
  .group.sel { outline: 2px solid var(--blue); outline-offset: 2px; }
  .glabel { position: absolute; inset-block-end: calc(100% + 6px); inset-inline-start: 0; max-width: 100%; padding: .2rem .55rem; border-radius: .4rem; background: color-mix(in srgb, var(--c, var(--muted)) 18%, var(--panel)); font-size: 1rem; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--text); }
  input.glabel { border: 1px solid var(--blue); outline: none; font: inherit; font-weight: 600; min-width: 12rem; }

  .edge .line, .line { fill: none; stroke: var(--c, var(--muted)); stroke-width: 2.5; }
  .edge.sel .line { stroke: var(--blue); }
  .line.drawing { stroke: var(--blue); stroke-dasharray: 6 5; }
  .edge .hit { fill: none; stroke: transparent; stroke-width: 16; cursor: pointer; pointer-events: stroke; }
  .elabel { position: absolute; transform: translate(-50%, -50%); }
  .elabel span, .elabel input { display: block; padding: .15rem .5rem; border: 1px solid var(--line); border-radius: .4rem; background: var(--panel); color: var(--text); font: inherit; font-size: .9rem; white-space: nowrap; }
  .elabel input { border-color: var(--blue); outline: none; }

  .box { position: absolute; border: 1px solid var(--blue); background: color-mix(in srgb, var(--blue) 10%, transparent); pointer-events: none; }
  .empty { position: absolute; inset: 0; display: grid; place-content: center; text-align: center; gap: .35rem; pointer-events: none; }
  .empty b { color: var(--text); font-size: 1.05rem; }
  .empty p { max-width: 26rem; font-size: .9rem; }

  .menu { position: absolute; z-index: 4; transform: translate(-50%, -100%); display: flex; align-items: center; gap: .25rem; padding: .3rem .4rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 8px 24px rgb(0 0 0 / .15); }
  .menu button { display: grid; place-items: center; width: 1.8rem; height: 1.8rem; border-radius: .4rem; color: var(--muted); }
  .menu button:hover { background: var(--soft); color: var(--text); }
  .menu .swatch { width: 1.1rem; height: 1.1rem; border-radius: 50%; border: 2px solid var(--panel); box-shadow: 0 0 0 1px var(--line); }
  .menu .swatch.none { background: linear-gradient(135deg, transparent 45%, var(--red) 45% 55%, transparent 55%), var(--panel); }
  .msep, .csep { width: 1px; height: 1.2rem; background: var(--line); }

  .add { position: absolute; z-index: 3; inset-block-end: 1rem; inset-inline-start: 50%; transform: translateX(-50%); display: flex; gap: .2rem; padding: .3rem; border: 1px solid var(--line); border-radius: .7rem; background: var(--panel); box-shadow: 0 8px 24px rgb(0 0 0 / .12); }
  .add button { display: flex; align-items: center; gap: .4rem; padding: .45rem .7rem; border-radius: .5rem; font-size: .85rem; color: var(--muted); }
  .add button:hover { background: var(--soft); color: var(--text); }
  .controls { position: absolute; z-index: 3; inset-block-start: 1rem; inset-inline-end: 1rem; display: grid; justify-items: center; gap: .15rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
  .controls button { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .controls button:hover:not(:disabled) { background: var(--soft); color: var(--text); }
  .controls button:disabled { opacity: .35; }
  .controls .csep { width: 1.2rem; height: 1px; margin-block: .15rem; }
</style>
