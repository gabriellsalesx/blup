<script lang="ts">
  import { onMount } from "svelte";
  import { EditorView, basicSetup } from "codemirror";
  import { keymap, Decoration, type DecorationSet } from "@codemirror/view";
  import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
  import { languages } from "@codemirror/language-data";
  import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
  import { tags } from "@lezer/highlight";
  import { StateField, RangeSetBuilder, type EditorState } from "@codemirror/state";
  import { look, setPref } from "./theme.svelte";
  import { spellcheck, wordAt, suggest, addWord, complete, skipAt } from "./spell";
  import { autocompletion, startCompletion, acceptCompletion, completionStatus, closeCompletion, moveCompletionSelection, type CompletionContext, type Completion } from "@codemirror/autocomplete";
  import { Prec } from "@codemirror/state";
  import { undo, redo } from "@codemirror/commands";
  import { undoHandlers } from "./undo";
  import { tables } from "./table";
  import Undo from "@lucide/svelte/icons/undo-2";
  import Redo from "@lucide/svelte/icons/redo-2";
  import { EditorSelection } from "@codemirror/state";
  import { wrap, prefix, block, link, insert } from "./mdcommands";
  import { PasteImage, PickNoteImage } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
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
  let { text, note, notes = [], onChange, onSave, onError }: {
    text: string; note: string; notes?: { name: string; path: string }[];
    onChange: (v: string) => void; onSave: () => void; onError?: (e: string) => void;
  } = $props();
  const te = t.editor;
  let host: HTMLDivElement;
  // Right-click on an underlined word: the dictionary's suggestions.
  let fix = $state<{ from: number; to: number; word: string; x: number; y: number; list: string[] | null } | null>(null);
  async function onMenu(e: MouseEvent) {
    const pos = view.posAtCoords({ x: e.clientX, y: e.clientY });
    const w = pos == null ? null : wordAt(view, pos);
    if (!w) return; // the system's menu
    e.preventDefault();
    fix = { ...w, x: Math.min(e.clientX, innerWidth - 240), y: Math.min(e.clientY, innerHeight - 260), list: null };
    const list = await suggest(w.word);
    if (fix?.from === w.from) fix.list = list;
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

  // [[ lists the notes, like Obsidian; the link takes the note's name (its
  // path when two notes share it).
  function noteLinks(ctx: CompletionContext) {
    const m = ctx.matchBefore(/\[\[[^\]\n|#]*/);
    if (!m) return null;
    const count = new Map<string, number>();
    for (const n of notes) count.set(n.name, (count.get(n.name) ?? 0) + 1);
    const options = notes.filter((n) => n.name !== "_index").map((n) => {
      const target = (count.get(n.name) ?? 0) > 1 ? n.path.replace(/\.md$/, "") : n.name;
      const folder = n.path.includes("/") ? n.path.slice(0, n.path.lastIndexOf("/")) : "";
      return {
        label: n.name, detail: folder, type: "text",
        apply: (v: EditorView, _c: unknown, from: number, to: number) => {
          const close = v.state.sliceDoc(to, to + 2) === "]]" ? 2 : v.state.sliceDoc(to, to + 1) === "]" ? 1 : 0;
          v.dispatch({ changes: { from, to: to + close, insert: target + "]]" }, selection: EditorSelection.cursor(from + target.length + 2) });
        },
      };
    });
    return { from: m.from + 2, options, validFor: /^[^\]\n|#]*$/ };
  }

  // While a word is typed (3 letters or more), like VS Code: corrections
  // when it is misspelled, then words of this note and of the dictionary
  // that start with it. Tab puts the chosen one in place.
  async function wordSuggestions(ctx: CompletionContext) {
    if (ctx.view && wordAt(ctx.view, ctx.pos)) return null; // fixSuggestions
    const m = ctx.matchBefore(/\p{L}[\p{L}\p{M}'’-]*/u);
    if (!m || m.to - m.from < 3 || !look.spellcheck || skipAt(ctx.view!, ctx.pos)) return null;
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
      } catch (e) {
        onError?.(String(e).replace(/^(Runtime)?Error: /, ""));
      }
    }
  }
  const images = (list: FileList | undefined | null) => [...(list ?? [])].filter((f) => f.type.startsWith("image/"));
  async function pickImage() {
    try {
      const md = await PickNoteImage(note);
      if (md) insert(view, md + "\n", true);
    } catch (e) { onError?.(String(e).replace(/^(Runtime)?Error: /, "")); }
  }

  // A new table, with the cursor in its first empty cell.
  function newTable() {
    block(view, `| ${te.col} 1 | ${te.col} 2 |\n| --- | --- |\n|   |   |`);
    const at = view.state.selection.main.head;
    setTimeout(() => {
      let best: HTMLElement | null = null;
      for (const el of host.querySelectorAll<HTMLElement>(".cm-md-table")) if ((el as any).cmWidget?.from <= at) best = el;
      best?.querySelector<HTMLElement>('[data-r="1"][data-c="0"]')?.focus();
    }, 30);
  }

  // The formatting bar.
  const H = /^#{1,6} /, LIST = /^(\s*)([-*+] \[[ x]\] |[-*+] |\d+\. )/, QUOTE = /^> ?/;
  const tools = [
    { key: "undo", label: te.undo + " (Ctrl+Z)", Icon: Undo, run: () => { undo(view); view.focus(); } },
    { key: "redo", label: te.redo + " (Ctrl+Shift+Z)", Icon: Redo, run: () => { redo(view); view.focus(); } },
    { key: "sep0" },
    { key: "heading", label: te.heading, Icon: Heading, run: () => prefix(view, "## ", H) },
    { key: "bold", label: te.bold + " (Ctrl+B)", Icon: Bold, run: () => wrap(view, "**", te.boldText) },
    { key: "italic", label: te.italic + " (Ctrl+I)", Icon: Italic, run: () => wrap(view, "*", te.italicText) },
    { key: "strike", label: te.strike, Icon: Strike, run: () => wrap(view, "~~", te.strikeText) },
    { key: "sep1" },
    { key: "list", label: te.list, Icon: List, run: () => prefix(view, "- ", LIST) },
    { key: "olist", label: te.olist, Icon: ListOrdered, run: () => prefix(view, (i) => `${i + 1}. `, LIST) },
    { key: "task", label: te.task, Icon: ListChecks, run: () => prefix(view, "- [ ] ", LIST) },
    { key: "quote", label: te.quote, Icon: Quote, run: () => prefix(view, "> ", QUOTE) },
    { key: "callout", label: te.callout, Icon: Info, run: () => block(view, `> [!tip] ${te.calloutTitle}\n> ${te.calloutText}`, [9, 9 + te.calloutTitle.length]) },
    { key: "sep2" },
    { key: "link", label: te.link + " (Ctrl+K)", Icon: Link, run: () => link(view, te.linkText) },
    { key: "image", label: te.image, Icon: ImageIcon, run: pickImage },
    { key: "code", label: te.code, Icon: Code, run: () => wrap(view, "`", te.codeText) },
    { key: "table", label: te.table, Icon: Table, run: newTable },
    { key: "hr", label: te.hr, Icon: Minus, run: () => block(view, "---") },
  ];

  onMount(() => {
    view = new EditorView({
      parent: host,
      doc: text,
      extensions: [
        basicSetup,
        markdown({ base: markdownLanguage, codeLanguages: languages }),
        syntaxHighlighting(markdownStyle),
        EditorView.lineWrapping,
        commentField,
        tables(t.editor.tableUi),
        EditorView.contentAttributes.of({ spellcheck: "false" }), // ours, below
        spellcheck,
        keymap.of([
          { key: "Mod-s", preventDefault: true, run: () => (onSave(), true) },
          { key: "Mod-b", preventDefault: true, run: (v) => (wrap(v, "**", te.boldText), true) },
          { key: "Mod-i", preventDefault: true, run: (v) => (wrap(v, "*", te.italicText), true) },
          { key: "Mod-k", preventDefault: true, run: (v) => (link(v, te.linkText), true) },
        ]),
        autocompletion({ override: [noteLinks, fixSuggestions, wordSuggestions], icons: false, defaultKeymap: false, activateOnTypingDelay: 150 }),
        // Tab takes the suggestion; Enter stays a new line.
        Prec.highest(keymap.of([
          { key: "Tab", run: (v) => (completionStatus(v.state) === "active" ? acceptCompletion(v) : false) },
          { key: "ArrowDown", run: (v) => (completionStatus(v.state) === "active" ? moveCompletionSelection(true)(v) : false) },
          { key: "ArrowUp", run: (v) => (completionStatus(v.state) === "active" ? moveCompletionSelection(false)(v) : false) },
          { key: "Escape", run: (v) => (completionStatus(v.state) === "active" ? closeCompletion(v) : false) },
        ])),
        EditorView.domEventHandlers({
          paste: (e) => {
            const files = images(e.clipboardData?.files);
            if (!files.length) return false;
            e.preventDefault();
            addImages(files);
            return true;
          },
          drop: (e, v) => {
            const files = images(e.dataTransfer?.files);
            if (!files.length) return false;
            e.preventDefault();
            addImages(files, v.posAtCoords({ x: e.clientX, y: e.clientY }) ?? undefined);
            return true;
          },
        }),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) onChange(u.state.doc.toString());
          // A click on a word in red opens its corrections.
          if (!u.docChanged && u.transactions.some((tr) => tr.isUserEvent("select.pointer"))) {
            const sel = u.state.selection.main;
            if (sel.empty && wordAt(u.view, sel.head)) setTimeout(() => startCompletion(u.view), 0);
          }
        }),
      ],
    });
    view.focus();
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

<svelte:window onclick={() => (fix = null)} onkeydown={(e) => e.key === "Escape" && (fix = null)} />

<div class="wrap">
  <div class="bar" role="toolbar" aria-label={te.toolbar}>
    {#each tools as tool (tool.key)}
      {#if tool.Icon}
        <button title={tool.label} aria-label={tool.label} onmousedown={(e) => e.preventDefault()} onclick={tool.run}><tool.Icon size={16} strokeWidth={1.9} /></button>
      {:else}<span class="sep"></span>{/if}
    {/each}
  </div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="editor" oncontextmenu={onMenu} class:wide={look.editorWidth === "wide"} style:--size="{look.editorFont}px" bind:this={host}></div>
</div>

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
  .bar button:hover { background: var(--soft); color: var(--text); }
  .bar .sep { width: 1px; height: 1.1rem; margin-inline: .3rem; background: var(--line); }
  .editor { flex: 1; min-height: 0; overflow: hidden; }
  .editor :global(.cm-tooltip-autocomplete) { border: 1px solid var(--line); border-radius: .5rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .25); overflow: hidden; }
  .editor :global(.cm-tooltip-autocomplete > ul) { font-family: inherit !important; font-size: .875rem; max-height: 16rem !important; }
  .editor :global(.cm-tooltip-autocomplete > ul > li) { padding: .35rem .7rem !important; color: var(--text); }
  .editor :global(.cm-tooltip-autocomplete > ul > li[aria-selected]) { background: var(--soft) !important; color: var(--text) !important; }
  .editor :global(.cm-completionDetail) { margin-inline-start: .6rem; color: var(--muted); font-style: normal; font-size: .78rem; }
  .editor :global(.cm-completionMatchedText) { text-decoration: none; font-weight: 700; color: var(--blue); }
  .editor :global(.cm-editor) { height: 100%; background: var(--panel); font-size: var(--size, 1rem); }
  .editor :global(.cm-editor.cm-focused) { outline: none; }
  .editor :global(.cm-scroller) { font-family: "Inter", ui-sans-serif, system-ui, sans-serif; line-height: 1.7; padding: 2rem 0; }
  .editor :global(.cm-content) { max-width: 46rem; margin-inline: auto; padding-inline: 2rem; caret-color: var(--text); }
  .editor.wide :global(.cm-content) { max-width: none; }
  .editor :global(.cm-obs-comment) { opacity: .75; }
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
  .editor :global(.cm-md-bar) { display: grid; place-items: center; padding: 0; border-radius: .35rem; background: var(--soft); color: var(--muted); font-size: .85rem; font-weight: 700; line-height: 1; opacity: 0; transition: opacity .12s, background .12s; }
  .editor :global(.cm-md-bar.top) { grid-area: 1 / 2; }
  .editor :global(.cm-md-bar.left) { grid-area: 2 / 1; }
  .editor :global(.cm-md-bar.right) { grid-area: 2 / 3; }
  .editor :global(.cm-md-bar.bottom) { grid-area: 3 / 2; }
  .editor :global(.cm-md-table:hover .cm-md-bar), .editor :global(.cm-md-table:focus-within .cm-md-bar) { opacity: 1; }
  .editor :global(.cm-md-bar:hover:not(:disabled)) { background: var(--blue); color: #fff; }
  .editor :global(.cm-md-bar.top:hover:not(:disabled)), .editor :global(.cm-md-bar.left:hover:not(:disabled)) { background: var(--red); }
  .editor :global(.cm-md-table:hover .cm-md-bar:disabled) { opacity: .3; cursor: default; }
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
  .editor :global(.cm-activeLine) { background: transparent; }
  .editor :global(.cm-line) { color: var(--text); }
  /* CodeMirror draws its own cursor and selection: visible in dark too. */
  .editor :global(.cm-cursor), .editor :global(.cm-dropCursor) { border-left: 2px solid var(--text); margin-left: -1px; }
  .editor :global(.cm-selectionBackground), .editor :global(.cm-focused .cm-selectionBackground), .editor :global(.cm-content ::selection) { background: color-mix(in srgb, var(--blue) 30%, transparent) !important; }
  .editor :global(.cm-selectionMatch) { background: color-mix(in srgb, var(--blue) 15%, transparent); }
</style>
