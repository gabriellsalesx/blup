// The editor showing the Markdown already drawn, like the page: the marks
// (**, *, ~~, ==, `, # , > , ---, ```, [ ]) hide and what they mean shows,
// and they come back while the cursor is on them, to edit them.
//
// - quotes: a bar at the left, the > hidden;
// - callouts: the type's icon and the title (its own or the type's name),
//   the [!type] hidden;
// - code blocks: a box with the language and a Copy button, the fences
//   hidden while the cursor is outside the block;
// - --- : a line across;
// - bold, italic, struck, highlighted and inline code without their marks;
// - headings without their #;
// - lists with a bullet, tasks with a box that ticks on a click;
// - images (![](…) and ![[….png]]) shown under their line.
import { EditorView, Decoration, ViewPlugin, WidgetType, type DecorationSet, type ViewUpdate } from "@codemirror/view";
import { type EditorState, type Range } from "@codemirror/state";
import { syntaxTree } from "@codemirror/language";
import { CALLOUTS, CALLOUT_ALIASES, calloutIcon, calloutName } from "./callouts";
import { frontEnd, type ImageSpan } from "./imagedrag";
import type { SyntaxNode } from "@lezer/common";

export type LiveText = { copy: string; copied: string; code: string; lang: string; remote: string; moveImage: string; removeImage: string };
export type LiveOptions = {
  text: LiveText;
  image: (src: string) => Promise<string>; // an image's data: URL ("" when missing)
  onCopy?: (code: string) => void;
  onImageDrag?: (span: ImageSpan | null) => void; // an image picked up to move (null: let go)
  onImagePoint?: (x: number, y: number) => void; // held by a corner, the finger or the mouse here
  onImageDrop?: (x: number, y: number) => void; // and let go here
  onImageRemove?: (span: ImageSpan) => void;
};

const hide = Decoration.replace({});
const quoteLine = Decoration.line({ class: "cm-quote" });
const codeLine = (cls: string) => Decoration.line({ class: "cm-codeblock" + cls });
const doneTask = Decoration.mark({ class: "cm-task-done" });

class Rule extends WidgetType {
  eq() { return true; }
  toDOM() { const s = document.createElement("span"); s.className = "cm-hr"; return s; }
}

class Bullet extends WidgetType {
  eq() { return true; }
  toDOM() { const s = document.createElement("span"); s.className = "cm-bullet"; s.textContent = "•"; return s; }
}

class Check extends WidgetType {
  constructor(readonly done: boolean) { super(); }
  eq(o: Check) { return o.done === this.done; }
  toDOM(view: EditorView) {
    const box = document.createElement("input");
    box.type = "checkbox";
    box.className = "cm-task-box";
    box.checked = this.done;
    box.addEventListener("mousedown", (e) => e.preventDefault());
    box.addEventListener("click", (e) => {
      e.preventDefault();
      // The widget stands for "- [ ]": the box is the [ ] after the mark.
      const line = view.state.doc.lineAt(view.posAtDOM(box));
      const m = line.text.match(/^(\s*(?:>\s?)*\s*(?:[-*+]|\d+[.)])\s+\[)[ xX]\]/);
      if (m) view.dispatch({ changes: { from: line.from + m[1].length, to: line.from + m[1].length + 1, insert: this.done ? " " : "x" } });
    });
    return box;
  }
  ignoreEvent() { return true; }
}

class CalloutHead extends WidgetType {
  constructor(readonly type: string, readonly name: string) { super(); }
  eq(o: CalloutHead) { return o.type === this.type && o.name === this.name; }
  toDOM() {
    const c = CALLOUTS.find((x) => x.type === this.type) ?? CALLOUTS[0];
    const s = document.createElement("span");
    s.className = "cm-callout-icon";
    s.innerHTML = calloutIcon(c, 17) + (this.name ? `<span class="cm-callout-name"></span>` : "");
    if (this.name) s.querySelector(".cm-callout-name")!.textContent = this.name;
    return s;
  }
}

