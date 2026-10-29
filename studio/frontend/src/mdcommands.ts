// The formatting bar's commands, on the selection (or the word or line at
// the cursor): marks around inline text, prefixes at the start of lines.
import type { EditorView } from "@codemirror/view";
import { EditorSelection } from "@codemirror/state";

// Puts mark around the selection, or takes it off when it is already there.
export function wrap(view: EditorView, mark: string, placeholder = "") {
  const { state } = view;
  view.dispatch(state.changeByRange((r) => {
    let { from, to } = r;
    if (from === to) {
      const w = state.wordAt(from);
      if (w) ({ from, to } = w);
    }
    const before = state.sliceDoc(from - mark.length, from), after = state.sliceDoc(to, to + mark.length);
    if (before === mark && after === mark) {
      return { changes: [{ from: from - mark.length, to: from }, { from: to, to: to + mark.length }], range: EditorSelection.range(from - mark.length, to - mark.length) };
    }
    const text = state.sliceDoc(from, to) || placeholder;
    return { changes: { from, to, insert: mark + text + mark }, range: EditorSelection.range(from + mark.length, from + mark.length + text.length) };
  }));
  view.focus();
}

// Gives every selected line the prefix (or takes it off when they all have
// it). other lists the prefixes it replaces (a heading level for another).
export function prefix(view: EditorView, pre: string | ((i: number) => string), other: RegExp) {
  const { state } = view;
  const lines = new Set<number>();
  for (const r of state.selection.ranges) {
    for (let n = state.doc.lineAt(r.from).number; n <= state.doc.lineAt(r.to).number; n++) lines.add(n);
  }
  const list = [...lines].map((n) => state.doc.line(n));
  const at = (i: number) => (typeof pre === "string" ? pre : pre(i));
  const all = list.every((l, i) => l.text.startsWith(at(i)));
  const changes = list.map((l, i) => {
    const old = l.text.match(other)?.[0] ?? "";
    return { from: l.from, to: l.from + old.length, insert: all ? "" : at(i) };
  });
  // The cursor moves along with the new prefix (it would stay before it).
  const set = state.changes(changes);
  view.dispatch({ changes: set, selection: state.selection.map(set, 1) });
  view.focus();
}

// Inserts a block (a table, a line, a callout) on lines of its own.
export function block(view: EditorView, text: string, select?: [number, number]) {
  const { state } = view;
  const r = state.selection.main;
  const line = state.doc.lineAt(r.from);
  const atStart = line.text.trim() === "";
  const before = atStart ? (line.number > 1 && state.doc.line(line.number - 1).text.trim() ? "\n" : "") : "\n\n";
  const insert = before + text + "\n";
  const from = atStart ? line.from : line.to;
  const start = from + before.length;
  view.dispatch({
    changes: { from, to: atStart ? line.to : from, insert },
    selection: select ? EditorSelection.range(start + select[0], start + select[1]) : EditorSelection.cursor(start + text.length),
  });
  view.focus();
}

// A link around the selection: [text](url), with the address selected.
export function link(view: EditorView, label: string) {
  const { state } = view;
  const r = state.selection.main;
  const text = state.sliceDoc(r.from, r.to) || label;
  const insert = `[${text}](https://)`;
  view.dispatch({ changes: { from: r.from, to: r.to, insert }, selection: EditorSelection.range(r.from + text.length + 3, r.from + insert.length - 1) });
  view.focus();
}

// Text at the cursor; ownLine puts it on a line of its own (an image).
export function insert(view: EditorView, text: string, ownLine = false) {
  const r = view.state.selection.main;
  if (ownLine && view.state.doc.lineAt(r.from).from !== r.from) text = "\n" + text;
  view.dispatch({ changes: { from: r.from, to: r.to, insert: text }, selection: EditorSelection.cursor(r.from + text.length) });
  view.focus();
}
