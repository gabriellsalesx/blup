<script lang="ts">
  import { onDestroy } from "svelte";
  import { OpenURL } from "./studio";
  import type { DBColumn } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";
  import Phone from "@lucide/svelte/icons/phone";
  import MessageCircle from "@lucide/svelte/icons/message-circle";
  import Mail from "@lucide/svelte/icons/mail";
  import External from "@lucide/svelte/icons/external-link";
  import X from "@lucide/svelte/icons/x";
  import Plus from "@lucide/svelte/icons/plus";
  import BackSheet from "./BackSheet.svelte";
  import Switch from "../ui/Switch.svelte";
  import NotePicker from "./NotePicker.svelte";
  import { optionColor, nextColor } from "./db";
  import Check from "@lucide/svelte/icons/check";
  import { typedValue } from "./fieldvalue";
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
  // What was typed, saved only when it changed (a number column keeps a
  // finite number).
  function commit(typed: string) {
    pending = null;
    const v = typedValue(column.type, typed, text);
    if (v !== undefined) onChange(v);
  }
  // What was typed when the sheet closes, with the field still in focus.
  let pending: string | null = null;
  onDestroy(() => { if (pending !== null) commit(pending); });

  // The choices' list, in the app's sheet.
  let menu = $state(false);
  let typed = $state("");
  const choices = $derived((column.options ?? []).filter((o) => o.value.toLowerCase().includes(typed.trim().toLowerCase())));
  function choose(v: string) {
    if (column.type === "multi") onChange(list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);
    else { onChange(v === text ? "" : v); menu = false; }
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
</script>

{#if column.type === "select" || column.type === "multi"}
  <button class={["chips", { compact }]} onclick={(e) => { e.stopPropagation(); typed = ""; menu = true; }} aria-haspopup="listbox" aria-label={column.name}>
    {#each list as v (v)}<span class="chip" style:--c={optionColor(option(v)?.color)}>{v}</span>{/each}
    {#if !list.length}<span class="empty">{compact ? "" : d.choose}</span>{/if}
  </button>
  {#if menu}
    <BackSheet title={column.name} onclose={() => (menu = false)}>
      {#snippet top()}<input type="text" bind:value={typed} placeholder={d.findOrCreate} aria-label={d.findOrCreate} enterkeyhint="done" onkeydown={(e) => { if (e.key === "Enter") { e.preventDefault(); if (choices[0] && choices[0].value.toLowerCase() === typed.trim().toLowerCase()) choose(choices[0].value); else create(); } }} />{/snippet}
      <ul role="listbox" aria-label={column.name}>
        {#each choices as o (o.value)}
          <li><button role="option" aria-selected={list.includes(o.value)} onclick={() => choose(o.value)}><span class="chip" style:--c={optionColor(o.color)}>{o.value}</span>{#if list.includes(o.value)}<span class="on"><Check size={20} /></span>{/if}</button></li>
        {/each}
        {#if typed.trim() && !option(typed.trim()) && onNewOption}
          <li><button onclick={create}><Plus size={20} />{d.createOption.replace("{name}", typed.trim())}</button></li>
        {/if}
      </ul>
      {#if list.length}<button class="clear" onclick={() => { onChange(column.type === "multi" ? [] : ""); menu = false; }}>{d.clear}</button>{/if}
    </BackSheet>
  {/if}
{:else if column.type === "check"}
  <Switch label={column.name} checked={value === true} onchange={onChange} />
{:else if column.type === "date"}
  <input class="fin" type="date" value={text.slice(0, 10)} onchange={(e) => onChange(e.currentTarget.value)} aria-label={column.name} />
{:else if column.type === "link"}
  <span class="frow">
    {#if linkName}<button class="flink" onclick={openLink}>{linkName}</button><button class="mini" title={d.clear} aria-label={d.clear} onclick={() => onChange("")}><X size={20} /></button>
    {:else}<button class="empty pick" onclick={() => (picking = true)}>{d.pickNote}</button>{/if}
  </span>
  {#if picking}<NotePicker title={column.name} {notes} onClose={() => (picking = false)} onPick={(n) => { picking = false; onChange(`[[${n.name}]]`); }} />{/if}
{:else}
  <span class="frow">
    <input class="fin" type={column.type === "number" ? "number" : column.type === "email" ? "email" : column.type === "phone" ? "tel" : column.type === "url" ? "url" : "text"}
      value={text} placeholder={compact ? "" : d.empty} aria-label={column.name}
      enterkeyhint="done" oninput={(e) => (pending = e.currentTarget.value)} onchange={(e) => commit(e.currentTarget.value)}
      onkeydown={(e) => { if (e.key === "Enter") e.currentTarget.blur(); }} />
    {#if text && column.type === "phone"}
      <button class="mini" title={d.call} aria-label={d.call} onclick={() => OpenURL("tel:" + digits(text))}><Phone size={20} /></button>
      <button class="mini" title={d.whatsapp} aria-label={d.whatsapp} onclick={() => OpenURL("https://wa.me/" + digits(text))}><MessageCircle size={20} /></button>
    {:else if text && column.type === "email"}
      <button class="mini" title={d.sendMail} aria-label={d.sendMail} onclick={() => OpenURL("mailto:" + text)}><Mail size={20} /></button>
    {:else if text && column.type === "url"}
      <button class="mini" title={d.openLink} aria-label={d.openLink} onclick={() => OpenURL(/^https?:/.test(text) ? text : "https://" + text)}><External size={20} /></button>
    {/if}
  </span>
{/if}

<style>
  .frow { display: flex; align-items: center; gap: .4rem; width: 100%; min-width: 0; }
  .fin { flex: 1; min-width: 0; width: 100%; min-height: 3rem; padding: .6rem .8rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--bg); color: var(--text); font: inherit; color-scheme: light dark; }
  .fin:focus { border-color: var(--blue); outline: none; }
  .mini { flex: none; display: grid; place-items: center; width: 3rem; height: 3rem; border: 1px solid var(--line); border-radius: .6rem; color: var(--blue-text); }
  .chips { display: flex; flex-wrap: wrap; align-items: center; gap: .35rem; width: 100%; min-height: 3rem; padding: .45rem .8rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--bg); text-align: start; }
  .chip { display: inline-block; padding: .05rem .5rem; border-radius: 999px; background: color-mix(in srgb, var(--c) 18%, transparent); color: color-mix(in srgb, var(--c) 70%, var(--text)); font-size: .82rem; font-weight: 500; white-space: nowrap; }
  .empty { color: var(--muted); }
  .pick { flex: 1; min-height: 3rem; padding-inline: .8rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--bg); text-align: start; }
  .flink { padding: .1rem .3rem; color: var(--blue); text-decoration: underline; text-underline-offset: 3px; text-align: start; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  ul { list-style: none; padding: 0; display: grid; gap: .6rem; }
  li button { display: flex; align-items: center; justify-content: space-between; gap: .6rem; width: 100%; min-height: 3.1rem; padding-inline: .85rem; border: 1px solid var(--line); border-radius: .75rem; text-align: start; }
  .on { display: grid; color: var(--blue-text); }
  .clear { min-height: 3rem; color: var(--muted); font-weight: 500; }
</style>
