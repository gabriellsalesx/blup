<script lang="ts">
  import { NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app } from "../lib/app.svelte";
  import NoteRow from "../ui/NoteRow.svelte";
  import { plural, t } from "../desk/i18n";
  import { PAGE, nearEnd } from "../lib/paging";

  let query = $state("");
  const term = $derived(query.trim().toLowerCase().replace(/^#/, ""));
  const tags = $derived([...new Set(app.notes.flatMap((n) => n.tags ?? []))].sort());
  // The text of the notes stays in Go: it looks there, a moment after the
  // typing stops, and says which notes have the term.
  let ids = $state.raw<string[]>([]);
  let looking: ReturnType<typeof setTimeout> | undefined;
  function search(text: string) {
    query = text;
    clearTimeout(looking);
    const q = term;
    if (!q) { ids = []; return; }
    looking = setTimeout(() => {
      NotesService.Search(q).then((found) => { if (q === term) ids = found ?? []; }, () => {});
    }, 150);
  }
  const found = $derived.by(() => {
    const byId = new Map(app.notes.map((n) => [n.id, n]));
    return term ? ids.flatMap((id) => byId.get(id) ?? []) : [];
  });
  // A new search starts again from the first piece.
  let limit = $derived.by(() => (void term, PAGE));
  const rows = $derived(found.length > limit ? found.slice(0, limit) : found);
</script>

<header>
  <h1>{t.search.title}</h1>
  <input type="text" aria-label={t.search.placeholder} placeholder={t.search.placeholder} autocomplete="off" value={query} oninput={(e) => search(e.currentTarget.value)} />
</header>
<div class="list">
  {#if term}
    <p class="count">{plural(found.length, t.common.notes1, t.common.notesN)}</p>
    {#each rows as note (note.id)}<NoteRow {note} />{/each}
    {#if rows.length < found.length}{#key limit}<div class="more" {@attach nearEnd(() => (limit += PAGE * 2))}></div>{/key}{/if}
  {:else}
    <p class="count">{t.search.tags}</p>
    <div class="tags">
      {#each tags as tag (tag)}<button onclick={() => search(`#${tag}`)}>#{tag}</button>{/each}
    </div>
  {/if}
</div>

<style>
  header { flex: none; display: grid; gap: .75rem; padding: calc(.75rem + var(--sat)) 1rem .75rem; }
  h1 { font-size: 1.6rem; font-weight: 700; letter-spacing: -.02em; }
  .list { flex: 1; min-height: 0; overflow-y: auto; }
  .more { height: 1px; }
  .count { padding: .5rem 1rem; font-size: .8rem; font-weight: 600; color: var(--muted); }
  .tags { display: flex; flex-wrap: wrap; gap: .5rem; padding: 0 1rem; }
  .tags button { padding: .45rem .8rem; border-radius: 999px; font-size: .875rem; color: var(--blue-text); background: color-mix(in srgb, var(--blue) 12%, transparent); }
</style>
