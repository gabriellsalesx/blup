// A canvas of the vault, in JSON Canvas 1.0 (jsoncanvas.org), the open
// format Obsidian made for its canvases: cards (text, a file of the vault,
// a web address, a group) placed on an endless board, and arrows between
// them. What this file doesn't know (other keys) is kept as it is.

export type Side = "top" | "right" | "bottom" | "left";
export type CNode = {
  id: string;
  type: "text" | "file" | "link" | "group";
  x: number; y: number; width: number; height: number;
  color?: string;
  text?: string; // text
  file?: string; subpath?: string; // file
  url?: string; // link
  label?: string; // group
  [k: string]: unknown;
};
export type CEdge = {
  id: string;
  fromNode: string; fromSide?: Side; fromEnd?: "none" | "arrow";
  toNode: string; toSide?: Side; toEnd?: "none" | "arrow";
  color?: string;
  label?: string;
  [k: string]: unknown;
};
// A card of a kind the Blup doesn't know (a plugin's), or an arrow to one:
// never shown nor read, only kept as it came, with its place in the list,
// to go back to the file when the canvas is saved.
export type Hidden = { at: number; item: Record<string, unknown> };
export type Canvas = { nodes: CNode[]; edges: CEdge[]; rest: Record<string, unknown>; hidden: { nodes: Hidden[]; edges: Hidden[] } };

// A canvas file may come from anywhere (another computer, through sync;
// another app): nothing in it is trusted. Each card and arrow is checked
// from scratch: what can be mended is (a size or position that isn't a
// number), what can't is left out (a repeated id, an arrow to a card that
// isn't there). A card of an unknown kind is set aside untouched, with the
// arrows to it, and written back as it was. The keys this file doesn't
// know stay as they are, so writing it back keeps them (Obsidian's or a
// plugin's).
export const LIMITS = { nodes: 20000, edges: 40000, coord: 1e7, size: 1e6 } as const;
const NODE_TYPES: readonly CNode["type"][] = ["text", "file", "link", "group"];
const SIDES: readonly Side[] = ["top", "right", "bottom", "left"];
const ENDS: readonly string[] = ["none", "arrow"];

type Obj = Record<string, unknown>;
const isObj = (v: unknown): v is Obj => typeof v === "object" && v !== null && !Array.isArray(v);
const oneOf = <T extends string>(list: readonly T[], v: unknown): v is T => typeof v === "string" && (list as readonly string[]).includes(v);
const finite = (v: unknown): v is number => typeof v === "number" && Number.isFinite(v);
// A position: a finite number within the board's limits (else 0).
const coord = (v: unknown) => (finite(v) ? Math.min(Math.max(v, -LIMITS.coord), LIMITS.coord) : 0);
// A size: a finite number, at least 1 (else the default).
const size = (v: unknown, def: number) => (finite(v) && v >= 1 ? Math.min(v, LIMITS.size) : def);

// The optional keys of a card or arrow that must be text: dropped when
// they aren't.
function textKeys(o: Obj, keys: string[]) {
  for (const k of keys) if (k in o && typeof o[k] !== "string") delete o[k];
}

function toNode(v: unknown): CNode | null {
  if (!isObj(v) || typeof v.id !== "string" || !v.id || !oneOf(NODE_TYPES, v.type)) return null;
  const type = v.type;
  // A card of a file without its file is nothing to show.
  if (type === "file" && typeof v.file !== "string") return null;
  const n: CNode = {
    ...v,
    id: v.id,
    type,
    x: coord(v.x),
    y: coord(v.y),
    width: size(v.width, 250),
    height: size(v.height, 60),
  };
  textKeys(n, ["color", "text", "file", "subpath", "url", "label"]);
  if (type === "text" && n.text === undefined) n.text = "";
  if (type === "link" && n.url === undefined) n.url = "";
  return n;
}

