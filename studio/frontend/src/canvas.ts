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
export type Canvas = { nodes: CNode[]; edges: CEdge[]; rest: Record<string, unknown> };

export function parseCanvas(text: string): Canvas {
  const j = text.trim() ? JSON.parse(text) : {};
  const { nodes, edges, ...rest } = j ?? {};
  return {
    nodes: (Array.isArray(nodes) ? nodes : []).filter((n: any) => n && typeof n.id === "string"),
    edges: (Array.isArray(edges) ? edges : []).filter((e: any) => e && typeof e.id === "string"),
    rest,
  };
}

// As Obsidian writes it: tabs, and a line per card and per arrow.
export function writeCanvas(c: Canvas): string {
  const list = (xs: unknown[]) => (xs.length ? "\n\t\t" + xs.map((x) => JSON.stringify(x)).join(",\n\t\t") + "\n\t" : "");
  const round = (n: CNode) => ({ ...n, x: Math.round(n.x), y: Math.round(n.y), width: Math.round(n.width), height: Math.round(n.height) });
  let out = "{\n\t\"nodes\":[" + list(c.nodes.map(round)) + "],\n\t\"edges\":[" + list(c.edges) + "]";
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
  return { d: `M${a.x},${a.y} C${c1.x},${c1.y} ${c2.x},${c2.y} ${b.x},${b.y}`, mid, end: b, endSide: bs };
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
