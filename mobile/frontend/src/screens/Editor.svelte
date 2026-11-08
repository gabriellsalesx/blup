<script lang="ts">
  import { onDestroy, onMount, untrack } from "svelte";
  import { EditorView, basicSetup } from "codemirror";
  import { tooltips } from "@codemirror/view";
  import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
  import { languages } from "@codemirror/language-data";
  import { syntaxHighlighting } from "@codemirror/language";
  import { undo, redo, undoDepth, redoDepth } from "@codemirror/commands";
  import ChevronLeft from "@lucide/svelte/icons/chevron-left";
  import CalendarClock from "@lucide/svelte/icons/calendar-clock";
  import CalendarPlus from "@lucide/svelte/icons/calendar-plus";
  import { fade } from "svelte/transition";
  import Check from "@lucide/svelte/icons/check";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import Cloud from "@lucide/svelte/icons/cloud";
  import Ellipsis from "@lucide/svelte/icons/ellipsis";
  import Folder from "@lucide/svelte/icons/folder";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Smartphone from "@lucide/svelte/icons/smartphone";
  import Table from "@lucide/svelte/icons/table";
  import Tag from "@lucide/svelte/icons/tag";
  import Bold from "@lucide/svelte/icons/bold";
  import Italic from "@lucide/svelte/icons/italic";
  import Heading from "@lucide/svelte/icons/heading";
  import Highlighter from "@lucide/svelte/icons/highlighter";
  import Code from "@lucide/svelte/icons/code";
  import ListOrdered from "@lucide/svelte/icons/list-ordered";
  import SquareCode from "@lucide/svelte/icons/square-code";
  import Strikethrough from "@lucide/svelte/icons/strikethrough";
  import ImagePlus from "@lucide/svelte/icons/image-plus";
  import Info from "@lucide/svelte/icons/info";
  import List from "@lucide/svelte/icons/list";
  import SquareCheckBig from "@lucide/svelte/icons/square-check-big";
  import Link2 from "@lucide/svelte/icons/link-2";
  import Undo2 from "@lucide/svelte/icons/undo-2";
  import Redo2 from "@lucide/svelte/icons/redo-2";
  import { NotesService, type Note } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message } from "../lib/app.svelte";
  import FolderPick from "../ui/FolderPick.svelte";
  import MentionedIn from "../ui/MentionedIn.svelte";
  import NotePanel from "../ui/NotePanel.svelte";
  import TagPick from "../ui/TagPick.svelte";
  import { markdownStyle } from "../editor/style";
  import { mdExtras } from "../editor/mdextras";
  import { linkConceal, headingFolds } from "../editor/livemd";
  import { forgetImages, livePreview } from "../editor/livepreview";
  import { prepareImage } from "../editor/pickimage";
  import { blockAt, dropPoint, moveImage, removeImage, scrollNear, type ImageSpan } from "../editor/imagedrag";
  import { wrap, prefix, heading, headingAt, insert, block, callout } from "../editor/mdcommands";
  import { CALLOUTS, calloutIcon, calloutName } from "../editor/callouts";
  import { mentions } from "../editor/mentions";
  import { targetAt } from "../editor/notelinks";
  import { tables, tableFrom } from "../editor/table";
  import "../editor/cm.css";
  import { fmt, shown, t } from "../desk/i18n";

  const LIST = /^(\s*)([-*+] \[[ x]\] |[-*+] |\d+\. )/;

  let { note }: { note: Note } = $props();

  // The note is read once: from here on the text being typed is the truth.
  let id = $state(untrack(() => note.id));
  // The properties stay out of the text: they show above it and change in
  // the side panel.
  const headOf = (text: string) => /^---\r?\n[\s\S]*?\r?\n---\r?\n?/.exec(text)?.[0] ?? "";
  let head = ""; // read with the text, in onMount
  // The name is the file's, as in the Studio.
  let name = $state(untrack(() => note.title));
  let title = untrack(() => note.title); // the file's, as last saved
  const fresh = untrack(() => !!app.place.fresh);
  let titleField = $state<HTMLInputElement>();
  let gone = false;
  let deleted = false;
  let body = $state("");
  let saved = $state("");
  let canUndo = $state(false);
  let canRedo = $state(false);
  let host = $state<HTMLDivElement>();
  let view: EditorView; // made once the text has come (start)
  let timer: ReturnType<typeof setTimeout>;

  async function save() {
    clearTimeout(timer);
    if (body === saved) return;
    const text = body;
    head = headOf(await NotesService.Save(id, head + text));
    saved = text;
    await app.refreshNotes(); // a pause in the typing: only the notes change
  }

  async function rename() {
    const next = name.trim();
    if (!next) name = title;
    if (!next || next === title) return;
    await save();
    try {
      const to = await NotesService.Rename(id, next);
      id = to;
      title = next;
      if (gone) await app.refresh();
      else await app.moved(to);
    } catch (e) {
      name = title;
      app.say({ text: message(e) });
    }
  }
  async function setProps(created: string, updated: string, tags: string[]) {
    try {
      await save();
      head = headOf(await NotesService.SetProps(id, created, updated, tags));
      await app.refresh();
    } catch (e) { app.say({ text: message(e) }); }
  }
  async function move(folder: string) {
    try {
      await save();
      if (folder !== note.folder) {
        id = await NotesService.Move(id, folder);
        await app.moved(id);
        app.sync();
      }
      app.back();
    } catch (e) { app.say({ text: message(e) }); }
  }
  async function newFolder(parent: string, name: string) {
    try {
      await move(await NotesService.CreateFolder(parent, name));
    } catch (e) { app.say({ text: message(e) }); }
  }
  // What the icon in the top bar stands for, on a tap. The Cloud gets the note
  // when the person leaves it, so "saved" there has to be true by then.
  let info = $state(false);
  let infoTimer: ReturnType<typeof setTimeout>;
  function showInfo() {
    info = !info;
    clearTimeout(infoTimer);
    if (info) infoTimer = setTimeout(() => (info = false), 4000);
  }
  const cloudOn = $derived(!!app.account?.vault && app.account.notes === "ready");
  const cloudText = $derived(
    app.syncing ? t.note.toCloud
    : app.syncError ? t.note.notYet
    : body === saved && app.lastSync >= note.updated ? t.note.savedCloud
    : t.note.onLeave);

  const more = $derived((note.tags?.length ?? 0) - 3); // tags beyond the ones shown

  // A day of the note's properties (2006-01-02), short, by the language.
  const shortDate = (d: string) => {
    if (!d) return t.note.noDate;
    const [y, m, day] = d.split("-").map(Number);
    return new Date(y, m - 1, day).toLocaleDateString(t.locale, { day: "2-digit", month: "2-digit", year: "numeric" });
  };

  function linkAt(pos: number) {
    const line = view.state.doc.lineAt(pos);
    const refs = app.notes.map((n) => ({ name: n.id.split("/").pop() ?? n.id, path: n.id }));
    return targetAt(line.text, pos - line.from, id, refs);
  }

  // The text comes from Go when the note opens: the list of notes carries none.
  onMount(() => {
    let closed = false;
    NotesService.Read(id).then((text) => {
      if (closed) return;
      head = headOf(text);
      body = saved = text.slice(head.length);
      start();
    }, (e) => app.say({ text: message(e) }));
    return () => {
      closed = true;
      if (!view) return; // closed before the text came
      leaveCell();
      view.destroy();
    };
  });

  function start() {
    forgetImages();
    view = new EditorView({
      parent: host,
      doc: body,
      extensions: [
        basicSetup,
        mentions(() => app.notes.map((n) => ({ name: n.title, path: n.id })), () => id),
        // The keyboard covers the window instead of shrinking it: the @ menu
        // has to fit above it.
        tooltips({ tooltipSpace: () => ({ top: 0, left: 0, right: innerWidth, bottom: document.querySelector("main")?.getBoundingClientRect().bottom ?? innerHeight }) }),
        markdown({ base: markdownLanguage, codeLanguages: languages }),
        syntaxHighlighting(markdownStyle),
        EditorView.lineWrapping,
        mdExtras,
        tables({ ...t.editor.tableUi }),
        linkConceal,
        headingFolds({ fold: t.editor.fold, unfold: t.editor.unfold }),
        livePreview({
          text: { copy: t.editor.copy, copied: t.editor.copied, code: t.editor.code, lang: t.note.codeLang, remote: t.note.loadImage, moveImage: t.editor.moveImage, removeImage: t.editor.removeImage },
          image: (src) => NotesService.Image(id, src),
          onImageDrag: (span) => { moving = span; if (span) NotesService.HapticMedium(); else dropLine = null; },
          onImagePoint: (x, y) => { scrollNear(view.contentDOM, y); showLine(x, y); },
          onImageDrop: landImage,
          onImageRemove: dropOut,
        }),
        EditorView.contentAttributes.of({ autocapitalize: "sentences", spellcheck: "true" }),
        EditorView.domEventHandlers({
          // A tap on a link opens it, instead of opening the keyboard.
          click: (e, v) => {
            const pos = v.posAtCoords({ x: e.clientX, y: e.clientY });
            // Below the text the nearest position is the last line's: not a link.
            const box = pos == null ? null : v.coordsAtPos(pos);
            const target = pos == null || !box || e.clientY > box.bottom ? null : linkAt(pos);
            if (!target) return false;
            e.preventDefault();
            v.contentDOM.blur();
            if ("note" in target) {
              if (target.note !== id) app.open(target.note);
            } else NotesService.OpenURL(target.url);
            return true;
          },
        }),
        EditorView.updateListener.of((u) => {
          canUndo = undoDepth(u.state) > 0;
          canRedo = redoDepth(u.state) > 0;
          level = headingAt(u.state, u.state.selection.main.head);
          if (u.selectionSet || u.docChanged) menu = "";
          if (!u.docChanged) return;
          body = u.state.doc.toString();
          clearTimeout(timer);
          timer = setTimeout(save, 500);
        }),
      ],
    });
    if (fresh) {
      titleField?.focus();
      titleField?.select();
    } else if (app.place.edit) {
      view.focus();
      view.dispatch({ selection: { anchor: body.length }, scrollIntoView: true });
    }
  }

  // The toolbar's two lists: they open above it, over the note.
  let menu = $state<"" | "heading" | "callout">("");
  let level = $state(0); // the heading at the cursor
  const toggle = (name: "heading" | "callout") => (menu = menu === name ? "" : name);
  function choose(run: () => void) {
    menu = "";
    run();
  }
  // A new table, with the cursor in its first empty cell, as in the Studio.
  function newTable() {
    menu = "";
    block(view, `| ${t.editor.col} 1 | ${t.editor.col} 2 |\n| --- | --- |\n|   |   |`);
    const at = view.state.selection.main.head;
    setTimeout(() => {
      let best: HTMLElement | null = null;
      for (const el of host?.querySelectorAll<HTMLElement>(".cm-md-table") ?? []) {
        const from = tableFrom(el);
        if (from !== undefined && from <= at) best = el;
      }
      const cell = best?.querySelector<HTMLElement>('[data-r="1"][data-c="0"]');
      cell?.focus();
      cell?.scrollIntoView({ block: "center" });
    }, 30);
  }

  // A picture from the phone goes on a block of its own, and the cursor to
  // the line after it (on its line, the Markdown would show in its place).
  let picker = $state<HTMLInputElement>();
  async function addImage() {
    const file = picker?.files?.[0];
    if (picker) picker.value = "";
    if (!file) return;
    try {
      const { name, data } = await prepareImage(file);
      block(view, await NotesService.AddImage(id, name, data));
      const next = view.state.doc.lineAt(view.state.selection.main.head + 1);
      view.dispatch({ changes: next.text.trim() ? { from: next.from, insert: "\n" } : undefined, selection: { anchor: next.from }, scrollIntoView: true });
      view.focus();
    } catch (e) {
      app.say({ text: e instanceof Error && e.message !== "image" ? message(e) : t.note.imageFail });
    }
  }

  // An image moves by holding one of the corners its focus shows (see
  // livepreview.ts): the blue line tells where it would land.
  let moving: ImageSpan | null = null;
  let dropLine = $state<{ x: number; y: number; width: number } | null>(null);
  function landing(x: number, y: number) {
    const pos = view.posAtCoords({ x, y }, false);
    const b = blockAt(view.state, pos);
    const top = view.lineBlockAt(b?.from ?? pos).top, bottom = view.lineBlockAt(b?.to ?? pos).bottom;
    const at = dropPoint(view.state, pos, y - view.documentTop > (top + bottom) / 2);
    return { at, y: view.documentTop + view.lineBlockAt(at)[at === view.state.doc.lineAt(at).from ? "top" : "bottom"] };
  }
  function showLine(x: number, y: number) {
    const r = view.contentDOM.getBoundingClientRect();
    dropLine = { x: r.left, y: landing(x, y).y, width: r.width };
  }
  function landImage(x: number, y: number) {
    if (!moving) return;
    const changes = moveImage(view.state, moving, landing(x, y).at);
    if (changes) view.dispatch({ changes, userEvent: "move.drop" });
    moving = dropLine = null;
  }
  function dropOut(span: ImageSpan) {
    view.dispatch({ changes: removeImage(view.state, span), userEvent: "delete" });
    app.say({ text: t.editor.imageRemoved, action: { label: t.editor.undo, run: () => undo(view) } });
  }

  // A table cell keeps what is typed until it loses the focus.
  function leaveCell() {
    const cell = document.activeElement as HTMLElement | null;
    if (cell?.closest(".cm-md-table")) cell.blur();
  }

  // Keeps the keyboard open: the button never takes the focus from the text.
  const hold = (e: Event) => e.preventDefault();

  async function remove() {
    await save(); // what was just typed goes to the trash too
    deleted = true;
    await app.trashNote(id);
    history.go(-2);
  }

  // Leaving an untouched new note does not keep an empty file.
  onDestroy(async () => {
    leaveCell();
    gone = true;
    clearTimeout(timer);
    if (deleted) return;
    if (fresh && !body.trim() && name.trim() === title) {
      await NotesService.Delete(id);
      await app.refresh();
      return;
    }
    await rename();
    await save();
    app.sync(); // the Cloud gets the note as the person leaves it, whenever the last sync was
  });
