// A drawing of the vault (.draw): the Studio's own sketches. Shapes,
// arrows and lines, free strokes, text and images on an endless board,
// saved as JSON:
//
//   {
//     "version":1,
//     "elements":[
//       {"id":"…","type":"rect","x":0,"y":0,"w":200,"h":100,"stroke":"ink",…}
//     ]
//   }
//
// Every element has a box (x, y, w, h). Lines, arrows and strokes keep
// their points inside it (relative to x, y), so moving one is moving its
// box and resizing it is scaling its points.

export type Kind = "rect" | "ellipse" | "diamond" | "arrow" | "line" | "pen" | "text" | "image";
export type Dash = "solid" | "dashed" | "dotted";
export type El = {
  id: string;
  type: Kind;
  x: number; y: number; w: number; h: number;
  points?: [number, number][]; // arrow, line, pen
  stroke: string; // a color of the palette, or #hex
  fill: string; // a color, or "none"
  fillStyle?: "tint" | "solid";
  width: number; // of the stroke
  dash: Dash;
  text?: string; // text, or the label inside a shape
  size?: number; // of the text
  file?: string; // image: its path in the vault
  [k: string]: unknown;
};
export type Doc = { elements: El[]; rest: Record<string, unknown> };

// A drawing may come from anywhere (another computer, through sync):
// nothing in it is trusted. Each element is checked from scratch: what can
// be mended is (a color, a stroke width, a number that isn't one), what
// can't is left out (an unknown kind, a line without its points, an image
// without its file, a repeated id). The keys this file doesn't know stay
// as they are, so writing it back keeps them.
export const LIMITS = { elements: 20000, points: 20000, coord: 1e7, size: 1e6 } as const;
const KINDS: readonly Kind[] = ["rect", "ellipse", "diamond", "arrow", "line", "pen", "text", "image"];
const DASHES: readonly Dash[] = ["solid", "dashed", "dotted"];
const FILL_STYLES: readonly string[] = ["tint", "solid"];

type Obj = Record<string, unknown>;
const isObj = (v: unknown): v is Obj => typeof v === "object" && v !== null && !Array.isArray(v);
const oneOf = <T extends string>(list: readonly T[], v: unknown): v is T => typeof v === "string" && (list as readonly string[]).includes(v);
const finite = (v: unknown): v is number => typeof v === "number" && Number.isFinite(v);
const clamp = (v: number, lim: number) => Math.min(Math.max(v, -lim), lim);
// A position or a (signed) size: finite and within the limits, else 0.
const num = (v: unknown, lim: number) => (finite(v) ? clamp(v, lim) : 0);

// A line's points: pairs of finite numbers; others are dropped.
function toPoints(v: unknown): [number, number][] {
  if (!Array.isArray(v)) return [];
  const out: [number, number][] = [];
  for (const p of v) {
    if (out.length >= LIMITS.points) break;
    if (Array.isArray(p) && p.length >= 2 && finite(p[0]) && finite(p[1])) out.push([clamp(p[0], LIMITS.coord), clamp(p[1], LIMITS.coord)]);
  }
  return out;
}

function toEl(v: unknown): El | null {
  if (!isObj(v) || typeof v.id !== "string" || !v.id || !oneOf(KINDS, v.type)) return null;
  const type = v.type;
  if (type === "image" && (typeof v.file !== "string" || !v.file)) return null;
  const e: El = {
    ...v,
    id: v.id,
    type,
    x: num(v.x, LIMITS.coord),
    y: num(v.y, LIMITS.coord),
    w: num(v.w, LIMITS.size),
    h: num(v.h, LIMITS.size),
    stroke: typeof v.stroke === "string" ? v.stroke : "ink",
    fill: typeof v.fill === "string" ? v.fill : "none",
    width: finite(v.width) && v.width >= 0 ? Math.min(v.width, 100) : 2,
    dash: oneOf(DASHES, v.dash) ? v.dash : "solid",
  };
  if (type === "arrow" || type === "line" || type === "pen") {
    const pts = toPoints(v.points);
    // A line or an arrow goes from a point to another; a stroke may be a dot.
    if (pts.length < (type === "pen" ? 1 : 2)) return null;
    e.points = pts;
  } else delete e.points;
  if ("fillStyle" in e && !oneOf(FILL_STYLES, e.fillStyle)) delete e.fillStyle;
  if ("text" in e && typeof e.text !== "string") delete e.text;
  if (type === "text" && e.text === undefined) e.text = "";
  if ("size" in e && !(finite(e.size) && e.size > 0)) delete e.size;
  else if (e.size !== undefined) e.size = Math.min(e.size, 10000);
  if ("file" in e && typeof e.file !== "string") delete e.file;
  return e;
}

