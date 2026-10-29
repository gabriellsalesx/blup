<script lang="ts">
  import { Clipboard } from "@wailsio/runtime";
  import { Attachments, AddAttachments, RenameAttachment, DeleteAttachment, OpenAttachment, Thumbnail } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Attachment } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Saved from "./Saved.svelte";
  import { saver } from "./saving.svelte";
  import { t } from "./i18n";
  import Upload from "@lucide/svelte/icons/upload";
  import FileText from "@lucide/svelte/icons/file-text";
  import Music from "@lucide/svelte/icons/music";
  import Video from "@lucide/svelte/icons/video";
  import FileIcon from "@lucide/svelte/icons/file";
  import X from "@lucide/svelte/icons/x";

  // The attachments folder as a gallery: images, PDFs, audio and video the
  // notes use. A file opens in a window with where it's used, rename, copy
  // (to paste in a note), open and delete.
  let { onOpenNote }: { onOpenNote: (path: string) => void } = $props();
  const an = t.attach;

  let list = $state<Attachment[]>([]);
  let thumbs = $state<Record<string, string>>({});
  let filter = $state<"all" | "image" | "other" | "unused">("all");
  let query = $state("");
  let open = $state<Attachment | null>(null);
  let newName = $state("");
  let asking = $state(false);
  let copied = $state(false);
  let loaded = $state(false);
  const s = saver();

  async function load() {
    list = (await Attachments()) ?? [];
    loaded = true;
    for (const a of list) if (a.kind === "image" && thumbs[a.name] === undefined) thumbs[a.name] = await Thumbnail(a.name).catch(() => "");
  }
  load();

  const shown = $derived(list.filter((a) =>
    (filter === "all" || (filter === "image" ? a.kind === "image" : filter === "other" ? a.kind !== "image" : !a.uses?.length)) &&
    a.name.toLowerCase().includes(query.trim().toLowerCase())));
  const size = (n: number) => (n < 1024 * 1024 ? `${Math.max(1, Math.round(n / 1024))} KB` : `${(n / 1024 / 1024).toFixed(1)} MB`);
  const stem = (n: string) => n.replace(/\.[^.]+$/, "");
  const ext = (n: string) => n.match(/\.[^.]+$/)?.[0] ?? "";
  const icons = { pdf: FileText, audio: Music, video: Video, other: FileIcon } as const;

  function show(a: Attachment) { open = a; newName = stem(a.name); asking = false; copied = false; }
  const add = () => s.run(async () => { await AddAttachments(); await load(); });
  const rename = () => s.run(async () => {
    if (!open || newName.trim() === stem(open.name)) return;
    const old = open.name;
    const name = await RenameAttachment(old, newName);
    thumbs[name] = thumbs[old];
    await load();
    open = list.find((a) => a.name === name) ?? null;
  });
  const remove = () => s.run(async () => { if (!open) return; await DeleteAttachment(open.name); open = null; await load(); });
  async function copy() {
    if (!open) return;
    await Clipboard.SetText(open.kind === "image" ? `![[${open.name}]]` : `[[${open.name}]]`);
    copied = true;
  }
</script>

