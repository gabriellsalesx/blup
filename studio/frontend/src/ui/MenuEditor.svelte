<script lang="ts" module>
  import type { MenuItem } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";

  // The menu as the editor holds it: each top entry with its submenu.
  export type Entry = { id: number; name: string; target: string };
  export type Group = { id: number; item: Entry; kids: Entry[] };

  let next = 0;
  const entry = (m: { name: string; target: string }): Entry => ({ id: ++next, name: m.name, target: m.target });

  // From the saved lines (Sub: in the submenu of the entry above).
  export function toGroups(menu: MenuItem[]): Group[] {
    const out: Group[] = [];
    for (const m of menu) {
      if (m.sub && out.length) out[out.length - 1].kids.push(entry(m));
      else out.push({ id: ++next, item: entry(m), kids: [] });
    }
    return out;
  }

  export function toMenu(groups: Group[]): MenuItem[] {
    return groups.flatMap((g) => [
      { name: g.item.name, target: g.item.target, sub: false },
      ...g.kids.map((k) => ({ name: k.name, target: k.target, sub: true })),
    ]);
  }

  export const newGroup = (m: { name: string; target: string }): Group => ({ id: ++next, item: entry(m), kids: [] });
</script>

<script lang="ts">
  import { t } from "../i18n";
  import ArrowUp from "@lucide/svelte/icons/arrow-up";
  import ArrowDown from "@lucide/svelte/icons/arrow-down";
  import X from "@lucide/svelte/icons/x";
  import GripVertical from "@lucide/svelte/icons/grip-vertical";
  import CornerDownRight from "@lucide/svelte/icons/corner-down-right";
  import CornerUpLeft from "@lucide/svelte/icons/corner-up-left";

  // The top menu: entries in order, each one able to hold a submenu.
  // Dragging an entry onto the middle of another puts it in that one's
  // submenu (which then only opens it, without a link); onto its top or
  // bottom edge, before or after it. The buttons do the same without a
  // mouse.
  let { groups = $bindable([]) }: { groups: Group[] } = $props();

  const l = t.look;
  const isLink = (m: Entry) => /^(https?:|mailto:)/.test(m.target);

  type Zone = "before" | "after" | "into";
  let drag = $state<{ g: number; k?: number } | null>(null); // ids
  let over = $state<{ g: number; k?: number; zone: Zone } | null>(null);

  const gi = (id: number) => groups.findIndex((g) => g.id === id);
  const dragHasKids = () => !!drag && drag.k === undefined && (groups[gi(drag.g)]?.kids.length ?? 0) > 0;

  function start(e: DragEvent, g: number, k?: number) {
    drag = { g, k };
    e.dataTransfer?.setData("text/plain", "menu");
    if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
    const row = (e.currentTarget as HTMLElement).closest(".m-row");
    if (row && e.dataTransfer) e.dataTransfer.setDragImage(row, 12, 12);
  }
  function end() {
    drag = null;
    over = null;
  }

  // Where a drop over a row lands: a top entry takes the dragged one in its
  // middle; a submenu entry only before or after.
  function zoneOf(e: DragEvent, g: number, k?: number): Zone | null {
    if (!drag) return null;
    if (drag.g === g && drag.k === k) return null;
    if (k === undefined && drag.k === undefined && drag.g === g) return null;
    if (k !== undefined && dragHasKids()) return null; // one level only
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const y = (e.clientY - r.top) / r.height;
    if (k !== undefined || dragHasKids()) return y < 0.5 ? "before" : "after";
    return y < 0.25 ? "before" : y > 0.75 ? "after" : "into";
  }
  function dragOver(e: DragEvent, g: number, k?: number) {
    const zone = zoneOf(e, g, k);
    if (!zone) {
      over = null;
      return;
    }
    e.preventDefault();
    over = { g, k, zone };
  }

  // Takes the dragged entry out: a whole group (with its submenu) or one
  // entry of a submenu.
  function take(): Group | null {
    if (!drag) return null;
    const i = gi(drag.g);
    if (i < 0) return null;
    if (drag.k === undefined) return groups.splice(i, 1)[0];
    const j = groups[i].kids.findIndex((x) => x.id === drag!.k);
    if (j < 0) return null;
    return { id: ++next, item: groups[i].kids.splice(j, 1)[0], kids: [] };
  }

  function drop(e: DragEvent) {
    e.preventDefault();
    const o = over;
    const moved = o && take();
    end();
    if (!o || !moved) return;
    const i = gi(o.g);
    if (i < 0) return;
    if (o.k === undefined) {
      if (o.zone === "into") groups[i].kids.push(moved.item);
      else groups.splice(o.zone === "before" ? i : i + 1, 0, moved);
    } else {
      const j = groups[i].kids.findIndex((x) => x.id === o.k);
      groups[i].kids.splice(o.zone === "before" ? j : j + 1, 0, moved.item);
    }
  }

  function move(i: number, d: number) {
    const j = i + d;
    if (j >= 0 && j < groups.length) [groups[i], groups[j]] = [groups[j], groups[i]];
  }
  function moveKid(g: Group, i: number, d: number) {
    const j = i + d;
    if (j >= 0 && j < g.kids.length) [g.kids[i], g.kids[j]] = [g.kids[j], g.kids[i]];
  }
  // Into the submenu of the entry above.
  function indent(i: number) {
    if (i < 1 || groups[i].kids.length) return;
    const [g] = groups.splice(i, 1);
    groups[i - 1].kids.push(g.item);
  }
  // Out of the submenu, right after its group.
  function outdent(i: number, k: number) {
    const [item] = groups[i].kids.splice(k, 1);
    groups.splice(i + 1, 0, { id: ++next, item, kids: [] });
  }
  // A group removed leaves its submenu in the menu, as top entries.
  function remove(i: number) {
    const [g] = groups.splice(i, 1);
    groups.splice(i, 0, ...g.kids.map((item) => ({ id: ++next, item, kids: [] })));
  }

  const cls = (g: number, k?: number) => (over && over.g === g && over.k === k ? `drop-${over.zone}` : "");
