<script lang="ts">
  import { Clipboard } from "@wailsio/runtime";
  import { RestoreSection, MoveItem, CopyNote, DeleteItem, NoteLink, OpenInObsidian, ShowInFolder } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { Node } from "../bindings/github.com/gabriellsalesx/blup/studio/models";
  import FilePlus from "@lucide/svelte/icons/file-plus";
  import FolderPlus from "@lucide/svelte/icons/folder-plus";
  import Copy from "@lucide/svelte/icons/copy";
  import FolderInput from "@lucide/svelte/icons/folder-input";
  import Link from "@lucide/svelte/icons/link";
  import External from "@lucide/svelte/icons/external-link";
  import FolderOpen from "@lucide/svelte/icons/folder-open";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash from "@lucide/svelte/icons/trash-2";
  import History from "@lucide/svelte/icons/history";
  import PenTool from "@lucide/svelte/icons/pen-tool";
  import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
  import { t } from "./i18n";
  import type { NewKind } from "./boards";
  import { toast, toastError } from "./toast.svelte";

  // The right-click menu of the file explorer, and its small dialogs
  // (move to, delete). Renaming happens in the tree itself (onRename).
  let { node, x, y, current, folders, boards = false, onClose, onChanged, onNew, onRename, onHistory }: {
    node: Node; x: number; y: number; current: string; folders: string[];
    onClose: () => void;
    onChanged: (change: { from: string; to: string | null; open?: string }) => void;
    boards?: boolean; // a vault of notes: new canvases and drawings too
    onNew: (kind: NewKind, folder: string) => void;
    onRename: (node: Node) => void;
    // Earlier versions on Blup Cloud (only when the plan keeps them).
    onHistory?: (path: string) => void;
  } = $props();
  const m = t.menu;

  let dialog = $state<"" | "move" | "delete">("");
  let filter = $state("");
  let error = $state("");
  let copied = $state(false);
  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");

  async function run(fn: () => Promise<void>) {
    error = "";
    try { await fn(); } catch (e) { error = msg(e); }
  }
  const base = (p: string) => p.split("/").pop()!.replace(/\.(draw|canvas|table|md)$/i, "");
  const move = (f: string) => run(async () => { const to = await MoveItem(node.path, f); onChanged({ from: node.path, to }); toast(t.toast.noteMoved.replace("{folder}", f || m.top)); });
  // To the Blup trash, with Undo in the toast (the tree follows the files).
  const remove = () => run(async () => {
    const where = await DeleteItem(node.path);
    onChanged({ from: node.path, to: null });
    toast(t.toast.noteDeleted.replace("{name}", node.name), { action: { label: t.toast.undo, run: () => RestoreSection(where).catch(toastError) } });
  });
  const copy = () => run(async () => { const to = await CopyNote(node.path); onChanged({ from: "", to, open: to }); toast(t.toast.noteCopied.replace("{name}", base(to))); });
  async function copyLink() {
    await Clipboard.SetText(await NoteLink(node.path, current));
    copied = true;
    toast(t.toast.linkCopied);
    setTimeout(onClose, 700);
  }
  const parent = node.path.includes("/") ? node.path.slice(0, node.path.lastIndexOf("/")) : "";
  const targets = $derived(["", ...folders].filter((f) => f !== parent && f !== node.path && !(node.dir && f.startsWith(node.path + "/")) && f.toLowerCase().includes(filter.toLowerCase())));
  const focus = (el: HTMLInputElement) => { el.focus(); el.select(); };

  // Keep the menu inside the window.
  let menuEl = $state<HTMLDivElement>();
  let pos = $state({ x, y });
  $effect(() => {
    if (!menuEl) return;
    const r = menuEl.getBoundingClientRect();
    pos = { x: Math.min(x, innerWidth - r.width - 8), y: Math.min(y, innerHeight - r.height - 8) };
  });
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onClose()} />

<div class="shade" class:dim={dialog} role="presentation" onclick={onClose} oncontextmenu={(e) => { e.preventDefault(); onClose(); }}></div>

