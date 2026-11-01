<script lang="ts">
  import { Tags } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import CalendarPlus from "@lucide/svelte/icons/calendar-plus";
  import CalendarClock from "@lucide/svelte/icons/calendar-clock";
  import Tag from "@lucide/svelte/icons/tag";
  import X from "@lucide/svelte/icons/x";
  import type { DBSchema } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import FieldInput from "./FieldInput.svelte";
  import { t } from "./i18n";

  // A vault note's properties, like Obsidian's, and only these: when it was
  // made, when it changed last (both filled in by themselves, see
  // Workspace.save, and editable), and its tags. onEdit changes them (the
  // workspace keeps props up to date).
  // A record of a database (database): its fields come after, each by its
  // column's kind.
  let { props, database, notes = [], onEdit, onNewOption, onOpenNote }: {
    props: Record<string, any>; database?: DBSchema;
    notes?: { name: string; path: string }[];
    onEdit: (key: string, value: any) => void;
    onNewOption?: (key: string, value: string, color: string) => void;
    onOpenNote?: (path: string) => void;
  } = $props();
  const p = t.noteProps;

  const day = (v: unknown) => String(v ?? "").slice(0, 10);
  const list = (v: any): string[] => (Array.isArray(v) ? v.map(String) : typeof v === "string" && v ? v.split(",").map((x) => x.trim()) : []).map((x) => x.replace(/^#/, "")).filter(Boolean);
  const tags = $derived(list(props.tags));

  // Every tag of the vault, for the suggestions (read once).
  let known = $state<string[]>([]);
  Tags().then((x: string[] | null) => (known = x ?? [])).catch(() => {});
  let draft = $state("");
  const suggestions = $derived(known.filter((k) => !tags.includes(k) && k.toLowerCase().includes(draft.trim().replace(/^#/, "").toLowerCase())).slice(0, 8));

  function setDate(key: string, v: string) {
    onEdit(key, v);
  }
  function setTags(next: string[]) {
    onEdit("tags", next);
  }
  function addTag(v = draft) {
    const tag = v.trim().replace(/^#/, "").replace(/\s+/g, "-");
    draft = "";
    if (tag && !tags.includes(tag)) setTags([...tags, tag]);
  }
  function key(e: KeyboardEvent) {
    if (e.key === "Enter" || e.key === ",") { e.preventDefault(); addTag(); }
    else if (e.key === "Backspace" && !draft && tags.length) setTags(tags.slice(0, -1));
  }
</script>

<section class="props" aria-label={p.title}>
  <h2>{p.title}</h2>
  <div class="row">
    <span class="k"><CalendarPlus size={16} />{p.created}</span>
    <input type="date" value={day(props.created)} onchange={(e) => setDate("created", e.currentTarget.value)} aria-label={p.created} />
  </div>
  <div class="row">
    <span class="k"><CalendarClock size={16} />{p.updated}</span>
    <input type="date" value={day(props.updated)} onchange={(e) => setDate("updated", e.currentTarget.value)} aria-label={p.updated} />
  </div>
  <div class="row">
    <span class="k"><Tag size={16} />{p.tags}</span>
    <div class="tags">
      {#each tags as tag (tag)}
        <span class="chip">{tag}<button title={p.remove.replace("{tag}", tag)} aria-label={p.remove.replace("{tag}", tag)} onclick={() => setTags(tags.filter((x) => x !== tag))}><X size={12} /></button></span>
      {/each}
      <span class="add">
        <input bind:value={draft} onkeydown={key} onblur={() => setTimeout(() => addTag(), 120)} placeholder={tags.length ? "" : p.addTag} aria-label={p.addTag} />
        {#if draft.trim() && suggestions.length}
          <ul class="sug" role="listbox">
            {#each suggestions as s (s)}<li><button onmousedown={(e) => { e.preventDefault(); addTag(s); }}>{s}</button></li>{/each}
          </ul>
        {/if}
      </span>
    </div>
  </div>
  {#each database?.columns ?? [] as c (c.key)}
    <div class="row">
      <span class="k">{c.name}</span>
      <div class="v"><FieldInput column={c} value={props[c.key]} {notes} {onOpenNote} onChange={(v) => onEdit(c.key, v)} onNewOption={onNewOption ? (v, color) => onNewOption(c.key, v, color) : undefined} /></div>
    </div>
  {/each}
</section>

<style>
  .props { display: grid; gap: .15rem; padding-block-end: .5rem; border-block-end: 1px solid var(--line); }
  h2 { font-size: 1rem; font-weight: 600; color: var(--muted); margin-block-end: .35rem; }
  .row { display: grid; grid-template-columns: 9.5rem minmax(0, 1fr); align-items: center; min-height: 2.1rem; }
  .k { display: flex; align-items: center; gap: .55rem; color: var(--muted); font-size: .92rem; }
  .v { min-width: 0; font-size: .92rem; }
  input[type="date"] { justify-self: start; padding: .2rem .4rem; border: 1px solid transparent; border-radius: .35rem; background: transparent; color: var(--text); font: inherit; font-size: .92rem; color-scheme: light dark; }
  input[type="date"]:hover, input[type="date"]:focus { border-color: var(--line); outline: none; }
  .tags { display: flex; flex-wrap: wrap; align-items: center; gap: .35rem; }
  .chip { display: inline-flex; align-items: center; gap: .2rem; padding: .12rem .3rem .12rem .6rem; border-radius: 999px; background: color-mix(in srgb, var(--blue) 16%, transparent); color: light-dark(#1d4ed8, #93c5fd); font-size: .88rem; }
  .chip button { display: grid; place-items: center; padding: .1rem; border-radius: 999px; color: inherit; opacity: .75; }
  .chip button:hover { opacity: 1; background: color-mix(in srgb, currentColor 18%, transparent); }
  .add { position: relative; }
  .add input { width: 8rem; padding: .2rem .3rem; border: 0; outline: none; background: transparent; color: var(--text); font: inherit; font-size: .9rem; }
  .sug { position: absolute; z-index: 5; inset-block-start: 100%; inset-inline-start: 0; min-width: 10rem; list-style: none; padding: .25rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--panel); box-shadow: 0 10px 28px rgb(0 0 0 / .2); }
  .sug button { width: 100%; padding: .3rem .5rem; border-radius: .35rem; text-align: start; font-size: .88rem; }
  .sug button:hover { background: var(--soft); }
</style>
