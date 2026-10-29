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
  import { undo, redo, undoDepth, redoDepth } from "@codemirror/commands";
  import { undoHandlers } from "./undo";
  import { tables } from "./table";
  import Undo from "@lucide/svelte/icons/undo-2";
  import Redo from "@lucide/svelte/icons/redo-2";
  import { EditorSelection } from "@codemirror/state";
  import { wrap, prefix, block, link, insert, heading, align, alignAt, headingAt, ALIGN_LINE, callout } from "./mdcommands";
  import { CALLOUTS, calloutIcon } from "./callouts";
  import { mdExtras } from "./mdextras";
  import Highlighter from "@lucide/svelte/icons/highlighter";
  import AlignLeft from "@lucide/svelte/icons/text-align-start";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
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
  let mouseButton = 0;
  // The suggestions come first, then the corrections list (if open) gives
  // way to the menu in one go: no blink between the two.
  async function onMenu(e: MouseEvent) {
    const pos = view.posAtCoords({ x: e.clientX, y: e.clientY });
    const w = pos == null ? null : wordAt(view, pos);
    if (!w) { closeCompletion(view); return; } // the system's menu
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
      for (const el of host.querySelectorAll<HTMLElement>(".cm-md-table")) if ((el as any).cmWidget?.from <= at) best = el;
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
      ...CALLOUTS.map((c) => ({ label: c.type === "quote" ? te.quoteSource : t.lang === "pt" ? c.pt : c.en, svg: calloutIcon(c), tint: c.color,
        run: () => callout(view, c.type, calloutFold ? "-" : "", c.type === "quote" ? te.sourceTitle : t.lang === "pt" ? c.pt : c.en, te.calloutText) })),
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
          // A right-click on a word in red doesn't move the cursor (that
          // would close the corrections list before the menu shows).
          mousedown: (e, v) => {
            mouseButton = e.button;
            if (e.button !== 2) return false;
            const pos = v.posAtCoords({ x: e.clientX, y: e.clientY });
            if (pos == null || !wordAt(v, pos)) return false;
            e.preventDefault();
            return true;
          },
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
  /* Callouts in their type's color (mdextras.ts), like on the site. */
  .editor :global(.cm-callout) { background: rgb(var(--c) / .1); border-inline-start: 3px solid rgb(var(--c)); padding-inline-start: .8rem !important; }
  .editor :global(.cm-callout-head) { color: rgb(var(--c)); font-weight: 700; border-start-start-radius: .3rem; padding-block-start: .3rem; }
  .editor :global(.cm-callout-end) { padding-block-end: .3rem; border-end-start-radius: .3rem; }
  .editor :global(.cm-callout-head *) { color: rgb(var(--c)) !important; }
  .editor :global(.cm-callout :is(span)) { font-style: normal; }
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
  .editor :global(.cm-activeLine) { background: transparent; }
  .editor :global(.cm-line) { color: var(--text); }
  /* CodeMirror draws its own cursor and selection: visible in dark too. */
  .editor :global(.cm-cursor), .editor :global(.cm-dropCursor) { border-left: 2px solid var(--text); margin-left: -1px; }
  .editor :global(.cm-selectionBackground), .editor :global(.cm-focused .cm-selectionBackground), .editor :global(.cm-content ::selection) { background: color-mix(in srgb, var(--blue) 30%, transparent) !important; }
  .editor :global(.cm-selectionMatch) { background: color-mix(in srgb, var(--blue) 15%, transparent); }
</style>