{#if !dialog}
  <div class="menu" bind:this={menuEl} style:left="{pos.x}px" style:top="{pos.y}px" role="menu">
    {#if node.dir}
      <button onclick={() => { onNew("note", node.path); onClose(); }}><FilePlus size={16} />{m.newNote}</button>
      {#if boards}
        <button onclick={() => { onNew("canvas", node.path); onClose(); }}><LayoutDashboard size={16} />{t.boards.newCanvas}</button>
        <button onclick={() => { onNew("drawing", node.path); onClose(); }}><PenTool size={16} />{t.boards.newDrawing}</button>
      {/if}
      <button onclick={() => { onNew("folder", node.path); onClose(); }}><FolderPlus size={16} />{m.newFolder}</button>
      <hr />
    {:else}
      {#if !node.path.toLowerCase().endsWith(".table")}<button onclick={copy}><Copy size={16} />{m.copy}</button>{/if}
    {/if}
    <button onclick={() => (dialog = "move")}><FolderInput size={16} />{m.move}</button>
    {#if !node.dir}
      <button onclick={copyLink}><Link size={16} />{copied ? m.copied : m.link}</button>
      {#if onHistory}<button onclick={() => { onHistory(node.path); onClose(); }}><History size={16} />{t.history.menu}</button>{/if}
    {/if}
    <hr />
    {#if node.dir || /\.(md|canvas)$/i.test(node.path)}<button onclick={() => { OpenInObsidian(node.path); onClose(); }}><External size={16} />{m.obsidian}</button>{/if}
    <button onclick={() => { ShowInFolder(node.path); onClose(); }}><FolderOpen size={16} />{m.show}</button>
    <hr />
    <button onclick={() => { onRename(node); onClose(); }}><Pencil size={16} />{m.rename}</button>
    <button class="danger" onclick={() => (dialog = "delete")}><Trash size={16} />{m.delete}</button>
  </div>
{:else}
  <div class="dialog" role="dialog" aria-modal="true">
    {#if dialog === "move"}
      <h2>{m.moveTitle.replace("{name}", node.name)}</h2>
      <input type="text" bind:value={filter} placeholder={m.filter} use:focus />
      <ul>
        {#each targets as f (f)}<li><button onclick={() => move(f)}><FolderInput size={15} />{f || m.top}</button></li>{/each}
      </ul>
    {:else}
      <h2>{m.deleteTitle.replace("{name}", node.name)}</h2>
      <p class="muted">{m.deleteHelp}</p>
    {/if}
    {#if error}<p class="bad">{error}</p>{/if}
    <div class="actions">
      <button class="btn ghost" onclick={onClose}>{t.sec.cancel}</button>
      {#if dialog === "delete"}<button class="btn danger-btn" onclick={remove}>{m.delete}</button>{/if}
    </div>
  </div>
{/if}

<style>
  .shade { position: fixed; inset: 0; z-index: 40; }
  .shade.dim { background: rgb(0 0 0 / .35); }
  .menu { position: fixed; z-index: 41; display: grid; min-width: 15rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .22); }
  .menu button { display: flex; align-items: center; gap: .6rem; padding: .42rem .6rem; border-radius: .35rem; font-size: .875rem; text-align: start; }
  .menu button > :global(*) { pointer-events: none; } /* the whole row takes the click */
  .menu button :global(svg) { color: var(--muted); }
  .menu button:hover { background: var(--soft); }
  .menu .danger, .menu .danger :global(svg) { color: var(--red); }
  hr { border: 0; border-block-start: 1px solid var(--line); margin: .25rem .2rem; }
  .dialog { position: fixed; z-index: 41; inset-block-start: 18%; inset-inline: 0; margin-inline: auto; width: min(28rem, calc(100% - 2rem)); display: grid; gap: .75rem; padding: 1.25rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); box-shadow: 0 20px 50px rgb(0 0 0 / .3); }
  .dialog h2 { font-size: 1rem; font-weight: 700; }
  .dialog p { font-size: .85rem; }
  .dialog ul { list-style: none; padding: 0; max-height: 16rem; overflow: auto; }
  .dialog li button { display: flex; align-items: center; gap: .5rem; width: 100%; padding: .4rem .5rem; border-radius: .35rem; font-size: .875rem; text-align: start; }
  .dialog li button:hover { background: var(--soft); }
  .danger-btn { background: var(--red); color: #fff; }
</style>
