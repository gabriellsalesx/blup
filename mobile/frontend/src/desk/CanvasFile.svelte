<script lang="ts">
  import { BoardFile, ReadNote } from "./studio";
  import FileText from "@lucide/svelte/icons/file-text";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import { cardHTML } from "./canvas";
  import { cardImages } from "./cardimages";
  import { t } from "./i18n";

  // A file of the vault on a canvas: a note shows its text, an image shows
  // itself, a drawing or another canvas shows its name (two clicks open it).
  let { file, version }: { file: string; version: number } = $props();

  const lower = $derived(file.toLowerCase());
  const image = $derived(/\.(png|jpe?g|webp|gif|svg|avif)$/.test(lower));
  const board = $derived(lower.endsWith(".canvas") ? "canvas" : lower.endsWith(".draw") ? "drawing" : "");
  const name = $derived(file.split("/").pop()!.replace(/\.(draw|canvas|table|md)$/i, ""));

  // Read again when the file changes on disk (version).
  const content = $derived.by(() => {
    version;
    if (image) return BoardFile(file).then((src) => ({ src, html: "" }));
    if (board || !lower.endsWith(".md")) return Promise.resolve({ src: "", html: "" });
    return ReadNote(file).then((n) => ({ src: "", html: cardHTML(n.body) }));
  });
</script>

{#await content}
  <div class="file"><b class="name">{name}</b></div>
{:then c}
  {#if c.src}
    <img src={c.src} alt={name} draggable="false" />
  {:else}
    <div class="file">
      <b class="name">
        {#if board === "drawing"}<PenTool size={15} />{:else if board === "canvas"}<LayoutDashboard size={15} />{:else}<FileText size={15} />{/if}
        {name}
      </b>
      {#if c.html}<div class="md" data-from={file} use:cardImages={{ from: file, html: c.html }}>{@html c.html}</div>{/if}
    </div>
  {/if}
{:catch}
  <div class="file missing"><b class="name">{name}</b><small>{t.boards.missing}</small></div>
{/await}

<style>
  .file { display: grid; align-content: start; gap: .4rem; height: 100%; }
  .name { display: flex; align-items: center; gap: .4rem; font-size: 1rem; }
  .missing small { color: var(--red); }
  img { display: block; width: 100%; height: 100%; object-fit: contain; border-radius: .4rem; }
</style>
