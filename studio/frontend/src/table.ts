// Markdown tables drawn as tables in the editor, like Obsidian's live
// preview: the cells are typed in place, Tab and Enter move between them,
// the "+" on the right and bottom edges add a column or a row, and a
// right-click on a cell inserts or deletes rows and columns. The note keeps
// the plain Markdown table.
import { EditorView, WidgetType, Decoration, type DecorationSet } from "@codemirror/view";
import { StateField, StateEffect, RangeSetBuilder, type EditorState } from "@codemirror/state";
import { syntaxTree } from "@codemirror/language";

export type TableText = {
  addRowAbove: string; addRowBelow: string; addColLeft: string; addColRight: string;
  delRow: string; delCol: string; delTable: string; asText: string; addRow: string; addCol: string; header: string; delLastRow: string; delLastCol: string;
};

type Table = { aligns: string[]; rows: string[][] }; // rows[0] is the header

// Cells split on | that are not escaped (\|).
function cells(line: string): string[] {
  let s = line.trim();
  if (s.startsWith("|")) s = s.slice(1);
  if (s.endsWith("|") && !s.endsWith("\\|")) s = s.slice(0, -1);
  return s.split(/(?<!\\)\|/).map((c) => c.trim().replace(/\\\|/g, "|"));
}

function parse(text: string): Table {
  const lines = text.split("\n").filter((l) => l.trim());
  const rows = lines.filter((_, i) => i !== 1).map(cells);
  const aligns = cells(lines[1] ?? "").map((a) => (a.startsWith(":") && a.endsWith(":") ? ":---:" : a.endsWith(":") ? "---:" : a.startsWith(":") ? ":---" : "---"));
  const n = Math.max(...rows.map((r) => r.length), aligns.length, 1);
  for (const r of rows) while (r.length < n) r.push("");
  while (aligns.length < n) aligns.push("---");
  return { aligns, rows };
}

function format(t: Table): string {
  const esc = (c: string) => c.replace(/\|/g, "\\|").replace(/\n/g, " ");
  const line = (r: string[]) => "| " + r.map((c) => esc(c) || " ").join(" | ") + " |";
  return [line(t.rows[0]), "| " + t.aligns.join(" | ") + " |", ...t.rows.slice(1).map(line)].join("\n").replace(/\|  \|/g, "|   |");
}

// "Show as text": the table at this position is left as Markdown until
// the cursor leaves it.
const asText = StateEffect.define<number | null>();

class TableWidget extends WidgetType {
  // from is where the table starts in the note; the widget on screen always
  // holds the latest one (cmWidget), as text before it changes.
  constructor(readonly text: string, readonly from: number, readonly labels: TableText) { super(); }
  eq(other: TableWidget) { return other.text === this.text && other.from === this.from; }
  ignoreEvent() { return true; }

  toDOM(view: EditorView) {
    const wrap = document.createElement("div");
    wrap.className = "cm-md-table";
    (wrap as any).cmWidget = this;
    this.draw(wrap, view);
    return wrap;
  }

  // While a cell is being typed in, the table on screen stays (redrawing it
  // would lose the cursor); it only takes the new text.
  // A change from elsewhere (undo, for instance) redraws it, keeping the
  // cell that had the cursor.
  updateDOM(dom: HTMLElement, view: EditorView) {
    (dom as any).cmWidget = this;
    const active = document.activeElement as HTMLElement | null;
    const inside = !!active && dom.contains(active);
    if (inside && (dom as any).cmText === this.text) return true;
    this.draw(dom, view);
    if (inside) (dom.querySelector(`[data-r="${active!.dataset.r}"][data-c="${active!.dataset.c}"]`) as HTMLElement | null)?.focus();
    return true;
  }