</script>

<div class="m-list" role="list">
  {#each groups as g, i (g.id)}
    <div class="m-group" role="listitem">
      <div class="m-row {cls(g.id)}" class:dragging={drag?.g === g.id && drag.k === undefined}
        role="group" aria-label={g.item.name}
        ondragover={(e) => dragOver(e, g.id)} ondragleave={() => (over = null)} ondrop={drop}>
        <span class="grip" draggable="true" title={l.drag} aria-hidden="true"
          ondragstart={(e) => start(e, g.id)} ondragend={end}><GripVertical size={16} /></span>
        <input type="text" bind:value={g.item.name} aria-label={l.linkName} placeholder={g.kids.length || !g.item.target ? l.groupName : ""} />
        {#if g.kids.length || !g.item.target}
          <span class="target muted">{g.kids.length ? l.group : l.groupEmpty}</span>
        {:else if isLink(g.item)}
          <input type="url" bind:value={g.item.target} placeholder={l.linkUrl} />
        {:else}
          <span class="target muted">{g.item.target}</span>
        {/if}
        <button class="icon-btn" title={t.up} disabled={i === 0} onclick={() => move(i, -1)}><ArrowUp size={16} /></button>
        <button class="icon-btn" title={t.down} disabled={i === groups.length - 1} onclick={() => move(i, 1)}><ArrowDown size={16} /></button>
        <button class="icon-btn" title={l.indent} disabled={i === 0 || g.kids.length > 0} onclick={() => indent(i)}><CornerDownRight size={16} /></button>
        <button class="icon-btn" title={t.remove} onclick={() => remove(i)}><X size={16} /></button>
      </div>
      {#if g.kids.length}
        <div class="m-kids">
          {#each g.kids as k, j (k.id)}
            <div class="m-row {cls(g.id, k.id)}" class:dragging={drag?.k === k.id}
              role="group" aria-label={k.name}
              ondragover={(e) => dragOver(e, g.id, k.id)} ondragleave={() => (over = null)} ondrop={drop}>
              <span class="grip" draggable="true" title={l.drag} aria-hidden="true"
                ondragstart={(e) => start(e, g.id, k.id)} ondragend={end}><GripVertical size={16} /></span>
              <input type="text" bind:value={k.name} aria-label={l.linkName} />
              {#if isLink(k)}<input type="url" bind:value={k.target} placeholder={l.linkUrl} />{:else}<span class="target muted">{k.target}</span>{/if}
              <button class="icon-btn" title={t.up} disabled={j === 0} onclick={() => moveKid(g, j, -1)}><ArrowUp size={16} /></button>
              <button class="icon-btn" title={t.down} disabled={j === g.kids.length - 1} onclick={() => moveKid(g, j, 1)}><ArrowDown size={16} /></button>
              <button class="icon-btn" title={l.outdent} onclick={() => outdent(i, j)}><CornerUpLeft size={16} /></button>
              <button class="icon-btn" title={t.remove} onclick={() => g.kids.splice(j, 1)}><X size={16} /></button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/each}
</div>

<style>
  .m-list { display: grid; gap: .5rem; }
  .m-group { display: grid; gap: .5rem; }
  .m-row { display: flex; gap: .5rem; align-items: center; border-radius: .5rem; outline: 2px solid transparent; outline-offset: 2px; transition: outline-color .12s, background-color .12s; }
  .m-row > input { flex: 1; min-width: 0; }
  .m-row.dragging { opacity: .45; }
  .m-row.drop-into { outline-color: var(--blue); background: color-mix(in srgb, var(--blue) 8%, transparent); }
  .m-row.drop-before { box-shadow: 0 -3px 0 var(--blue); }
  .m-row.drop-after { box-shadow: 0 3px 0 var(--blue); }
  .m-kids { display: grid; gap: .5rem; margin-inline-start: 1.25rem; padding-inline-start: .75rem; border-inline-start: 2px solid var(--line); }
  .grip { display: grid; place-items: center; width: 1.25rem; flex: none; color: var(--muted); cursor: grab; }
  .grip:active { cursor: grabbing; }
  .target { flex: 1; min-width: 0; font-size: .85rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding-inline: .5rem; }
</style>
