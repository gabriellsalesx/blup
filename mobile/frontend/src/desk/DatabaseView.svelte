<script lang="ts">
  import { onDestroy, onMount, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { NotesChanged } from "../../bindings/events";
  import { AddRecord, DeleteItem, ReadDatabase, RestoreSection, SaveDatabaseSchema, SetProps } from "./studio";
  import type { DBColumn, DBRecord, DBSchema, DBView } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Table2 from "@lucide/svelte/icons/table-2";
  import Kanban from "@lucide/svelte/icons/kanban";
  import Search from "@lucide/svelte/icons/search";
  import Plus from "@lucide/svelte/icons/plus";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import FileText from "@lucide/svelte/icons/file-text";
  import SlidersHorizontal from "@lucide/svelte/icons/sliders-horizontal";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import ArrowUp from "@lucide/svelte/icons/arrow-up";
  import ArrowDown from "@lucide/svelte/icons/arrow-down";
  import X from "@lucide/svelte/icons/x";
  import Check from "@lucide/svelte/icons/check";
  import BackSheet from "./BackSheet.svelte";
  import FieldInput from "./FieldInput.svelte";
  import { TYPES, asText, compare, keyOf, matches, nextColor, optionColor } from "./db";
  import { toast, toastError } from "./toast.svelte";
  import { t } from "./i18n";

  // A database of the vault, for anyone: a table to fill like a
  // spreadsheet (click a cell and write), or a board to drag cards between
  // columns (Hot, Warm, Cold). Each record is a note: its name opens it.
  // The search looks in every field and in the notes' text; a choice
  // filters; a column's title sorts and edits it; "+" adds a column.
  let { path, notes, onOpenNote, onStatus, onError, onGone, bindSave }: {
    path: string; notes: { name: string; path: string }[];
    bindSave?: (save: () => Promise<void>) => void; // the workspace saves first before a rename, move or delete
    onOpenNote: (path: string) => void;
    onStatus: (s: "saved" | "saving" | "unsaved") => void;
    onError: (e: string) => void;
    onGone?: () => void; // deleted or renamed elsewhere (a sync)
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

  // A record on a phone: a list of cards, each opening a sheet to fill in,
  // one big field after the other. A new one is the same sheet, with its name.
  let record = $state<DBRecord | null>(null);
  let draft = $state<{ name: string; props: Record<string, unknown> } | null>(null);
  function newRecord(group = "") {
    const props: Record<string, unknown> = {};
    if (view?.type === "board" && view.groupBy && group) props[view.groupBy] = group;
    else if (view?.filter && view.match) props[view.filter] = view.match;
    draft = { name: "", props };
  }
  async function add() {
    const job = draft, name = draft?.name.trim();
    if (!job || !name) return;
    draft = null;
    try { await AddRecord(path, name, $state.snapshot(job.props)); await load(); } catch (e) { onError(clean(e)); }
  }
  // The sheet goes first: its step in the history must not stay under the note.
  function goNote(p: string) {
    record = null;
    setTimeout(() => onOpenNote(p), 150);
  }
  async function remove(r: DBRecord) {
    try {
      const where = await DeleteItem(r.path);
      records = records.filter((x) => x.path !== r.path);
      record = null;
      toast(t.toast.noteDeleted.replace("{name}", r.name), { action: { label: t.toast.undo, run: () => RestoreSection(where).then(load).catch(toastError) } });
    } catch (e) { onError(clean(e)); }
  }

  // Columns: add, rename, change the kind, edit the choices, sort, delete.
  let colMenu = $state<{ key: string } | null>(null);
  let colList = $state(false);
  const menuCol = $derived(colMenu ? byKey.get(colMenu.key) : undefined);
  function addColumn() {
    if (!schema) return;
    const key = keyOf(d.newColumn, schema.columns.map((c) => c.key));
    schema.columns.push({ key, name: d.newColumn, type: "text", options: [] });
    saveSchema();
    colMenu = { key };
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
  const cardFields = $derived((schema?.columns ?? []).filter((c) => c.key !== groupCol?.key).slice(0, 3));

  // Changed on disk (another window, an AI, sync): read again. Paths
  // compare in any case, as the other views do.
  const off = Events.On(NotesChanged, async (ev) => {
    const self = path.toLowerCase();
    const dir = (path.replace(/[^/]*$/, "") + (schema?.records ?? "") + "/").toLowerCase();
    const list = ev.data.split("\n").map((p) => p.toLowerCase());
    // The database itself gone (deleted or renamed by a sync): closed, and
    // its columns never saved again.
    if (list.includes(self) && !(await ReadDatabase(path).then(() => true, () => false))) {
      clearTimeout(timer);
      dirty = false;
      schema = null; // flush() writes nothing from here on
      onGone?.();
      return;
    }
    // Read again in place (the view, the search and the menus stay),
    // unless a change of its own is waiting to be saved.
    if (list.some((p) => p === self || p.startsWith(dir)) && !dirty) load();
  });
  onDestroy(() => { off(); flush(); });
  const focus = (el: HTMLInputElement) => { setTimeout(() => el.focus(), 260); };
  const asList = (v: unknown) => (Array.isArray(v) ? v.map(String) : v ? [String(v)] : []);
  const optionOf = (c: DBColumn, v: string) => c.options?.find((o) => o.value === v);
</script>

{#snippet values(r: DBRecord)}
  {#each cardFields as c (c.key)}
    {#if asText(r.props?.[c.key])}
      {#if c.type === "select" || c.type === "multi"}
        {#each asList(r.props?.[c.key]) as v (v)}<span class="chip" style:--c={optionColor(optionOf(c, v)?.color)}>{v}</span>{/each}
      {:else if typeof r.props?.[c.key] === "boolean"}<small><span class="muted">{c.name}:</span> <Check size={12} aria-hidden="true" /></small>
      {:else}<small><span class="muted">{c.name}:</span> {asText(r.props?.[c.key])}</small>{/if}
    {/if}
  {/each}
{/snippet}

{#snippet card(r: DBRecord)}
  <button type="button" class="rec" onclick={() => (record = r)}>
    <span class="body"><b>{r.name}</b>{#if cardFields.some((c) => asText(r.props?.[c.key]))}<span class="vals">{@render values(r)}</span>{/if}</span>
    <ChevronRight size={20} />
  </button>
{/snippet}

{#snippet field(c: DBColumn, value: unknown, change: (v: unknown) => void)}
  <div class="field">
    {#if c.type !== "check"}<span class="label">{c.name}</span>{/if}
    <FieldInput column={c} {value} {notes} onOpenNote={goNote} onChange={change} onNewOption={(v, color) => newOption(c, v, color)} />
  </div>
{/snippet}

{#if schema && view}
  <div class="db">
    <div class="bar">
      <div class="views" role="tablist" aria-label={d.views}>
        {#each schema.views as v (v.id)}
          <button role="tab" aria-selected={v.id === view.id} class={{ on: v.id === view.id }} onclick={() => (viewId = v.id)}>
            {#if v.type === "board"}<Kanban size={18} />{:else}<Table2 size={18} />{/if}{v.name}
          </button>
        {/each}
        {#if !schema.views.some((v) => v.type === "board") && selects.length}
          <button onclick={addBoard}><Plus size={18} />{d.viewBoard}</button>
        {/if}
      </div>
      <button class="cols" aria-label={t.dbx.columns} onclick={() => (colList = true)}><SlidersHorizontal size={21} /></button>
    </div>
    <div class="find">
      <label class="search"><Search size={18} /><input type="text" bind:value={query} placeholder={d.search} aria-label={d.search} enterkeyhint="search" /></label>
      {#if selects.length}
        <select class="pick" value={view.filter && view.match ? `${view.filter}\u0000${view.match}` : ""} aria-label={d.filter}
          onchange={(e) => { const [f, m] = e.currentTarget.value.split("\u0000"); view.filter = f ?? ""; view.match = m ?? ""; saveSchema(); }}>
          <option value="">{d.all}</option>
          {#each selects as c (c.key)}
            <optgroup label={c.name}>{#each c.options ?? [] as o (o.value)}<option value={`${c.key}\u0000${o.value}`}>{o.value}</option>{/each}</optgroup>
          {/each}
        </select>
      {/if}
      {#if view.type === "board" && selects.length > 1}
        <select class="pick" value={view.groupBy} aria-label={d.groupBy} onchange={(e) => { view.groupBy = e.currentTarget.value; saveSchema(); }}>
          {#each selects as c (c.key)}<option value={c.key}>{c.name}</option>{/each}
        </select>
      {/if}
    </div>

    {#if view.type === "table"}
      <div class="list">
        {#each shown as r (r.path)}{@render card(r)}{/each}
        {#if !shown.length}<p class="muted none">{records.length ? d.nothingFound : t.dbx.none}</p>{/if}
      </div>
    {:else}
      <div class="lanes">
        {#each lanes as lane (lane.value)}
          <section class="lane" style:--c={lane.value ? optionColor(lane.color) : "var(--muted)"}>
            <header><span class="dot"></span>{lane.value || d.noValue}<span class="n">{shown.filter((r) => asText(r.props?.[groupCol!.key]) === lane.value).length}</span></header>
            <div class="cards">
              {#each shown.filter((r) => asText(r.props?.[groupCol!.key]) === lane.value) as r (r.path)}{@render card(r)}{/each}
              <button class="lane-add" onclick={() => newRecord(lane.value)}><Plus size={18} />{d.newRecord}</button>
            </div>
          </section>
        {/each}
      </div>
    {/if}
    <button class="fab" aria-label={d.newRecord} onclick={() => newRecord()}><Plus size={26} /></button>
  </div>

  {#if draft}
    <BackSheet title={t.dbx.newRecord} onclose={() => (draft = null)}>
      <div class="form">
        <div class="field"><span class="label">{d.name}</span><input class="name" type="text" bind:value={draft.name} {@attach focus} aria-label={d.name} enterkeyhint="done" onkeydown={(e) => { if (e.key === "Enter") e.currentTarget.blur(); }} /></div>
        {#each schema.columns as c (c.key)}{@render field(c, draft.props[c.key], (v) => { if (draft) draft.props[c.key] = v; })}{/each}
        <button type="button" class="make" disabled={!draft.name.trim()} onclick={add}>{t.dbx.add}</button>
      </div>
    </BackSheet>
  {/if}

  {#if record}
    {@const r = record}
    <BackSheet title={r.name} onclose={() => (record = null)}>
      <div class="form">
        {#each schema.columns as c (c.key)}{@render field(c, r.props?.[c.key], (v) => set(r, c.key, v))}{/each}
        <button class="item" onclick={() => goNote(r.path)}><FileText size={20} />{t.dbx.openNote}</button>
        <button class="item danger" onclick={() => remove(r)}><Trash2 size={20} />{d.deleteRecord}</button>
      </div>
    </BackSheet>
  {/if}

  {#if colList}
    <BackSheet title={t.dbx.columns} onclose={() => (colList = false)}>
      <div class="form">
        <button class="item" onclick={() => (colMenu = { key: "" })}><span class="grow">{d.name}</span><ChevronRight size={20} /></button>
        {#each schema.columns as c (c.key)}
          <button class="item" onclick={() => (colMenu = { key: c.key })}><span class="grow">{c.name}</span><small class="muted">{d.types[c.type as keyof typeof d.types]}</small><ChevronRight size={20} /></button>
        {/each}
        <button class="addopt" onclick={addColumn}><Plus size={20} />{d.addColumn}</button>
      </div>
    </BackSheet>
  {/if}

  {#if colMenu}
    <BackSheet title={menuCol?.name ?? d.name} onclose={() => (colMenu = null)}>
    <div class="colmenu">
      {#if menuCol}
        <input type="text" value={menuCol.name} aria-label={d.columnName} onchange={(e) => { menuCol.name = e.currentTarget.value.trim() || menuCol.name; saveSchema(); }} onkeydown={(e) => { if (e.key === "Enter") e.currentTarget.blur(); }} />
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
                <button class="x" title={d.removeChoice} aria-label={d.removeChoice} onclick={() => { menuCol.options = (menuCol.options ?? []).filter((_, k) => k !== i); saveSchema(); }}><X size={18} /></button>
              </div>
            {/each}
            <button class="addopt" onclick={() => newOption(menuCol, d.newChoice, nextColor(menuCol.options?.at(-1)?.color))}><Plus size={20} />{d.addChoice}</button>
          </div>
        {/if}
      {/if}
      <button class="item" onclick={() => sortBy(colMenu!.key, false)}><ArrowUp size={20} />{d.sortAsc}</button>
      <button class="item" onclick={() => sortBy(colMenu!.key, true)}><ArrowDown size={20} />{d.sortDesc}</button>
      {#if menuCol}<button class="item danger" onclick={() => deleteColumn(menuCol.key)}><Trash2 size={20} />{d.deleteColumn}</button>{/if}
    </div>
    </BackSheet>
  {/if}
{/if}

<style>
  .db { position: absolute; inset: 0; display: flex; flex-direction: column; background: var(--bg); }
  .bar { flex: none; display: flex; align-items: center; gap: .5rem; padding: .6rem 1rem .4rem; }
  .views { flex: 1; min-width: 0; display: flex; gap: .4rem; overflow-x: auto; scrollbar-width: none; }
  .views button { flex: none; display: flex; align-items: center; gap: .45rem; min-height: 2.6rem; padding-inline: .85rem; border-radius: .6rem; color: var(--muted); font-weight: 600; }
  .views .on { background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .cols { flex: none; display: grid; place-items: center; width: 2.75rem; height: 2.75rem; color: var(--muted); }
  .find { flex: none; display: flex; align-items: center; gap: .5rem; padding: .2rem 1rem .7rem; border-block-end: 1px solid var(--line); }
  .search { flex: 1; min-width: 0; position: relative; display: block; }
  .search :global(svg) { position: absolute; inset-block-start: .85rem; inset-inline-start: .75rem; color: var(--muted); }
  .search input { width: 100%; min-height: 2.9rem; padding-inline: 2.4rem .75rem; }
  .pick { flex: none; max-width: 38%; min-height: 2.9rem; padding-inline: .6rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--bg); color: var(--text); }
  .list { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; display: grid; align-content: start; gap: .6rem; padding: .8rem 1rem 6rem; touch-action: pan-y; }
  .none { padding-block: 2rem; text-align: center; }
  .rec { display: flex; align-items: center; gap: .6rem; width: 100%; min-height: 3.4rem; padding: .7rem .85rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); text-align: start; color: var(--muted); }
  .rec:active { background: var(--soft); }
  .body { flex: 1; min-width: 0; display: grid; gap: .35rem; color: var(--text); }
  .body b { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vals { display: flex; flex-wrap: wrap; align-items: center; gap: .3rem .5rem; }
  .vals small { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: .85rem; }
  .chip { padding: .05rem .5rem; border-radius: 999px; background: color-mix(in srgb, var(--c) 18%, transparent); color: color-mix(in srgb, var(--c) 70%, var(--text)); font-size: .82rem; font-weight: 500; }
  .lanes { flex: 1; min-height: 0; display: flex; gap: .7rem; overflow-x: auto; scroll-snap-type: x mandatory; padding: .8rem 1rem; scroll-padding-inline: 1rem; }
  .lane { flex: 0 0 84%; display: flex; flex-direction: column; min-height: 0; scroll-snap-align: start; border-radius: .75rem; background: color-mix(in srgb, var(--c) 8%, var(--panel)); }
  .lane header { flex: none; display: flex; align-items: center; gap: .5rem; padding: .75rem .85rem .4rem; font-weight: 600; }
  .dot { width: .7rem; height: .7rem; border-radius: 50%; background: var(--c); }
  .n { margin-inline-start: auto; color: var(--muted); font-size: .9rem; }
  .cards { flex: 1; min-height: 0; overflow-y: auto; display: grid; align-content: start; gap: .5rem; padding: .3rem .6rem 5rem; }
  .lane-add { display: flex; align-items: center; gap: .5rem; min-height: 3rem; padding-inline: .6rem; color: var(--muted); font-weight: 500; }
  .fab { position: absolute; inset-inline-end: 1rem; inset-block-end: calc(1rem + var(--sab)); z-index: 3; display: grid; place-items: center; width: 3.5rem; height: 3.5rem; border-radius: 50%; background: var(--blue); color: #fff; box-shadow: 0 6px 16px rgb(0 0 0 / .3); }
  .form { display: grid; gap: .9rem; }
  .field { display: grid; gap: .35rem; }
  .label { font-size: .9rem; font-weight: 600; color: var(--muted); }
  .name { width: 100%; min-height: 3rem; }
  .make { min-height: 3rem; border-radius: .6rem; background: var(--blue); color: #fff; font-weight: 600; }
  .make:disabled { opacity: .45; }
  .grow { flex: 1; min-width: 0; }

  .colmenu { display: grid; gap: .75rem; }
  .kind { display: grid; gap: .3rem; font-size: .9rem; color: var(--muted); }
  .opts { display: grid; gap: .6rem; }
  .opt { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: .3rem; }
  .sw { width: 1.6rem; height: 1.6rem; border-radius: 50%; }
  .x { display: grid; place-items: center; width: 2.75rem; height: 2.75rem; border-radius: .3rem; color: var(--muted); }
  .x:hover { background: var(--soft); color: var(--red); }
  .addopt, .item { display: flex; align-items: center; gap: .75rem; min-height: 3.1rem; padding-inline: .85rem; border: 1px solid var(--line); border-radius: .75rem; font-weight: 500; text-align: start; }
  .addopt { color: var(--muted); }
  .addopt:hover, .item:hover { background: var(--soft); }
  .item.danger { color: var(--red); }
</style>
