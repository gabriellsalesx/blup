<script lang="ts">
  import { onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { NoteGraph } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import { t } from "./i18n";

  // "Mentioned in": the notes (and canvases) with a link to this one, under
  // the editor; a click opens one. It always starts open.
  let { note, onOpenNote }: { note: string; onOpenNote: (path: string) => void } = $props();
  const tb = t.backlinks;

  let from = $state<{ path: string; name: string; folder: string }[]>([]);
  let open = $state(true);
  // Read again when notes change on disk (a link added or taken out).
  let changes = $state(0);
  let wait: ReturnType<typeof setTimeout> | undefined;
  const off = Events.On("notesChanged", () => { clearTimeout(wait); wait = setTimeout(() => changes++, 400); });
  onDestroy(() => { off(); clearTimeout(wait); });
  $effect(() => {
    const me = note;
    changes;
    NoteGraph().then((g) => {
      if (me !== note) return;
      const names = new Map((g.nodes ?? []).map((n) => [n.path, n]));
      from = (g.edges ?? []).filter((e) => e.to === me).map((e) => names.get(e.from)!).filter(Boolean);
    }).catch(() => (from = []));
  });
  const toggle = () => (open = !open);
</script>

<section class="back" aria-label={tb.title}>
  <button class="head" onclick={toggle} aria-expanded={open}>
    <span class={["chev", { open }]}><ChevronRight size={14} /></span>{tb.title} <span class="n">{from.length}</span>
  </button>
  {#if open}
    {#if from.length}
      <ul>
        {#each from as n (n.path)}
          <li><button onclick={() => onOpenNote(n.path)}><b>{n.name}</b>{#if n.folder}<small class="muted">{n.folder}</small>{/if}</button></li>
        {/each}
      </ul>
    {:else}
      <p class="muted none">{tb.none}</p>
    {/if}
  {/if}
</section>

<style>
  .back { max-height: 40%; overflow: auto; border-block-start: 1px solid var(--line); padding: .35rem 1rem .5rem; background: var(--panel); }
  .head { display: flex; align-items: center; gap: .35rem; padding: .25rem .3rem; border-radius: .35rem; font-size: .85rem; font-weight: 600; color: var(--muted); }
  .head:hover { color: var(--text); }
  .chev { display: inline-grid; transition: transform .15s; }
  .chev.open { transform: rotate(90deg); }
  .n { padding: 0 .4rem; border-radius: 999px; background: var(--soft); font-size: .75rem; }
  ul { list-style: none; padding: .2rem 0 0 1.2rem; display: grid; gap: .1rem; }
  li button { display: flex; align-items: baseline; gap: .5rem; width: 100%; padding: .3rem .5rem; border-radius: .35rem; text-align: start; font-size: .88rem; }
  li button:hover { background: var(--soft); }
  li b { font-weight: 500; color: var(--blue); }
  li small { font-size: .75rem; }
  .none { padding: .3rem 1.5rem; font-size: .85rem; }
</style>