</script>

<header>
  <button aria-label={t.common.back} onclick={() => app.back()}><ChevronLeft size={26} /></button>
  <span class="gap"></span>
  <button class="status" aria-label={body === saved ? t.note.saved : t.note.saving} onpointerdown={hold} onmousedown={hold} onclick={showInfo}>
    {#if body === saved}
      <span class="state saved"><Check size={12} strokeWidth={3.5} /></span>
    {:else}
      <span class="state saving"><RefreshCw size={11} strokeWidth={3} /></span>
    {/if}
  </button>
  <button aria-label={t.note.about} onclick={() => app.go({ ...app.place, sheet: true })}><Ellipsis size={24} /></button>
  {#if info}
    <div class="info" role="status" transition:fade={{ duration: 120 }}>
      <p><Smartphone size={16} />{body === saved ? t.note.savedHere : t.note.savingHere}</p>
      {#if cloudOn}<p><Cloud size={16} />{cloudText}</p>{/if}
    </div>
  {/if}
</header>

<div class="page">
  <div class="chips">
    <button type="button" class="chip" aria-label={fmt(t.note.folder, { name: note.folder || t.folderPick.none })} onclick={() => app.go({ ...app.place, pick: "folder" })}><Folder size={15} /><span>{note.folder || t.folderPick.none}</span><ChevronDown size={15} /></button>
    {#each (note.tags ?? []).slice(0, 3) as tag (tag)}<button type="button" class="chip tag" onclick={() => app.go({ ...app.place, pick: "tags" })}><span>{tag}</span></button>{/each}
    {#if more > 0}<button type="button" class="chip tag" aria-label={fmt(t.note.moreTags, { n: more })} onclick={() => app.go({ ...app.place, pick: "tags" })}><span>+{more}</span></button>{/if}
    <button type="button" class="chip" aria-label={t.note.pickTags} onclick={() => app.go({ ...app.place, pick: "tags" })}><Tag size={15} />{#if !note.tags?.length}<span>{t.noteProps.addTag}</span>{/if}</button>
  </div>
  <input bind:this={titleField} class="title" type="text" bind:value={name} aria-label={t.note.name} spellcheck="false" enterkeyhint="next"
    onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); view.focus(); } }} onblur={rename} />
  <button type="button" class="props" aria-label={t.note.props} onclick={() => app.go({ ...app.place, sheet: true })}>
    <span><CalendarPlus size={15} />{t.noteProps.created}<b>{shortDate(note.created)}</b></span>
    <span><CalendarClock size={15} />{t.noteProps.updated}<b>{shortDate(note.changed)}</b></span>
  </button>
  <div class="editor" bind:this={host}></div>
  <MentionedIn {id} />
</div>

{#if app.keyboard}
  <div class="bar">
    {#if menu === "heading"}
      <div class="pop" role="menu" aria-label={t.note.headings}>
        <button role="menuitem" onpointerdown={hold} onmousedown={hold} onclick={() => choose(() => heading(view, 0))}><span class="mark">¶</span>{t.editor.normal}{#if !level}<i><Check size={18} /></i>{/if}</button>
        {#each [1, 2, 3, 4] as n (n)}
          <button role="menuitem" style:font-size={["", "1.35em", "1.2em", "1.08em", "1em"][n]} onpointerdown={hold} onmousedown={hold} onclick={() => choose(() => heading(view, n))}><span class="mark">H{n}</span>{fmt(t.editor.headingN, { n })}{#if level === n}<i><Check size={18} /></i>{/if}</button>
        {/each}
      </div>
    {:else if menu === "callout"}
      <div class="pop" role="menu" aria-label={t.note.callouts}>
        {#each CALLOUTS as c (c.type)}
          <button role="menuitem" onpointerdown={hold} onmousedown={hold} onclick={() => choose(() => callout(view, c.type, "", c.type === "quote" ? t.editor.sourceTitle : calloutName(c, shown.lang), t.editor.calloutText))}>
            <span class="mark">{@html calloutIcon(c, 20)}</span>{c.type === "quote" ? t.editor.quoteSource : calloutName(c, shown.lang)}
          </button>
        {/each}
      </div>
    {/if}
    <div class="tools" role="toolbar" aria-label={t.editor.toolbar}>
      <div class="scroll">
        <button aria-label={t.note.headings} aria-expanded={menu === "heading"} class={{ on: menu === "heading" }} onpointerdown={hold} onmousedown={hold} onclick={() => toggle("heading")}>{#if level}<b>H{level}</b>{:else}<Heading size={21} />{/if}</button>
        <button aria-label={t.editor.bold} onpointerdown={hold} onmousedown={hold} onclick={() => wrap(view, "**")}><Bold size={21} /></button>
        <button aria-label={t.editor.italic} onpointerdown={hold} onmousedown={hold} onclick={() => wrap(view, "*")}><Italic size={21} /></button>
        <button aria-label={t.editor.strike} onpointerdown={hold} onmousedown={hold} onclick={() => wrap(view, "~~")}><Strikethrough size={21} /></button>
        <button aria-label={t.editor.highlight} onpointerdown={hold} onmousedown={hold} onclick={() => wrap(view, "==")}><Highlighter size={21} /></button>
        <button aria-label={t.editor.list} onpointerdown={hold} onmousedown={hold} onclick={() => prefix(view, "- ", LIST)}><List size={21} /></button>
        <button aria-label={t.editor.olist} onpointerdown={hold} onmousedown={hold} onclick={() => prefix(view, (i) => `${i + 1}. `, LIST)}><ListOrdered size={21} /></button>
        <button aria-label={t.editor.task} onpointerdown={hold} onmousedown={hold} onclick={() => prefix(view, "- [ ] ", LIST)}><SquareCheckBig size={21} /></button>
        <button aria-label={t.editor.table} onpointerdown={hold} onmousedown={hold} onclick={newTable}><Table size={21} /></button>
        <button aria-label={t.editor.code} onpointerdown={hold} onmousedown={hold} onclick={() => wrap(view, "`")}><Code size={21} /></button>
        <button aria-label={t.editor.codeBlock} onpointerdown={hold} onmousedown={hold} onclick={() => block(view, "```\n\n```", [4, 4])}><SquareCode size={21} /></button>
        <button aria-label={t.note.image} onpointerdown={hold} onmousedown={hold} onclick={() => { menu = ""; picker?.click(); }}><ImagePlus size={21} /></button>
        <button aria-label={t.note.callout} aria-expanded={menu === "callout"} class={{ on: menu === "callout" }} onpointerdown={hold} onmousedown={hold} onclick={() => toggle("callout")}><Info size={21} /></button>
        <button aria-label={t.editor.linkNote} onpointerdown={hold} onmousedown={hold} onclick={() => insert(view, "[[]]")}><Link2 size={21} /></button>
      </div>
      <button aria-label={t.editor.undo} aria-disabled={!canUndo} onpointerdown={hold} onmousedown={hold} onclick={() => undo(view)}><Undo2 size={21} /></button>
      <button aria-label={t.editor.redo} aria-disabled={!canRedo} onpointerdown={hold} onmousedown={hold} onclick={() => redo(view)}><Redo2 size={21} /></button>
    </div>
  </div>
{/if}

<input bind:this={picker} type="file" accept="image/*" hidden onchange={addImage} />

{#if dropLine}<div class="drop-line" style:left="{dropLine.x}px" style:top="{dropLine.y}px" style:width="{dropLine.width}px"></div>{/if}
{#if app.place.sheet}
  <NotePanel {note} onprops={setProps} onremove={remove} />
{/if}
{#if app.place.pick === "folder"}
  <FolderPick current={note.folder} onpick={move} oncreate={newFolder} />
{:else if app.place.pick === "tags"}
  <TagPick {note} onchange={(tags) => setProps(note.created, note.changed, tags)} />
{/if}

<style>
  header { position: relative; flex: none; display: flex; align-items: center; gap: .25rem; padding: calc(.25rem + var(--sat)) .25rem .25rem; border-block-end: 1px solid var(--line); }
  header button { display: grid; place-items: center; width: 2.9rem; height: 2.9rem; }
  .gap { flex: 1; }
  .state { flex: none; display: grid; place-items: center; width: 1.2rem; height: 1.2rem; border-radius: 50%; }
  .saved { background: #16a34a; color: #fff; }
  .saving { background: var(--soft); color: var(--text); }
  .saving :global(svg) { animation: turn 1s linear infinite; }
  @keyframes turn { to { transform: rotate(1turn); } }
  .page { flex: 1; min-height: 0; overflow-y: auto; display: flex; flex-direction: column; }
  .chips { flex: none; display: flex; flex-wrap: wrap; gap: .4rem; padding: .9rem 1.5rem 0; }
  .chip { display: inline-flex; align-items: center; gap: .35rem; max-width: 100%; min-height: 2.1rem; padding-inline: .6rem; border: 1px solid var(--line); border-radius: .6rem; color: var(--muted); font-size: .85rem; font-weight: 500; }
  .chip { max-width: 60%; }
  .chip.tag { border-color: transparent; background: color-mix(in srgb, var(--blue) 14%, transparent); }
  .chip.tag span { color: var(--blue-text); }
  .info { position: absolute; inset-block-start: calc(100% - .25rem); inset-inline-end: .75rem; z-index: 5; display: grid; gap: .4rem; padding: .6rem .8rem; border: 1px solid var(--line); border-radius: .7rem; background: var(--panel); box-shadow: 0 6px 16px rgb(0 0 0 / .2); font-size: .85rem; font-weight: 500; }
  .info p { display: flex; align-items: center; gap: .45rem; }
  .info :global(svg) { color: var(--blue-text); }
  .chip span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text); }
  .title { flex: none; width: 100%; padding: .5rem 1.5rem .4rem; border: 0; outline: none; background: transparent; color: var(--text); font: inherit; font-size: 1.6rem; font-weight: 700; letter-spacing: -.02em; }
  .props { flex: none; display: grid; gap: .35rem; margin-inline: 1.5rem; padding-block: .25rem .9rem; border-block-end: 1px solid var(--line); text-align: start; font-size: .82rem; color: var(--muted); }
  .props span { display: flex; flex-wrap: wrap; align-items: center; gap: .4rem; }
  .props b { font-weight: 500; color: var(--text); }
  .editor { flex: 1 0 auto; display: flex; flex-direction: column; }
  .bar { position: relative; flex: none; }
  .tools { display: flex; align-items: center; padding: .2rem .4rem; background: var(--panel); border-block-start: 1px solid var(--line); }
  .scroll { flex: 1; min-width: 0; display: flex; gap: .1rem; overflow-x: auto; scrollbar-width: none; touch-action: pan-x; }
  .tools b { font-size: .95rem; font-weight: 700; color: var(--blue-text); }
  .tools .on { background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .drop-line { position: fixed; z-index: 5; height: 2px; margin-block-start: -1px; border-radius: 1px; background: var(--blue); pointer-events: none; }
  .pop { position: absolute; inset-inline: .5rem; inset-block-end: calc(100% + .35rem); z-index: 6; display: grid; max-height: 40vh; overflow-y: auto; overscroll-behavior: contain; padding: .3rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); box-shadow: 0 8px 24px rgb(0 0 0 / .25); }
  .pop button { display: flex; align-items: center; gap: .75rem; min-height: 2.9rem; padding-inline: .6rem; border-radius: .5rem; font-weight: 600; text-align: start; }
  .pop button:active { background: var(--soft); }
  .pop .mark { flex: none; display: grid; place-items: center; width: 1.8rem; font-size: .85rem; font-weight: 700; color: var(--muted); }
  .pop i { display: grid; margin-inline-start: auto; color: var(--blue-text); }
  .tools button { flex: none; display: grid; place-items: center; width: 2.6rem; height: 2.75rem; border-radius: .5rem; color: var(--muted); }
  .tools button:active { background: var(--soft); color: var(--text); }
  .tools [aria-disabled="true"], .tools [aria-disabled="true"]:active { background: none; color: var(--muted); opacity: .35; }
</style>