function toEdge(v: unknown, ids: Set<string>): CEdge | null {
  if (!isObj(v) || typeof v.id !== "string" || !v.id) return null;
  const { fromNode, toNode } = v;
  if (typeof fromNode !== "string" || typeof toNode !== "string" || !ids.has(fromNode) || !ids.has(toNode)) return null;
  const e: CEdge = { ...v, id: v.id, fromNode, toNode };
  for (const k of ["fromSide", "toSide"]) if (k in e && !oneOf(SIDES, e[k])) delete e[k];
  for (const k of ["fromEnd", "toEnd"]) if (k in e && !oneOf(ENDS, e[k])) delete e[k];
  textKeys(e, ["color", "label"]);
  return e;
}

// A card of a kind this file doesn't know: an object with an id and a
// kind that is some other text.
const unknownKind = (v: unknown): v is Obj =>
  isObj(v) && typeof v.id === "string" && !!v.id && typeof v.type === "string" && !!v.type && !oneOf(NODE_TYPES, v.type);

// An arrow kept aside: its ends must be cards of the canvas, shown or not.
const hiddenEdge = (v: unknown, ids: Set<string>): v is Obj =>
  isObj(v) && typeof v.id === "string" && !!v.id && typeof v.fromNode === "string" && typeof v.toNode === "string" && ids.has(v.fromNode) && ids.has(v.toNode);

type Item<T> = { at: number; shown?: T; hidden?: Obj };

// The first of each id (shown or kept aside), no more than max in all.
function sort<T extends { id: string }>(items: Item<T>[], max: number) {
  const seen = new Set<string>();
  const shown: T[] = [];
  const hidden: Hidden[] = [];
  for (const it of items) {
    const id = it.shown ? it.shown.id : it.hidden ? (it.hidden.id as string) : "";
    if (!id || seen.has(id)) continue;
    if (seen.size >= max) break;
    seen.add(id);
    if (it.shown) shown.push(it.shown);
    else hidden.push({ at: it.at, item: it.hidden! });
  }
  return { shown, hidden };
}

export function parseCanvas(text: string): Canvas {
  const j: unknown = text.trim() ? JSON.parse(text) : {};
  const empty = (): Canvas => ({ nodes: [], edges: [], rest: {}, hidden: { nodes: [], edges: [] } });
  if (!isObj(j)) return empty();
  const { nodes, edges, ...rest } = j;
  const n = sort<CNode>((Array.isArray(nodes) ? nodes : []).map((v, at) => {
    const shown = toNode(v);
    return shown ? { at, shown } : unknownKind(v) ? { at, hidden: v } : { at };
  }), LIMITS.nodes);
  const ids = new Set(n.shown.map((x) => x.id));
  const all = new Set([...ids, ...n.hidden.map((h) => h.item.id as string)]);
  const e = sort<CEdge>((Array.isArray(edges) ? edges : []).map((v, at) => {
    const shown = toEdge(v, ids);
    return shown ? { at, shown } : hiddenEdge(v, all) ? { at, hidden: v } : { at };
  }), LIMITS.edges);
  return { nodes: n.shown, edges: e.shown, rest, hidden: { nodes: n.hidden, edges: e.hidden } };
}

// Puts what was kept aside back in its place in the list.
function merge(shown: unknown[], hidden: Hidden[]): unknown[] {
  const out = [...shown];
  for (const h of [...hidden].sort((a, b) => a.at - b.at)) out.splice(Math.min(h.at, out.length), 0, h.item);
  return out;
}

// As Obsidian writes it: tabs, and a line per card and per arrow.
export function writeCanvas(c: Canvas): string {
  const list = (xs: unknown[]) => (xs.length ? "\n\t\t" + xs.map((x) => JSON.stringify(x)).join(",\n\t\t") + "\n\t" : "");
  const round = (n: CNode) => ({ ...n, x: Math.round(n.x), y: Math.round(n.y), width: Math.round(n.width), height: Math.round(n.height) });
  // An arrow kept aside goes back only while both its cards are there.
  const ids = new Set([...c.nodes.map((n) => n.id), ...c.hidden.nodes.map((h) => h.item.id)]);
  const edges = c.hidden.edges.filter((h) => ids.has(h.item.fromNode) && ids.has(h.item.toNode));
  let out = "{\n\t\"nodes\":[" + list(merge(c.nodes.map(round), c.hidden.nodes)) + "],\n\t\"edges\":[" + list(merge(c.edges, edges)) + "]";
  for (const [k, v] of Object.entries(c.rest)) out += ",\n\t" + JSON.stringify(k) + ":" + JSON.stringify(v);
  return out + "\n}";
}

