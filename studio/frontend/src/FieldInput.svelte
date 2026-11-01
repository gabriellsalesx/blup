<script lang="ts">
  import { OpenURL } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import type { DBColumn } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Phone from "@lucide/svelte/icons/phone";
  import MessageCircle from "@lucide/svelte/icons/message-circle";
  import Mail from "@lucide/svelte/icons/mail";
  import External from "@lucide/svelte/icons/external-link";
  import X from "@lucide/svelte/icons/x";
  import Plus from "@lucide/svelte/icons/plus";
  import NotePicker from "./NotePicker.svelte";
  import { optionColor, nextColor } from "./db";
  import { t } from "./i18n";

  // One field of a database's record, by its column's kind: written in a
  // table's cell (compact) or in the record's Properties. A choice that
  // doesn't exist yet can be made right there (onNewOption).
  let { column, value, compact = false, notes = [], onChange, onNewOption, onOpenNote }: {
    column: DBColumn; value: unknown; compact?: boolean;
    notes?: { name: string; path: string }[];
    onChange: (v: unknown) => void;
    onNewOption?: (value: string, color: string) => void;
    onOpenNote?: (path: string) => void;
  } = $props();
  const d = t.db;

  const text = $derived(value == null ? "" : String(value));
  const list = $derived(Array.isArray(value) ? value.map(String) : value ? [String(value)] : []);
  const option = (v: string) => column.options?.find((o) => o.value === v);
  const commit = (v: string) => { if (v !== text) onChange(v); };

  // The choices' list, under the field, over everything (a table's cell
  // would cut it).
  let menu = $state<{ x: number; y: number } | null>(null);
  let typed = $state("");
  function openMenu(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = { x: Math.min(r.left, innerWidth - 230), y: Math.min(r.bottom + 4, innerHeight - 280) };
    typed = "";
  }
  let menuEl = $state<HTMLDivElement>();
  $effect(() => {
    if (!menu) return;
    const close = (e: PointerEvent) => { if (!menuEl?.contains(e.target as Node)) menu = null; };
    addEventListener("pointerdown", close, true);
    return () => removeEventListener("pointerdown", close, true);
  });
  const choices = $derived((column.options ?? []).filter((o) => o.value.toLowerCase().includes(typed.trim().toLowerCase())));
  function choose(v: string) {
    if (column.type === "multi") onChange(list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);
    else { onChange(v === text ? "" : v); menu = null; }
  }
  function create() {
    const v = typed.trim();
    if (!v) return;
    if (!option(v)) onNewOption?.(v, nextColor(column.options?.at(-1)?.color));
    choose(v);
    typed = "";
  }

  let picking = $state(false);
  const linkName = $derived(text.replace(/^\[\[|\]\]$/g, "").split("|")[0]);
  function openLink() {
    const n = notes.find((x) => x.name === linkName || x.path.replace(/\.md$/, "") === linkName);
    if (n) onOpenNote?.(n.path);
  }
  const digits = (s: string) => s.replace(/\D/g, "");
  const focus = (el: HTMLInputElement) => el.focus();
</script>