// The top of a code block: its language and the Copy button (alone, at the
// right, while the fences show).
class CodeHead extends WidgetType {
  constructor(readonly lang: string, readonly code: string, readonly raw: boolean, readonly opts: LiveOptions) { super(); }
  eq(o: CodeHead) { return o.lang === this.lang && o.code === this.code && o.raw === this.raw; }
  toDOM() {
    const bar = document.createElement("span");
    bar.className = "cm-code-head" + (this.raw ? " raw" : "");
    if (!this.raw) {
      const l = document.createElement("span");
      l.className = "cm-code-lang";
      l.textContent = this.lang;
      bar.append(l);
    }
    const b = document.createElement("button");
    b.type = "button";
    b.className = "cm-code-copy";
    b.textContent = this.opts.text.copy;
    b.addEventListener("mousedown", (e) => { e.preventDefault(); e.stopPropagation(); });
    b.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      this.opts.onCopy?.(this.code);
      b.textContent = this.opts.text.copied;
      setTimeout(() => (b.textContent = this.opts.text.copy), 1400);
    });
    bar.append(b);
    return bar;
  }
  ignoreEvent() { return true; }
}

// The images already loaded, by their address in the note; the web
// addresses the writer chose to load (an image from the web tells its
// server the note was opened, so it waits for a click).
const loaded = new Map<string, string>();
const allowed = new Set<string>();

// Addresses are relative to the note: another note starts over.
export function forgetImages() { loaded.clear(); }

// A click or a tap on a picture gives it the focus: a line around it, a
// button that moves it and one that takes it out of the note. While it has
// the focus, dragging the picture itself (or the move button) carries it to
// another line: the blue line tells where. Nothing resizes it. Without the
// focus, the mouse can still drag it (dragstart below). Delete removes it,
// and Esc or a click anywhere else lets go of it.
const TRASH = `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10 11v6"/><path d="M14 11v6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`;
// Lucide "move".
const MOVE = `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 2v20"/><path d="m15 19-3 3-3-3"/><path d="m19 9 3 3-3 3"/><path d="M2 12h20"/><path d="m5 9-3 3 3 3"/><path d="m9 5 3-3 3 3"/></svg>`;
// How far a finger goes on the picture before it counts as moving it.
const SLOP = 8;
let letGo: (() => void) | null = null;

