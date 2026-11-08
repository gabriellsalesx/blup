<script lang="ts">
  import { onMount, type Snippet } from "svelte";
  import { EditorView, basicSetup } from "codemirror";
  import { keymap, Decoration, type DecorationSet } from "@codemirror/view";
  import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
  import { languages } from "@codemirror/language-data";
  import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
  import { tags } from "@lezer/highlight";
  import { StateField, RangeSetBuilder, type EditorState } from "@codemirror/state";
  import { look, setPref } from "./theme.svelte";
  import { spellcheck, wordAt, suggest, addWord, complete, skipAt } from "./spell";
  import { autocompletion, startCompletion, acceptCompletion, completionStatus, closeCompletion, moveCompletionSelection, selectedCompletion, type CompletionContext, type Completion } from "@codemirror/autocomplete";
  import { Prec } from "@codemirror/state";
  import { undo, redo, undoDepth, redoDepth } from "@codemirror/commands";
  import { undoHandlers } from "./undo";
  import { tableFrom, tables } from "./table";
  import { toast } from "./toast.svelte";
  import ContextMenu, { type MenuEntry } from "./ContextMenu.svelte";
  import NotePicker from "./NotePicker.svelte";
  import { Clipboard } from "@wailsio/runtime";
  import LinkIcon from "@lucide/svelte/icons/link-2";
  import External from "@lucide/svelte/icons/external-link";
  import Brush from "@lucide/svelte/icons/paintbrush";
  import Pilcrow from "@lucide/svelte/icons/pilcrow";
  import ListPlus from "@lucide/svelte/icons/list-plus";
  import Scissors from "@lucide/svelte/icons/scissors";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import Paste from "@lucide/svelte/icons/clipboard-paste";
  import PastePlain from "@lucide/svelte/icons/clipboard-type";
  import SelectAll from "@lucide/svelte/icons/square-dashed";
  import CommentIcon from "@lucide/svelte/icons/eye-off";
  import CodeBlock from "@lucide/svelte/icons/square-code";
  import Undo from "@lucide/svelte/icons/undo-2";
  import Redo from "@lucide/svelte/icons/redo-2";
  import { EditorSelection } from "@codemirror/state";
  import { wrap, prefix, block, link, insert, heading, align, alignAt, headingAt, ALIGN_LINE, callout } from "./mdcommands";
  import { CALLOUTS, calloutIcon, calloutName } from "./callouts";
  import { mdExtras } from "./mdextras";
  import { linkConceal, headingFolds } from "./livemd";
  import { forgetImages, livePreview } from "./livepreview";
  import { blockAt, dataImage, dropPoint, moveImage, removeImage, scrollNear, type ImageSpan } from "./imagedrag";
  import Highlighter from "@lucide/svelte/icons/highlighter";
  import AlignLeft from "@lucide/svelte/icons/text-align-start";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import { PasteImage, PickNoteImage, OpenURL, NoteImage } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import { targetAt, type Target } from "./notelinks";
  import Bold from "@lucide/svelte/icons/bold";
  import Italic from "@lucide/svelte/icons/italic";
  import Strike from "@lucide/svelte/icons/strikethrough";
  import Heading from "@lucide/svelte/icons/heading";
  import List from "@lucide/svelte/icons/list";
  import ListOrdered from "@lucide/svelte/icons/list-ordered";
  import ListChecks from "@lucide/svelte/icons/list-checks";
  import Quote from "@lucide/svelte/icons/quote";
  import Info from "@lucide/svelte/icons/info";
  import Link from "@lucide/svelte/icons/link";
  import ImageIcon from "@lucide/svelte/icons/image";
  import Code from "@lucide/svelte/icons/code";
  import Table from "@lucide/svelte/icons/table";
  import Minus from "@lucide/svelte/icons/minus";
  import { t } from "./i18n";

  // Markdown colors from the Studio's tokens, readable in light and dark.
  const markdownStyle = HighlightStyle.define([
    { tag: tags.heading1, fontWeight: "800", fontSize: "1.5em" },
    { tag: tags.heading2, fontWeight: "700", fontSize: "1.3em" },
    { tag: tags.heading3, fontWeight: "700", fontSize: "1.15em" },
    { tag: [tags.heading4, tags.heading5, tags.heading6], fontWeight: "700" },
    { tag: tags.strong, fontWeight: "700" },
    { tag: tags.emphasis, fontStyle: "italic" },
    { tag: tags.strikethrough, textDecoration: "line-through" },
    { tag: [tags.link, tags.url], color: "var(--blue)" },
    { tag: tags.monospace, fontFamily: "ui-monospace, monospace", fontSize: ".9em", backgroundColor: "var(--soft)", borderRadius: "4px" },
    { tag: [tags.processingInstruction, tags.meta, tags.contentSeparator], color: "var(--muted)" },
    { tag: tags.quote, color: "var(--muted)", fontStyle: "italic" },
  ]);

  // Obsidian comments (%% text %%, on one line or several) never reach the
  // site: they show faded and in italics, with a hint on hover. Code keeps
  // its %% as written, like the theme does.
  const commentMark = Decoration.mark({ class: "cm-obs-comment", attributes: { title: t.editor.comment } });
  function comments(state: EditorState): DecorationSet {
    const b = new RangeSetBuilder<Decoration>();
    const doc = state.doc.toString();
    if (doc.includes("%%")) {
      for (const m of doc.matchAll(/(```[\s\S]*?```|`[^`\n]*`)|%%[\s\S]*?%%/g)) {
        if (!m[1]) b.add(m.index!, m.index! + m[0].length, commentMark);
      }
    }
    return b.finish();
  }
  const commentField = StateField.define<DecorationSet>({
    create: comments,
    update: (deco, tr) => (tr.docChanged ? comments(tr.state) : deco),
    provide: (f) => EditorView.decorations.from(f),
  });

  // A Markdown editor. onChange gets every edit; onSave, Ctrl/Cmd+S.
  // note is the note's path (for the images' links); notes, every note (for
  // [[ ).
  // onOpenNote: a link to another note was clicked (the workspace opens it,
  // and the preview follows).
  // onNewBoard (a vault of notes): makes a canvas or a drawing and gives its
  // path, for the / menu. header: what scrolls with the note, above its
  // text (a vault's title and properties).
  // atEnd: the cursor starts at the end of the text (the note of the day);
  // onReady says it is in place.
  let { text, note, notes = [], header, atEnd = false, onReady, onChange, onSave, onError, onOpenNote, onNewBoard }: {
    text: string; note: string; notes?: { name: string; path: string }[];
    onChange: (v: string) => void; onSave: () => void; onError?: (e: string) => void;
    onOpenNote?: (path: string) => void;
    onNewBoard?: (kind: "canvas" | "drawing") => Promise<string>;
    header?: Snippet;
    atEnd?: boolean;
    onReady?: () => void;
  } = $props();
  const te = t.editor;
  let host: HTMLDivElement;
  // Right-click on an underlined word: the dictionary's suggestions.
  let fix = $state<{ from: number; to: number; word: string; x: number; y: number; list: string[] | null } | null>(null);
  let mouseButton = 0;

  // The editor's right-click menu.
  let ctx = $state<{ x: number; y: number } | null>(null);
  let picking = $state(false); // the note picker of "Link to a note"
  const sel = () => view.state.selection.main;
  const selected = () => view.state.sliceDoc(sel().from, sel().to);
  // A link to another note in the format the site understands: a Markdown
  // link with the path relative to this note ([Name](../Folder/Name.md)).
  function noteHref(target: string) {
    const from = note.split("/").slice(0, -1);
    const to = target.split("/");
    let i = 0;
    while (i < from.length && i < to.length - 1 && from[i] === to[i]) i++;
    const parts = [...from.slice(i).map(() => ".."), ...to.slice(i)];
    return parts.map((p) => p.replace(/%/g, "%25").replace(/ /g, "%20").replace(/\(/g, "%28").replace(/\)/g, "%29")).join("/");
  }
  // The link under pos (see notelinks.ts).
  function linkAt(v: EditorView, pos: number): Target | null {
    const line = v.state.doc.lineAt(pos);
    return targetAt(line.text, pos - line.from, note, notes);
  }

  const mdLink = (text: string, n: { path: string }) => `[${text.replace(/[[\]]/g, "\\$&")}](${noteHref(n.path)})`;
  function linkToNote(n: { name: string; path: string }) {
    picking = false;
    insert(view, mdLink(selected() || n.name, n));
  }
  async function copySel(cut: boolean) {
    await Clipboard.SetText(selected());
    if (cut) view.dispatch({ changes: { from: sel().from, to: sel().to, insert: "" } });
    view.focus();
  }
  async function paste(plain: boolean) {
    let text = await Clipboard.Text().catch(() => "");
    if (plain) text = text.replace(/[*_~`#>]|\[([^\]]*)\]\([^)]*\)/g, (m, link) => link ?? "");
    insert(view, text);
  }
  const menuItems = (): MenuEntry[] => {
    const has = !sel().empty;
    return [
      { label: te.linkNote, Icon: LinkIcon, run: () => (picking = true) },
      { label: te.linkOut, Icon: External, run: () => link(view, te.linkText), hint: "Ctrl+K" },
      "sep",
      { label: te.format, Icon: Brush, sub: [
        { label: te.bold, Icon: Bold, run: () => wrap(view, "**", te.boldText), hint: "Ctrl+B" },
        { label: te.italic, Icon: Italic, run: () => wrap(view, "*", te.italicText), hint: "Ctrl+I" },
        { label: te.strike, Icon: Strike, run: () => wrap(view, "~~", te.strikeText) },
        { label: te.highlight, Icon: Highlighter, run: () => wrap(view, "==", te.highlightText) },
        { label: te.code, Icon: Code, run: () => wrap(view, "`", te.codeText) },
        { label: te.commentText, Icon: CommentIcon, run: () => wrap(view, "%%", te.commentPh) },
      ] },
      { label: te.paragraph, Icon: Pilcrow, sub: [
        { label: te.normal, run: () => heading(view, 0) },
        ...[1, 2, 3, 4].map((n) => ({ label: te.headingN.replace("{n}", String(n)), run: () => heading(view, n) })),
        "sep",
        { label: te.list, Icon: List, run: () => prefix(view, "- ", LIST) },
        { label: te.olist, Icon: ListOrdered, run: () => prefix(view, (i) => `${i + 1}. `, LIST) },
        { label: te.task, Icon: ListChecks, run: () => prefix(view, "- [ ] ", LIST) },
        { label: te.quote, Icon: Quote, run: () => prefix(view, "> ", QUOTE) },
      ] },
      { label: te.insertMenu, Icon: ListPlus, sub: [
        { label: te.table, Icon: Table, run: newTable },
        { label: te.callout, Icon: Info, run: () => callout(view, "tip", calloutFold ? "-" : "", calloutName(CALLOUTS.find((c) => c.type === "tip")!, t.lang), te.calloutText) },
        { label: te.image, Icon: ImageIcon, run: pickImage },
        { label: te.codeBlock, Icon: CodeBlock, run: () => block(view, "```\n\n```", [4, 4]) },
        { label: te.hr, Icon: Minus, run: () => block(view, "---") },
      ] },
      "sep",
      { label: te.cut, Icon: Scissors, run: () => copySel(true), disabled: !has, hint: "Ctrl+X" },
      { label: te.copy, Icon: CopyIcon, run: () => copySel(false), disabled: !has, hint: "Ctrl+C" },
      { label: te.paste, Icon: Paste, run: () => paste(false), hint: "Ctrl+V" },
      { label: te.pastePlain, Icon: PastePlain, run: () => paste(true) },
      { label: te.selectAll, Icon: SelectAll, run: () => { view.dispatch({ selection: { anchor: 0, head: view.state.doc.length } }); view.focus(); }, hint: "Ctrl+A" },
    ];
  };
  // The suggestions come first, then the corrections list (if open) gives
  // way to the menu in one go: no blink between the two.
  async function onMenu(e: MouseEvent) {
    const pos = view.posAtCoords({ x: e.clientX, y: e.clientY });
    // On a link: the cursor goes in it, which shows it as written, to edit.
    if (pos != null && linkAt(view, pos) && view.state.selection.main.empty) view.dispatch({ selection: EditorSelection.cursor(pos) });
    const w = pos == null ? null : wordAt(view, pos);
    if (!w) { e.preventDefault(); closeCompletion(view); ctx = { x: e.clientX, y: e.clientY }; return; }
    e.preventDefault();
    const at = { x: Math.min(e.clientX, innerWidth - 240), y: Math.min(e.clientY, innerHeight - 260) };
    const list = await suggest(w.word);
    closeCompletion(view);
    fix = { ...w, ...at, list };
  }
  function replace(s: string) {
    if (!fix) return;
    view.dispatch({ changes: { from: fix.from, to: fix.to, insert: s } });
    fix = null;
    view.focus();
  }
  function learn() {
    if (!fix) return;
    addWord(fix.word);
    setPref("words", [...(look.words ?? []), fix.word]);
    fix = null;
    view.focus();
  }
  let view: EditorView;

  // [[ lists the notes; the chosen one becomes a Markdown
  // link ([Name](path.md)), which the site turns into a link.
  function noteLinks(ctx: CompletionContext) {
    const m = ctx.matchBefore(/\[\[[^\]\n|#]*/);
    if (!m) return null;
    const options = notes.filter((n) => n.name !== "_index").map((n) => {
      const folder = n.path.includes("/") ? n.path.slice(0, n.path.lastIndexOf("/")) : "";
      return {
        label: n.name, detail: folder, type: "text",
        apply: (v: EditorView, _c: unknown, from: number, to: number) => {
          const close = v.state.sliceDoc(to, to + 2) === "]]" ? 2 : v.state.sliceDoc(to, to + 1) === "]" ? 1 : 0;
          const link = mdLink(n.name, n);
          v.dispatch({ changes: { from: from - 2, to: to + close, insert: link }, selection: EditorSelection.cursor(from - 2 + link.length) });
        },
      };
    });
    return { from: m.from + 2, options, validFor: /^[^\]\n|#]*$/ };
  }

  // @ mentions (at the start of a line or after a space, so e-mails don't
  // open it): a note of the site (its link, [@Name](path.md)) or a date.
  const startsWord = (ctx: CompletionContext, at: number) => at === ctx.state.doc.lineAt(at).from || /\s/.test(ctx.state.sliceDoc(at - 1, at));
  const dateText = (d: Date) => d.toLocaleDateString(t.locale, { day: "2-digit", month: "2-digit", year: "numeric" });
  function dates(): Completion[] {
    const day = (n: number) => { const d = new Date(); d.setDate(d.getDate() + n); return d; };
    const long = (d: Date) => d.toLocaleDateString(t.locale, { weekday: "long", day: "numeric", month: "long" });
    const list: [string, Date][] = [[te.today, day(0)], [te.tomorrow, day(1)], [te.yesterday, day(-1)]];
    // The next days of the week, by name ("sexta-feira").
    for (let n = 2; n <= 7; n++) list.push([day(n).toLocaleDateString(t.locale, { weekday: "long" }), day(n)]);
    return list.map(([label, d], i) => ({ label, detail: long(d), type: "mention", section: { name: te.secDates, rank: 1 }, boost: -i, apply: dateText(d) }));
  }
  function mentions(ctx: CompletionContext) {
    const m = ctx.matchBefore(/@[^\s@\[\]()]*/u);
    if (!m || !startsWord(ctx, m.from) || skipAt(ctx.view!, m.from)) return null;
    const noteOptions: Completion[] = notes.filter((n) => n.name !== "_index" && n.path !== note).map((n) => ({
      label: n.name, detail: n.path.includes("/") ? n.path.slice(0, n.path.lastIndexOf("/")) : "", type: "mention",
      section: { name: te.secNotes, rank: 0 },
      apply: (v: EditorView, _c: Completion, from: number, to: number) => {
        // The @ stays in the link's text: a mention reads "@Name", in blue.
        const md = mdLink("@" + n.name, n);
        v.dispatch({ changes: { from: from - 1, to, insert: md }, selection: EditorSelection.cursor(from - 1 + md.length) });
      },
    }));
    const dateOptions = dates().map((o) => ({ ...o, apply: (v: EditorView, _c: Completion, from: number, to: number) => {
      const s = o.apply as string;
      v.dispatch({ changes: { from: from - 1, to, insert: s }, selection: EditorSelection.cursor(from - 1 + s.length) });
    } }));
    return { from: m.from + 1, options: [...noteOptions, ...dateOptions], validFor: /^[^\s@\[\]()]*$/u };
  }

  // / actions (at the start of a line or after a space): what the toolbar
  // and the right-click menu do, by typing.
  function actions(ctx: CompletionContext) {
    const m = ctx.matchBefore(/\/[\p{L}\d]*/u);
    if (!m || !startsWord(ctx, m.from) || skipAt(ctx.view!, m.from)) return null;
    const act = (label: string, detail: string, run: () => void, rank = 0): Completion => ({
      label, detail, type: "action", boost: -rank,
      apply: (v: EditorView, _c: Completion, from: number, to: number) => {
        v.dispatch({ changes: { from: from - 1, to, insert: "" } });
        run();
      },
    });
    let r = 0;
    const list: Completion[] = [
      ...[1, 2, 3].map((n) => act(te.headingN.replace("{n}", String(n)), "#".repeat(n), () => heading(view, n), r++)),
      act(te.list, "-", () => prefix(view, "- ", LIST), r++),
      act(te.olist, "1.", () => prefix(view, (i) => `${i + 1}. `, LIST), r++),
      act(te.task, "[ ]", () => prefix(view, "- [ ] ", LIST), r++),
      act(te.quote, ">", () => prefix(view, "> ", QUOTE), r++),
      act(te.callout, "", () => callout(view, "tip", "", calloutName(CALLOUTS.find((c) => c.type === "tip")!, t.lang), te.calloutText), r++),
      act(te.table, "", newTable, r++),
      act(te.codeBlock, "```", () => block(view, "```\n\n```", [4, 4]), r++),
      act(te.hr, "---", () => block(view, "---"), r++),
      act(te.linkNote, "", () => (picking = true), r++),
      act(te.linkOut, "", () => link(view, te.linkText), r++),
      act(te.image, "", pickImage, r++),
      act(te.todayDate, "", () => insert(view, dateText(new Date())), r++),
    ];
    if (onNewBoard) {
      const board = (kind: "canvas" | "drawing") => async () => {
        try {
          const path = await onNewBoard(kind);
          if (path) insert(view, mdLink(path.split("/").pop()!.replace(/\.(draw|canvas)$/i, ""), { path }));
        } catch (e) { onError?.(String(e).replace(/^(Runtime)?Error: /, "")); }
      };
      list.push(act(t.boards.newDrawing, "", board("drawing"), r++), act(t.boards.newCanvas, "", board("canvas"), r++));
    }
    return { from: m.from + 1, options: list, validFor: /^[\p{L}\d]*$/u };
  }

  // While a word is typed (3 letters or more), like VS Code: corrections
  // when it is misspelled, then words of this note and of the dictionary
  // that start with it. Tab puts the chosen one in place.
  async function wordSuggestions(ctx: CompletionContext) {
    if (ctx.view && wordAt(ctx.view, ctx.pos)) return null; // fixSuggestions
    const m = ctx.matchBefore(/\p{L}[\p{L}\p{M}'’-]*/u);
    if (!m || m.to - m.from < 3 || !look.spellcheck || skipAt(ctx.view!, ctx.pos)) return null;
    if (/[@/]/.test(ctx.state.sliceDoc(m.from - 1, m.from))) return null; // the @ and / menus
    if (!ctx.explicit && ctx.state.sliceDoc(m.to, m.to + 1).match(/\p{L}/u)) return null; // inside a word
    const word = m.text;
    const r = await complete(word);
    if (ctx.aborted) return null;
    const seen = new Set<string>([word]);
    const options: Completion[] = [];
    const add = (label: string, type: string, detail: string, boost: number) => {
      if (seen.has(label)) return;
      seen.add(label);
      options.push({ label, type, detail, boost });
    };
    r.fixes.forEach((w, i) => add(w, "fix", te.fix, 99 - i));
    const inNote = ctx.state.doc.toString().match(/\p{L}[\p{L}\p{M}'’-]{3,}/gu) ?? [];
    const low = word.toLowerCase();
    for (const w of new Set(inNote)) if (w.toLowerCase().startsWith(low) && w.length > word.length) add(w, "text", te.inNote, 50);
    r.words.forEach((w, i) => add(w, "text", "", 20 - i));
    return options.length ? { from: m.from, options: options.slice(0, 10), filter: false } : null;
  }

  // The cursor on a word marked in red (a click on it, or typing in it):
  // its corrections, and Tab swaps the whole word.
  async function fixSuggestions(ctx: CompletionContext) {
    const w = ctx.view && wordAt(ctx.view, ctx.pos);
    if (!w || !look.spellcheck) return null;
    const list = await suggest(w.word);
    if (ctx.aborted || !list.length) return null;
    const fixes = [...list.filter((x) => !x.includes(" ")), ...list.filter((x) => x.includes(" "))].slice(0, 6);
    return { from: w.from, to: w.to, options: fixes.map((label, i) => ({ label, type: "fix", detail: te.fix, boost: 99 - i })), filter: false };
  }

  // Images pasted or dropped on the note go to the attachments folder; the
  // note gets their Markdown.
  async function addImages(files: File[], at?: number) {
    if (at != null) view.dispatch({ selection: EditorSelection.cursor(at) });
    for (const f of files) {
      const ext = f.type.split("/")[1]?.replace("jpeg", "jpg").replace("svg+xml", "svg") ?? "png";
      const name = f.name && f.name.includes(".") ? f.name : `image.${ext}`;
      try {
        const data = await new Promise<string>((ok, fail) => {
          const r = new FileReader();
          r.onload = () => ok(String(r.result).split(",")[1] ?? "");
          r.onerror = () => fail(r.error);
          r.readAsDataURL(f);
        });
        insert(view, await PasteImage(note, name, data) + "\n", true);
        toast(t.toast.imagePasted);
      } catch (e) {
        onError?.(String(e).replace(/^(Runtime)?Error: /, ""));
      }
    }
  }
  const images = (list: FileList | undefined | null) => [...(list ?? [])].filter((f) => f.type.startsWith("image/"));
  // The images of a paste or a drop: its files, or an image that came as
  // its address (data:image/…), which must not land in the text.
  function imagesIn(data: DataTransfer | null) {
    const files = images(data?.files);
    if (files.length) return files;
    const file = dataImage(data?.getData("text/plain"));
    return file ? [file] : [];
  }

  // Moving an image with the mouse (see imagedrag.ts). The listeners are on
  // the editor's box, in the capture phase: an image or a table under the
  // pointer would keep the event from the editor.
  let moving: ImageSpan | null = null;
  let dropLine = $state<{ x: number; y: number; width: number } | null>(null);
  // Where the image would land, with the pointer there.
  function landing(x: number, y: number) {
    const pos = view.posAtCoords({ x, y }, false);
    const b = blockAt(view.state, pos);
    const top = view.lineBlockAt(b?.from ?? pos).top, bottom = view.lineBlockAt(b?.to ?? pos).bottom;
    const at = dropPoint(view.state, pos, y - view.documentTop > (top + bottom) / 2);
    return { at, y: view.documentTop + view.lineBlockAt(at)[at === view.state.doc.lineAt(at).from ? "top" : "bottom"] };
  }
  // Both enter and over must be taken, or WebKit refuses the drop.
  function dragOver(e: DragEvent) {
    if (!moving) return;
    e.preventDefault();
    e.stopPropagation();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
    showLine(e.clientX, e.clientY);
  }
  function showLine(x: number, y: number) {
    const r = view.contentDOM.getBoundingClientRect();
    dropLine = { x: r.left, y: landing(x, y).y, width: r.width };
  }
  function dragLeave(e: DragEvent) {
    if (!host.contains(e.relatedTarget as Node | null)) dropLine = null;
  }
  function dropImage(e: DragEvent) {
    if (!moving) return;
    e.preventDefault();
    e.stopPropagation();
    landImage(e.clientX, e.clientY);
  }
  function landImage(x: number, y: number) {
    if (!moving) return;
    const changes = moveImage(view.state, moving, landing(x, y).at);
    if (changes) view.dispatch({ changes, userEvent: "move.drop" });
    moving = dropLine = null;
    view.focus();
  }
  function dropOut(span: ImageSpan) {
    view.dispatch({ changes: removeImage(view.state, span), userEvent: "delete" });
    view.focus();
    toast(te.imageRemoved, { action: { label: te.undo, run: () => undo(view) } });
  }
  async function pickImage() {
    try {
      const md = await PickNoteImage(note);
      if (md) insert(view, md + "\n", true);
    } catch (e) { onError?.(String(e).replace(/^(Runtime)?Error: /, "")); }
  }

  let menuOpen = $state(""); // the toolbar menu open (headings, alignment)
  // The menu floats over the window (the editor's column would cut it),
  // under its button, kept inside the window.
  let menuPos = $state({ x: 0, y: 0 });
  function openMenu(key: string, btn: HTMLElement) {
    if (menuOpen === key) { menuOpen = ""; return; }
    const r = btn.getBoundingClientRect();
    menuPos = { x: Math.max(8, Math.min(r.left, innerWidth - 232)), y: r.bottom + 4 };
    menuOpen = key;
  }
  // What the toolbar shows for the cursor's place: its alignment, its
  // heading level.
  let curAlign = $state<"left" | "center" | "right" | "justify">("left");
  let curHead = $state(0);
  // Undo and redo are greyed out when there is nothing to undo or redo.
  let canUndo = $state(false), canRedo = $state(false);
  type MenuItem = { label: string; run: () => void; svg?: string; size?: string; tint?: string; check?: () => boolean; keep?: boolean };
  let calloutFold = $state(false); // new callouts start folded
  // The menus' icons, drawn here (a picture of each choice: where the lines
  // sit, how big the heading is).
  const lines = (x: number[][]) => `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">${x.map(([a, b], i) => `<path d="M${a} ${5 + i * 4.67}H${b}"/>`).join("")}</svg>`;
  const alignIcons = {
    left: lines([[3, 21], [3, 15], [3, 19], [3, 12]]),
    center: lines([[3, 21], [7, 17], [5, 19], [8, 16]]),
    right: lines([[3, 21], [9, 21], [5, 21], [12, 21]]),
    justify: lines([[3, 21], [3, 21], [3, 21], [3, 14]]),
  };
  const textIcon = (label: string, size: number) => `<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><text x="12" y="16.5" text-anchor="middle" font-size="${size}" font-weight="700" fill="currentColor" font-family="Inter, system-ui, sans-serif">${label}</text></svg>`;

  // A new table, with the cursor in its first empty cell.
  function newTable() {
    block(view, `| ${te.col} 1 | ${te.col} 2 |\n| --- | --- |\n|   |   |`);
    const at = view.state.selection.main.head;
    setTimeout(() => {
      let best: HTMLElement | null = null;
      for (const el of host.querySelectorAll<HTMLElement>(".cm-md-table")) { const from = tableFrom(el); if (from !== undefined && from <= at) best = el; }
      best?.querySelector<HTMLElement>('[data-r="1"][data-c="0"]')?.focus();
    }, 30);
  }

  // The formatting bar.
  const H = /^#{1,6} /, LIST = /^(\s*)([-*+] \[[ x]\] |[-*+] |\d+\. )/, QUOTE = /^> ?/;
  const tools = [
    { key: "undo", label: te.undo + " (Ctrl+Z)", Icon: Undo, run: () => { undo(view); view.focus(); }, off: () => !canUndo },
    { key: "redo", label: te.redo + " (Ctrl+Shift+Z)", Icon: Redo, run: () => { redo(view); view.focus(); }, off: () => !canRedo },
    { key: "sep0" },
    { key: "heading", label: te.heading, Icon: Heading, svg: () => (curHead ? textIcon("H" + curHead, 11) : textIcon("H", 17)), items: [
      { label: te.normal, run: () => heading(view, 0), size: "", svg: textIcon("¶", 15) },
      ...[1, 2, 3, 4].map((n) => ({ label: te.headingN.replace("{n}", String(n)), run: () => heading(view, n), size: ["", "1.35em", "1.2em", "1.08em", "1em"][n], svg: textIcon("H" + n, 11) })),
    ] as MenuItem[] },
    { key: "bold", label: te.bold + " (Ctrl+B)", Icon: Bold, run: () => wrap(view, "**", te.boldText) },
    { key: "italic", label: te.italic + " (Ctrl+I)", Icon: Italic, run: () => wrap(view, "*", te.italicText) },
    { key: "strike", label: te.strike, Icon: Strike, run: () => wrap(view, "~~", te.strikeText) },
    { key: "mark", label: te.highlight, Icon: Highlighter, run: () => wrap(view, "==", te.highlightText) },
    { key: "align", label: te.align, Icon: AlignLeft, svg: () => alignIcons[curAlign], items: [
      { label: te.alignLeft, svg: alignIcons.left, run: () => align(view, null) },
      { label: te.alignCenter, svg: alignIcons.center, run: () => align(view, "text-center") },
      { label: te.alignRight, svg: alignIcons.right, run: () => align(view, "text-right") },
      { label: te.alignJustify, svg: alignIcons.justify, run: () => align(view, "text-justify") },
    ] as MenuItem[] },
    { key: "sep1" },
    { key: "list", label: te.list, Icon: List, run: () => prefix(view, "- ", LIST) },
    { key: "olist", label: te.olist, Icon: ListOrdered, run: () => prefix(view, (i) => `${i + 1}. `, LIST) },
    { key: "task", label: te.task, Icon: ListChecks, run: () => prefix(view, "- [ ] ", LIST) },
    { key: "quote", label: te.quote, Icon: Quote, run: () => prefix(view, "> ", QUOTE) },
    { key: "callout", label: te.callout, Icon: Info, svg: () => calloutIcon(CALLOUTS.find((c) => c.type === "info")!, 17).replace(/stroke="rgb\([^)]*\)"/, 'stroke="currentColor"'), items: [
      // The quote callout is the one with a source (the plain quote is the
      // ❝ button): its title asks for the author.
      ...CALLOUTS.map((c) => ({ label: c.type === "quote" ? te.quoteSource : calloutName(c, t.lang), svg: calloutIcon(c), tint: c.color,
        run: () => callout(view, c.type, calloutFold ? "-" : "", c.type === "quote" ? te.sourceTitle : calloutName(c, t.lang), te.calloutText) })),
      { label: te.foldable, run: () => { calloutFold = !calloutFold; }, check: () => calloutFold, keep: true },
    ] as MenuItem[] },
    { key: "sep2" },
    { key: "link", label: te.link + " (Ctrl+K)", Icon: Link, run: () => link(view, te.linkText) },
    { key: "image", label: te.image, Icon: ImageIcon, run: pickImage },
    { key: "code", label: te.code, Icon: Code, run: () => wrap(view, "`", te.codeText) },
    { key: "table", label: te.table, Icon: Table, run: newTable },
    { key: "hr", label: te.hr, Icon: Minus, run: () => block(view, "---") },
  ];

  onMount(() => {
    forgetImages();
    view = new EditorView({
      parent: host,
      doc: text,
      extensions: [
        basicSetup,
        markdown({ base: markdownLanguage, codeLanguages: languages }),
        syntaxHighlighting(markdownStyle),
        EditorView.lineWrapping,
        commentField,
        mdExtras,
        linkConceal,
        headingFolds({ fold: te.fold, unfold: te.unfold }),
        livePreview({
          text: { copy: te.copy, copied: te.copied, code: te.code, lang: t.lang, remote: te.remoteImage, moveImage: te.moveImage, removeImage: te.removeImage },
          image: (src) => NoteImage(note, src),
          onCopy: (code) => void Clipboard.SetText(code),
          onImageDrag: (span) => { moving = span; if (!span) dropLine = null; },
          onImagePoint: (x, y) => { scrollNear(view.contentDOM, y); showLine(x, y); },
          onImageDrop: landImage,
          onImageRemove: dropOut,
        }),
        tables(t.editor.tableUi),
        EditorView.contentAttributes.of({ spellcheck: "false" }), // ours, below
        spellcheck,
        keymap.of([
          { key: "Mod-s", preventDefault: true, run: () => (onSave(), true) },
          { key: "Mod-b", preventDefault: true, run: (v) => (wrap(v, "**", te.boldText), true) },
          { key: "Mod-i", preventDefault: true, run: (v) => (wrap(v, "*", te.italicText), true) },
          { key: "Mod-k", preventDefault: true, run: (v) => (link(v, te.linkText), true) },
        ]),
        autocompletion({ override: [noteLinks, mentions, actions, fixSuggestions, wordSuggestions], icons: false, defaultKeymap: false, activateOnTypingDelay: 150 }),
        // Tab takes the suggestion; Enter stays a new line.
        Prec.highest(keymap.of([
          // Enter takes the choice of the @ and / menus (elsewhere, it is a
          // new line, below).
          { key: "Enter", run: (v) => (completionStatus(v.state) === "active" && ["mention", "action"].includes(selectedCompletion(v.state)?.type ?? "") ? acceptCompletion(v) : false) },
          // Enter at the end of an aligned paragraph: the new line goes after
          // its hidden {.text-center}, which stays with the paragraph.
          { key: "Enter", run: (v) => {
            const sel = v.state.selection.main, line = v.state.doc.lineAt(sel.head);
            if (!sel.empty || sel.head !== line.to) return false;
            // The cursor is before the hidden mark, or (End) just after it.
            const next = ALIGN_LINE.test(line.text) ? line : line.number < v.state.doc.lines ? v.state.doc.line(line.number + 1) : null;
            if (!next || !ALIGN_LINE.test(next.text)) return false;
            v.dispatch({ changes: { from: next.to, insert: "\n\n" }, selection: { anchor: next.to + 2 }, scrollIntoView: true });
            return true;
          } },
          { key: "Tab", run: (v) => (completionStatus(v.state) === "active" ? acceptCompletion(v) : false) },
          { key: "ArrowDown", run: (v) => (completionStatus(v.state) === "active" ? moveCompletionSelection(true)(v) : false) },
          { key: "ArrowUp", run: (v) => (completionStatus(v.state) === "active" ? moveCompletionSelection(false)(v) : false) },
          { key: "Escape", run: (v) => (completionStatus(v.state) === "active" ? closeCompletion(v) : false) },
        ])),
        EditorView.domEventHandlers({
          // A right-click on a word in red, or on the selection, doesn't move
          // the cursor (that would close the corrections list, or lose the
          // selection, before the menu shows).
          mousedown: (e, v) => {
            mouseButton = e.button;
            if (e.button !== 2) return false;
            const pos = v.posAtCoords({ x: e.clientX, y: e.clientY });
            const inSel = pos != null && v.state.selection.ranges.some((r) => !r.empty && pos >= r.from && pos <= r.to);
            // A right-click on the selection keeps it (the menu acts on it).
            if (pos == null || (!inSel && !wordAt(v, pos))) return false;
            e.preventDefault();
            return true;
          },
          // A link to another note opens it; an address on the web opens
          // in the browser with Ctrl/Cmd (a plain click edits the text).
          click: (e, v) => {
            if (e.button !== 0 || !v.state.selection.main.empty) return false;
            const pos = v.posAtCoords({ x: e.clientX, y: e.clientY });
            const target = pos == null ? null : linkAt(v, pos);
            if (!target) return false;
            if ("note" in target) {
              if (target.note === note) return false;
              e.preventDefault();
              onOpenNote?.(target.note);
              return true;
            }
            if (!(e.ctrlKey || e.metaKey)) return false;
            e.preventDefault();
            OpenURL(target.url);
            return true;
          },
          // The hand over a link that opens something.
          mousemove: (e, v) => {
            const pos = v.posAtCoords({ x: e.clientX, y: e.clientY });
            const target = pos == null ? null : linkAt(v, pos);
            const hand = !!target && ("note" in target ? target.note !== note : e.ctrlKey || e.metaKey);
            v.contentDOM.style.cursor = hand ? "pointer" : "";
            return false;
          },
          paste: (e) => {
            const files = imagesIn(e.clipboardData);
            if (!files.length) return false;
            e.preventDefault();
            addImages(files);
            return true;
          },
          drop: (e, v) => {
            const files = imagesIn(e.dataTransfer);
            if (!files.length) return false;
            e.preventDefault();
            addImages(files, v.posAtCoords({ x: e.clientX, y: e.clientY }) ?? undefined);
            return true;
          },
        }),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) onChange(u.state.doc.toString());
          canUndo = undoDepth(u.state) > 0;
          canRedo = redoDepth(u.state) > 0;
          if (u.docChanged || u.selectionSet) {
            const head = u.state.selection.main.head;
            curAlign = alignAt(u.state, head);
            curHead = headingAt(u.state, head);
          }
          // A click on a word in red opens its corrections (a right-click
          // has its own menu, onMenu).
          if (!u.docChanged && mouseButton !== 2 && u.transactions.some((tr) => tr.isUserEvent("select.pointer"))) {
            const sel = u.state.selection.main;
            if (sel.empty && wordAt(u.view, sel.head)) setTimeout(() => startCompletion(u.view), 0);
          }
        }),
      ],
    });
    view.focus();
    if (atEnd) view.dispatch({ selection: EditorSelection.cursor(view.state.doc.length), scrollIntoView: true });
    onReady?.();
    // Ctrl+Z from the window (see undo.ts) undoes in the editor.
    const onUndo = (cmd: "undo" | "redo") => {
      if (!view.hasFocus) return false;
      (cmd === "undo" ? undo : redo)(view);
      return true;
    };
    undoHandlers.add(onUndo);
    return () => { undoHandlers.delete(onUndo); view.destroy(); };
  });
</script>

<svelte:window onclick={() => { fix = null; menuOpen = ""; }} onkeydown={(e) => e.key === "Escape" && (fix = null)} />

<div class="wrap">
  <div class="bar" role="toolbar" aria-label={te.toolbar}>
    {#each tools as tool (tool.key)}
      {#if tool.items}
        <!-- A button with a small menu (headings, alignment). -->
        <div class="drop">
          <button class="has-menu" class:on={menuOpen === tool.key} title={tool.label} aria-label={tool.label} aria-haspopup="menu" onmousedown={(e) => e.preventDefault()}
            onclick={(e) => { e.stopPropagation(); openMenu(tool.key, e.currentTarget); }}><span class="now">{@html tool.svg()}</span><ChevronDown size={11} /></button>
          {#if menuOpen === tool.key}
            <div class="drop-menu" role="menu" style:left="{menuPos.x}px" style:top="{menuPos.y}px">
              {#each tool.items as item (item.label)}
                {#if item.check}<hr />{/if}
                <button role={item.check ? "menuitemcheckbox" : "menuitem"} aria-checked={item.check ? item.check() : undefined} onmousedown={(e) => e.preventDefault()}
                  onclick={(e) => { if (item.keep) e.stopPropagation(); else menuOpen = ""; item.run(); }}>
                  {#if item.check}<span class="ico box" class:on={item.check()}>{#if item.check()}✓{/if}</span>
                  {:else if item.svg}<span class="ico" style:--tint={item.tint ?? null} class:tinted={item.tint}>{@html item.svg}</span>{/if}<span style:font-size={item.size || null} class:big={item.size}>{item.label}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if tool.Icon}
        <button title={tool.label} aria-label={tool.label} disabled={tool.off?.()} onmousedown={(e) => e.preventDefault()} onclick={tool.run}><tool.Icon size={16} strokeWidth={1.9} /></button>
      {:else}<span class="sep"></span>{/if}
    {/each}
  </div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <!-- The note scrolls as one page: the header, then the text. -->
  <div class={["page", { wide: look.editorWidth === "wide", headed: !!header }]}>
    {#if header}<div class="head">{@render header()}</div>{/if}
    <div class="editor" oncontextmenu={onMenu} class:wide={look.editorWidth === "wide"} style:--size="{look.editorFont}px" ondragentercapture={dragOver} ondragovercapture={dragOver} ondragleavecapture={dragLeave} ondropcapture={dropImage} bind:this={host}></div>
  </div>
  {#if dropLine}<div class="drop-line" style:left="{dropLine.x}px" style:top="{dropLine.y}px" style:width="{dropLine.width}px"></div>{/if}
</div>

{#if ctx}<ContextMenu x={ctx.x} y={ctx.y} items={menuItems()} onClose={() => { ctx = null; view.focus(); }} />{/if}
{#if picking}<NotePicker {notes} onPick={linkToNote} onClose={() => { picking = false; view.focus(); }} />{/if}

{#if fix}
  <div class="fix" role="menu" tabindex="-1" style:left="{fix.x}px" style:top="{fix.y}px" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
    {#if fix.list === null}<small class="muted">{t.editor.looking}</small>
    {:else if fix.list.length}{#each fix.list as s (s)}<button class="sug" onclick={() => replace(s)}>{s}</button>{/each}
    {:else}<small class="muted">{t.editor.none}</small>{/if}
    <hr />
    <button onclick={learn}>{t.editor.add.replace("{word}", fix.word)}</button>
  </div>
{/if}

<style>
  .wrap { height: 100%; display: flex; flex-direction: column; }
  .bar { display: flex; flex-wrap: wrap; justify-content: center; align-items: center; gap: .1rem; padding: .3rem .6rem; border-block-end: 1px solid var(--line); background: var(--panel); }
  .bar button { display: grid; place-items: center; width: 1.9rem; height: 1.9rem; border-radius: .35rem; color: var(--muted); }
  .bar button:hover:not(:disabled) { background: var(--soft); color: var(--text); }
  .bar button:disabled { opacity: .35; cursor: default; }
  .drop { position: relative; }
  .bar .has-menu { display: flex; gap: .05rem; width: auto; padding-inline: .35rem; }
  .bar .has-menu .now { display: inline-grid; place-items: center; }
  .bar .has-menu .now :global(svg) { width: 17px; height: 17px; }
  .bar .has-menu.on { background: var(--soft); color: var(--text); }
  .drop-menu { position: fixed; z-index: 60; max-height: 70vh; overflow-y: auto; display: grid; min-width: 11rem; padding: .3rem; border: 1px solid var(--line); border-radius: .55rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .25); }
  /* The whole row is the button: the icon and the text don't take the
     clicks (between them the Linux window missed them). */
  .drop-menu button { display: grid; grid-template-columns: auto 1fr; align-items: center; gap: .55rem; width: 100%; height: auto; place-items: center start; padding: .4rem .6rem; border-radius: .35rem; font-size: .875rem; text-align: start; white-space: nowrap; cursor: pointer; }
  .drop-menu button > :global(*) { pointer-events: none; }
  .drop-menu .ico { display: inline-grid; place-items: center; width: 1.4rem; height: 1.4rem; flex: none; border-radius: .3rem; background: var(--soft); color: var(--text); }
  .drop-menu button:hover { background: var(--soft); }
  .drop-menu .big { font-weight: 700; }
  .drop-menu .ico.tinted { background: rgb(var(--tint) / .14); }
  .drop-menu .ico.box { border: 1.5px solid var(--line); background: none; font-size: .8rem; font-weight: 700; }
  .drop-menu .ico.box.on { border-color: var(--blue); background: var(--blue); color: #fff; }
  .drop-menu hr { border: 0; border-block-start: 1px solid var(--line); margin: .25rem .2rem; }
  .bar .sep { width: 1px; height: 1.1rem; margin-inline: .3rem; background: var(--line); }
  .page { flex: 1; min-height: 0; overflow-y: auto; display: flex; flex-direction: column; background: var(--panel); }
  .head { width: 100%; max-width: 46rem; margin-inline: auto; padding: 2.25rem 2rem 0; }
  .page.wide .head { max-width: none; }
  .editor { flex: 1; }
  .editor :global(.cm-tooltip-autocomplete) { border: 1px solid var(--line); border-radius: .5rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .25); overflow: hidden; }
  .editor :global(.cm-tooltip-autocomplete > ul) { font-family: inherit !important; font-size: .875rem; max-height: 16rem !important; }
  .editor :global(.cm-tooltip-autocomplete > ul > li) { padding: .35rem .7rem !important; color: var(--text); }
  .editor :global(.cm-tooltip-autocomplete > ul > li[aria-selected]) { background: var(--soft) !important; color: var(--text) !important; }
  .editor :global(.cm-completionDetail) { margin-inline-start: .6rem; color: var(--muted); font-style: normal; font-size: .78rem; }
  .editor :global(.cm-completionMatchedText) { text-decoration: none; font-weight: 700; color: var(--blue); }
  .editor :global(.cm-editor) { min-height: 100%; background: var(--panel); font-size: var(--size, 1rem); }
  .editor :global(.cm-editor.cm-focused) { outline: none; }
  .editor :global(.cm-scroller) { overflow: visible; font-family: "Inter", ui-sans-serif, system-ui, sans-serif; line-height: 1.7; padding: 2rem 0; }
  .headed .editor :global(.cm-scroller) { padding-block-start: .75rem; }
  .editor :global(.cm-content) { max-width: 46rem; margin-inline: auto; padding-inline: 2rem; caret-color: var(--text); }
  .editor.wide :global(.cm-content) { max-width: none; }
  .editor :global(.cm-obs-comment) { opacity: .75; }
  /* Callouts in their type's color (mdextras.ts), like on the site. */
  .editor :global(.cm-callout) { background: rgb(var(--c) / .1); border-inline-start: 3px solid rgb(var(--c)); padding-inline-start: .8rem !important; }
  .editor :global(.cm-callout-head) { color: rgb(var(--c)); font-weight: 700; border-start-start-radius: .3rem; padding-block-start: .3rem; }
  .editor :global(.cm-callout-end) { padding-block-end: .3rem; border-end-start-radius: .3rem; }
  .editor :global(.cm-callout-head *) { color: rgb(var(--c)) !important; }
  .editor :global(.cm-callout :is(span)) { font-style: normal; }
  /* The Markdown drawn as the page (livepreview.ts). */
  .editor :global(.cm-quote) { border-inline-start: 3px solid var(--blue); padding-inline-start: .9rem !important; color: var(--muted); }
  .editor :global(.cm-callout-icon) { display: inline-flex; align-items: center; gap: .4rem; vertical-align: -3px; margin-inline-end: .35rem; }
  .editor :global(.cm-callout-name) { font-weight: 700; }
  .editor :global(.cm-codeblock) { background: var(--soft); padding-inline: 1rem !important; font-family: ui-monospace, monospace; font-size: .9em; }
  .editor :global(.cm-codeblock-first) { border-start-start-radius: .5rem; border-start-end-radius: .5rem; padding-block-start: .35rem; }
  .editor :global(.cm-codeblock-last) { border-end-start-radius: .5rem; border-end-end-radius: .5rem; padding-block-end: .35rem; }
  .editor :global(.cm-code-head) { display: inline-flex; align-items: center; justify-content: space-between; width: 100%; font-family: "Inter", ui-sans-serif, system-ui, sans-serif; }
  .editor :global(.cm-code-head.raw) { width: auto; float: inline-end; }
  .editor :global(.cm-code-lang) { color: var(--muted); font-size: .8rem; text-transform: lowercase; }
  .editor :global(.cm-code-copy) { padding: .1rem .55rem; border: 1px solid var(--line); border-radius: .35rem; background: var(--panel); color: var(--muted); font-size: .75rem; cursor: pointer; }
  .editor :global(.cm-code-copy:hover) { color: var(--blue); border-color: var(--blue); }
  .editor :global(.cm-hr) { display: inline-block; width: 100%; height: 0; vertical-align: middle; border-block-start: 1px solid var(--line); }
  .editor :global(.cm-bullet) { display: inline-block; width: 1ch; color: var(--muted); font-weight: 700; }
  .editor :global(.cm-task-box) { width: 1rem; height: 1rem; margin: 0 .25rem 0 0; vertical-align: -2px; accent-color: var(--blue); cursor: pointer; }
  .editor :global(.cm-task-done) { color: var(--muted); text-decoration: line-through; }
  .drop-line { position: fixed; z-index: 5; height: 2px; margin-block-start: -1px; border-radius: 1px; background: var(--blue); pointer-events: none; }
  .editor :global(.cm-image) { position: relative; display: block; width: fit-content; max-width: 100%; margin-block: .3rem; outline: none; }
  .editor :global(.cm-image.moving) { opacity: .5; }
  .editor :global(.cm-image.focused img) { outline: 2px solid var(--blue); outline-offset: 2px; }
  .editor :global(.cm-image.focused) { touch-action: none; cursor: move; }
  .editor :global(.cm-image-move) { position: absolute; inset: .6rem auto auto .6rem; display: grid; place-items: center; width: 2rem; height: 2rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--bg); color: var(--blue); cursor: move; touch-action: none; }
  .editor :global(.cm-image-remove) { position: absolute; inset: .6rem .6rem auto auto; display: grid; place-items: center; width: 2rem; height: 2rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--bg); color: var(--red); cursor: pointer; }
  .editor :global(.cm-image img) { display: block; max-width: 100%; max-height: 28rem; border-radius: .4rem; }
  .editor :global(.cm-image-load) { padding: .35rem .7rem; border: 1px dashed var(--line); border-radius: .4rem; color: var(--muted); font-size: .85em; cursor: pointer; }
  .editor :global(.cm-image-load:hover) { border-color: var(--blue); color: var(--blue); }
  .editor :global(.cm-image.missing) { color: var(--muted); font-size: .85em; }
  .editor :global(.cm-highlight) { background: color-mix(in srgb, #facc15 38%, transparent); border-radius: 3px; }
  .editor :global(.cm-align-center) { text-align: center; }
  .editor :global(.cm-align-right) { text-align: end; }
  .editor :global(.cm-align-justify) { text-align: justify; }
  .editor :global(.cm-align-attr) { color: var(--muted); font-size: .8em; opacity: .7; }
  /* A misspelled word: highlighted in red, like a marker pen. */
  .editor :global(.cm-misspelled) { background: color-mix(in srgb, var(--red) 24%, transparent); border-radius: 3px; box-shadow: inset 0 -2px 0 color-mix(in srgb, var(--red) 70%, transparent); }
  .editor :global(.cm-completionIcon) { display: none; }
  .editor :global(.cm-tooltip-autocomplete > ul > li[aria-selected]::after) { content: "Tab"; float: inline-end; margin-inline-start: 1rem; padding: 0 .35rem; border: 1px solid var(--line); border-radius: .3rem; font-size: .7rem; color: var(--muted); }
  .editor :global(.cm-completionLabel) { font-weight: 500; }
  /* Tables drawn as tables (table.ts). */
  /* A grid: the − bars on top and on the left, the table, the + bars on
     the right and at the bottom; each bar as long as the table's side. */
  .editor :global(.cm-md-table) { display: inline-grid; grid-template-columns: 1.1rem minmax(0, auto) 1.1rem; grid-template-rows: 1.1rem auto 1.1rem; gap: .25rem; max-width: 100%; padding-block: .2rem; }
  .editor :global(.cm-md-table-inner) { grid-area: 2 / 2; min-width: 0; overflow-x: auto; }
  .editor :global(.cm-md-table table) { border-collapse: collapse; font-size: .95em; }
  .editor :global(.cm-md-table th), .editor :global(.cm-md-table td) { min-width: 6rem; padding: .4rem .7rem; border: 1px solid var(--line); vertical-align: top; outline: none; white-space: pre-wrap; }
  .editor :global(.cm-md-table th) { background: var(--soft); font-weight: 700; }
  .editor :global(.cm-md-table :is(th, td):focus) { box-shadow: inset 0 0 0 2px var(--blue); }
  .editor :global(.cm-md-table th:empty::before) { content: attr(data-placeholder); color: var(--muted); font-weight: 500; }
  .editor :global(.cm-md-bar) { display: grid; place-items: center; padding: 0; border-radius: .35rem; background: var(--soft); color: var(--muted); font-size: .85rem; font-weight: 700; line-height: 1; visibility: hidden; }
  .editor :global(.cm-md-bar.corner) { grid-area: 1 / 1; }
  .editor :global(.cm-md-bar.top) { grid-area: 1 / 2; }
  .editor :global(.cm-md-bar.left) { grid-area: 2 / 1; }
  .editor :global(.cm-md-bar.right) { grid-area: 2 / 3; }
  .editor :global(.cm-md-bar.bottom) { grid-area: 3 / 2; }
  /* No fade: the bars just show while the mouse or the cursor is in the
     table (a fade flickered in the Linux window). */
  .editor :global(.cm-md-table:hover .cm-md-bar), .editor :global(.cm-md-table:focus-within .cm-md-bar) { visibility: visible; }
  .editor :global(.cm-md-bar:hover:not(:disabled)) { background: var(--blue); color: #fff; }
  .editor :global(.cm-md-bar:is(.top, .left, .corner):hover:not(:disabled)) { background: var(--red); }
  .editor :global(.cm-md-bar:disabled) { opacity: .3; cursor: default; }
  :global(.cm-md-menu) { position: fixed; z-index: 70; display: grid; min-width: 13rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .28); }
  :global(.cm-md-menu button) { padding: .4rem .6rem; border-radius: .35rem; font-size: .85rem; text-align: start; }
  :global(.cm-md-menu button:hover:not(:disabled)) { background: var(--soft); }
  :global(.cm-md-menu button:disabled) { opacity: .4; }
  :global(.cm-md-menu button.danger) { color: var(--red); }
  :global(.cm-md-menu hr) { border: 0; border-block-start: 1px solid var(--line); margin: .25rem .2rem; }
  .fix { position: fixed; z-index: 50; display: grid; min-width: 13rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .28); }
  .fix button { padding: .4rem .6rem; border-radius: .35rem; font-size: .85rem; text-align: start; }
  .fix button:hover { background: var(--soft); }
  .fix .sug { font-weight: 600; }
  .fix small { padding: .4rem .6rem; font-size: .8rem; }
  .fix hr { border: 0; border-block-start: 1px solid var(--line); margin: .25rem .2rem; }
  .editor :global(.cm-obs-comment), .editor :global(.cm-obs-comment *) { color: var(--muted) !important; font-style: italic; }
  .editor :global(.cm-gutters) { display: none; }
  /* Links as their text (livemd.ts), in the links' purple. */
  .editor :global(.cm-link-text) { color: var(--blue); text-decoration: underline; text-underline-offset: 3px; cursor: pointer; }
  /* The chevron that folds a heading, in the margin at its left: on hover,
     or always when folded. */
  .editor :global(.cm-heading-line) { position: relative; }
  .editor :global(.cm-fold-chevron) { position: absolute; inset-inline-start: -1.35rem; inset-block-start: 50%; translate: 0 -50%; display: grid; place-items: center; width: 1.1rem; height: 1.1rem; border-radius: .3rem; color: var(--muted); cursor: pointer; opacity: 0; transition: opacity .1s; }
  .editor :global(.cm-heading-line:hover .cm-fold-chevron), .editor :global(.cm-fold-chevron.folded) { opacity: 1; }
  .editor :global(.cm-fold-chevron:hover) { background: var(--soft); color: var(--text); }
  .editor :global(.cm-fold-chevron.folded svg) { rotate: -90deg; }
  .editor :global(.cm-foldPlaceholder) { margin-inline-start: .4rem; padding: 0 .35rem; border: 1px solid var(--line); border-radius: .3rem; background: var(--soft); color: var(--muted); cursor: pointer; }
  .editor :global(.cm-activeLine) { background: transparent; }
  .editor :global(.cm-line) { color: var(--text); }
  /* CodeMirror draws its own cursor and selection: visible in dark too. */
  .editor :global(.cm-cursor), .editor :global(.cm-dropCursor) { border-left: 2px solid var(--text); margin-left: -1px; }
  .editor :global(.cm-selectionBackground), .editor :global(.cm-focused .cm-selectionBackground), .editor :global(.cm-content ::selection) { background: color-mix(in srgb, var(--blue) 30%, transparent) !important; }
  .editor :global(.cm-selectionMatch) { background: color-mix(in srgb, var(--blue) 15%, transparent); }
</style>
