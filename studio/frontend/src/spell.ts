// The spell check in the editor: wavy underlines on the words the chosen
// dictionary doesn't know, in the part of the note on screen. Code, links,
// addresses and wikilinks are left alone. The words come from a worker
// (spell.worker.ts), so typing never waits for it.
import { ViewPlugin, Decoration, type DecorationSet, type EditorView, type ViewUpdate } from "@codemirror/view";
import { RangeSetBuilder } from "@codemirror/state";
import { syntaxTree } from "@codemirror/language";
import SpellWorker from "./spell.worker.ts?worker";

const worker = new SpellWorker();
let nextId = 0;
const waiting = new Map<number, (v: any) => void>();
worker.onmessage = (e) => { waiting.get(e.data.id)?.(e.data); waiting.delete(e.data.id); };
function ask(msg: object): Promise<any> {
  const id = ++nextId;
  return new Promise((resolve) => { waiting.set(id, resolve); worker.postMessage({ ...msg, id }); });
}

// What the worker said about each word, for the current dictionary.
let known = new Map<string, boolean>();
let generation = 0; // grows when the dictionary changes: the editors redraw
const listeners = new Set<() => void>();

export function setupSpell(lang: string, words: string[]) {
  worker.postMessage({ type: "setup", lang, words: [...words] });
  known = new Map();
  generation++;
  listeners.forEach((f) => f());
}
export function addWord(word: string) {
  worker.postMessage({ type: "add", word });
  known.set(word, true);
  generation++;
  listeners.forEach((f) => f());
}
export async function suggest(word: string): Promise<string[]> {
  return (await ask({ type: "suggest", word })).suggestions ?? [];
}

const WORD = /\p{L}[\p{L}\p{M}'’-]*\p{L}|\p{L}/gu;
// Not words to check: code, links' addresses, raw HTML, front matter.
const SKIP = new Set(["FencedCode", "CodeBlock", "InlineCode", "CodeText", "URL", "Autolink", "HTMLTag", "HTMLBlock", "LinkLabel", "Comment", "CommentBlock"]);
// Also attributes ({.text-center}), wikilinks, addresses, math, e-mails, HTML, callout types ([!tip])
// and file names (logo.png).
const SKIP_TEXT = /\{\.[^}\n]*\}|\[\[[^\]]*\]\]|\[![^\]]*\]|https?:\/\/\S+|\$[^$\n]+\$|\S+@\S+\.\S+|<[^>]+>|[\p{L}\d_-]+\.[a-z0-9]{2,4}\b/gu;

function words(view: EditorView) {
  const out: { from: number; to: number; word: string }[] = [];
  const tree = syntaxTree(view.state);
  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    const skip: [number, number][] = [];
    tree.iterate({ from, to, enter: (n) => { if (SKIP.has(n.name)) { skip.push([n.from, n.to]); return false; } } });
    for (const m of text.matchAll(SKIP_TEXT)) skip.push([from + m.index!, from + m.index! + m[0].length]);
    for (const m of text.matchAll(WORD)) {
      const a = from + m.index!, b = a + m[0].length;
      const word = m[0].replace(/^['’-]+|['’-]+$/g, "");
      if (word.length < 2 || skip.some(([x, y]) => a < y && b > x)) continue;
      if (/^\p{Lu}+$/u.test(word) && word.length <= 6) continue; // acronyms
      // Glued to digits, paths or a file extension: a name, not a word.
      const doc = view.state.doc, prev = a > 0 ? doc.sliceString(a - 1, a) : "", next = doc.sliceString(b, b + 2);
      if (/[\d_/\\@]/.test(prev + next[0]) || /^\.\p{L}/u.test(next)) continue;
      out.push({ from: a, to: b, word });
    }
  }
  return out;
}

const mark = Decoration.mark({ class: "cm-misspelled" });

export const spellcheck = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet = Decoration.none;
    timer: ReturnType<typeof setTimeout> | undefined;
    seen = generation;
    redraw = () => this.schedule(0);
    constructor(readonly view: EditorView) {
      listeners.add(this.redraw);
      this.schedule(0);
    }
    update(u: ViewUpdate) {
      if (u.docChanged) this.decorations = this.decorations.map(u.changes);
      if (u.docChanged || u.viewportChanged) this.schedule(400);
    }
    schedule(ms: number) {
      clearTimeout(this.timer);
      this.timer = setTimeout(() => this.run(), ms);
    }
    async run() {
      if (!enabled) { this.set(Decoration.none); return; }
      const list = words(this.view);
      const unknown = [...new Set(list.map((w) => w.word).filter((w) => !known.has(w)))];
      if (unknown.length) {
        const gen = generation;
        const res = await ask({ type: "check", words: unknown });
        if (gen !== generation || res.error) return;
        const wrong = new Set<string>(res.wrong);
        for (const w of unknown) known.set(w, !wrong.has(w));
      }
      const b = new RangeSetBuilder<Decoration>();
      const max = this.view.state.doc.length;
      for (const w of list) if (known.get(w.word) === false && w.to <= max) b.add(w.from, w.to, mark);
      this.set(b.finish());
    }
    set(d: DecorationSet) {
      this.decorations = d;
      this.view.dispatch({}); // redraw with the new marks
    }
    destroy() { clearTimeout(this.timer); listeners.delete(this.redraw); }
  },
  { decorations: (v) => v.decorations },
);

let enabled = true;
export function enableSpell(on: boolean) {
  enabled = on;
  generation++;
  listeners.forEach((f) => f());
}

// The misspelled word under a position, if any.
export function wordAt(view: EditorView, pos: number): { from: number; to: number; word: string } | null {
  const plugin = view.plugin(spellcheck);
  if (!plugin) return null;
  let hit: { from: number; to: number } | null = null;
  plugin.decorations.between(pos, pos, (from, to) => { hit = { from, to }; });
  if (!hit) return null;
  const { from, to } = hit as { from: number; to: number };
  return { from, to, word: view.state.sliceDoc(from, to) };
}

// Words for the word being typed: corrections when it is misspelled, then
// dictionary words that start with it.
export async function complete(word: string): Promise<{ fixes: string[]; words: string[] }> {
  if (!enabled) return { fixes: [], words: [] };
  const r = await ask({ type: "complete", word });
  return { fixes: r.fixes ?? [], words: r.words ?? [] };
}

// Inside code, a link's address or a wikilink nothing is suggested.
export function skipAt(view: EditorView, pos: number) {
  let skip = false;
  for (let n: any = syntaxTree(view.state).resolveInner(pos, -1); n; n = n.parent) if (SKIP.has(n.name)) skip = true;
  const line = view.state.doc.lineAt(pos);
  const before = line.text.slice(0, pos - line.from);
  const ticks = (before.match(/`/g) ?? []).length;
  return skip || ticks % 2 === 1 || /\[\[[^\]]*$|https?:\/\/\S*$/.test(before);
}
