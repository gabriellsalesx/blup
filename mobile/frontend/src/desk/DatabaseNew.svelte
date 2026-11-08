<script lang="ts">
  import Users from "@lucide/svelte/icons/users";
  import ListChecks from "@lucide/svelte/icons/list-checks";
  import BookOpen from "@lucide/svelte/icons/book-open";
  import Table2 from "@lucide/svelte/icons/table-2";
  import { templates, type Template } from "./db";
  import Sheet from "../ui/Sheet.svelte";
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

<Sheet title={d.newTitle} onclose={onClose}>
  <form class="form" onsubmit={create}>
    <div class="kinds" role="radiogroup" aria-label={d.newTitle}>
      {#each list as tpl (tpl.id)}
        {@const Icon = icons[tpl.id as keyof typeof icons]}
        <button type="button" class={["kind", { on: pick.id === tpl.id }]} role="radio" aria-checked={pick.id === tpl.id} onclick={() => choose(tpl)}>
          <span class="dot"><Icon size={20} /></span><b>{tpl.name}</b><small>{tpl.about}</small>
        </button>
      {/each}
    </div>
    <input type="text" bind:value={name} aria-label={d.dbName} placeholder={d.dbName} enterkeyhint="done" oninput={() => (typed = true)} />
    <button class="make" disabled={!name.trim()}>{d.create}</button>
  </form>
</Sheet>

<style>
  .form { display: grid; gap: 1rem; }
  .kinds { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .6rem; }
  .kind { display: grid; justify-items: center; align-content: start; gap: .4rem; padding: .85rem .6rem; border: 1px solid var(--line); border-radius: .75rem; text-align: center; }
  .kind.on { border-color: var(--blue); background: color-mix(in srgb, var(--blue) 8%, transparent); }
  .kind b { font-size: .95rem; font-weight: 600; }
  .kind small { font-size: .8rem; line-height: 1.3; color: var(--muted); }
  .dot { display: grid; place-items: center; width: 2.6rem; height: 2.6rem; border-radius: .6rem; background: color-mix(in srgb, var(--blue) 14%, transparent); color: var(--blue-text); }
  .make { min-height: 3rem; border-radius: .6rem; background: var(--blue); color: #fff; font-weight: 600; }
  .make:disabled { opacity: .45; }
</style>
