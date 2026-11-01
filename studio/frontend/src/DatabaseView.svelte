<script lang="ts">
  import { onDestroy, onMount, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { AddRecord, DeleteItem, ReadDatabase, RestoreSection, SaveDatabaseSchema, SetProps } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { DBColumn, DBRecord, DBSchema, DBView } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Table2 from "@lucide/svelte/icons/table-2";
  import Kanban from "@lucide/svelte/icons/kanban";
  import Search from "@lucide/svelte/icons/search";
  import Plus from "@lucide/svelte/icons/plus";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import ArrowUp from "@lucide/svelte/icons/arrow-up";
  import ArrowDown from "@lucide/svelte/icons/arrow-down";
  import X from "@lucide/svelte/icons/x";
  import FieldInput from "./FieldInput.svelte";
  import { TYPES, asText, compare, keyOf, matches, nextColor, optionColor } from "./db";
  import { toast, toastError } from "./toast.svelte";
  import { t } from "./i18n";

  // A database of the vault, for anyone: a table to fill like a
  // spreadsheet (click a cell and write), or a board to drag cards between
  // columns (Hot, Warm, Cold). Each record is a note: its name opens it.
  // The search looks in every field and in the notes' text; a choice
  // filters; a column's title sorts and edits it; "+" adds a column.
  let { path, notes, onOpenNote, onStatus, onError, bindSave }: {
    path: string; notes: { name: string; path: string }[];
    bindSave?: (save: () => Promise<void>) => void; // the workspace saves first before a rename, move or delete
    onOpenNote: (path: string) => void;
    onStatus: (s: "saved" | "saving" | "unsaved") => void;
    onError: (e: string) => void;
  } = $props();
  const d = t.db;
  const clean = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");

  // The schema with its lists always there (the file may have none).
  type Schema = DBSchema & { columns: DBColumn[]; views: DBView[] };
  let schema = $state<Schema | null>(null);
  let records = $state<DBRecord[]>([]);
  let viewId = $state("");
  let query = $state("");
  const view = $derived(schema?.views.find((v) => v.id === viewId) ?? schema?.views[0]);
  const byKey = $derived(new Map((schema?.columns ?? []).map((c) => [c.key, c])));
  const selects = $derived((schema?.columns ?? []).filter((c) => c.type === "select"));

  async function load() {
    try {
      const db = await ReadDatabase(untrack(() => path));
      schema = { ...db.schema, columns: db.schema.columns ?? [], views: db.schema.views ?? [] };
      records = db.records ?? [];
      if (!viewId) viewId = schema.views[0]?.id ?? "";
    } catch (e) { onError(clean(e)); }
  }
  load();

  // The columns and views, saved a moment after a change.
  let timer: ReturnType<typeof setTimeout> | undefined;
  let dirty = false;
  function saveSchema() {
    dirty = true;
    onStatus("unsaved");
    clearTimeout(timer);
    timer = setTimeout(flush, 300);
  }
  async function flush() {
    clearTimeout(timer);
    if (!dirty || !schema) return;
    dirty = false;
    onStatus("saving");
    try { await SaveDatabaseSchema(path, $state.snapshot(schema) as DBSchema); onStatus("saved"); }
    catch (e) { dirty = true; onStatus("unsaved"); onError(clean(e)); }
  }
  onMount(() => bindSave?.(flush));

  // What is shown: the search, the view's filter and its order.
  const shown = $derived.by(() => {
    if (!view) return records;
    let list = records.filter((r) => matches(r, query.trim()));
    if (view.filter && view.match) list = list.filter((r) => asText(r.props?.[view.filter!]).split(", ").includes(view.match!));
    const desc = view.sort?.startsWith("-"), key = (view.sort ?? "").replace(/^-/, "");
    return [...list].sort((a, b) => (desc ? -1 : 1) * compare(a, b, key, byKey.get(key)));
  });

  // A cell changed: the record's note gets it.
  async function set(r: DBRecord, key: string, value: unknown) {
    r.props = { ...(r.props ?? {}), [key]: value };
    try { await SetProps(r.path, [{ key, value }]); } catch (e) { onError(clean(e)); }
  }

  // New records: named in a row on top (or in a board's column).
  let adding = $state<string | null>(null); // the board column it goes in ("" the table)
  let newName = $state("");
  async function add(group = "") {
    const name = newName.trim();
    adding = null; newName = "";
    if (!name) return;
    const props: Record<string, unknown> = {};
    if (view?.type === "board" && view.groupBy && group) props[view.groupBy] = group;
    else if (view?.filter && view.match) props[view.filter] = view.match;
    try { await AddRecord(path, name, props); await load(); } catch (e) { onError(clean(e)); }
  }
  async function remove(r: DBRecord) {
    try {
      const where = await DeleteItem(r.path);
      records = records.filter((x) => x !== r);
      toast(t.toast.noteDeleted.replace("{name}", r.name), { action: { label: t.toast.undo, run: () => RestoreSection(where).then(load).catch(toastError) } });
    } catch (e) { onError(clean(e)); }
  }

  // Columns: add, rename, change the kind, edit the choices, sort, delete.
  let colMenu = $state<{ key: string; x: number; y: number } | null>(null);
  const menuCol = $derived(colMenu ? byKey.get(colMenu.key) : undefined);
  function openCol(e: MouseEvent, key: string) {
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    colMenu = { key, x: Math.min(r.left, innerWidth - 260), y: r.bottom + 4 };
  }
  function addColumn(e: MouseEvent) {
    if (!schema) return;
    const key = keyOf(d.newColumn, schema.columns.map((c) => c.key));
    schema.columns.push({ key, name: d.newColumn, type: "text", options: [] });
    saveSchema();
    openCol(e, key);
  }
  function deleteColumn(key: string) {
    if (!schema) return;
    schema.columns = schema.columns.filter((c) => c.key !== key);
    for (const v of schema.views) { if (v.groupBy === key) v.groupBy = selects.find((c) => c.key !== key)?.key ?? ""; if (v.filter === key) { v.filter = ""; v.match = ""; } }
    colMenu = null;
    saveSchema();
  }
  function sortBy(key: string, desc: boolean) {
    if (!view) return;
    view.sort = (desc ? "-" : "") + key;
    colMenu = null;
    saveSchema();
  }
  // A choice renamed: the records that had it get the new name too.
  function renameOption(c: DBColumn, o: { value: string }, typed: string) {
    const was = o.value, now = typed.trim();
    if (!now || now === was) return;
    o.value = now;
    saveSchema();
    for (const r of records) {
      const v = r.props?.[c.key];
      if (v === was) set(r, c.key, now);
      else if (Array.isArray(v) && v.includes(was)) set(r, c.key, v.map((x) => (x === was ? now : x)));
    }
  }
  function newOption(c: DBColumn, value: string, color: string) {
    c.options = [...(c.options ?? []), { value, color }];
    saveSchema();
  }

  // Views: the table and the board (by a choice column).
  function addBoard() {
    if (!schema || !selects.length) return;
    const v: DBView = { id: "board-" + Date.now().toString(36), name: d.viewBoard, type: "board", groupBy: selects[0].key, sort: "", filter: "", match: "" };
    schema.views.push(v);
    viewId = v.id;
    saveSchema();
  }

  // The board: a column for each choice (and one for none); dragging a
  // card to another column changes its choice.
  const groupCol = $derived(view?.type === "board" ? byKey.get(view.groupBy ?? "") : undefined);
  const lanes = $derived(groupCol ? [...(groupCol.options ?? []).map((o) => ({ value: o.value, color: o.color })), { value: "", color: "" }] : []);
  let dragged: DBRecord | null = null;
  let overLane = $state<string | null>(null);
  function drop(lane: string) {
    const r = dragged;
    dragged = null; overLane = null;
    if (!r || !groupCol || asText(r.props?.[groupCol.key]) === lane) return;
    set(r, groupCol.key, lane);
  }
  const cardFields = $derived((schema?.columns ?? []).filter((c) => c.key !== groupCol?.key).slice(0, 3));

  // Changed on disk (another window, an AI, sync): read again.
  const off = Events.On("notesChanged", (ev: any) => {
    const dir = path.replace(/[^/]*$/, "") + (schema?.records ?? "") + "/";
    const list = String(ev.data ?? "").split("\n");
    // Read again in place (the view, the search and the menus stay),
    // unless a change of its own is waiting to be saved.
    if (list.some((p) => p === path || p.startsWith(dir)) && !dirty) load();
  });
  onDestroy(() => { off(); flush(); });
  const focus = (el: HTMLInputElement) => { el.focus(); el.select(); };
</script>

<svelte:window onclick={() => (colMenu = null)} />

{#if schema && view}
  <div class="db">
    <div class="bar">
      <div class="views" role="tablist" aria-label={d.views}>
        {#each schema.views as v (v.id)}
          <button role="tab" aria-selected={v.id === view.id} class={{ on: v.id === view.id }} onclick={() => (viewId = v.id)}>
            {#if v.type === "board"}<Kanban size={15} />{:else}<Table2 size={15} />{/if}{v.name}
          </button>
        {/each}
        {#if !schema.views.some((v) => v.type === "board") && selects.length}
          <button class="ghost" onclick={addBoard}><Plus size={14} />{d.viewBoard}</button>
        {/if}
      </div>
      <span class="grow"></span>
      <!-- The system's own fields (app.css); the icon sits inside. -->
      <label class="search"><Search size={15} /><input type="text" bind:value={query} placeholder={d.search} aria-label={d.search} /></label>
      {#if selects.length}
        <select class="filter" value={view.filter && view.match ? `${view.filter}\u0000${view.match}` : ""} aria-label={d.filter}
            onchange={(e) => { const [f, m] = e.currentTarget.value.split("\u0000"); view.filter = f ?? ""; view.match = m ?? ""; saveSchema(); }}>
            <option value="">{d.filter}: {d.all}</option>
            {#each selects as c (c.key)}
              <optgroup label={c.name}>{#each c.options ?? [] as o (o.value)}<option value={`${c.key}\u0000${o.value}`}>{o.value}</option>{/each}</optgroup>
            {/each}
          </select>
      {/if}
      {#if view.type === "board" && selects.length > 1}
        <select class="group" value={view.groupBy} aria-label={d.groupBy} onchange={(e) => { view.groupBy = e.currentTarget.value; saveSchema(); }}>
          {#each selects as c (c.key)}<option value={c.key}>{d.groupByCol.replace("{name}", c.name)}</option>{/each}
        </select>
      {/if}
      <button class="btn" onclick={() => { adding = view.type === "board" ? lanes[0]?.value ?? "" : ""; newName = ""; }}><Plus size={15} />{d.newRecord}</button>
    </div>

    {#if view.type === "table"}
      <div class="scroll">
        <table>
          <thead>
            <tr>
              <th class="name"><button onclick={(e) => openCol(e, "")}>{d.name}{#if view.sort === "" || view.sort === "-"}{#if view.sort === "-"}<ArrowDown size={12} />{:else}<ArrowUp size={12} />{/if}{/if}</button></th>
              {#each schema.columns as c (c.key)}
                <th><button onclick={(e) => openCol(e, c.key)}>{c.name}{#if view.sort === c.key}<ArrowUp size={12} />{:else if view.sort === "-" + c.key}<ArrowDown size={12} />{/if}</button></th>
              {/each}
              <th class="addcol"><button title={d.addColumn} aria-label={d.addColumn} onclick={addColumn}><Plus size={15} /></button></th>
            </tr>
          </thead>
          <tbody>
            {#if adding === ""}
              <tr class="new"><td colspan={schema.columns.length + 2}><input type="text" bind:value={newName} placeholder={d.newRecordPh} use:focus onkeydown={(e) => { if (e.key === "Enter") add(); if (e.key === "Escape") adding = null; }} onblur={() => add()} /></td></tr>
            {/if}
            {#each shown as r (r.path)}
              <tr>
                <td class="name"><button class="open" onclick={() => onOpenNote(r.path)}>{r.name}</button><button class="del" title={d.deleteRecord} aria-label={d.deleteRecord} onclick={() => remove(r)}><Trash2 size={14} /></button></td>
                {#each schema.columns as c (c.key)}
                  <td><FieldInput column={c} value={r.props?.[c.key]} compact {notes} {onOpenNote} onChange={(v) => set(r, c.key, v)} onNewOption={(v, color) => newOption(c, v, color)} /></td>
                {/each}
                <td></td>
              </tr>
            {/each}
          </tbody>
        </table>
        {#if !shown.length && adding === null}
          <p class="empty muted">{records.length ? d.nothingFound : d.emptyLead}</p>
        {/if}
      </div>
    {:else}
      <div class="board">
        {#each lanes as lane (lane.value)}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <section class={["lane", { over: overLane === lane.value }]} style:--c={lane.value ? optionColor(lane.color) : "var(--muted)"}
            ondragover={(e) => { e.preventDefault(); overLane = lane.value; }} ondragleave={() => (overLane = null)} ondrop={() => drop(lane.value)}>
            <header><span class="dot"></span>{lane.value || d.noValue}<span class="n">{shown.filter((r) => asText(r.props?.[groupCol!.key]) === lane.value).length}</span></header>
            {#if adding === lane.value}
              <input type="text" bind:value={newName} placeholder={d.newRecordPh} use:focus onkeydown={(e) => { if (e.key === "Enter") add(lane.value); if (e.key === "Escape") adding = null; }} onblur={() => add(lane.value)} />
            {/if}
            {#each shown.filter((r) => asText(r.props?.[groupCol!.key]) === lane.value) as r (r.path)}
              <button class="bcard" draggable="true" ondragstart={() => (dragged = r)} ondragend={() => { dragged = null; overLane = null; }} onclick={() => onOpenNote(r.path)}>
                <b>{r.name}</b>
                {#each cardFields as c (c.key)}{#if asText(r.props?.[c.key])}<small><span class="muted">{c.name}:</span> {asText(r.props?.[c.key])}</small>{/if}{/each}
              </button>
            {/each}
            <button class="lane-add" onclick={() => { adding = lane.value; newName = ""; }}><Plus size={14} />{d.newRecord}</button>
          </section>
        {/each}
      </div>
    {/if}
  </div>

  {#if colMenu}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="colmenu" style:left="{colMenu.x}px" style:top="{colMenu.y}px" onclick={(e) => e.stopPropagation()}>
      {#if menuCol}
        <input type="text" value={menuCol.name} use:focus aria-label={d.columnName} onchange={(e) => { menuCol.name = e.currentTarget.value.trim() || menuCol.name; saveSchema(); }} onkeydown={(e) => { if (e.key === "Enter") e.currentTarget.blur(); }} />
        <label class="kind"><span>{d.kind}</span>
          <select value={menuCol.type} onchange={(e) => { menuCol.type = e.currentTarget.value; saveSchema(); }}>
            {#each TYPES as ty (ty)}<option value={ty}>{d.types[ty]}</option>{/each}
          </select>
        </label>
        {#if menuCol.type === "select" || menuCol.type === "multi"}
          <div class="opts">
            {#each menuCol.options ?? [] as o, i (i)}
              <div class="opt">
                <button class="sw" style:background={optionColor(o.color)} title={d.color} aria-label={d.color} onclick={() => { o.color = nextColor(o.color); saveSchema(); }}></button>
                <input type="text" value={o.value} aria-label={d.choice} onchange={(e) => renameOption(menuCol, o, e.currentTarget.value)} />
                <button class="x" title={d.removeChoice} aria-label={d.removeChoice} onclick={() => { menuCol.options = (menuCol.options ?? []).filter((_, k) => k !== i); saveSchema(); }}><X size={13} /></button>
              </div>
            {/each}
            <button class="addopt" onclick={() => newOption(menuCol, d.newChoice, nextColor(menuCol.options?.at(-1)?.color))}><Plus size={13} />{d.addChoice}</button>
          </div>
        {/if}
      {/if}
      <button class="item" onclick={() => sortBy(colMenu!.key, false)}><ArrowUp size={14} />{d.sortAsc}</button>
      <button class="item" onclick={() => sortBy(colMenu!.key, true)}><ArrowDown size={14} />{d.sortDesc}</button>
      {#if menuCol}<button class="item danger" onclick={() => deleteColumn(menuCol.key)}><Trash2 size={14} />{d.deleteColumn}</button>{/if}
    </div>
  {/if}
{/if}

<style>
  .db { position: absolute; inset: 0; display: grid; grid-template-rows: auto minmax(0, 1fr); background: var(--panel); }
  .bar { display: flex; flex-wrap: wrap; align-items: center; gap: .5rem; padding: .65rem 1rem; border-block-end: 1px solid var(--line); }
  .views { display: flex; gap: .2rem; }
  .views button { display: flex; align-items: center; gap: .35rem; padding: .35rem .65rem; border-radius: .45rem; font-size: .86rem; color: var(--muted); }
  .views button:hover { background: var(--soft); color: var(--text); }
  .views button.on { background: var(--soft); color: var(--text); font-weight: 600; }
  .views .ghost { border: 1px dashed var(--line); }
  .grow { flex: 1; }
  .search { position: relative; display: block; width: 15rem; color: var(--muted); }
  .search :global(svg) { position: absolute; inset-inline-start: .65rem; inset-block-start: 50%; translate: 0 -50%; pointer-events: none; }
  .search input { padding-inline-start: 2.1rem; }
  .filter, .group { width: auto; min-width: 8rem; }
  .btn { display: flex; align-items: center; gap: .35rem; }

  .scroll { overflow: auto; padding: 0 1rem 2rem; }
  table { border-collapse: collapse; min-width: 100%; font-size: .9rem; }
  th, td { border-block-end: 1px solid var(--line); text-align: start; vertical-align: middle; }
  th { position: sticky; inset-block-start: 0; z-index: 1; background: var(--panel); }
  th button { display: flex; align-items: center; gap: .3rem; width: 100%; padding: .55rem .5rem; font-size: .8rem; font-weight: 600; color: var(--muted); text-align: start; white-space: nowrap; }
  th button:hover { color: var(--text); background: var(--soft); }
  th.addcol button { justify-content: center; width: 2.4rem; }
  td { min-width: 9rem; max-width: 18rem; padding: .15rem .2rem; }
  td.name { position: relative; min-width: 12rem; }
  .open { padding: .3rem .4rem; border-radius: .35rem; font-weight: 600; color: var(--text); text-align: start; }
  .open:hover { color: var(--blue); text-decoration: underline; text-underline-offset: 3px; }
  .del { position: absolute; inset-inline-end: .2rem; inset-block-start: 50%; translate: 0 -50%; display: grid; place-items: center; width: 1.6rem; height: 1.6rem; border-radius: .35rem; color: var(--muted); opacity: 0; }
  tr:hover .del { opacity: 1; }
  .del:hover { background: var(--soft); color: var(--red); }
  tbody tr:hover { background: color-mix(in srgb, var(--soft) 60%, transparent); }
  .new td { padding: .4rem .2rem; }
  .empty { padding: 2.5rem 1rem; text-align: center; }

  .board { display: flex; gap: .8rem; overflow: auto; padding: 1rem; align-items: flex-start; }
  .lane { flex: none; width: 16rem; display: grid; gap: .45rem; padding: .6rem; border-radius: .7rem; background: color-mix(in srgb, var(--c) 7%, var(--bg)); border: 1px solid transparent; }
  .lane.over { border-color: var(--c); }
  .lane header { display: flex; align-items: center; gap: .45rem; font-size: .86rem; font-weight: 600; }
  .dot { width: .6rem; height: .6rem; border-radius: 50%; background: var(--c); }
  .n { margin-inline-start: auto; padding: 0 .45rem; border-radius: 999px; background: var(--soft); font-size: .75rem; color: var(--muted); }
  .bcard { display: grid; gap: .2rem; padding: .6rem .7rem; border: 1px solid var(--line); border-radius: .55rem; background: var(--panel); text-align: start; cursor: grab; box-shadow: 0 1px 2px rgb(0 0 0 / .06); }
  .bcard:hover { border-color: var(--muted); }
  .bcard b { font-size: .9rem; }
  .bcard small { font-size: .78rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .lane-add { display: flex; align-items: center; gap: .3rem; padding: .35rem .5rem; border-radius: .4rem; font-size: .82rem; color: var(--muted); }
  .lane-add:hover { background: var(--soft); color: var(--text); }

  .colmenu { position: fixed; z-index: 80; width: 15.5rem; display: grid; gap: .3rem; padding: .45rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .28); }
  .kind { display: grid; gap: .2rem; font-size: .75rem; color: var(--muted); }
  .opt input { padding-block: .35rem; }
  .opts { display: grid; gap: .25rem; padding: .3rem 0; border-block: 1px solid var(--line); }
  .opt { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .3rem; }
  .sw { width: 1rem; height: 1rem; border-radius: 50%; }
  .x { display: grid; place-items: center; width: 1.5rem; height: 1.5rem; border-radius: .3rem; color: var(--muted); }
  .x:hover { background: var(--soft); color: var(--red); }
  .addopt, .item { display: flex; align-items: center; gap: .45rem; padding: .35rem .45rem; border-radius: .35rem; font-size: .85rem; text-align: start; }
  .addopt { color: var(--muted); }
  .addopt:hover, .item:hover { background: var(--soft); }
  .item.danger { color: var(--red); }
</style>
