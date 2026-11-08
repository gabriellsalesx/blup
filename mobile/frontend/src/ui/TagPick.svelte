<script lang="ts">
  import { untrack } from "svelte";
  import Check from "@lucide/svelte/icons/check";
  import Plus from "@lucide/svelte/icons/plus";
  import { NotesService, type Note } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app } from "../lib/app.svelte";
  import Sheet from "./Sheet.svelte";
  import { fmt, t } from "../desk/i18n";

  // Picks the note's tags, or makes one. A #tag written in the text is on
  // because of the text: it changes there.
  let { note, onchange }: { note: Note; onchange: (tags: string[]) => void } = $props();

  let query = $state("");
  const own = $derived(note.propTags ?? []);
  const clean = (v: string) => v.trim().replace(/^#/, "").replace(/\s+/g, "-");
  const q = $derived(clean(query).toLowerCase());
  // The note's tags come first, and stay put while others are marked.
  const had = untrack(() => note.tags ?? []);
  const known = $derived([...new Set([...app.notes.flatMap((n) => n.tags ?? []), ...app.extraTags])].sort((a, b) => Number(had.includes(b)) - Number(had.includes(a)) || a.localeCompare(b, t.locale)));
  const shown = $derived(known.filter((t) => t.toLowerCase().includes(q)));
  const inText = (t: string) => (note.tags ?? []).includes(t) && !own.includes(t);

  function toggle(tag: string) {
    if (inText(tag)) return;
    NotesService.Haptic();
    onchange(own.includes(tag) ? own.filter((t) => t !== tag) : [...own, tag]);
  }
  function create() {
    onchange([...own, clean(query)]);
    query = "";
  }
</script>

<Sheet title={t.tagPick.title} tall onclose={() => app.back()}>
  {#snippet top()}<input type="text" bind:value={query} placeholder={t.common.searchOrCreate} aria-label={t.tagPick.search} autocapitalize="off" />{/snippet}
  {#if q && !known.some((t) => t.toLowerCase() === q)}
    <button type="button" class="row" onclick={create}><span class="tile"><Plus size={19} /></span><span class="name">{fmt(t.tagPick.newNamed, { name: clean(query) })}</span></button>
  {/if}
  <div class="list">
    {#each shown as tag (tag)}
      {@const on = (note.tags ?? []).includes(tag)}
      <button type="button" class="row" role="checkbox" aria-checked={on} aria-disabled={inText(tag)} onclick={() => toggle(tag)}>
        <span class="pill">{tag}</span>{#if inText(tag)}<small>{t.tagPick.inText}</small>{/if}<span class={["box", { on }]}>{#if on}<Check size={16} strokeWidth={3} />{/if}</span>
      </button>
    {:else}
      {#if !q}<p>{t.tagPick.first}</p>{/if}
    {/each}
  </div>
</Sheet>

<style>
  .list { display: grid; }
  .row { display: flex; align-items: center; gap: .75rem; width: 100%; min-height: 3.1rem; text-align: start; }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .tile { flex: none; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .pill { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: .3rem .65rem; border-radius: .5rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); font-size: .9rem; font-weight: 500; }
  small { font-size: .8rem; color: var(--muted); }
  .box { flex: none; display: grid; place-items: center; width: 1.6rem; height: 1.6rem; margin-inline-start: auto; border: 1.5px solid var(--line); border-radius: .45rem; }
  .box.on { border-color: var(--blue); background: var(--blue); color: #fff; }
  [aria-disabled="true"] .box { opacity: .4; }
  p { font-size: .9rem; color: var(--muted); }
</style>