export function parseDraw(text: string): Doc {
  const j: unknown = text.trim() ? JSON.parse(text) : {};
  if (!isObj(j)) return { elements: [], rest: {} };
  const { elements, version: _v, ...rest } = j;
  const seen = new Set<string>();
  const out: El[] = [];
  for (const raw of Array.isArray(elements) ? elements : []) {
    const e = toEl(raw);
    if (!e || seen.has(e.id)) continue;
    if (out.length >= LIMITS.elements) break;
    seen.add(e.id);
    out.push(e);
  }
  return { elements: out, rest };
}

const r2 = (n: number) => Math.round(n * 100) / 100;

// Tabs, and a line per element (a change shows as a changed line).
export function writeDraw(d: Doc): string {
  const els = d.elements.map((e) => JSON.stringify({ ...e, x: r2(e.x), y: r2(e.y), w: r2(e.w), h: r2(e.h), ...(e.points ? { points: e.points.map(([x, y]) => [r2(x), r2(y)]) } : {}) }));
  let out = "{\n\t\"version\":1,\n\t\"elements\":[" + (els.length ? "\n\t\t" + els.join(",\n\t\t") + "\n\t" : "") + "]";
  for (const [k, v] of Object.entries(d.rest)) out += ",\n\t" + JSON.stringify(k) + ":" + JSON.stringify(v);
  return out + "\n}";
}

export function newId() {
  const b = crypto.getRandomValues(new Uint8Array(8));
  return Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
}