// Ids like Obsidian's: 16 hex digits.
export function newId() {
  const b = crypto.getRandomValues(new Uint8Array(8));
  return Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
}

// The six colors of Obsidian's canvas (1 to 6); any other is a #hex.
export const COLORS: Record<string, string> = { "1": "#e93147", "2": "#ec7500", "3": "#e0ac00", "4": "#08b94e", "5": "#00bfbc", "6": "#7852ee" };
export const colorOf = (c?: string) => (c ? COLORS[c] ?? (/^#[0-9a-f]{3,8}$/i.test(c) ? c : undefined) : undefined);

// The middle of a side of a card.
export function sidePoint(n: CNode, side: Side) {
  switch (side) {
    case "top": return { x: n.x + n.width / 2, y: n.y };
    case "bottom": return { x: n.x + n.width / 2, y: n.y + n.height };
    case "left": return { x: n.x, y: n.y + n.height / 2 };
    case "right": return { x: n.x + n.width, y: n.y + n.height / 2 };
  }
}
const NORMAL: Record<Side, { x: number; y: number }> = { top: { x: 0, y: -1 }, bottom: { x: 0, y: 1 }, left: { x: -1, y: 0 }, right: { x: 1, y: 0 } };

// The side of a card facing a point (an arrow without a side, or one being
// dropped).
export function sideToward(n: CNode, p: { x: number; y: number }): Side {
  const dx = (p.x - (n.x + n.width / 2)) / Math.max(n.width, 1);
  const dy = (p.y - (n.y + n.height / 2)) / Math.max(n.height, 1);
  if (Math.abs(dx) > Math.abs(dy)) return dx > 0 ? "right" : "left";
  return dy > 0 ? "bottom" : "top";
}

const center = (n: CNode) => ({ x: n.x + n.width / 2, y: n.y + n.height / 2 });

// An arrow's curve: from a side of one card to a side of another, leaving
// and arriving straight out of the sides. Its middle is where the label goes.
export function edgePath(from: CNode, to: CNode, e: Pick<CEdge, "fromSide" | "toSide">) {
  const fs = e.fromSide ?? sideToward(from, center(to));
  const ts = e.toSide ?? sideToward(to, center(from));
  return curve(sidePoint(from, fs), fs, sidePoint(to, ts), ts);
}

export function curve(a: { x: number; y: number }, as: Side, b: { x: number; y: number }, bs: Side | null) {
  const d = Math.hypot(b.x - a.x, b.y - a.y);
  const k = Math.min(Math.max(d * 0.4, 30), 250);
  const c1 = { x: a.x + NORMAL[as].x * k, y: a.y + NORMAL[as].y * k };
  const c2 = bs ? { x: b.x + NORMAL[bs].x * k, y: b.y + NORMAL[bs].y * k } : b;
  // The point at the middle of the curve (t = .5).
  const mid = { x: (a.x + 3 * c1.x + 3 * c2.x + b.x) / 8, y: (a.y + 3 * c1.y + 3 * c2.y + b.y) / 8 };
  // The arrow tips, drawn (a marker painted with context-stroke comes out
  // black in WebKit): along the curve's last and first stretch.
  const last = c2.x === b.x && c2.y === b.y ? c1 : c2;
  return { d: `M${a.x},${a.y} C${c1.x},${c1.y} ${c2.x},${c2.y} ${b.x},${b.y}`, mid, end: b, endSide: bs, head: tip(last, b), tail: tip(c1, a) };
}

// A filled triangle pointing at to, coming from from.
function tip(from: { x: number; y: number }, to: { x: number; y: number }) {
  const r = Math.atan2(to.y - from.y, to.x - from.x);
  const p = (s: number) => `${to.x - 15 * Math.cos(r + s)},${to.y - 15 * Math.sin(r + s)}`;
  return `M${to.x},${to.y} L${p(0.42)} L${p(-0.42)} Z`;
}

// The box around some cards.
export function bounds(nodes: CNode[]) {
  if (!nodes.length) return { x: 0, y: 0, width: 0, height: 0 };
  const x = Math.min(...nodes.map((n) => n.x)), y = Math.min(...nodes.map((n) => n.y));
  const r = Math.max(...nodes.map((n) => n.x + n.width)), b = Math.max(...nodes.map((n) => n.y + n.height));
  return { x, y, width: r - x, height: b - y };
}

// Whether card n lies inside group g (to move with it).
export const inside = (n: CNode, g: CNode) => n !== g && n.x >= g.x && n.y >= g.y && n.x + n.width <= g.x + g.width && n.y + n.height <= g.y + g.height;

// The Markdown of a card, as HTML, for reading: headings, lists, tasks,
// quotes, code, bold, italic, highlights and links (to notes, [[wiki]]
// too, or to the web). Everything is escaped first; links become buttons
// that carry their address in data-href, nothing runs on its own.
export function cardHTML(md: string): string {
  const out: string[] = [];
  let list: "ul" | "ol" | "" = "";
  let code = false;
  const close = () => { if (list) { out.push(`</${list}>`); list = ""; } };
  for (const raw of md.replace(/\r/g, "").split("\n")) {
    if (/^```/.test(raw)) { close(); out.push(code ? "</code></pre>" : "<pre><code>"); code = !code; continue; }
    if (code) { out.push(esc(raw) + "\n"); continue; }
    const line = raw.trimEnd();
    let m: RegExpExecArray | null;
    if ((m = /^(#{1,6})\s+(.*)$/.exec(line))) { close(); out.push(`<h${m[1].length}>${inline(m[2])}</h${m[1].length}>`); }
    else if ((m = /^\s*[-*+]\s+\[( |x|X)\]\s+(.*)$/.exec(line))) { open("ul"); out.push(`<li class="task${m[1] === " " ? "" : " done"}">${inline(m[2])}</li>`); }
    else if ((m = /^\s*[-*+]\s+(.*)$/.exec(line))) { open("ul"); out.push(`<li>${inline(m[1])}</li>`); }
    else if ((m = /^\s*\d+[.)]\s+(.*)$/.exec(line))) { open("ol"); out.push(`<li>${inline(m[1])}</li>`); }
    else if ((m = /^>\s?(.*)$/.exec(line))) { close(); out.push(`<blockquote>${inline(m[1])}</blockquote>`); }
    else if (/^(-{3,}|\*{3,})$/.test(line)) { close(); out.push("<hr>"); }
    else if (!line.trim()) { close(); }
    else { close(); out.push(`<p>${inline(line)}</p>`); }
  }
  if (code) out.push("</code></pre>");
  close();
  return out.join("");

  function open(kind: "ul" | "ol") {
    if (list === kind) return;
    close();
    out.push(`<${kind}>`);
    list = kind;
  }
}

const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");

function inline(s: string): string {
  // Code first, kept out of the rest.
  const codes: string[] = [];
  s = s.replace(/`([^`]+)`/g, (_, c) => `\u0000${codes.push(esc(c)) - 1}\u0000`);
  s = esc(s);
  s = s.replace(/!?\[\[([^\]|#]+)(#[^\]|]*)?(?:\|([^\]]*))?\]\]/g, (_, name, _h, alias) => `<button class="lnk" data-href="[[${name}]]">${alias || name}</button>`);
  s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, text, href) => `<button class="lnk" data-href="${href}">${text}</button>`);
  s = s.replace(/\*\*([^*]+)\*\*/g, "<b>$1</b>").replace(/__([^_]+)__/g, "<b>$1</b>");
  s = s.replace(/(^|[^*])\*([^*]+)\*/g, "$1<i>$2</i>").replace(/(^|\W)_([^_]+)_(?=\W|$)/g, "$1<i>$2</i>");
  s = s.replace(/~~([^~]+)~~/g, "<s>$1</s>").replace(/==([^=]+)==/g, "<mark>$1</mark>");
  return s.replace(/\u0000(\d+)\u0000/g, (_, i) => `<code>${codes[+i]}</code>`);
}