{#if column.type === "select" || column.type === "multi"}
  <button class={["chips", { compact }]} onclick={(e) => { e.stopPropagation(); openMenu(e); }} aria-haspopup="listbox" aria-label={column.name}>
    {#each list as v (v)}<span class="chip" style:--c={optionColor(option(v)?.color)}>{v}</span>{/each}
    {#if !list.length}<span class="empty">{compact ? "" : d.choose}</span>{/if}
  </button>
  {#if menu}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="menu" bind:this={menuEl} style:left="{menu.x}px" style:top="{menu.y}px" onclick={(e) => e.stopPropagation()}>
      <input bind:value={typed} placeholder={d.findOrCreate} use:focus onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); if (choices[0] && choices[0].value.toLowerCase() === typed.trim().toLowerCase()) choose(choices[0].value); else create(); } if (e.key === "Escape") menu = null; }} />
      <ul role="listbox" aria-label={column.name}>
        {#each choices as o (o.value)}
          <li><button role="option" aria-selected={list.includes(o.value)} onclick={() => choose(o.value)}><span class="chip" style:--c={optionColor(o.color)}>{o.value}</span>{#if list.includes(o.value)}<span class="on">✓</span>{/if}</button></li>
        {/each}
        {#if typed.trim() && !option(typed.trim()) && onNewOption}
          <li><button onclick={create}><Plus size={14} />{d.createOption.replace("{name}", typed.trim())}</button></li>
        {/if}
      </ul>
      {#if list.length}<button class="clear" onclick={() => { onChange(column.type === "multi" ? [] : ""); menu = null; }}>{d.clear}</button>{/if}
    </div>
  {/if}
{:else if column.type === "check"}
  <input type="checkbox" checked={value === true} onchange={(e) => onChange(e.currentTarget.checked)} aria-label={column.name} />
{:else if column.type === "date"}
  <input class="field" type="date" value={text.slice(0, 10)} onchange={(e) => onChange(e.currentTarget.value)} aria-label={column.name} />
{:else if column.type === "link"}
  <span class="row">
    {#if linkName}<button class="link" onclick={openLink}>{linkName}</button><button class="mini" title={d.clear} aria-label={d.clear} onclick={() => onChange("")}><X size={13} /></button>
    {:else}<button class="empty pick" onclick={() => (picking = true)}>{d.pickNote}</button>{/if}
  </span>
  {#if picking}<NotePicker title={column.name} {notes} onClose={() => (picking = false)} onPick={(n) => { picking = false; onChange(`[[${n.name}]]`); }} />{/if}
{:else}
  <span class="row">
    <input class="field" type={column.type === "number" ? "number" : column.type === "email" ? "email" : column.type === "phone" ? "tel" : column.type === "url" ? "url" : "text"}
      value={text} placeholder={compact ? "" : d.empty} aria-label={column.name}
      onchange={(e) => commit(column.type === "number" && e.currentTarget.value !== "" ? (Number(e.currentTarget.value) as unknown as string) : e.currentTarget.value)}
      onkeydown={(e) => { if (e.key === "Enter") e.currentTarget.blur(); }} />
    {#if text && column.type === "phone"}
      <button class="mini" title={d.call} aria-label={d.call} onclick={() => OpenURL("tel:" + digits(text))}><Phone size={13} /></button>
      <button class="mini" title={d.whatsapp} aria-label={d.whatsapp} onclick={() => OpenURL("https://wa.me/" + digits(text))}><MessageCircle size={13} /></button>
    {:else if text && column.type === "email"}
      <button class="mini" title={d.sendMail} aria-label={d.sendMail} onclick={() => OpenURL("mailto:" + text)}><Mail size={13} /></button>
    {:else if text && column.type === "url"}
      <button class="mini" title={d.openLink} aria-label={d.openLink} onclick={() => OpenURL(/^https?:/.test(text) ? text : "https://" + text)}><External size={13} /></button>
    {/if}
  </span>
{/if}

<style>
  .row { display: flex; align-items: center; gap: .15rem; width: 100%; min-width: 0; }
  .field { flex: 1; min-width: 0; width: 100%; padding: .25rem .4rem; border: 1px solid transparent; border-radius: .35rem; background: transparent; color: var(--text); font: inherit; color-scheme: light dark; }
  .field:hover, .field:focus { border-color: var(--line); outline: none; background: var(--panel); }
  .mini { display: grid; place-items: center; flex: none; width: 1.5rem; height: 1.5rem; border-radius: .3rem; color: var(--muted); }
  .mini:hover { background: var(--soft); color: var(--blue); }
  .chips { display: flex; flex-wrap: wrap; align-items: center; gap: .25rem; width: 100%; min-height: 1.8rem; padding: .2rem .35rem; border-radius: .35rem; text-align: start; }
  .chips:hover { background: var(--soft); }
  .chip { display: inline-block; padding: .05rem .5rem; border-radius: 999px; background: color-mix(in srgb, var(--c) 18%, transparent); color: color-mix(in srgb, var(--c) 70%, var(--text)); font-size: .82rem; font-weight: 500; white-space: nowrap; }
  .empty { color: var(--muted); font-size: .85rem; }
  .pick { padding: .25rem .4rem; border-radius: .35rem; }
  .pick:hover { background: var(--soft); color: var(--text); }
  .link { padding: .1rem .3rem; color: var(--blue); text-decoration: underline; text-underline-offset: 3px; text-align: start; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  input[type="checkbox"] { width: 1.05rem; height: 1.05rem; margin: .3rem .4rem; accent-color: var(--blue); }
  .menu { position: fixed; z-index: 80; width: 14rem; display: grid; gap: .3rem; padding: .4rem; border: 1px solid var(--line); border-radius: .55rem; background: var(--panel); box-shadow: 0 12px 34px rgb(0 0 0 / .28); }
  .menu input { width: 100%; padding: .35rem .5rem; border: 1px solid var(--line); border-radius: .4rem; background: var(--bg); color: var(--text); font: inherit; font-size: .85rem; outline: none; }
  .menu ul { list-style: none; padding: 0; max-height: 14rem; overflow: auto; display: grid; gap: .1rem; }
  .menu li button { display: flex; align-items: center; justify-content: space-between; gap: .4rem; width: 100%; padding: .3rem .4rem; border-radius: .35rem; font-size: .85rem; text-align: start; }
  .menu li button:hover { background: var(--soft); }
  .on { color: var(--blue); font-weight: 700; }
  .clear { padding: .3rem; border-radius: .35rem; font-size: .8rem; color: var(--muted); }
  .clear:hover { background: var(--soft); color: var(--text); }
</style>
