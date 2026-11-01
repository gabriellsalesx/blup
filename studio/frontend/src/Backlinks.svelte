<script lang="ts">
  import { onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { Mentions } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Mention } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import { t } from "./i18n";

  // "Mentioned in": the notes (and canvases) with a link to this one, under
  // the editor, each with the lines where it mentions it (the mention in
  // blue); a click opens one. It starts folded; open or folded, as
  // the user leaves it, it stays so (on this computer).
  let { note, onOpenNote }: { note: string; onOpenNote: (path: string) => void } = $props();
  const tb = t.backlinks;

  let from = $state<Mention[]>([]);
  let open = $state((() => { try { return localStorage.getItem("blup.backlinks") === "open"; } catch { return false; } })());
  // Read again when notes change on disk (a link added or taken out).
  let changes = $state(0);
  let wait: ReturnType<typeof setTimeout> | undefined;
  const off = Events.On("notesChanged", () => { clearTimeout(wait); wait = setTimeout(() => changes++, 400); });
  onDestroy(() => { off(); clearTimeout(wait); });
  $effect(() => {
    const me = note;
    changes;
    Mentions(me).then((list) => {
      if (me === note) from = list ?? [];
    }).catch(() => (from = []));
  });
  function toggle() {
    open = !open;
    try { localStorage.setItem("blup.backlinks", open ? "open" : "closed"); } catch {}
  }
</script>

<section class="back" aria-label={tb.title}>
  <button class="head" onclick={toggle} aria-expanded={open}>
    <span class={["chev", { open }]}><ChevronRight size={14} /></span>{tb.title} <span class="n">{from.length}</span>
  </button>
  {#if open}
    {#if from.length}
      <ul>
        {#each from as n (n.path)}
          <li><button onclick={() => onOpenNote(n.path)}>
            <span class="who"><b>{n.name}</b>{#if n.folder}<small class="muted">{n.folder}</small>{/if}</span>
            {#each n.lines ?? [] as l, i (i)}<span class="line">{l.before}<mark>{l.link}</mark>{l.after}</span>{/each}
          </button></li>
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
  li button { display: grid; gap: .15rem; width: 100%; padding: .35rem .5rem; border-radius: .35rem; text-align: start; font-size: .88rem; }
  .who { display: flex; align-items: baseline; gap: .5rem; }
  .line { color: var(--muted); font-size: .82rem; line-height: 1.45; }
  .line mark { background: none; color: var(--blue); font-weight: 600; }
  li button:hover { background: var(--soft); }
  li b { font-weight: 500; color: var(--blue); }
  li small { font-size: .75rem; }
  .none { padding: .3rem 1.5rem; font-size: .85rem; }
</style>
