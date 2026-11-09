<script lang="ts">
  import X from "@lucide/svelte/icons/x";
  import FileText from "@lucide/svelte/icons/file-text";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import TextField from "./ui/TextField.svelte";
  import Switch from "./Switch.svelte";
  import { t } from "./i18n";
  import { KINDS, KIND_ICON, type Event, type Kind } from "./calendar";

  // An event to add or change: what it is, its kind (each with its color
  // and icon), the day, the time or the whole day, how it repeats and when
  // to remind. Everything but the name comes filled in.
  let { ev, isNew, onSave, onDelete, onNotes, onClose }: {
    ev: Event;
    isNew: boolean;
    onSave: (ev: Event) => void;
    onDelete: (ev: Event) => void;
    onNotes: (ev: Event) => void;
    onClose: () => void;
  } = $props();
  const c = t.cal;

  // svelte-ignore state_referenced_locally
  const e = $state({ ...ev });
  let allDay = $state(!e.time);
  let repeatTouched = false; // the repeat was picked by hand: a new kind keeps it

  function kind(k: Kind) {
    e.kind = k;
    if (isNew && !repeatTouched) e.repeat = k === "birthday" ? "yearly" : "";
  }
  function setAllDay(v: boolean) {
    allDay = v;
    if (v) {
      e.time = "";
      if (e.remind === "15m" || e.remind === "1h") e.remind = "1d";
    } else if (!e.time) e.time = "09:00";
  }
  // A reminder minutes or an hour before only makes sense with a time.
  const reminds = $derived((allDay ? ["", "0", "1d", "1w"] : ["", "0", "15m", "1h", "1d", "1w"]) as (keyof typeof c.reminds)[]);
  const remindLabel = (r: keyof typeof c.reminds) => (r === "0" && allDay ? c.remindOnDay : c.reminds[r]);

  function submit(ev?: SubmitEvent) {
    ev?.preventDefault();
    if (!e.title.trim() || !e.date) return;
    onSave({ ...e, title: e.title.trim(), time: allDay ? "" : e.time } as Event);
  }
</script>

<svelte:window onkeydown={(k) => k.key === "Escape" && onClose()} />

<div class="shade" role="presentation" onclick={onClose}></div>
<div class="dialog" role="dialog" aria-modal="true" aria-label={isNew ? c.newTitle : c.editTitle}>
  <form class="form" onsubmit={submit}>
    <header><h2>{isNew ? c.newTitle : c.editTitle}</h2><button type="button" class="close" title={c.cancel} aria-label={c.cancel} onclick={onClose}><X size={18} /></button></header>

    <TextField label={c.what} bind:value={e.title} placeholder={c.placeholders[e.kind as Kind]} autofocus={isNew} />

    <div class="field">
      <span>{c.kind}</span>
      <div class="kinds" role="radiogroup" aria-label={c.kind}>
        {#each KINDS as k (k)}
          {@const Icon = KIND_ICON[k]}
          <button type="button" role="radio" aria-checked={e.kind === k} class={["kind", `k-${k}`, { on: e.kind === k }]} onclick={() => kind(k)}>
            <Icon size={16} /><span>{c.kinds[k]}</span>
          </button>
        {/each}
      </div>
    </div>

    <div class="two">
      <label class="field"><span>{c.date}</span><input type="date" bind:value={e.date} required /></label>
      {#if !allDay}<label class="field"><span>{c.time}</span><input type="time" bind:value={e.time} required /></label>{/if}
    </div>
    <Switch label={c.allDay} checked={allDay} onchange={setAllDay} />

    <div class="two">
      <label class="field"><span>{c.repeat}</span>
        <select bind:value={e.repeat} onchange={() => (repeatTouched = true)}>
          {#each Object.entries(c.repeats) as [v, label] (v)}<option value={v}>{label}</option>{/each}
        </select>
      </label>
      <label class="field"><span>{c.remind}</span>
        <select bind:value={e.remind}>
          {#each reminds as r (r)}<option value={r}>{remindLabel(r)}</option>{/each}
        </select>
      </label>
    </div>
    {#if e.remind}<p class="muted help">{c.remindHelp}</p>{/if}
    {#if !isNew}<p class="muted help">{c.notesHelp}</p>{/if}

    <div class="actions">
      {#if !isNew}
        <button type="button" class="btn ghost danger" onclick={() => onDelete(ev)}><Trash2 size={15} />{c.del}</button>
        <button type="button" class="btn ghost" onclick={() => onNotes(ev)}><FileText size={15} />{c.notes}</button>
      {/if}
      <span class="grow"></span>
      <button type="button" class="btn ghost" onclick={onClose}>{c.cancel}</button>
      <button class="btn" disabled={!e.title.trim() || !e.date}>{c.save}</button>
    </div>
  </form>
</div>

<style>
  .shade { position: fixed; inset: 0; z-index: 60; background: rgb(0 0 0 / .45); }
  .dialog { position: fixed; z-index: 61; top: 50%; left: 50%; transform: translate(-50%, -50%); width: min(34rem, calc(100% - 2rem)); max-height: calc(100% - 2rem); overflow: auto; padding: 1.25rem; border: 1px solid var(--line); border-radius: .85rem; background: var(--panel); box-shadow: 0 24px 60px rgb(0 0 0 / .35); }
  .form { display: grid; gap: .9rem; }
  header { display: flex; align-items: center; }
  h2 { flex: 1; font-size: 1.1rem; font-weight: 700; }
  .close { display: grid; place-items: center; padding: .3rem; border-radius: .4rem; color: var(--muted); }
  .close:hover { background: var(--soft); color: var(--text); }
  .kinds { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .4rem; }
  .kind { display: flex; align-items: center; gap: .45rem; padding: .5rem .6rem; border: 1px solid var(--line); border-radius: .55rem; font-size: .85rem; font-weight: 600; color: var(--text); }
  .kind :global(svg) { color: var(--k); }
  .kind:hover { border-color: var(--muted); }
  .kind.on { border-color: var(--k); background: color-mix(in srgb, var(--k) 12%, transparent); }
  .two { display: grid; grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr)); gap: .75rem; }
  .help { font-size: .8rem; margin-top: -.4rem; }
  .actions { display: flex; flex-wrap: wrap; align-items: center; gap: .5rem; }
  .grow { flex: 1; }
  .danger { color: var(--red); }
  .k-appointment { --k: var(--blue); }
  .k-birthday { --k: #db2777; }
  .k-exam { --k: #d97706; }
  .k-interview { --k: #0d9488; }
  .k-deadline { --k: #dc2626; }
  .k-reminder { --k: #16a34a; }
</style>
