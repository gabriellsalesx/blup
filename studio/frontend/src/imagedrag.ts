// Moving an image in the note by dragging it. What moves is its Markdown,
// never the picture itself: the editor shows the picture from a data: URL,
// and the browser would otherwise drop that whole address in the text.
//
// It lands between two blocks of the note (before or after a paragraph, a
// list, a table, a callout, a code block), never in the middle of one.
import { syntaxTree } from "@codemirror/language";
import type { ChangeSpec, EditorState } from "@codemirror/state";

export type ImageSpan = { from: number; to: number };

const ALIGNED = /^\{\.text-\w+\}$/; // the line of a paragraph's alignment

// The end of the front matter at the top (--- … ---): left as written.
export function frontEnd(state: EditorState) {
  const doc = state.doc;
  if (doc.lines < 2 || doc.line(1).text !== "---") return 0;
  for (let n = 2; n <= doc.lines; n++) if (/^(---|\.\.\.)$/.test(doc.line(n).text)) return doc.line(n).to;
  return 0;
}

// The block of the note at pos (null on a blank line).
export function blockAt(state: EditorState, pos: number) {
  let n = syntaxTree(state).resolveInner(pos, 1);
  if (n.name === "Document") n = syntaxTree(state).resolveInner(pos, -1);
  while (n.parent && n.parent.name !== "Document") n = n.parent;
  return n.parent && pos >= n.from && pos <= n.to ? { name: n.name, from: n.from, to: n.to } : null;
}

// Where an image dropped at pos goes: the edge of the block there.
export function dropPoint(state: EditorState, pos: number, below: boolean) {
  const front = frontEnd(state);
  if (pos <= front) return front;
  const b = blockAt(state, pos);
  if (!b) return state.doc.lineAt(pos).from;
  return below ? b.to : Math.max(b.from, front);
}

// The changes that take the image at span to the point at (from dropPoint);
// null when it would stay where it is.
export function moveImage(state: EditorState, span: ImageSpan, at: number): ChangeSpec[] | null {
  const doc = state.doc;
  let { from, to } = span;
  let md = state.sliceDoc(from, to);
  const line = doc.lineAt(from);
  const block = blockAt(state, from);
  const rest = block ? state.sliceDoc(block.from, from) + state.sliceDoc(to, block.to) : "";
  if (block?.name === "Paragraph" && rest.split("\n").every((l) => !l.trim() || ALIGNED.test(l.trim()))) {
    // The image is the whole paragraph: its alignment goes with it, and so
    // does the blank line after it.
    md = state.sliceDoc(block.from, block.to);
    from = block.from;
    to = block.to;
    for (let i = 0; i < 2 && state.sliceDoc(to, to + 1) === "\n"; i++) to++;
  } else if (line.text.trim() === md) {
    // Alone on its line, in a paragraph with other lines.
    from = line.from;
    to = Math.min(line.to + 1, doc.length);
  }
  if (at >= from && at <= to) return null;
  const left = state.sliceDoc(Math.max(0, at - 2), at), right = state.sliceDoc(at, at + 2);
  const before = at === 0 || left.endsWith("\n\n") ? "" : left.endsWith("\n") ? "\n" : "\n\n";
  const after = !state.sliceDoc(at).trim() || right.startsWith("\n\n") ? "" : right.startsWith("\n") ? "\n" : "\n\n";
  return [{ from, to }, { from: at, insert: before + md + after }];
}

// The change that takes the image at span out of the note, with its line
// when it is alone there. The file stays in the attachments.
export function removeImage(state: EditorState, span: ImageSpan): ChangeSpec {
  const doc = state.doc, line = doc.lineAt(span.from);
  if (line.text.trim() !== state.sliceDoc(span.from, span.to)) return span;
  let to = Math.min(line.to + 1, doc.length);
  const blankBefore = line.number === 1 || !doc.line(line.number - 1).text.trim();
  if (blankBefore && !doc.lineAt(to).text.trim()) to = Math.min(to + 1, doc.length);
  return { from: line.from, to };
}

// The note scrolls by itself while an image is held near its top or bottom.
export function scrollNear(el: HTMLElement, y: number) {
  for (let p = el.parentElement; p; p = p.parentElement) {
    if (!/auto|scroll/.test(getComputedStyle(p).overflowY) || p.scrollHeight <= p.clientHeight) continue;
    const r = p.getBoundingClientRect();
    if (y < r.top + 64) p.scrollBy(0, -16);
    else if (y > r.bottom - 64) p.scrollBy(0, 16);
    return;
  }
}

// An image in the form of its address (data:image/…;base64,…), as a file.
// null for anything else.
export function dataImage(text: string | undefined) {
  const m = text?.trim().match(/^data:(image\/[\w.+-]+);base64,([A-Za-z0-9+/=\s]+)$/);
  if (!m) return null;
  try {
    const bytes = Uint8Array.from(atob(m[2].replace(/\s/g, "")), (c) => c.charCodeAt(0));
    return new File([bytes], "", { type: m[1] });
  } catch {
    return null; // not base64 after all: plain text, then
  }
}
