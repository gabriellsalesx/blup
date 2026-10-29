<script lang="ts">
  import { Clipboard } from "@wailsio/runtime";
  import { RenameItem, MoveItem, CopyNote, DeleteItem, NoteLink, OpenInObsidian, ShowInFolder } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
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
  import { t } from "./i18n";

  // The right-click menu of the file explorer, and its small dialogs
  // (rename, move to, delete).
  let { node, x, y, current, folders, onClose, onChanged, onNew }: {
    node: Node; x: number; y: number; current: string; folders: string[];
    onClose: () => void;
    onChanged: (change: { from: string; to: string | null; open?: string }) => void;
    onNew: (kind: "note" | "folder", folder: string) => void;
  } = $props();
  const m = t.menu;

  let dialog = $state<"" | "rename" | "move" | "delete">("");
  let name = $state(node.name);
  let filter = $state("");
  let error = $state("");
  let copied = $state(false);
  const msg = (e: unknown) => String(e).replace(/^(Runtime)?Error: /, "");

  async function run(fn: () => Promise<void>) {
    error = "";
    try { await fn(); } catch (e) { error = msg(e); }
  }
  const rename = () => run(async () => { const to = await RenameItem(node.path, name); onChanged({ from: node.path, to }); });
  const move = (f: string) => run(async () => { const to = await MoveItem(node.path, f); onChanged({ from: node.path, to }); });
  const remove = () => run(async () => { await DeleteItem(node.path); onChanged({ from: node.path, to: null }); });
  const copy = () => run(async () => { const to = await CopyNote(node.path); onChanged({ from: "", to, open: to }); });
  async function copyLink() {
    await Clipboard.SetText(await NoteLink(node.path, current));
    copied = true;
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
      <button onclick={() => { onNew("folder", node.path); onClose(); }}><FolderPlus size={16} />{m.newFolder}</button>
      <hr />
    {:else}
      <button onclick={copy}><Copy size={16} />{m.copy}</button>
    {/if}
    <button onclick={() => (dialog = "move")}><FolderInput size={16} />{m.move}</button>
    {#if !node.dir}
      <button onclick={copyLink}><Link size={16} />{copied ? m.copied : m.link}</button>
    {/if}
    <hr />
    <button onclick={() => { OpenInObsidian(node.path); onClose(); }}><External size={16} />{m.obsidian}</button>
    <button onclick={() => { ShowInFolder(node.path); onClose(); }}><FolderOpen size={16} />{m.show}</button>
    <hr />
    <button onclick={() => (dialog = "rename")}><Pencil size={16} />{m.rename}</button>
    <button class="danger" onclick={() => (dialog = "delete")}><Trash size={16} />{m.delete}</button>
  </div>
{:else}
  <div class="dialog" role="dialog" aria-modal="true">
    {#if dialog === "rename"}
      <h2>{m.renameTitle}</h2>
      <input type="text" bind:value={name} use:focus onkeydown={(e) => e.key === "Enter" && rename()} />
      <p class="muted">{m.renameHelp}</p>
    {:else if dialog === "move"}
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
      {#if dialog === "rename"}<button class="btn" onclick={rename}>{m.rename}</button>{/if}
      {#if dialog === "delete"}<button class="btn danger-btn" onclick={remove}>{m.delete}</button>{/if}
    </div>
  </div>
{/if}

<style>
  .shade { position: fixed; inset: 0; z-index: 40; }
  .shade.dim { background: rgb(0 0 0 / .35); }
  .menu { position: fixed; z-index: 41; display: grid; min-width: 15rem; padding: .3rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .22); }
  .menu button { display: flex; align-items: center; gap: .6rem; padding: .42rem .6rem; border-radius: .35rem; font-size: .875rem; text-align: start; }
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