function focusPicture(view: EditorView, box: HTMLElement, size: number, opts: LiveOptions) {
  if (box.classList.contains("focused")) return;
  letGo?.();
  const span = () => { const from = view.posAtDOM(box); return { from, to: from + size }; };

  // Carries the picture from a pointer already held on el; started at once
  // by the move button, after SLOP by the picture itself (a tap stays a tap).
  const carry = (el: HTMLElement, e: PointerEvent, now: boolean) => {
    const x0 = e.clientX, y0 = e.clientY;
    let on = false;
    el.setPointerCapture(e.pointerId);
    const start = () => {
      on = true;
      box.classList.add("moving");
      opts.onImageDrag?.(span());
    };
    if (now) start();
    const move = (ev: PointerEvent) => {
      if (!on && Math.hypot(ev.clientX - x0, ev.clientY - y0) < SLOP) return;
      if (!on) start();
      opts.onImagePoint?.(ev.clientX, ev.clientY);
    };
    const end = (ev: PointerEvent) => {
      el.removeEventListener("pointermove", move);
      el.removeEventListener("pointerup", end);
      el.removeEventListener("pointercancel", end);
      if (!on) return;
      box.classList.remove("moving");
      if (ev.type === "pointerup") opts.onImageDrop?.(ev.clientX, ev.clientY);
      opts.onImageDrag?.(null);
    };
    el.addEventListener("pointermove", move);
    el.addEventListener("pointerup", end);
    el.addEventListener("pointercancel", end);
  };

  const mover = document.createElement("button");
  mover.type = "button";
  mover.className = "cm-image-move";
  mover.title = opts.text.moveImage;
  mover.setAttribute("aria-label", opts.text.moveImage);
  mover.innerHTML = MOVE;
  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "cm-image-remove";
  remove.title = opts.text.removeImage;
  remove.setAttribute("aria-label", opts.text.removeImage);
  remove.innerHTML = TRASH;
  remove.addEventListener("click", (e) => { e.stopPropagation(); opts.onImageRemove?.(span()); });
  const parts: HTMLElement[] = [mover, remove];

  // The notes on a phone slide away from under a finger: not this picture.
  const keepTouch = (e: TouchEvent) => e.stopPropagation();
  const fromButton = (e: PointerEvent) => { e.preventDefault(); e.stopPropagation(); carry(mover, e, true); };
  const fromPicture = (e: PointerEvent) => {
    if (e.target !== box && !(e.target as HTMLElement).matches("img")) return;
    e.preventDefault();
    e.stopPropagation();
    carry(box, e, false);
  };
  mover.addEventListener("pointerdown", fromButton);
  box.addEventListener("touchstart", keepTouch, { passive: true });
  box.addEventListener("pointerdown", fromPicture);

  const outside = (e: Event) => { if (!box.contains(e.target as Node)) letGo?.(); };
  const keys = (e: KeyboardEvent) => {
    if (e.key === "Delete" || e.key === "Backspace") { e.preventDefault(); opts.onImageRemove?.(span()); }
    else if (e.key === "Escape") { e.preventDefault(); letGo?.(); view.focus(); }
  };
  box.classList.add("focused");
  box.append(...parts);
  box.tabIndex = -1;
  box.addEventListener("keydown", keys);
  document.addEventListener("pointerdown", outside, true);
  box.focus({ preventScroll: true }); // out of the text: on a phone the keyboard goes
  letGo = () => {
    letGo = null;
    document.removeEventListener("pointerdown", outside, true);
    box.removeEventListener("keydown", keys);
    box.removeEventListener("touchstart", keepTouch);
    box.removeEventListener("pointerdown", fromPicture);
    box.classList.remove("focused", "moving");
    box.removeAttribute("tabindex");
    for (const part of parts) part.remove();
  };
}

class Picture extends WidgetType {
  // size: how long its Markdown is.
  constructor(readonly src: string, readonly alt: string, readonly size: number, readonly opts: LiveOptions) { super(); }
  eq(o: Picture) { return o.src === this.src && o.alt === this.alt && o.size === this.size; }
  toDOM(view: EditorView) {
    const box = document.createElement("span");
    box.className = "cm-image";
    const img = document.createElement("img");
    img.alt = this.alt;
    const show = (url: string) => {
      if (!url) { box.classList.add("missing"); box.textContent = this.alt || this.src; return; }
      img.src = url;
      img.onload = () => view.requestMeasure();
      box.append(img);
    };
    // Dragging it carries its Markdown, not the address the picture shows
    // from (see imagedrag.ts).
    img.addEventListener("dragstart", (e) => {
      const from = view.posAtDOM(box), to = from + this.size;
      e.dataTransfer?.clearData();
      e.dataTransfer?.setData("text/plain", view.state.sliceDoc(from, to));
      if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
      this.opts.onImageDrag?.({ from, to });
    });
    img.addEventListener("dragend", () => this.opts.onImageDrag?.(null));
    img.addEventListener("click", () => focusPicture(view, box, this.size, this.opts));
    if (/^https?:/i.test(this.src) && !allowed.has(this.src)) {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "cm-image-load";
      let host = this.src;
      try { host = new URL(this.src).host; } catch {}
      b.textContent = this.opts.text.remote.replace("{host}", host);
      b.addEventListener("mousedown", (e) => { e.preventDefault(); e.stopPropagation(); });
      b.addEventListener("click", (e) => { e.preventDefault(); e.stopPropagation(); allowed.add(this.src); b.remove(); show(this.src); });
      box.append(b);
    } else if (/^(https?:|data:image\/)/i.test(this.src)) show(this.src);
    else if (loaded.has(this.src)) show(loaded.get(this.src)!);
    else this.opts.image(this.src).then((url) => { if (url) loaded.set(this.src, url); show(url); }, () => show(""));
    return box;
  }
  destroy(dom: HTMLElement) { if (dom.classList.contains("focused")) letGo?.(); }
}