  draw(wrap: HTMLElement, view: EditorView) {
    (wrap as any).cmText = this.text;
    const t = parse(this.text);
    const L = this.labels;
    wrap.replaceChildren();
    const table = document.createElement("table");
    const write = (next: Table, focus?: [number, number]) => {
      const from = (wrap as any).cmWidget.from;
      const old = (wrap as any).cmText as string;
      const text = format(next);
      (wrap as any).cmText = text;
      (wrap as any).cmDirty = false;
      view.dispatch({ changes: { from, to: from + old.length, insert: text } });
      if (focus) {
        // Drawn from the table just written (the latest widget).
        (wrap as any).cmWidget.draw(wrap, view);
        (wrap.querySelector(`[data-r="${focus[0]}"][data-c="${focus[1]}"]`) as HTMLElement | null)?.focus();
      }
    };
    const read = (): Table => ({
      aligns: t.aligns,
      rows: t.rows.map((r, ri) => r.map((_, ci) => (wrap.querySelector(`[data-r="${ri}"][data-c="${ci}"]`) as HTMLElement)?.innerText.replace(/\n/g, " ") ?? "")),
    });
    t.rows.forEach((row, ri) => {
      const tr = document.createElement("tr");
      row.forEach((cell, ci) => {
        const td = document.createElement(ri === 0 ? "th" : "td");
        td.contentEditable = "plaintext-only";
        td.spellcheck = false;
        td.dataset.r = String(ri);
        td.dataset.c = String(ci);
        td.textContent = cell;
        const a = t.aligns[ci];
        td.style.textAlign = a === ":---:" ? "center" : a === "---:" ? "right" : "left";
        if (ri === 0 && !cell) td.dataset.placeholder = L.header;
        // Typing stays in the cell; the note gets it half a second after the
        // typing stops, or when the cursor leaves the table.
        td.addEventListener("input", () => {
          (wrap as any).cmDirty = true;
          clearTimeout((wrap as any).cmTimer);
          (wrap as any).cmTimer = setTimeout(() => (wrap as any).cmDirty && (wrap as any).cmSave?.(), 500);
        });
        td.addEventListener("keydown", (e) => {
          const rows = t.rows.length, cols = row.length;
          const go = (r: number, c: number) => {
            const n = read();
            if (r >= rows) n.rows.push(Array(cols).fill(""));
            if (r >= rows || (wrap as any).cmDirty) write(n, [r, c]);
            else (wrap.querySelector(`[data-r="${r}"][data-c="${c}"]`) as HTMLElement)?.focus();
          };
          if (e.key === "Tab") {
            e.preventDefault();
            let r = ri, c = ci + (e.shiftKey ? -1 : 1);
            if (c >= cols) { c = 0; r++; }
            if (c < 0) { c = cols - 1; r--; }
            if (r >= 0) go(r, c);
          } else if (e.key === "Enter") {
            e.preventDefault();
            go(ri + 1, ci);
          } else if (e.key === "Escape") {
            (e.target as HTMLElement).blur();
            view.focus();
          }
        });
        td.addEventListener("contextmenu", (e) => {
          e.preventDefault();
          e.stopPropagation();
          menu(e, [
            [L.addRowAbove, ri > 0 ? () => { const n = read(); n.rows.splice(ri, 0, Array(row.length).fill("")); write(n, [ri, ci]); } : null],
            [L.addRowBelow, () => { const n = read(); n.rows.splice(ri + 1, 0, Array(row.length).fill("")); write(n, [ri + 1, ci]); }],
            [L.addColLeft, () => { const n = read(); n.rows.forEach((r) => r.splice(ci, 0, "")); n.aligns.splice(ci, 0, "---"); write(n, [ri, ci]); }],
            [L.addColRight, () => { const n = read(); n.rows.forEach((r) => r.splice(ci + 1, 0, "")); n.aligns.splice(ci + 1, 0, "---"); write(n, [ri, ci + 1]); }],
            null,
            [L.delRow, ri > 0 ? () => { const n = read(); n.rows.splice(ri, 1); write(n, [Math.min(ri, n.rows.length - 1), ci]); } : null],
            [L.delCol, row.length > 1 ? () => { const n = read(); n.rows.forEach((r) => r.splice(ci, 1)); n.aligns.splice(ci, 1); write(n, [ri, Math.max(0, ci - 1)]); } : null],
            null,
            [L.asText, () => { const from = (wrap as any).cmWidget.from; view.dispatch({ effects: asText.of(from), selection: { anchor: from } }); view.focus(); }],
            [L.delTable, () => { const from = (wrap as any).cmWidget.from; view.dispatch({ changes: { from, to: from + ((wrap as any).cmText as string).length, insert: "" } }); view.focus(); }, true],
          ]);
        });
        tr.append(td);
      });
      table.append(tr);
    });
    // Bars on the edges, shown on hover: + on the right and at the bottom
    // add a column and a row; − on the left and at the top take the last
    // column and the last row away.
    const cols = t.aligns.length, rows = t.rows.length;
    const redraw = () => (wrap as any).cmWidget.draw(wrap, view);
    const addCol = bar("right", "+", L.addCol, () => { const n = read(); n.rows.forEach((r) => r.push("")); n.aligns.push("---"); write(n, [0, n.aligns.length - 1]); });
    const addRow = bar("bottom", "+", L.addRow, () => { const n = read(); n.rows.push(Array(n.aligns.length).fill("")); write(n, [n.rows.length - 1, 0]); });
    const delCol = bar("left", "−", L.delLastCol, cols > 1 ? () => { const n = read(); n.rows.forEach((r) => r.pop()); n.aligns.pop(); write(n); redraw(); } : null);
    const delRow = bar("top", "−", L.delLastRow, rows > 2 ? () => { const n = read(); n.rows.pop(); write(n); redraw(); } : null);
    const inner = document.createElement("div");
    inner.className = "cm-md-table-inner";
    inner.append(table);
    wrap.append(delRow, delCol, inner, addCol, addRow);
  }
}

