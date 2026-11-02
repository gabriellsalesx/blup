// The formatting bar's commands, on the selection (or the word or line at
// the cursor): marks around inline text, prefixes at the start of lines.
import type { EditorView } from "@codemirror/view";
import { EditorSelection, type EditorState } from "@codemirror/state";

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

// The heading level of the lines (0 = plain text).
export function heading(view: EditorView, level: number) {
  const { state } = view;
  const lines = new Set<number>();
  for (const r of state.selection.ranges) {
    for (let n = state.doc.lineAt(r.from).number; n <= state.doc.lineAt(r.to).number; n++) lines.add(n);
  }
  const changes = [...lines].map((n) => {
    const l = state.doc.line(n);
    const old = l.text.match(/^#{1,6} /)?.[0] ?? "";
    return { from: l.from, to: l.from + old.length, insert: level ? "#".repeat(level) + " " : "" };
  });
  const set = state.changes(changes);
  view.dispatch({ changes: set, selection: state.selection.map(set, 1) });
  view.focus();
}

// Alignment of the paragraph (or heading) at the cursor: a {.text-center}
// line under the paragraph, or at the end of a heading; null = the normal
// (left) alignment.
export const ALIGN_LINE = /^\{\.text-(center|right|justify)\}\s*$/;
const ALIGN_END = /\s*\{\.text-(center|right|justify)\}\s*$/;
export function align(view: EditorView, cls: string | null) {
  const { state } = view;
  const doc = state.doc;
  const HEAD = /^#{1,6} /;
  const breaks = (text: string) => !text.trim() || ALIGN_LINE.test(text) || HEAD.test(text);
  const changes: { from: number; to?: number; insert?: string }[] = [];
  const done = new Set<number>();
  // Every paragraph and heading the selection touches (all of them when
  // everything is selected).
  for (const r of state.selection.ranges) {
    for (let n = doc.lineAt(r.from).number; n <= doc.lineAt(r.to).number; n++) {
      let line = doc.line(n);
      if (ALIGN_LINE.test(line.text)) {
        if (n === 1) continue;
        line = doc.line(n - 1);
      }
      if (!line.text.trim() || done.has(line.number)) continue;
      if (HEAD.test(line.text)) {
        done.add(line.number);
        const text = line.text.replace(ALIGN_END, "");
        changes.push({ from: line.from, to: line.to, insert: text + (cls ? ` {.${cls}}` : "") });
        continue;
      }
      let start = line, end = line;
      while (start.number > 1 && !breaks(doc.line(start.number - 1).text)) start = doc.line(start.number - 1);
      while (end.number < doc.lines && !breaks(doc.line(end.number + 1).text)) end = doc.line(end.number + 1);
      for (let k = start.number; k <= end.number; k++) done.add(k);
      const next = end.number < doc.lines ? doc.line(end.number + 1) : null;
      if (next && ALIGN_LINE.test(next.text)) {
        done.add(next.number);
        changes.push(cls ? { from: next.from, to: next.to, insert: `{.${cls}}` } : { from: end.to, to: next.to });
      } else if (cls) {
        changes.push({ from: end.to, insert: `\n{.${cls}}` });
      }
    }
  }
  if (changes.length) view.dispatch({ changes });
  view.focus();
}

// The alignment and heading level where the cursor is, for the toolbar.
type Align = "left" | "center" | "right" | "justify";
const isAlign = (v: string | undefined): v is Align => v === "left" || v === "center" || v === "right" || v === "justify";
export function alignAt(state: EditorState, pos: number): Align {
  const doc = state.doc;
  let line = doc.lineAt(pos);
  if (ALIGN_LINE.test(line.text) && line.number > 1) line = doc.line(line.number - 1);
  if (/^#{1,6} /.test(line.text)) { const a = line.text.match(/\{\.text-(center|right|justify)\}\s*$/)?.[1]; return isAlign(a) ? a : "left"; }
  if (!line.text.trim()) return "left";
  let end = line;
  while (end.number < doc.lines) {
    const next = doc.line(end.number + 1);
    if (ALIGN_LINE.test(next.text)) { const a = next.text.match(ALIGN_LINE)?.[1]; return isAlign(a) ? a : "left"; }
    if (!next.text.trim() || /^#{1,6} /.test(next.text)) break;
    end = next;
  }
  return "left";
}
export function headingAt(state: EditorState, pos: number) {
  return state.doc.lineAt(pos).text.match(/^(#{1,6}) /)?.[1].length ?? 0;
}

// A callout of type: around the selected lines, or a new one with its
// title selected. Inside a callout, it changes that callout's type. fold
// is "" (always open), "-" (starts folded) or "+" (open, can fold), for
// a new callout.
export function callout(view: EditorView, type: string, fold: string, title: string, text: string) {
  const { state } = view;
  const doc = state.doc;
  const r = state.selection.main;
  for (let n = doc.lineAt(r.head).number; n >= 1; n--) {
    const l = doc.line(n);
    if (!l.text.startsWith(">")) break;
    const m = l.text.match(/^(>\s*\[!)(\w+)\]([+-]?)/);
    if (m) {
      const from = l.from + m[1].length;
      view.dispatch({ changes: { from, to: from + m[2].length, insert: type } }); // folding stays as it was
      view.focus();
      return;
    }
  }
  const head = `> [!${type}]${fold} `;
  if (!r.empty) {
    const first = doc.lineAt(r.from), last = doc.lineAt(r.to);
    const body = doc.sliceString(first.from, last.to).split("\n").map((l) => (l.trim() ? "> " + l : ">")).join("\n");
    const insert = head + title + "\n" + body;
    view.dispatch({ changes: { from: first.from, to: last.to, insert }, selection: EditorSelection.range(first.from + head.length, first.from + head.length + title.length) });
    view.focus();
    return;
  }
  block(view, `${head}${title}\n> ${text}`, [head.length, head.length + title.length]);
}