const IMAGE_FILE = /\.(png|jpe?g|webp|gif|svg|avif)$/i;
const WIKI_IMAGE = /!\[\[([^\]|\n]+?)(\|[^\]\n]*)?\]\]/g;

function build(view: EditorView, opts: LiveOptions): DecorationSet {
  const { state } = view;
  const doc = state.doc;
  const sel = state.selection.ranges;
  const touched = (from: number, to: number) => sel.some((r) => r.from <= to && r.to >= from);
  const lineTouched = (pos: number) => { const l = doc.lineAt(pos); return touched(l.from, l.to); };
  const out: Range<Decoration>[] = [];
  const front = frontEnd(state);
  const quoted = new Set<number>(); // lines of quotes already done

  for (const { from, to } of view.visibleRanges) {
    syntaxTree(state).iterate({
      from, to,
      enter: (n) => {
        if (n.from < front) return n.to > front;
        switch (n.name) {
          case "HeaderMark": {
            // # of a heading (not the underline of a Setext one).
            const line = doc.lineAt(n.from);
            if (!/^#/.test(state.sliceDoc(n.from, n.to)) || n.from !== line.from + (line.text.length - line.text.trimStart().length)) return;
            if (!lineTouched(n.from)) out.push(hide.range(n.from, Math.min(n.to + 1, line.to)));
            return;
          }
          case "Emphasis": case "StrongEmphasis": case "Strikethrough": case "InlineCode": {
            if (touched(n.from, n.to)) return;
            const mark = n.name === "Strikethrough" ? "StrikethroughMark" : n.name === "InlineCode" ? "CodeMark" : "EmphasisMark";
            const c = n.node.cursor();
            if (c.firstChild()) do { if (c.name === mark) out.push(hide.range(c.from, c.to)); } while (c.nextSibling());
            return n.name !== "InlineCode";
          }
          case "HorizontalRule": {
            if (!lineTouched(n.from)) out.push(Decoration.replace({ widget: new Rule() }).range(n.from, n.to));
            return false;
          }
          case "FencedCode": {
            const first = doc.lineAt(n.from), last = doc.lineAt(n.to);
            const node = n.node;
            const marks = node.getChildren("CodeMark");
            const text = node.getChild("CodeText");
            const info = node.getChild("CodeInfo");
            const closed = marks.length > 1 && doc.lineAt(marks[marks.length - 1].from).number > first.number;
            const inside = touched(n.from, n.to);
            const code = text ? state.sliceDoc(text.from, text.to) : "";
            const lang = info ? state.sliceDoc(info.from, info.to) : "";
            for (let k = first.number; k <= last.number; k++) {
              const cls = (k === first.number ? " cm-codeblock-first" : "") + (k === last.number ? " cm-codeblock-last" : "");
              out.push(codeLine(cls).range(doc.line(k).from));
            }
            const head = new CodeHead(lang || opts.text.code, code, inside, opts);
            if (inside) out.push(Decoration.widget({ widget: head, side: 1 }).range(first.to));
            else {
              out.push(Decoration.replace({ widget: head }).range(first.from, first.to));
              if (closed && last.number > first.number) out.push(hide.range(last.from, last.to));
            }
            return false;
          }
          case "Blockquote": {
            const first = doc.lineAt(n.from), last = doc.lineAt(n.to);
            const head = first.text.match(/^>\s*\[!(\w+)\]([+-]?)\s*/);
            for (let k = first.number; k <= last.number; k++) {
              if (quoted.has(k)) continue;
              quoted.add(k);
              const line = doc.line(k);
              const m = line.text.match(/^(\s*>\s?)+/);
              if (!head) out.push(quoteLine.range(line.from));
              if (lineTouched(line.from)) continue;
              if (head && k === first.number) {
                // [!tip] Title: the icon, then the title (or the type's name).
                const type = head[1].toLowerCase();
                const known = CALLOUT_ALIASES[type] ?? type;
                const c = CALLOUTS.find((x) => x.type === known);
                const own = line.text.slice(head[0].length).trim();
                const name = own ? "" : c ? calloutName(c, opts.text.lang) : head[1];
                out.push(Decoration.replace({ widget: new CalloutHead(c?.type ?? "note", name) }).range(line.from, line.from + head[0].length));
              } else if (m) out.push(hide.range(line.from, line.from + m[0].length));
            }
            return true;
          }
          case "ListMark": {
            const item = n.node.parent;
            const task = item?.getChild("Task");
            const mark = state.sliceDoc(n.from, n.to);
            if (task) {
              const box = task.getChild("TaskMarker");
              if (!box) return;
              const done = /x/i.test(state.sliceDoc(box.from, box.to));
              if (!touched(n.from, box.to)) out.push(Decoration.replace({ widget: new Check(done) }).range(n.from, box.to));
              if (done && box.to < task.to) out.push(doneTask.range(box.to, task.to));
            } else if (/^[-*+]$/.test(mark) && item?.parent?.name === "BulletList" && !touched(n.from, n.to)) {
              out.push(Decoration.replace({ widget: new Bullet() }).range(n.from, n.to));
            }
            return;
          }
          case "Image": {
            const url = n.node.getChild("URL");
            if (!url || lineTouched(n.from)) return false;
            const raw = state.sliceDoc(n.from, n.to);
            const alt = raw.match(/^!\[([^\]]*)\]/)?.[1] ?? "";
            out.push(Decoration.replace({ widget: new Picture(state.sliceDoc(url.from, url.to), alt, raw.length, opts) }).range(n.from, n.to));
            return false;
          }
        }
      },
    });

    // What the parser doesn't know: ![[image.png]] and ==highlight==.
    const text = state.sliceDoc(from, to);
    for (const m of text.matchAll(WIKI_IMAGE)) {
      const a = from + m.index!, b = a + m[0].length;
      if (a < front || !IMAGE_FILE.test(m[1].trim()) || lineTouched(a) || inCode(state, a)) continue;
      out.push(Decoration.replace({ widget: new Picture("[[" + m[1].trim() + "]]", m[1].trim(), m[0].length, opts) }).range(a, b));
    }
    for (const m of text.matchAll(/==(?=\S)[^=\n]+?==/g)) {
      const a = from + m.index!, b = a + m[0].length;
      if (a < front || touched(a, b) || inCode(state, a)) continue;
      out.push(hide.range(a, a + 2), hide.range(b - 2, b));
    }
  }
  return Decoration.set(out, true);
}

function inCode(state: EditorState, pos: number) {
  for (let n: SyntaxNode | null = syntaxTree(state).resolveInner(pos, 1); n; n = n.parent) {
    if (/^(InlineCode|FencedCode|CodeBlock|CodeText)$/.test(n.name)) return true;
  }
  return false;
}

export function livePreview(opts: LiveOptions) {
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;
      constructor(view: EditorView) { this.decorations = build(view, opts); }
      update(u: ViewUpdate) {
        const parsed = syntaxTree(u.startState) !== syntaxTree(u.state);
        if (u.docChanged || u.selectionSet || u.viewportChanged || parsed) this.decorations = build(u.view, opts);
      }
    },
    { decorations: (v) => v.decorations },
  );
}
