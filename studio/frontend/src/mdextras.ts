// What the editor shows of Markdown the site understands but the editor's
// parser doesn't: ==highlight== in yellow, and paragraphs aligned by a
// {.text-center} line (the line itself hidden).
import { EditorView, Decoration, type DecorationSet } from "@codemirror/view";
import { StateField, type EditorState, type Range } from "@codemirror/state";
import { ALIGN_LINE } from "./mdcommands";

const highlight = Decoration.mark({ class: "cm-highlight" });
const attr = Decoration.mark({ class: "cm-align-attr" });
const hide = Decoration.replace({});
const aligned = (cls: string) => Decoration.line({ class: "cm-align-" + cls });

function build(state: EditorState): DecorationSet {
  const out: Range<Decoration>[] = [];
  const doc = state.doc;
  const text = doc.toString();
  if (text.includes("==")) {
    for (const m of text.matchAll(/==(?=\S)[^=\n]+?==/g)) out.push(highlight.range(m.index!, m.index! + m[0].length));
  }
  if (text.includes("{.text-")) {
    for (let n = 1; n <= doc.lines; n++) {
      const line = doc.line(n);
      const own = line.text.match(ALIGN_LINE);
      const end = !own && /^#{1,6} /.test(line.text) ? line.text.match(/\{\.text-(center|right|justify)\}\s*$/) : null;
      // The {.text-center} itself is hidden: the paragraph shows aligned.
      if (own) {
        const prev = n > 1 ? doc.line(n - 1) : null;
        if (prev && prev.text.trim()) out.push(hide.range(prev.to, line.to));
        else out.push(attr.range(line.from, line.to));
        for (let k = n - 1; k >= 1 && doc.line(k).text.trim(); k--) out.push(aligned(own[1]).range(doc.line(k).from));
      } else if (end) {
        out.push(aligned(end[1]).range(line.from));
        out.push(hide.range(line.from + end.index!, line.to));
      }
    }
  }
  return Decoration.set(out, true);
}

export const mdExtras = StateField.define<DecorationSet>({
  create: build,
  update: (d, tr) => (tr.docChanged ? build(tr.state) : d),
  provide: (f) => EditorView.decorations.from(f),
});
