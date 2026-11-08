<script lang="ts">
  import X from "@lucide/svelte/icons/x";
  import Users from "@lucide/svelte/icons/users";
  import ListChecks from "@lucide/svelte/icons/list-checks";
  import BookOpen from "@lucide/svelte/icons/book-open";
  import Table2 from "@lucide/svelte/icons/table-2";
  import { templates, type Template } from "./db";
  import TextField from "./ui/TextField.svelte";
  import { t } from "./i18n";

  // A new database: pick what it is for (each comes with its columns and a
  // board ready) and its name, the template's by default. One click and
  // Enter is enough.
  let { onCreate, onClose }: { onCreate: (title: string, tpl: Template) => void; onClose: () => void } = $props();
  const d = t.db;
  const list = templates();
  const icons = { clients: Users, tasks: ListChecks, reading: BookOpen, empty: Table2 } as const;
  let pick = $state(list[0]);
  let name = $state(list[0].name);
  let typed = false; // the name was written by hand: picking keeps it
  function choose(tpl: Template) {
    pick = tpl;
    if (!typed) name = tpl.id === "empty" ? d.tEmptyName : tpl.name;
  }
  function create(e?: SubmitEvent) {
    e?.preventDefault();
    if (name.trim()) onCreate(name.trim(), pick);
  }
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onClose()} />

<div class="shade" role="presentation" onclick={onClose}></div>
<div class="dialog" role="dialog" aria-modal="true" aria-label={d.newTitle}>
<form class="form" onsubmit={create}>
  <header><h2>{d.newTitle}</h2><button type="button" class="close" title={t.attach.close} aria-label={t.attach.close} onclick={onClose}><X size={18} /></button></header>
  <p class="muted">{d.newLead}</p>
  <div class="choices">
    {#each list as tpl (tpl.id)}
      {@const Icon = icons[tpl.id as keyof typeof icons]}
      <button type="button" class={["choice", { on: pick.id === tpl.id }]} onclick={() => choose(tpl)}>
        <span class="ico"><Icon size={18} /></span><b>{tpl.name}</b><small class="muted">{tpl.about}</small>
      </button>
    {/each}
  </div>
  <TextField label={d.dbName} bind:value={name} autofocus oninput={() => (typed = true)} />
  <div class="actions">
    <button type="button" class="btn ghost" onclick={onClose}>{t.ob.cancel}</button>
    <button class="btn" disabled={!name.trim()}>{d.create}</button>
  </div>
</form>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 60; background: rgb(0 0 0 / .45); }
  .form { display: grid; gap: .9rem; }
  .dialog { position: fixed; z-index: 61; inset: 0; margin: auto; width: min(34rem, calc(100% - 2rem)); height: fit-content; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  header { display: flex; align-items: center; }
  h2 { flex: 1; font-size: 1.1rem; font-weight: 700; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  p { font-size: .9rem; }
  .choices { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .5rem; }
  .choice { display: grid; grid-template-columns: auto minmax(0, 1fr); grid-template-rows: auto auto; column-gap: .6rem; align-items: center; padding: .7rem .8rem; border: 1px solid var(--line); border-radius: .6rem; text-align: start; }
  .choice:hover { border-color: var(--muted); }
  .choice.on { border-color: var(--blue); background: color-mix(in srgb, var(--blue) 8%, transparent); }
  .ico { grid-row: 1 / 3; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .5rem; background: var(--soft); color: var(--blue); }
  .choice b { font-size: .92rem; }
  .choice small { font-size: .78rem; }
  .actions { display: flex; justify-content: flex-end; gap: .5rem; }
</style>
