<script lang="ts">
  import { onDestroy, onMount, tick, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { BoardFile, ReadBoard, SaveBoard, SaveBoardImage } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import MousePointer from "@lucide/svelte/icons/mouse-pointer-2";
  import Hand from "@lucide/svelte/icons/hand";
  import Square from "@lucide/svelte/icons/square";
  import Circle from "@lucide/svelte/icons/circle";
  import Diamond from "@lucide/svelte/icons/diamond";
  import ArrowIcon from "@lucide/svelte/icons/move-up-right";
  import Minus from "@lucide/svelte/icons/minus";
  import Pencil from "@lucide/svelte/icons/pencil";
  import TypeIcon from "@lucide/svelte/icons/type";
  import Eraser from "@lucide/svelte/icons/eraser";
  import ImagePlus from "@lucide/svelte/icons/image-plus";
  import Download from "@lucide/svelte/icons/download";
  import ZoomIn from "@lucide/svelte/icons/zoom-in";
  import ZoomOut from "@lucide/svelte/icons/zoom-out";
  import Scan from "@lucide/svelte/icons/scan";
  import Maximize2 from "@lucide/svelte/icons/maximize-2";
  import Minimize2 from "@lucide/svelte/icons/minimize-2";
  import Undo2 from "@lucide/svelte/icons/undo-2";
  import Redo2 from "@lucide/svelte/icons/redo-2";
  import Copy from "@lucide/svelte/icons/copy";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import BringToFront from "@lucide/svelte/icons/bring-to-front";
  import SendToBack from "@lucide/svelte/icons/send-to-back";
  import { arrowHead, bounds, colorOf, dashArray, fitPoints, isPath, isShape, LINE, measure, newId, normalize, PALETTE, parseDraw, penPath, scaleTo, snapAngle, snapBox, worldPoints, writeDraw, type Box, type Dash, type El, type Kind } from "./draw";
  import { scheme } from "./theme.svelte";
  import { toast } from "./toast.svelte";
  import { t } from "./i18n";

  // A drawing of the vault: the Studio's own sketchbook. Pick a tool on
  // top (or its key), draw on the endless board; the style of what is
  // selected (or of what comes next) is on the left. Saved by itself a
  // moment after each change.
  //
  // Moving around: the hand (or Space, or the middle button) drags the
  // board, the wheel scrolls it, Ctrl and the wheel (or a pinch) zoom.
  let { path, full, onFull, onStatus, onError, onOutside, bindSave }: {
    path: string;
    bindSave?: (save: () => Promise<void>) => void; // the workspace saves first before a rename, move or delete
    full: boolean; // full screen: the drawing alone in the window
    onFull: () => void;
    onStatus: (s: "saved" | "saving" | "unsaved") => void;
    onError: (e: string) => void;
    onOutside: () => void; // changed on disk by someone else
  } = $props();
  const td = t.draw;

  type Tool = "select" | "hand" | Exclude<Kind, "image"> | "eraser";
  const TOOLS: { tool: Tool; key: string; Icon: typeof Square; label: string }[] = [
    { tool: "hand", key: "h", Icon: Hand, label: td.hand },
    { tool: "select", key: "v", Icon: MousePointer, label: td.select },
    { tool: "rect", key: "r", Icon: Square, label: td.rect },
    { tool: "diamond", key: "d", Icon: Diamond, label: td.diamond },
    { tool: "ellipse", key: "o", Icon: Circle, label: td.ellipse },
    { tool: "arrow", key: "a", Icon: ArrowIcon, label: td.arrow },
    { tool: "line", key: "l", Icon: Minus, label: td.line },
    { tool: "pen", key: "p", Icon: Pencil, label: td.pen },
    { tool: "text", key: "t", Icon: TypeIcon, label: td.text },
    { tool: "eraser", key: "e", Icon: Eraser, label: td.eraser },
  ];
  let tool = $state<Tool>("select");

  let elements = $state<El[]>([]);
  let rest: Record<string, unknown> = {};
  let loaded = $state(false);
  let host = $state<HTMLDivElement>();
  let pan = $state({ x: 0, y: 0 });
  let zoom = $state(1);
  let selected = $state<Set<string>>(new Set());
  let editing = $state(""); // the element whose text is being written
  let space = $state(false); // Space held: the hand for a moment
  const clean = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");
  const byId = $derived(new Map(elements.map((e) => [e.id, e])));
  const chosen = $derived(elements.filter((e) => selected.has(e.id)));
  const ink = (c: string) => colorOf(c, scheme.dark);

  // The style of what comes next (and of what is selected, when changed).
  let style = $state({ stroke: "ink", fill: "none", fillStyle: "tint" as "tint" | "solid", width: 2, dash: "solid" as Dash, size: 20 });

  // Images on the drawing: read from the vault once.
  let images = $state<Record<string, string>>({});
  function loadImage(file: string) {
    if (file in images) return;
    images[file] = "";
    BoardFile(file).then((src) => (images[file] = src)).catch(() => {});
  }

  // Loading and saving.
  let written = "";
  let dirty = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  ReadBoard(untrack(() => path)).then(async (text) => {
    written = text;
    const d = parseDraw(text);
    elements = d.elements;
    rest = d.rest;
    for (const e of elements) if (e.type === "image" && e.file) loadImage(e.file);
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
      written = writeDraw({ elements: $state.snapshot(elements) as El[], rest });
      await SaveBoard(path, written);
      onStatus(dirty ? "unsaved" : "saved");
    } catch (e) {
      dirty = true;
      onStatus("unsaved");
      onError(clean(e));
    }
  }

  // Undo and redo: the whole drawing before each change.
  // Undo keeps copies of the drawing (typed, no JSON round trip).
  let undoStack: El[][] = [], redoStack: El[][] = [];
  let canUndo = $state(false), canRedo = $state(false);
  const snap = (): El[] => $state.snapshot(elements);
  function remember() {
    undoStack.push(snap());
    if (undoStack.length > 200) undoStack.shift();
    redoStack = [];
    canUndo = true; canRedo = false;
  }
  function restore(from: El[][], to: El[][]) {
    const s = from.pop();
    if (!s) return;
    to.push(snap());
    elements = s;
    selected = new Set(); editing = "";
    canUndo = undoStack.length > 0; canRedo = redoStack.length > 0;
    changed();
  }
  const undo = () => restore(undoStack, redoStack);
  const redo = () => restore(redoStack, undoStack);

  // The board's pixels and the drawing's own units.
  function toWorld(cx: number, cy: number) {
    const r = host!.getBoundingClientRect();
    return { x: (cx - r.left - pan.x) / zoom, y: (cy - r.top - pan.y) / zoom };
  }
  function zoomAt(cx: number, cy: number, z: number) {
    const r = host!.getBoundingClientRect();
    const nz = Math.min(Math.max(z, 0.1), 5);
    const wx = (cx - r.left - pan.x) / zoom, wy = (cy - r.top - pan.y) / zoom;
    pan = { x: cx - r.left - wx * nz, y: cy - r.top - wy * nz };
    zoom = nz;
  }
  function zoomBy(f: number) {
    const r = host!.getBoundingClientRect();
    zoomAt(r.left + r.width / 2, r.top + r.height / 2, zoom * f);
  }

  // Ctrl and + or - (the row's or the keypad's, any keyboard layout) zoom,
  // Ctrl+0 frames what was made; also while the files panel has the focus,
  // which it keeps after the board was opened from it.
  function zoomKey(e: KeyboardEvent) {
    if (!(e.ctrlKey || e.metaKey) || e.altKey || (e.target as HTMLElement)?.closest?.("[role=dialog], [role=menu]")) return false;
    const c = e.code, k = e.key;
    if (k === "+" || k === "=" || c === "Equal" || c === "NumpadAdd") zoomBy(1.2);
    else if (k === "-" || c === "Minus" || c === "NumpadSubtract") zoomBy(1 / 1.2);
    else if (k === "0" || c === "Digit0" || c === "Numpad0") fit();
    else return false;
    e.preventDefault();
    return true;
  }
  function fit() {
    if (!host) return;
    const r = host.getBoundingClientRect();
    if (!elements.length) { zoom = 1; pan = { x: r.width / 2, y: r.height / 2 }; return; }
    const b = bounds(elements);
    const z = Math.min(Math.max(Math.min((r.width - 160) / Math.max(b.w, 1), (r.height - 200) / Math.max(b.h, 1)), 0.1), 1);
    zoom = z;
    pan = { x: r.width / 2 - (b.x + b.w / 2) * z, y: r.height / 2 - (b.y + b.h / 2) * z };
  }
  function wheel(e: WheelEvent) {
    e.preventDefault();
    if (e.ctrlKey || e.metaKey) zoomAt(e.clientX, e.clientY, zoom * Math.exp(-e.deltaY * 0.01));
    else pan = { x: pan.x - e.deltaX, y: pan.y - e.deltaY };
  }

  // Dragging, by tool: moving the board, drawing an element, moving or
  // resizing the selection, a box that selects, or the eraser.
  type Drag =
    | { kind: "pan"; x: number; y: number; px: number; py: number }
    | { kind: "draw"; id: string; x: number; y: number; pts?: [number, number][] }
    | { kind: "move"; x: number; y: number; start: Map<string, { x: number; y: number }>; moved: boolean }
    | { kind: "resize"; corner: number; box: Box; start: Map<string, { b: Box; pts?: [number, number][] }>; moved: boolean }
    | { kind: "end"; id: string; which: number; pts: [number, number][]; moved: boolean }
    | { kind: "box"; a: { x: number; y: number }; b: { x: number; y: number } }
    | { kind: "erase"; gone: boolean };
  let drag = $state<Drag | null>(null);

  function down(e: PointerEvent) {
    if (e.button !== 0 && e.button !== 1) return;
    if (editing) { finishText(); return; }
    host!.setPointerCapture(e.pointerId);
    const p = toWorld(e.clientX, e.clientY);
    if (e.button === 1 || tool === "hand" || space) {
      drag = { kind: "pan", x: e.clientX, y: e.clientY, px: pan.x, py: pan.y };
      return;
    }
    const hit = elAt(e);
    if (tool === "select") {
      if (hit) {
        if (e.shiftKey) {
          const s = new Set(selected);
          if (s.has(hit.id)) s.delete(hit.id); else s.add(hit.id);
          selected = s;
        } else if (!selected.has(hit.id)) selected = new Set([hit.id]);
        drag = { kind: "move", x: e.clientX, y: e.clientY, start: new Map(chosen.map((c) => [c.id, { x: c.x, y: c.y }])), moved: false };
      } else {
        if (!e.shiftKey) selected = new Set();
        drag = { kind: "box", a: p, b: p };
      }
      return;
    }
    if (tool === "eraser") {
      drag = { kind: "erase", gone: false };
      erase(e);
      return;
    }
    if (tool === "text") { addText(p); return; }
    // A new element, drawn by dragging.
    remember();
    const base = { id: newId(), stroke: style.stroke, fill: style.fill, fillStyle: style.fillStyle, width: style.width, dash: style.dash };
    if (tool === "pen" || tool === "arrow" || tool === "line") {
      const el: El = { ...base, type: tool, x: p.x, y: p.y, w: 0, h: 0, points: [[0, 0], [0, 0]], fill: "none" };
      elements.push(el);
      drag = { kind: "draw", id: el.id, x: p.x, y: p.y, pts: [[p.x, p.y]] };
    } else {
      elements.push({ ...base, type: tool, x: p.x, y: p.y, w: 0, h: 0 });
      drag = { kind: "draw", id: base.id, x: p.x, y: p.y };
    }
    selected = new Set();
  }

  // The element under the pointer.
  function elAt(e: { clientX: number; clientY: number }) {
    const el = document.elementFromPoint(e.clientX, e.clientY)?.closest("[data-el]") as HTMLElement | null;
    return el ? byId.get(el.dataset.el!) : undefined;
  }

  function move(e: PointerEvent) {
    const d = drag;
    if (!d) return;
    const p = toWorld(e.clientX, e.clientY);
    if (d.kind === "pan") pan = { x: d.px + e.clientX - d.x, y: d.py + e.clientY - d.y };
    else if (d.kind === "draw") {
      const el = byId.get(d.id);
      if (!el) return;
      if (el.type === "pen") {
        d.pts!.push([p.x, p.y]);
        fitPoints(el, d.pts!);
      } else if (isPath(el)) {
        let dx = p.x - d.x, dy = p.y - d.y;
        if (e.shiftKey) [dx, dy] = snapAngle(dx, dy);
        el.points = [[0, 0], [dx, dy]];
        el.x = d.x; el.y = d.y; el.w = dx; el.h = dy;
      } else {
        let w = p.x - d.x, h = p.y - d.y;
        if (e.shiftKey) ({ w, h } = snapBox(w, h));
        el.w = w; el.h = h;
      }
    } else if (d.kind === "move") {
      const dx = (e.clientX - d.x) / zoom, dy = (e.clientY - d.y) / zoom;
      if (!d.moved && Math.hypot(dx, dy) * zoom < 3) return;
      if (!d.moved) { remember(); d.moved = true; }
      for (const [id, s] of d.start) { const el = byId.get(id); if (el) { el.x = s.x + dx; el.y = s.y + dy; } }
    } else if (d.kind === "resize") {
      if (!d.moved) { remember(); d.moved = true; }
      // The corner dragged moves; the one across stays.
      const b0 = d.box;
      const ax = d.corner === 0 || d.corner === 3 ? b0.x + b0.w : b0.x;
      const ay = d.corner === 0 || d.corner === 1 ? b0.y + b0.h : b0.y;
      let w = p.x - ax, h = p.y - ay;
      if (e.shiftKey && b0.w && b0.h) { const k = Math.max(Math.abs(w) / b0.w, Math.abs(h) / b0.h); w = Math.sign(w || 1) * b0.w * k; h = Math.sign(h || 1) * b0.h * k; }
      const nb = { x: Math.min(ax, ax + w), y: Math.min(ay, ay + h), w: Math.abs(w), h: Math.abs(h) };
      const sx = b0.w ? nb.w / b0.w : 1, sy = b0.h ? nb.h / b0.h : 1;
      for (const [id, s] of d.start) {
        const el = byId.get(id);
        if (!el) continue;
        scaleTo(el, s.b, s.pts, { x: nb.x + (s.b.x - b0.x) * sx, y: nb.y + (s.b.y - b0.y) * sy, w: s.b.w * sx, h: s.b.h * sy });
        if (el.type === "text") Object.assign(el, measure(el.text ?? "", el.size ?? 20));
      }
    } else if (d.kind === "end") {
      const el = byId.get(d.id);
      if (!el) return;
      if (!d.moved) { remember(); d.moved = true; }
      const pts = d.pts.map((q) => [...q] as [number, number]);
      let to: [number, number] = [p.x, p.y];
      const other = pts[d.which === 0 ? pts.length - 1 : 0];
      if (e.shiftKey) { const [dx, dy] = snapAngle(p.x - other[0], p.y - other[1]); to = [other[0] + dx, other[1] + dy]; }
      pts[d.which === 0 ? 0 : pts.length - 1] = to;
      fitPoints(el, pts);
    } else if (d.kind === "box") d.b = p;
    else if (d.kind === "erase") erase(e);
  }

  function up() {
    const d = drag;
    drag = null;
    if (!d) return;
    if (d.kind === "draw") {
      const el = byId.get(d.id);
      if (!el) return;
      normalize(el);
      // A click without dragging leaves nothing behind.
      if (Math.max(el.w, el.h) < 3 && el.type !== "pen") { elements = elements.filter((x) => x.id !== el.id); undoStack.pop(); canUndo = undoStack.length > 0; return; }
      if (el.type !== "pen") { selected = new Set([el.id]); tool = "select"; }
      changed();
    } else if ((d.kind === "move" || d.kind === "resize" || d.kind === "end") && d.moved) {
      for (const el of chosen) normalize(el);
      changed();
    } else if (d.kind === "box") {
      const x = Math.min(d.a.x, d.b.x), y = Math.min(d.a.y, d.b.y), w = Math.abs(d.a.x - d.b.x), h = Math.abs(d.a.y - d.b.y);
      if (w < 2 && h < 2) return;
      const inBox = elements.filter((el) => el.x >= x && el.y >= y && el.x + el.w <= x + w && el.y + el.h <= y + h).map((el) => el.id);
      selected = new Set([...selected, ...inBox]);
    } else if (d.kind === "erase" && d.gone) changed();
  }

  function erase(e: PointerEvent) {
    const hit = elAt(e);
    if (!hit || drag?.kind !== "erase") return;
    if (!drag.gone) { remember(); drag.gone = true; }
    elements = elements.filter((x) => x.id !== hit.id);
  }

  // The corners of the selection, to resize; the ends of a single line.
  function downCorner(e: PointerEvent, corner: number) {
    e.stopPropagation();
    host!.setPointerCapture(e.pointerId);
    drag = { kind: "resize", corner, box: bounds(chosen), start: new Map(chosen.map((c) => [c.id, { b: { x: c.x, y: c.y, w: c.w, h: c.h }, pts: c.points?.map((q) => [...q] as [number, number]) }])), moved: false };
  }
  function downEnd(e: PointerEvent, el: El, which: number) {
    e.stopPropagation();
    host!.setPointerCapture(e.pointerId);
    drag = { kind: "end", id: el.id, which, pts: worldPoints(el), moved: false };
  }

  // Text: on its own (the text tool, or two clicks on the board), or inside
  // a shape (two clicks on it).
  let draft = $state("");
  function addText(p: { x: number; y: number }) {
    remember();
    const el: El = { id: newId(), type: "text", x: p.x, y: p.y - style.size * 0.6, w: 10, h: style.size * LINE, stroke: style.stroke, fill: "none", width: style.width, dash: "solid", text: "", size: style.size };
    elements.push(el);
    startText(el, false);
  }
  function startText(el: El, keep = true) {
    if (keep) remember();
    selected = new Set([el.id]);
    draft = el.text ?? "";
    editing = el.id;
    tool = "select";
  }
  function typed() {
    const el = byId.get(editing);
    if (el?.type === "text") Object.assign(el, measure(draft, el.size ?? 20));
  }
  function finishText() {
    const el = byId.get(editing);
    editing = "";
    if (!el) return;
    el.text = draft;
    if (el.type === "text") {
      if (!draft.trim()) { elements = elements.filter((x) => x.id !== el.id); selected = new Set(); }
      else Object.assign(el, measure(draft, el.size ?? 20));
    } else if (!draft.trim()) delete el.text;
    changed();
  }
  function dbl(e: MouseEvent) {
    if (tool !== "select") return;
    const hit = elAt(e);
    if (hit && (hit.type === "text" || isShape(hit))) startText(hit);
    else if (!hit) addText(toWorld(e.clientX, e.clientY));
  }
  const focus = (el: HTMLTextAreaElement) => { el.focus(); el.setSelectionRange(el.value.length, el.value.length); };

  // The style: changing it changes what is selected, and what comes next.
  function setStyle<K extends keyof typeof style>(k: K, v: (typeof style)[K]) {
    style[k] = v;
    if (!chosen.length) return;
    remember();
    for (const el of chosen) {
      if (k === "size") { if (el.type === "text" || el.text) { el.size = v as number; if (el.type === "text") Object.assign(el, measure(el.text ?? "", el.size)); } }
      else if (k === "fill" || k === "fillStyle") { if (isShape(el)) Object.assign(el, { [k]: v }); }
      else Object.assign(el, { [k]: v });
    }
    changed();
  }
  // What the style panel shows: the selection's, else the next one's.
  const shown = $derived(chosen[0] ? { stroke: chosen[0].stroke, fill: chosen[0].fill, fillStyle: chosen[0].fillStyle ?? "tint", width: chosen[0].width, dash: chosen[0].dash, size: chosen[0].size ?? style.size } : style);
  const showFill = $derived(chosen.length ? chosen.some(isShape) : tool === "rect" || tool === "ellipse" || tool === "diamond");
  const showText = $derived(chosen.length ? chosen.some((c) => c.type === "text" || !!c.text) : tool === "text");
  const showPanel = $derived(chosen.length > 0 || !["select", "hand", "eraser"].includes(tool));

  function remove() {
    if (!chosen.length) return;
    remember();
    elements = elements.filter((e) => !selected.has(e.id));
    selected = new Set();
    changed();
  }
  function duplicate(list = chosen, offset = 16) {
    if (!list.length) return;
    remember();
    const copies = list.map((e) => ({ ...structuredClone($state.snapshot(e)), id: newId(), x: e.x + offset, y: e.y + offset }) as El);
    elements.push(...copies);
    selected = new Set(copies.map((c) => c.id));
    changed();
  }
  function order(front: boolean) {
    if (!chosen.length) return;
    remember();
    const rest = elements.filter((e) => !selected.has(e.id)), mine = elements.filter((e) => selected.has(e.id));
    elements = front ? [...rest, ...mine] : [...mine, ...rest];
    changed();
  }

  // Copy and paste: elements inside the Studio; an image or text from
  // other apps.
  let clip: El[] = [];
  async function paste(e: ClipboardEvent) {
    if (editing || (e.target as HTMLElement)?.closest?.("input, textarea")) return;
    const file = [...(e.clipboardData?.files ?? [])].find((f) => f.type.startsWith("image/"));
    if (file) { e.preventDefault(); await addImage(file); return; }
    const text = e.clipboardData?.getData("text/plain");
    if (clip.length && text === "blup-draw") { e.preventDefault(); duplicate(clip, 24); return; }
    if (text?.trim()) {
      e.preventDefault();
      const c = center();
      remember();
      const el: El = { id: newId(), type: "text", x: c.x, y: c.y, w: 0, h: 0, stroke: style.stroke, fill: "none", width: style.width, dash: "solid", text: text.trim(), size: style.size };
      Object.assign(el, measure(el.text!, el.size!));
      elements.push(el);
      selected = new Set([el.id]);
      changed();
    }
  }
  function copy(e: ClipboardEvent) {
    if (editing || !chosen.length || (e.target as HTMLElement)?.closest?.("input, textarea")) return;
    e.preventDefault();
    clip = $state.snapshot(chosen) as El[];
    e.clipboardData?.setData("text/plain", "blup-draw");
  }

  // Images: pasted, dropped or picked; kept in the attachments folder.
  let picker = $state<HTMLInputElement>();
  async function addImage(file: File, at = center()) {
    try {
      const data = await new Promise<string>((ok, bad) => {
        const r = new FileReader();
        r.onload = () => ok(String(r.result));
        r.onerror = () => bad(r.error);
        r.readAsDataURL(file);
      });
      const size = await new Promise<{ w: number; h: number }>((ok) => {
        const img = new Image();
        img.onload = () => ok({ w: img.naturalWidth, h: img.naturalHeight });
        img.onerror = () => ok({ w: 400, h: 300 });
        img.src = data;
      });
      const rel = await SaveBoardImage(path, file.name || "image.png", data.slice(data.indexOf(",") + 1));
      images[rel] = data;
      const k = Math.min(1, 480 / Math.max(size.w, size.h));
      remember();
      const el: El = { id: newId(), type: "image", x: at.x - (size.w * k) / 2, y: at.y - (size.h * k) / 2, w: size.w * k, h: size.h * k, stroke: "ink", fill: "none", width: 0, dash: "solid", file: rel };
      elements.push(el);
      selected = new Set([el.id]);
      tool = "select";
      changed();
    } catch (e) { onError(clean(e)); }
  }
  function drop(e: DragEvent) {
    const file = [...(e.dataTransfer?.files ?? [])].find((f) => f.type.startsWith("image/"));
    if (!file) return;
    e.preventDefault();
    addImage(file, toWorld(e.clientX, e.clientY));
  }
  function center() {
    const r = host!.getBoundingClientRect();
    return toWorld(r.left + r.width / 2, r.top + r.height / 2);
  }

  // Export: the drawing as a PNG in the attachments folder.
  let world = $state<SVGGElement>();
  async function exportPNG() {
    if (!elements.length || !world) return;
    try {
      const b = bounds(elements), pad = 24;
      const bg = getComputedStyle(host!).backgroundColor;
      const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${b.w + pad * 2}" height="${b.h + pad * 2}" viewBox="${b.x - pad} ${b.y - pad} ${b.w + pad * 2} ${b.h + pad * 2}"><rect x="${b.x - pad}" y="${b.y - pad}" width="${b.w + pad * 2}" height="${b.h + pad * 2}" fill="${bg}"/>${world.innerHTML}</svg>`;
      const img = new Image();
      await new Promise((ok, bad) => { img.onload = ok; img.onerror = bad; img.src = "data:image/svg+xml;charset=utf-8," + encodeURIComponent(svg); });
      const c = document.createElement("canvas");
      c.width = (b.w + pad * 2) * 2; c.height = (b.h + pad * 2) * 2;
      const g = c.getContext("2d")!;
      g.scale(2, 2);
      g.drawImage(img, 0, 0);
      const png = c.toDataURL("image/png");
      const name = (path.split("/").pop() ?? "drawing").replace(/\.draw$/i, "") + ".png";
      const rel = await SaveBoardImage(path, name, png.slice(png.indexOf(",") + 1));
      toast(td.exported.replace("{path}", rel));
    } catch (e) { onError(clean(e)); }
  }

  function key(e: KeyboardEvent) {
    if ((e.target as HTMLElement)?.closest?.("input, textarea, [contenteditable]")) {
      if (e.key === "Escape") { e.preventDefault(); finishText(); }
      return;
    }
    if (!host?.isConnected) return;
    if (zoomKey(e)) return;
    // Keys go to the board unless something else has the focus (the files
    // panel, the ribbon, a window over it); Ctrl+Alt is the Studio's.
    if ((e.target as HTMLElement)?.closest?.(".side, .ribbon, [role=dialog], [role=menu]") || e.altKey) return;
    const mod = e.ctrlKey || e.metaKey, k = e.key.toLowerCase();
    if (e.key === " " && !space) { space = true; e.preventDefault(); return; }
    if (mod && k === "z") { e.preventDefault(); if (e.shiftKey) redo(); else undo(); }
    else if (mod && k === "y") { e.preventDefault(); redo(); }
    else if (mod && k === "a") { e.preventDefault(); selected = new Set(elements.map((x) => x.id)); tool = "select"; }
    else if (mod && k === "d") { e.preventDefault(); duplicate(); }
    else if (mod) return;
    else if (e.key === "Delete" || e.key === "Backspace") { e.preventDefault(); remove(); }
    else if (e.key === "Escape") { if (full && !chosen.length && tool === "select") onFull(); selected = new Set(); tool = "select"; }
    else if (k === "f") { e.preventDefault(); onFull(); }
    else if (e.key === "Enter" && chosen.length === 1 && (chosen[0].type === "text" || isShape(chosen[0]))) { e.preventDefault(); startText(chosen[0]); }
    else if (e.key.startsWith("Arrow") && chosen.length) {
      e.preventDefault();
      const s = e.shiftKey ? 10 : 1;
      remember();
      for (const el of chosen) { el.x += k === "arrowleft" ? -s : k === "arrowright" ? s : 0; el.y += k === "arrowup" ? -s : k === "arrowdown" ? s : 0; }
      changed();
    } else {
      const found = TOOLS.find((x) => x.key === k);
      if (found) { tool = found.tool; if (found.tool !== "select") selected = new Set(); }
    }
  }

  // Changed on disk by something else (an AI in the terminal, sync):
  // opened again, unless there are changes of its own.
  const off = Events.On("notesChanged", async (ev) => {
    const list = (ev.data?.paths ?? []).map((p) => p.toLowerCase());
    if (!list.includes(path.toLowerCase())) return;
    const now = await ReadBoard(path).catch(() => written);
    if (now.trimEnd() !== written.trimEnd() && !dirty) onOutside();
  });
  onMount(() => bindSave?.(save));
  onDestroy(() => { off(); save(); });

  const sel = $derived(chosen.length && !editing && drag?.kind !== "draw" ? bounds(chosen) : null);
  const line = $derived(chosen.length === 1 && isPath(chosen[0]) && chosen[0].type !== "pen" ? chosen[0] : null);
  const box = $derived(drag?.kind === "box" ? drag : null);
  const editEl = $derived(editing ? byId.get(editing) : undefined);
  const cursor = $derived(drag?.kind === "pan" ? "grabbing" : tool === "hand" || space ? "grab" : tool === "select" ? "default" : tool === "text" ? "text" : "crosshair");
  const WIDTHS = [1, 2, 4];
  const SIZES = [16, 20, 28, 40];
</script>

<svelte:window onkeydown={key} onkeyup={(e) => { if (e.key === " ") space = false; }} onpaste={paste} oncopy={copy} />

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="draw" bind:this={host} style:cursor={cursor} style:background-position="{pan.x}px {pan.y}px" style:background-size="{20 * zoom}px {20 * zoom}px"
  role="application" aria-label={td.board} onmousedown={(e) => { if (tool === "text" || editing) e.preventDefault(); }} onpointerdown={down} onpointermove={move} onpointerup={up} onwheel={wheel} ondblclick={dbl}
  ondragover={(e) => e.preventDefault()} ondrop={drop}>
  <svg class="paper">
    <g transform="translate({pan.x} {pan.y}) scale({zoom})">
      <g bind:this={world}>
        {#each elements as el (el.id)}
          {@const c = ink(el.stroke)}
          {@const dash = dashArray(el.dash, el.width)}
          {@const fill = el.fill === "none" ? "none" : ink(el.fill)}
          {@const fo = el.fillStyle === "solid" ? 1 : 0.25}
          <g data-el={el.id} class:erasing={tool === "eraser"}>
            {#if el.type === "rect"}
              <rect x={Math.min(el.x, el.x + el.w)} y={Math.min(el.y, el.y + el.h)} width={Math.abs(el.w)} height={Math.abs(el.h)} rx={Math.min(8, Math.abs(el.w) / 4, Math.abs(el.h) / 4)} fill={fill} fill-opacity={fo} stroke={c} stroke-width={el.width} stroke-dasharray={dash} stroke-linecap="round" pointer-events="all" />
            {:else if el.type === "ellipse"}
              <ellipse cx={el.x + el.w / 2} cy={el.y + el.h / 2} rx={Math.abs(el.w / 2)} ry={Math.abs(el.h / 2)} fill={fill} fill-opacity={fo} stroke={c} stroke-width={el.width} stroke-dasharray={dash} stroke-linecap="round" pointer-events="all" />
            {:else if el.type === "diamond"}
              <polygon points="{el.x + el.w / 2},{el.y} {el.x + el.w},{el.y + el.h / 2} {el.x + el.w / 2},{el.y + el.h} {el.x},{el.y + el.h / 2}" fill={fill} fill-opacity={fo} stroke={c} stroke-width={el.width} stroke-dasharray={dash} stroke-linejoin="round" stroke-linecap="round" pointer-events="all" />
            {:else if el.type === "arrow" || el.type === "line"}
              {@const pts = worldPoints(el)}
              <path d={"M" + pts.map((p) => p.join(",")).join(" L")} fill="none" stroke={c} stroke-width={el.width} stroke-dasharray={dash} stroke-linecap="round" />
              {#if el.type === "arrow" && pts.length > 1}<path d={arrowHead(pts[pts.length - 2], pts[pts.length - 1], el.width)} fill="none" stroke={c} stroke-width={el.width} stroke-linecap="round" stroke-linejoin="round" />{/if}
              <path d={"M" + pts.map((p) => p.join(",")).join(" L")} fill="none" stroke="transparent" pointer-events="stroke" stroke-width={Math.max(14, el.width + 10)} />
            {:else if el.type === "pen"}
              <path d={penPath(worldPoints(el))} fill="none" stroke={c} stroke-width={el.width} stroke-dasharray={dash} stroke-linecap="round" stroke-linejoin="round" />
              <path d={penPath(worldPoints(el))} fill="none" stroke="transparent" pointer-events="stroke" stroke-width={Math.max(14, el.width + 10)} />
            {:else if el.type === "image"}
              <image href={images[el.file ?? ""] || undefined} x={el.x} y={el.y} width={el.w} height={el.h} preserveAspectRatio="none" />
              <rect x={el.x} y={el.y} width={el.w} height={el.h} fill="transparent" stroke={images[el.file ?? ""] ? "none" : c} stroke-dasharray="4 4" pointer-events="all" />
            {/if}
            {#if el.text && editing !== el.id}
              {@const size = el.size ?? 20}
              {@const lines = el.text.split("\n")}
              {#if el.type === "text"}
                <rect x={el.x} y={el.y} width={el.w} height={el.h} fill="transparent" pointer-events="all" />
                <text x={el.x} y={el.y} fill={c} font-size={size} font-family="Inter, ui-sans-serif, system-ui, sans-serif" pointer-events="all">
                  {#each lines as l, i (i)}<tspan x={el.x} y={el.y + size * (0.95 + i * LINE)}>{l || " "}</tspan>{/each}
                </text>
              {:else}
                {@const top = el.y + el.h / 2 - (lines.length * size * LINE) / 2}
                <text fill={c} font-size={size} font-family="Inter, ui-sans-serif, system-ui, sans-serif" text-anchor="middle" pointer-events="none">
                  {#each lines as l, i (i)}<tspan x={el.x + el.w / 2} y={top + size * (0.95 + i * LINE)}>{l || " "}</tspan>{/each}
                </text>
              {/if}
            {/if}
          </g>
        {/each}
      </g>

      <!-- The selection: its box and corners, or a line's two ends. -->
      {#if line && !drag}
        {@const pts = worldPoints(line)}
        <circle class="handle" role="presentation" cx={pts[0][0]} cy={pts[0][1]} r={6 / zoom} stroke-width={1.5 / zoom} onpointerdown={(e) => downEnd(e, line, 0)} />
        <circle class="handle" role="presentation" cx={pts[pts.length - 1][0]} cy={pts[pts.length - 1][1]} r={6 / zoom} stroke-width={1.5 / zoom} onpointerdown={(e) => downEnd(e, line, 1)} />
      {:else if sel}
        {@const m = 6 / zoom}
        <rect class="selbox" x={sel.x - m} y={sel.y - m} width={sel.w + 2 * m} height={sel.h + 2 * m} stroke-width={1 / zoom} stroke-dasharray="{4 / zoom} {3 / zoom}" />
        {#if drag?.kind !== "move"}
          {#each [[sel.x - m, sel.y - m], [sel.x + sel.w + m, sel.y - m], [sel.x + sel.w + m, sel.y + sel.h + m], [sel.x - m, sel.y + sel.h + m]] as [cx, cy], i (i)}
            <rect class="handle corner{i}" role="presentation" x={cx - 5 / zoom} y={cy - 5 / zoom} width={10 / zoom} height={10 / zoom} rx={2 / zoom} stroke-width={1.5 / zoom} onpointerdown={(e) => downCorner(e, i)} />
          {/each}
        {/if}
      {/if}
      {#if box}
        <rect class="boxsel" x={Math.min(box.a.x, box.b.x)} y={Math.min(box.a.y, box.b.y)} width={Math.abs(box.a.x - box.b.x)} height={Math.abs(box.a.y - box.b.y)} stroke-width={1 / zoom} />
      {/if}
    </g>
  </svg>

  <!-- Writing: over the text (or in the middle of the shape). -->
  {#if editEl}
    {@const size = (editEl.size ?? style.size) * zoom}
    {@const shape = editEl.type !== "text"}
    <textarea class="write" class:shape bind:value={draft} use:focus oninput={typed} onblur={finishText} onpointerdown={(e) => e.stopPropagation()}
      style:color={ink(editEl.stroke)} style:font-size="{size}px" style:line-height={LINE}
      style:left="{shape ? editEl.x * zoom + pan.x : editEl.x * zoom + pan.x - 2}px" style:top="{shape ? (editEl.y + editEl.h / 2) * zoom + pan.y - size * LINE / 2 : editEl.y * zoom + pan.y}px"
      style:width="{shape ? editEl.w * zoom : Math.max(editEl.w * zoom + size, 40)}px" style:height="{(draft.split('\n').length) * size * LINE + 4}px"
      placeholder={shape ? "" : td.textPh} rows="1"></textarea>
  {/if}

  {#if loaded && !elements.length && tool === "select"}
    <div class="empty muted"><b>{td.emptyTitle}</b><p>{td.emptyLead}</p></div>
  {/if}
</div>

<!-- The tools, on top. -->
<div class="tools" role="toolbar" aria-label={td.tools}>
  {#each TOOLS as x (x.tool)}
    <button class={{ on: tool === x.tool }} title="{x.label} ({x.key.toUpperCase()})" aria-label={x.label} onclick={() => { tool = x.tool; if (x.tool !== "select") selected = new Set(); }}>
      <x.Icon size={17} strokeWidth={1.75} /><small>{x.key.toUpperCase()}</small>
    </button>
    {#if x.tool === "select" || x.tool === "pen"}<span class="tsep"></span>{/if}
  {/each}
  <button title={td.image} aria-label={td.image} onclick={() => picker?.click()}><ImagePlus size={17} strokeWidth={1.75} /></button>
  <input type="file" accept="image/*" hidden bind:this={picker} onchange={(e) => { const f = e.currentTarget.files?.[0]; if (f) addImage(f); e.currentTarget.value = ""; }} />
</div>

<!-- The style of the selection (or of what comes next). -->
{#if showPanel}
  <div class="stylepanel" role="group" aria-label={td.style}>
    <span class="plabel">{td.stroke}</span>
    <div class="swrow">
      {#each PALETTE as c (c)}
        <button class={["sw", { on: shown.stroke === c }]} style:background={ink(c)} title={td.colors[c]} aria-label={td.colors[c]} onclick={() => setStyle("stroke", c)}></button>
      {/each}
    </div>
    {#if showFill}
      <span class="plabel">{td.fill}</span>
      <div class="swrow">
        <button class={["sw", "none", { on: shown.fill === "none" }]} title={td.noFill} aria-label={td.noFill} onclick={() => setStyle("fill", "none")}></button>
        {#each PALETTE.filter((c) => c !== "ink") as c (c)}
          <button class={["sw", { on: shown.fill === c }]} style:background={ink(c)} title={td.colors[c]} aria-label={td.colors[c]} onclick={() => setStyle("fill", c)}></button>
        {/each}
      </div>
      {#if shown.fill !== "none"}
        <div class="swrow seg">
          <button class={{ on: shown.fillStyle !== "solid" }} onclick={() => setStyle("fillStyle", "tint")}>{td.tint}</button>
          <button class={{ on: shown.fillStyle === "solid" }} onclick={() => setStyle("fillStyle", "solid")}>{td.solid}</button>
        </div>
      {/if}
    {/if}
    <span class="plabel">{td.width}</span>
    <div class="swrow seg">
      {#each WIDTHS as w (w)}
        <button class={{ on: shown.width === w }} title="{w}px" aria-label="{w}px" onclick={() => setStyle("width", w)}><span class="wline" style:height="{w + 1}px"></span></button>
      {/each}
    </div>
    <span class="plabel">{td.dash}</span>
    <div class="swrow seg">
      {#each ["solid", "dashed", "dotted"] as const as d (d)}
        <button class={{ on: shown.dash === d }} title={td[d]} aria-label={td[d]} onclick={() => setStyle("dash", d)}><svg width="26" height="6" aria-hidden="true"><line x1="2" y1="3" x2="24" y2="3" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-dasharray={dashArray(d, 2)} /></svg></button>
      {/each}
    </div>
    {#if showText}
      <span class="plabel">{td.size}</span>
      <div class="swrow seg">
        {#each SIZES as s, i (s)}
          <button class={{ on: shown.size === s }} onclick={() => setStyle("size", s)}>{["P", "M", "G", "GG"][i]}</button>
        {/each}
      </div>
    {/if}
    {#if chosen.length}
      <span class="plabel">{td.actions}</span>
      <div class="swrow acts">
        <button title={td.toBack} aria-label={td.toBack} onclick={() => order(false)}><SendToBack size={16} /></button>
        <button title={td.toFront} aria-label={td.toFront} onclick={() => order(true)}><BringToFront size={16} /></button>
        <button title="{td.duplicate} (Ctrl+D)" aria-label={td.duplicate} onclick={() => duplicate()}><Copy size={16} /></button>
        <button title="{td.remove} (Delete)" aria-label={td.remove} onclick={remove}><Trash2 size={16} /></button>
      </div>
    {/if}
  </div>
{/if}

<div class="controls" role="toolbar" aria-label={td.view}>
  <button title={td.zoomOut} aria-label={td.zoomOut} onclick={() => zoomBy(1 / 1.2)}><ZoomOut size={16} strokeWidth={1.75} /></button>
  <button class="pct" title={td.fit} onclick={fit}>{Math.round(zoom * 100)}%</button>
  <button title={td.zoomIn} aria-label={td.zoomIn} onclick={() => zoomBy(1.2)}><ZoomIn size={16} strokeWidth={1.75} /></button>
  <button title={td.fit} aria-label={td.fit} onclick={fit}><Scan size={16} strokeWidth={1.75} /></button>
  <button title="{full ? td.exitFull : td.full} (F)" aria-label={full ? td.exitFull : td.full} onclick={onFull}>{#if full}<Minimize2 size={16} strokeWidth={1.75} />{:else}<Maximize2 size={16} strokeWidth={1.75} />{/if}</button>
  <span class="csep"></span>
  <button title="{td.undo} (Ctrl+Z)" aria-label={td.undo} disabled={!canUndo} onclick={undo}><Undo2 size={16} strokeWidth={1.75} /></button>
  <button title="{td.redo} (Ctrl+Shift+Z)" aria-label={td.redo} disabled={!canRedo} onclick={redo}><Redo2 size={16} strokeWidth={1.75} /></button>
</div>
<button class="export btn ghost" disabled={!elements.length} onclick={exportPNG}><Download size={15} />{td.export}</button>

<style>
  .draw { position: absolute; inset: 0; overflow: hidden; touch-action: none; -webkit-user-select: none; user-select: none; background-color: var(--bg);
    background-image: radial-gradient(circle, color-mix(in srgb, var(--muted) 35%, transparent) 1px, transparent 1.2px); }
  .paper { position: absolute; inset: 0; width: 100%; height: 100%; }
  .erasing:hover { opacity: .35; }
  .selbox { fill: none; stroke: var(--blue); pointer-events: none; }
  .boxsel { fill: color-mix(in srgb, var(--blue) 10%, transparent); stroke: var(--blue); pointer-events: none; }
  .handle { fill: var(--panel); stroke: var(--blue); cursor: move; }
  .corner0, .corner2 { cursor: nwse-resize; }
  .corner1, .corner3 { cursor: nesw-resize; }
  .write { position: absolute; z-index: 2; padding: 0; border: 0; outline: 1px dashed var(--blue); outline-offset: 2px; background: transparent; resize: none; overflow: hidden; white-space: pre; font-family: "Inter", ui-sans-serif, system-ui, sans-serif; }
  .write.shape { text-align: center; }
  .empty { position: absolute; inset: 0; display: grid; place-content: center; text-align: center; gap: .35rem; pointer-events: none; }
  .empty b { color: var(--text); font-size: 1.05rem; }
  .empty p { max-width: 28rem; font-size: .9rem; }

  .tools { position: absolute; z-index: 3; inset-block-start: .8rem; inset-inline-start: 50%; transform: translateX(-50%); display: flex; align-items: center; gap: .15rem; padding: .3rem; border: 1px solid var(--line); border-radius: .7rem; background: var(--panel); box-shadow: 0 8px 24px rgb(0 0 0 / .1); }
  .tools button { position: relative; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .5rem; color: var(--muted); }
  .tools button:hover { background: var(--soft); color: var(--text); }
  .tools button.on { background: color-mix(in srgb, var(--blue) 16%, transparent); color: var(--blue); }
  .tools small { position: absolute; inset-inline-end: 3px; inset-block-end: 1px; font-size: .55rem; opacity: .7; }
  .tsep { width: 1px; height: 1.4rem; margin-inline: .2rem; background: var(--line); }

  .stylepanel { position: absolute; z-index: 3; inset-block-start: 4.2rem; inset-inline-start: .8rem; display: grid; gap: .35rem; width: 13.5rem; padding: .75rem; border: 1px solid var(--line); border-radius: .7rem; background: var(--panel); box-shadow: 0 8px 24px rgb(0 0 0 / .1); }
  .plabel { font-size: .72rem; color: var(--muted); margin-block-start: .2rem; }
  .swrow { display: flex; flex-wrap: wrap; gap: .3rem; }
  .sw { width: 1.35rem; height: 1.35rem; border-radius: .35rem; border: 1px solid var(--line); }
  .sw.on { outline: 2px solid var(--blue); outline-offset: 1px; }
  .sw.none { background: linear-gradient(135deg, transparent 45%, var(--red) 45% 55%, transparent 55%), var(--panel); }
  .seg button { flex: 1; display: grid; place-items: center; min-height: 1.9rem; padding: 0 .4rem; border: 1px solid var(--line); border-radius: .4rem; font-size: .78rem; color: var(--muted); }
  .seg button:hover { color: var(--text); }
  .seg button.on { border-color: var(--blue); color: var(--text); background: color-mix(in srgb, var(--blue) 10%, transparent); }
  .wline { display: block; width: 1.4rem; border-radius: 2px; background: currentColor; }
  .acts button { display: grid; place-items: center; width: 2rem; height: 2rem; border: 1px solid var(--line); border-radius: .4rem; color: var(--muted); }
  .acts button:hover { color: var(--text); background: var(--soft); }

  .controls { position: absolute; z-index: 3; inset-block-end: .9rem; inset-inline-start: .8rem; display: flex; align-items: center; gap: .1rem; padding: .25rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
  .controls button { display: grid; place-items: center; min-width: 2rem; height: 2rem; border-radius: .4rem; color: var(--muted); }
  .controls button:hover:not(:disabled) { background: var(--soft); color: var(--text); }
  .controls button:disabled { opacity: .35; }
  .controls .pct { min-width: 3.2rem; font-size: .8rem; font-variant-numeric: tabular-nums; }
  .csep { width: 1px; height: 1.2rem; margin-inline: .2rem; background: var(--line); }
  .export { position: absolute; z-index: 3; inset-block-start: .8rem; inset-inline-end: .8rem; display: flex; gap: .4rem; align-items: center; background: var(--panel); }
</style>
