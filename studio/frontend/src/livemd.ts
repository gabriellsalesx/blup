// The editor reading more like the page: links show as their text, and
// headings fold.
//
// Links ([[Note]], [[Note|text]], [text](path.md), [text](https://…)) show
// only their text, underlined; their brackets and address come back while
// the cursor is on the link (typing, the arrows, or a right-click), to
// edit them. Images and code stay as written.
//
// Headings fold like Obsidian's: a chevron at their left (on hover, or
// always when folded) hides everything down to the next heading of the
// same level or higher.
import { EditorView, Decoration, ViewPlugin, WidgetType, type DecorationSet, type ViewUpdate } from "@codemirror/view";
import { RangeSetBuilder, type EditorState } from "@codemirror/state";
import { foldable, foldedRanges, foldEffect, unfoldEffect, syntaxTree } from "@codemirror/language";

const hide = Decoration.replace({});
const linkText = Decoration.mark({ class: "cm-link-text" });

const WIKI = /(?<!!)\[\[([^\]|\n]+?)(\|([^\]\n]*))?\]\]/g;
const MD = /(?<![!\\])\[((?:[^\]\\\n]|\\.)+)\]\(([^)\s]+)(\s+"[^"]*")?\)/g;

// Inside code (inline or a block): left as it is.
function inCode(state: EditorState, pos: number) {
  for (let n: any = syntaxTree(state).resolveInner(pos, 1); n; n = n.parent) {
    if (/^(InlineCode|FencedCode|CodeBlock|CodeText)$/.test(n.name)) return true;
  }
  return false;
}

function links(view: EditorView): DecorationSet {
  const { state } = view;
  const sel = state.selection.ranges;
  const touched = (from: number, to: number) => sel.some((r) => r.from <= to && r.to >= from);
  const found: { from: number; to: number; parts: [number, number, Decoration][] }[] = [];
  for (const { from, to } of view.visibleRanges) {
    const text = state.sliceDoc(from, to);
    for (const m of text.matchAll(WIKI)) {
      const a = from + m.index!, b = a + m[0].length;
      if (touched(a, b) || inCode(state, a)) continue;
      // [[target|text]]: the text shows; without it, the target's name.
      const shown = m[3] !== undefined ? m[3] : m[1];
      const textFrom = m[3] !== undefined ? a + 2 + m[1].length + 1 : a + 2;
      found.push({ from: a, to: b, parts: [[a, textFrom, hide], [textFrom, textFrom + shown.length, linkText], [textFrom + shown.length, b, hide]] });
    }
    for (const m of text.matchAll(MD)) {
      const a = from + m.index!, b = a + m[0].length;
      if (touched(a, b) || inCode(state, a)) continue;
      const textTo = a + 1 + m[1].length;
      found.push({ from: a, to: b, parts: [[a, a + 1, hide], [a + 1, textTo, linkText], [textTo, b, hide]] });
    }
  }
  found.sort((x, y) => x.from - y.from);
  const b = new RangeSetBuilder<Decoration>();
  let last = -1;
  for (const f of found) {
    if (f.from < last) continue; // overlapping matches: the first wins
    for (const [x, y, d] of f.parts) if (y > x) b.add(x, y, d);
    last = f.to;
  }
  return b.finish();
}

export const linkConceal = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    constructor(view: EditorView) { this.decorations = links(view); }
    update(u: ViewUpdate) {
      if (u.docChanged || u.selectionSet || u.viewportChanged) this.decorations = links(u.view);
    }
  },
  { decorations: (v) => v.decorations },
);

// The chevron of a heading.
class Chevron extends WidgetType {
  constructor(readonly folded: boolean, readonly title: string) { super(); }
  eq(o: Chevron) { return o.folded === this.folded; }
  toDOM(view: EditorView) {
    const b = document.createElement("span");
    b.className = "cm-fold-chevron" + (this.folded ? " folded" : "");
    b.setAttribute("role", "button");
    b.setAttribute("aria-label", this.title);
    b.title = this.title;
    b.innerHTML = `<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg>`;
    b.addEventListener("mousedown", (e) => {
      e.preventDefault();
      e.stopPropagation();
      toggleAt(view, view.posAtDOM(b));
    });
    return b;
  }
  ignoreEvent() { return true; }
}

function toggleAt(view: EditorView, pos: number) {
  const line = view.state.doc.lineAt(pos);
  let open: { from: number; to: number } | null = null;
  foldedRanges(view.state).between(line.to, line.to + 1, (from, to) => { if (from === line.to) open = { from, to }; });
  if (open) { view.dispatch({ effects: unfoldEffect.of(open) }); return; }
  const range = foldable(view.state, line.from, line.to);
  if (range) view.dispatch({ effects: foldEffect.of(range) });
}

const headingLine = Decoration.line({ class: "cm-heading-line" });

function chevrons(view: EditorView, title: { fold: string; unfold: string }): DecorationSet {
  const { state } = view;
  const b = new RangeSetBuilder<Decoration>();
  const folded = new Set<number>();
  foldedRanges(state).between(0, state.doc.length, (from) => { folded.add(from); });
  for (const { from, to } of view.visibleRanges) {
    for (let pos = from; pos <= to; ) {
      const line = state.doc.lineAt(pos);
      if (/^#{1,6} /.test(line.text) && !inCode(state, line.from) && foldable(state, line.from, line.to)) {
        const isFolded = folded.has(line.to);
        b.add(line.from, line.from, headingLine);
        b.add(line.from, line.from, Decoration.widget({ widget: new Chevron(isFolded, isFolded ? title.unfold : title.fold), side: -1 }));
      }
      pos = line.to + 1;
    }
  }
  return b.finish();
}

export function headingFolds(title: { fold: string; unfold: string }) {
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;
      constructor(view: EditorView) { this.decorations = chevrons(view, title); }
      update(u: ViewUpdate) {
        const folds = u.transactions.some((tr) => tr.effects.some((e) => e.is(foldEffect) || e.is(unfoldEffect)));
        if (u.docChanged || u.viewportChanged || folds) this.decorations = chevrons(u.view, title);
      }
    },
    { decorations: (v) => v.decorations },
  );
}