function bar(side: string, sign: string, title: string, run: (() => void) | null) {
  const b = document.createElement("button");
  b.className = "cm-md-bar " + side;
  b.title = title;
  // Drawn, not typed: a + or − glyph never sits in the exact middle.
  const path = sign === "+" ? "M12 5v14M5 12h14" : "M5 12h14";
  b.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="${path}"/></svg>`;
  b.addEventListener("mousedown", (e) => e.preventDefault());
  if (run) b.addEventListener("click", run);
  else b.disabled = true;
  return b;
}

// The small menu of a cell (danger items in red).
function menu(e: MouseEvent, items: ([string, (() => void) | null, boolean?] | null)[]) {
  document.querySelector(".cm-md-menu")?.remove();
  const m = document.createElement("div");
  m.className = "cm-md-menu";
  for (const it of items) {
    if (!it) { m.append(document.createElement("hr")); continue; }
    const [label, run, danger] = it;
    const b = document.createElement("button");
    b.textContent = label;
    if (danger) b.className = "danger";
    if (!run) b.disabled = true;
    else b.addEventListener("click", () => { m.remove(); run(); });
    m.append(b);
  }
  document.body.append(m);
  const r = m.getBoundingClientRect();
  m.style.left = Math.min(e.clientX, innerWidth - r.width - 8) + "px";
  m.style.top = Math.min(e.clientY, innerHeight - r.height - 8) + "px";
  const close = (ev: Event) => { if (!m.contains(ev.target as Node)) { m.remove(); removeEventListener("mousedown", close, true); } };
  setTimeout(() => addEventListener("mousedown", close, true));
}

export function tables(labels: TableText) {
  const build = (state: EditorState, raw: number | null): DecorationSet => {
    const b = new RangeSetBuilder<Decoration>();
    const doc = state.doc;
    let last = -1; // where the previous table ended: tables never overlap
    syntaxTree(state).iterate({
      enter: (n) => {
        if (n.name !== "Table") return;
        if (n.from <= last || (raw !== null && raw >= n.from && raw <= n.to)) return false;
        // Whole lines, and every row line after it (a tree still parsing
        // may stop short): a block widget covers its lines.
        const first = doc.lineAt(n.from);
        let end = doc.lineAt(n.to);
        while (end.number < doc.lines && /^\s*\|/.test(doc.line(end.number + 1).text)) end = doc.line(end.number + 1);
        // A table has a header and a |---| line; anything less stays text.
        if (end.number === first.number || !/^\s*\|?\s*:?-+/.test(doc.line(first.number + 1).text)) return false;
        b.add(first.from, end.to, Decoration.replace({ widget: new TableWidget(state.sliceDoc(first.from, end.to), first.from, labels), block: true }));
        last = end.to;
        return false;
      },
    });
    return b.finish();
  };
  return StateField.define<{ raw: number | null; deco: DecorationSet }>({
    create: (state) => ({ raw: null, deco: build(state, null) }),
    update(v, tr) {
      let raw = v.raw;
      for (const e of tr.effects) if (e.is(asText)) raw = e.value;
      if (raw !== null) {
        raw = tr.changes.mapPos(raw);
        // Leaving the table shown as text draws it again.
        const head = tr.state.selection.main.head;
        let inside = false;
        syntaxTree(tr.state).iterate({ from: raw, to: raw, enter: (n) => { if (n.name === "Table" && head >= n.from && head <= n.to) inside = true; } });
        if (!inside && tr.selection) raw = null;
      }
      // The tables come from the syntax tree, which may still be parsing:
      // a newer tree redraws them too.
      const reparsed = syntaxTree(tr.state) !== syntaxTree(tr.startState);
      if (!tr.docChanged && !reparsed && raw === v.raw && !tr.effects.length) return v;
      return { raw, deco: build(tr.state, raw) };
    },
    provide: (f) => EditorView.decorations.from(f, (v) => v.deco),
  });
}