// The palette. "ink" is the color of text: dark on light, light on dark.
export const PALETTE = ["ink", "red", "orange", "yellow", "green", "blue", "purple", "gray"] as const;
const HEX: Record<string, string> = { red: "#e03131", orange: "#f08c00", yellow: "#e0b000", green: "#2f9e44", blue: "#1971c2", purple: "#7048e8", gray: "#868e96" };
export function colorOf(c: string, dark: boolean) {
  if (c === "ink") return dark ? "#e9ecef" : "#1e1e1e";
  return HEX[c] ?? (/^#[0-9a-f]{3,8}$/i.test(c) ? c : dark ? "#e9ecef" : "#1e1e1e");
}

export const isPath = (e: El) => e.type === "arrow" || e.type === "line" || e.type === "pen";
export const isShape = (e: El) => e.type === "rect" || e.type === "ellipse" || e.type === "diamond";

// A line's points (world) into its box: the box around them, the points
// relative to it.
export function fitPoints(e: El, world: [number, number][]) {
  const xs = world.map((p) => p[0]), ys = world.map((p) => p[1]);
  const x = Math.min(...xs), y = Math.min(...ys);
  e.x = x; e.y = y;
  e.w = Math.max(...xs) - x; e.h = Math.max(...ys) - y;
  e.points = world.map(([px, py]) => [px - x, py - y]);
}
export const worldPoints = (e: El) => (e.points ?? []).map(([x, y]) => [e.x + x, e.y + y] as [number, number]);

// A negative box (drawn up or to the left) made positive.
export function normalize(e: El) {
  if (isPath(e)) { fitPoints(e, worldPoints(e)); return; }
  if (e.w < 0) { e.x += e.w; e.w = -e.w; }
  if (e.h < 0) { e.y += e.h; e.h = -e.h; }
}

// Resizing: the element scaled from its box (b0) to a new one.
export function scaleTo(e: El, b0: Box, pts0: [number, number][] | undefined, b: Box) {
  e.x = b.x; e.y = b.y; e.w = b.w; e.h = b.h;
  if (pts0) {
    const sx = b0.w ? b.w / b0.w : 1, sy = b0.h ? b.h / b0.h : 1;
    e.points = pts0.map(([x, y]) => [x * sx, y * sy]);
  }
  if (e.type === "text" && b0.h) e.size = Math.max(6, (e.size ?? 20) * (Math.abs(b.h) / b0.h));
}

export type Box = { x: number; y: number; w: number; h: number };
export function bounds(els: El[]): Box {
  if (!els.length) return { x: 0, y: 0, w: 0, h: 0 };
  const x = Math.min(...els.map((e) => e.x)), y = Math.min(...els.map((e) => e.y));
  return { x, y, w: Math.max(...els.map((e) => e.x + e.w)) - x, h: Math.max(...els.map((e) => e.y + e.h)) - y };
}

// Shift while drawing: squares and circles, lines at 45 degrees.
export function snapBox(w: number, h: number) {
  const s = Math.max(Math.abs(w), Math.abs(h));
  return { w: Math.sign(w || 1) * s, h: Math.sign(h || 1) * s };
}
export function snapAngle(dx: number, dy: number) {
  const a = Math.round(Math.atan2(dy, dx) / (Math.PI / 4)) * (Math.PI / 4), d = Math.hypot(dx, dy);
  return [Math.cos(a) * d, Math.sin(a) * d] as [number, number];
}

// A free stroke, smoothed: curves through the middles of its points.
export function penPath(pts: [number, number][]) {
  if (!pts.length) return "";
  if (pts.length < 3) return `M${pts[0][0]},${pts[0][1]} ` + pts.map(([x, y]) => `L${x},${y}`).join(" ");
  let d = `M${pts[0][0]},${pts[0][1]}`;
  for (let i = 1; i < pts.length - 1; i++) {
    const mx = (pts[i][0] + pts[i + 1][0]) / 2, my = (pts[i][1] + pts[i + 1][1]) / 2;
    d += ` Q${pts[i][0]},${pts[i][1]} ${mx},${my}`;
  }
  const l = pts[pts.length - 1];
  return d + ` L${l[0]},${l[1]}`;
}

// The two strokes of an arrow's head at the end of a line.
export function arrowHead(from: [number, number], to: [number, number], width: number) {
  const a = Math.atan2(to[1] - from[1], to[0] - from[0]);
  const len = Math.min(Math.max(10, width * 5), Math.hypot(to[0] - from[0], to[1] - from[1]) * 0.6 || 10);
  const p = (s: number) => `${to[0] - len * Math.cos(a + s)},${to[1] - len * Math.sin(a + s)}`;
  return `M${p(0.45)} L${to[0]},${to[1]} L${p(-0.45)}`;
}

export function dashArray(dash: Dash, width: number) {
  if (dash === "dashed") return `${width * 4} ${width * 3}`;
  if (dash === "dotted") return `0 ${width * 2.5}`;
  return undefined;
}

// Text: its lines and size. The width is measured with the Studio's font.
export const LINE = 1.25;
let ctx: CanvasRenderingContext2D | null = null;
export function measure(text: string, size: number) {
  ctx ??= document.createElement("canvas").getContext("2d");
  const lines = text.split("\n");
  if (ctx) ctx.font = `${size}px Inter, ui-sans-serif, system-ui, sans-serif`;
  const w = Math.max(...lines.map((l) => ctx?.measureText(l || " ").width ?? l.length * size * 0.55));
  return { w: Math.ceil(w) + 2, h: Math.ceil(lines.length * size * LINE) };
}
