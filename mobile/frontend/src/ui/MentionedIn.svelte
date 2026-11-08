<script lang="ts">
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import type { Mention } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import { NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app } from "../lib/app.svelte";
  import { t } from "../desk/i18n";

  // "Mencionada em", at the foot of the note, as in the Studio: the notes with
  // a link to this one and the lines where they have it. It stays as left.
  let { id }: { id: string } = $props();

  let from = $state<Mention[]>([]);
  let open = $state((() => { try { return localStorage.getItem("blup.backlinks") === "open"; } catch { return false; } })());
  $effect(() => {
    const me = id;
    void app.notes; // read again when the notes change
    NotesService.Mentions(me).then((list) => { if (me === id) from = list ?? []; }).catch(() => (from = []));
  });
  function toggle() {
    open = !open;
    try { localStorage.setItem("blup.backlinks", open ? "open" : "closed"); } catch {}
  }
</script>

<section aria-label={t.backlinks.title}>
  <button type="button" class="head" aria-expanded={open} onclick={toggle}>
    <span class={["chev", { open }]}><ChevronRight size={16} /></span>{t.backlinks.title} <span class="n">{from.length}</span>
  </button>
  {#if open}
    <div class="list">
      {#each from as n (n.path)}
        <button type="button" class="who" onclick={() => app.go({ note: n.path })}>
          <b>{n.name}</b>
          {#each (n.lines ?? []).slice(0, 2) as l, i (i)}<span class="line">{l.before}<mark>{l.link}</mark>{l.after}</span>{/each}
        </button>
      {:else}
        <p>{t.backlinks.none}</p>
      {/each}
    </div>
  {/if}
</section>

<style>
  section { flex: none; display: grid; gap: .25rem; margin-inline: 1.5rem; padding-block: .6rem 1.5rem; border-block-start: 1px solid var(--line); }
  .head { display: flex; align-items: center; gap: .4rem; min-height: 2.5rem; font-size: .9rem; font-weight: 600; color: var(--muted); }
  .chev { display: grid; transition: rotate .15s; }
  .chev.open { rotate: 90deg; }
  .n { padding-inline: .45rem; border-radius: 999px; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); font-size: .8rem; }
  /* Many mentions don't push the note away: the list stops at a height and
     scrolls inside, as in the Studio. */
  .list { display: grid; align-content: start; max-height: min(18rem, 38vh); overflow-y: auto; overscroll-behavior: contain; touch-action: pan-y; }
  .who { display: grid; gap: .15rem; padding-block: .45rem; text-align: start; }
  .who b { font-weight: 600; color: var(--blue-text); }
  .line { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; line-clamp: 2; overflow: hidden; font-size: .85rem; line-height: 1.45; color: var(--muted); }
  mark { background: none; color: var(--blue-text); font-weight: 600; }
  p { font-size: .85rem; color: var(--muted); }
</style>