<div class="panel">
  <h1>{an.title}</h1>
  <p class="lead">{an.lead}</p>

  <div class="bar-row">
    <div class="tabs" role="tablist">
      {#each [["all", an.all], ["image", an.images], ["other", an.others], ["unused", an.unused]] as [k, label] (k)}
        <button role="tab" aria-selected={filter === k} class:on={filter === k} onclick={() => (filter = k as typeof filter)}>{label}</button>
      {/each}
    </div>
    <input type="text" bind:value={query} placeholder={an.search} />
    <button class="btn" onclick={add}><Upload size={16} />{an.add}</button>
  </div>
  <Saved state={s.state} />

  {#if loaded && !shown.length}
    <p class="empty muted">{list.length ? an.none : an.empty}</p>
  {:else}
    <div class="grid">
      {#each shown as a (a.name)}
        {@const Icon = icons[a.kind as keyof typeof icons] ?? FileIcon}
        <button class="tile" onclick={() => show(a)} title={a.name}>
          <span class="thumb">{#if a.kind === "image" && thumbs[a.name]}<img src={thumbs[a.name]} alt="" loading="lazy" />{:else}<Icon size={30} strokeWidth={1.5} />{/if}</span>
          <span class="name">{a.name}</span>
          <small class:unused={!a.uses?.length}>{a.uses?.length ? an.usedIn.replace("{n}", String(a.uses.length)) : an.notUsed} · {size(a.size)}</small>
        </button>
      {/each}
    </div>
  {/if}
</div>

{#if open}
  {@const Icon = icons[open.kind as keyof typeof icons] ?? FileIcon}
  <div class="shade" role="presentation" onclick={() => (open = null)}></div>
  <div class="dialog" role="dialog" aria-modal="true" aria-label={open.name}>
    <header class="head">
      <h2>{open.name}</h2>
      <button class="close" title={an.close} aria-label={an.close} onclick={() => (open = null)}><X size={18} /></button>
    </header>
    <div class="big">{#if open.kind === "image" && thumbs[open.name]}<img src={thumbs[open.name]} alt="" />{:else}<Icon size={48} strokeWidth={1.4} />{/if}</div>
    <label class="field"><span>{an.name}</span>
      <div class="row"><input type="text" bind:value={newName} onkeydown={(e) => e.key === "Enter" && rename()} /><span class="muted">{ext(open.name)}</span>
        <button class="btn ghost" disabled={!newName.trim() || newName.trim() === stem(open.name)} onclick={rename}>{an.rename}</button></div>
      <small>{an.renameHelp}</small>
    </label>
    <div class="field"><span>{an.uses}</span>
      {#if open.uses?.length}
        <ul class="uses">{#each open.uses as u (u)}<li><button class="link" onclick={() => { open = null; onOpenNote(u); }}>{u.replace(/\.md$/, "")}</button></li>{/each}</ul>
      {:else}<small>{an.notUsedHelp}</small>{/if}
    </div>
    <div class="actions">
      {#if !asking}
        <button class="btn ghost danger-text" onclick={() => (asking = true)}>{an.delete}</button>
        <span class="grow"></span>
        <button class="btn ghost" onclick={() => open && OpenAttachment(open.name)}>{an.openApp}</button>
        <button class="btn" onclick={copy}>{copied ? an.copied : an.copy}</button>
      {:else}
        <!-- Asking to delete: only the question and its answers. -->
        <span class="q">{open.uses?.length ? an.deleteUsedQ.replace("{n}", String(open.uses.length)) : an.deleteQ}</span>
        <span class="grow"></span>
        <button class="btn ghost" onclick={() => (asking = false)}>{t.sec.cancel}</button>
        <button class="btn danger" onclick={remove}>{an.delete}</button>
      {/if}
    </div>
  </div>
{/if}

<style>
  .bar-row { display: flex; flex-wrap: wrap; gap: .6rem; align-items: center; }
  .bar-row input { flex: 1; min-width: 10rem; }
  .tabs { display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--line); border-radius: .6rem; }
  .tabs button { padding: .3rem .7rem; border-radius: .45rem; font-size: .85rem; font-weight: 600; color: var(--muted); }
  .tabs button.on { background: var(--soft); color: var(--text); }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr)); gap: .75rem; }
  .tile { display: grid; gap: .3rem; padding: .5rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); text-align: start; }
  .tile:hover { border-color: var(--blue); }
  .thumb { display: grid; place-items: center; aspect-ratio: 4 / 3; border-radius: .4rem; overflow: hidden; background: var(--soft); color: var(--muted); }
  .thumb img { width: 100%; height: 100%; object-fit: cover; }
  .name { font-size: .82rem; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .tile small { font-size: .72rem; color: var(--muted); }
  .tile small.unused { color: var(--red); }
  .empty { padding: 2rem; border: 1px dashed var(--line); border-radius: .75rem; text-align: center; }
  .shade { position: fixed; inset: 0; z-index: 40; background: rgb(0 0 0 / .45); }
  /* Centered in the window, like the settings. */
  .dialog { position: fixed; z-index: 41; inset: 0; margin: auto; width: min(34rem, calc(100% - 2rem)); height: fit-content; max-height: calc(100% - 2rem); overflow: auto; display: grid; gap: 1rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  /* The name and the X stay at the top while the rest scrolls. */
  .head { position: sticky; z-index: 1; inset-block-start: -1.25rem; display: flex; align-items: center; gap: .75rem; margin: -1.25rem -1.25rem -.25rem; padding: 1.1rem 1.25rem .6rem; background: var(--panel); border-block-end: 1px solid var(--line); }
  .head h2 { flex: 1; min-width: 0; font-size: 1rem; font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .close { flex: none; display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  .big { display: grid; place-items: center; min-height: 8rem; max-height: 18rem; border-radius: .6rem; overflow: hidden; background: var(--soft); color: var(--muted); }
  .big img { max-width: 100%; max-height: 18rem; object-fit: contain; }
  .uses { list-style: none; padding: 0; display: grid; gap: .2rem; font-size: .875rem; }
  .link { color: var(--blue); text-align: start; }
  .link:hover { text-decoration: underline; }
  .actions { display: flex; flex-wrap: wrap; gap: .5rem; align-items: center; }
  .grow { flex: 1; }
  .q { font-size: .85rem; color: var(--muted); }
  .danger-text { color: var(--red); }
  .btn.danger { background: var(--red); color: #fff; }
</style>
