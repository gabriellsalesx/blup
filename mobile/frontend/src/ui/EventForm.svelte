<script lang="ts">
  import { untrack } from "svelte";
  import X from "@lucide/svelte/icons/x";
  import FileText from "@lucide/svelte/icons/file-text";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import { KINDS, KIND_ICON, type Event, type Kind } from "../lib/calendar";
  import { t } from "../desk/i18n";
  import Switch from "./Switch.svelte";

  // An event to add or change: the Studio's form, on a screen of its own.
  let { ev, isNew, onSave, onDelete, onNotes, onClose }: {
    ev: Event;
    isNew: boolean;
    onSave: (ev: Event) => void;
    onDelete: (ev: Event) => void;
    onNotes: (ev: Event) => void;
    onClose: () => void;
  } = $props();

  const e = $state(untrack(() => ({ ...ev })));
  const fresh = untrack(() => isNew);
  let allDay = $state(!e.time);
  let repeatTouched = false; // the repeat was picked by hand: a new kind keeps it

  function kind(k: Kind) {
    e.kind = k;
    if (fresh && !repeatTouched) e.repeat = k === "birthday" ? "yearly" : "";
  }
  function setAllDay(v: boolean) {
    allDay = v;
    if (v) {
      e.time = "";
      if (e.remind === "15m" || e.remind === "1h") e.remind = "1d";
    } else if (!e.time) e.time = "09:00";
  }
  // A reminder minutes or an hour before only makes sense with a time.
  type Remind = keyof typeof t.cal.reminds;
  const reminds = $derived((allDay ? ["", "0", "1d", "1w"] : ["", "0", "15m", "1h", "1d", "1w"]) as Remind[]);
  const remindLabel = (r: Remind) => (r === "0" && allDay ? t.cal.remindOnDay : t.cal.reminds[r]);

  function submit(event: SubmitEvent) {
    event.preventDefault();
    if (!e.title.trim() || !e.date) return;
    onSave({ ...e, title: e.title.trim(), time: allDay ? "" : e.time } as Event);
  }
</script>

<form onsubmit={submit} aria-label={isNew ? t.cal.newTitle : t.cal.editTitle}>
  <header>
    <button type="button" class="icon" aria-label={t.cal.cancel} onclick={onClose}><X size={24} /></button>
    <h2>{isNew ? t.cal.newTitle : t.cal.editTitle}</h2>
    <button class="save" disabled={!e.title.trim() || !e.date}>{t.cal.save}</button>
  </header>
  <div class="body">
    <label class="field"><span>{t.cal.what}</span>
      <input type="text" bind:value={e.title} placeholder={t.cal.placeholders[e.kind as Kind]} autocapitalize="sentences" />
    </label>

    <div class="field">
      <span>{t.cal.kind}</span>
      <div class="kinds" role="radiogroup" aria-label={t.cal.kind}>
        {#each KINDS as k (k)}
          {@const Icon = KIND_ICON[k]}
          <button type="button" role="radio" aria-checked={e.kind === k} class={["kind", `k-${k}`, { on: e.kind === k }]} onclick={() => kind(k)}>
            <Icon size={17} /><span>{t.cal.kinds[k]}</span>
          </button>
        {/each}
      </div>
    </div>

    <div class="two">
      <label class="field"><span>{t.cal.date}</span><input type="date" bind:value={e.date} required /></label>
      {#if !allDay}<label class="field"><span>{t.cal.time}</span><input type="time" bind:value={e.time} required /></label>{/if}
    </div>
    <Switch label={t.cal.allDay} checked={allDay} onchange={setAllDay} />

    <label class="field"><span>{t.cal.repeat}</span>
      <select bind:value={e.repeat} onchange={() => (repeatTouched = true)}>
        {#each Object.entries(t.cal.repeats) as [v, label] (v)}<option value={v}>{label}</option>{/each}
      </select>
    </label>
    <label class="field"><span>{t.cal.remind}</span>
      <select bind:value={e.remind}>
        {#each reminds as r (r)}<option value={r}>{remindLabel(r)}</option>{/each}
      </select>
    </label>
    {#if e.remind}<p class="help">{t.calx.remindHelp}</p>{/if}
    <Switch label={t.calx.alarm} hint={allDay ? t.calx.alarmDay : t.calx.alarmTime} checked={!!e.alarm} onchange={(on) => (e.alarm = on)} />

    {#if !isNew}
      <p class="help">{t.cal.notesHelp}</p>
      <button type="button" class="wide" onclick={() => onNotes(ev)}><FileText size={18} />{t.cal.notes}</button>
      <button type="button" class="wide danger" onclick={() => onDelete(ev)}><Trash2 size={18} />{t.cal.del}</button>
    {/if}
  </div>
</form>

<style>
  form { position: absolute; inset: 0; z-index: 15; display: flex; flex-direction: column; background: var(--bg); }
  header { flex: none; display: flex; align-items: center; gap: .25rem; padding: calc(.25rem + var(--sat)) .75rem .25rem .25rem; border-block-end: 1px solid var(--line); }
  h2 { flex: 1; min-width: 0; font-size: 1.05rem; font-weight: 700; }
  .icon { display: grid; place-items: center; width: 2.9rem; height: 2.9rem; }
  .save { min-height: 2.5rem; padding-inline: 1.1rem; border-radius: .6rem; background: var(--blue); color: #fff; font-weight: 600; }
  .save:disabled { opacity: .45; }
  .body { flex: 1; min-height: 0; overflow-y: auto; display: grid; align-content: start; gap: 1rem; padding: 1rem 1rem calc(1.5rem + var(--sab)); }
  .field { display: grid; gap: .35rem; font-size: .875rem; min-width: 0; }
  .field > span { font-weight: 600; }
  .field input, .field select { font-size: 1rem; }
  .two { display: grid; grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr)); gap: .75rem; }
  .kinds { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .5rem; }
  .kind { display: flex; align-items: center; gap: .5rem; min-height: 2.9rem; padding-inline: .75rem; border: 1px solid var(--line); border-radius: .6rem; font-size: .9rem; font-weight: 500; background: var(--panel); }
  .kind :global(svg) { flex: none; color: var(--k); }
  .kind.on { border-color: var(--k); background: color-mix(in srgb, var(--k) 12%, transparent); font-weight: 600; }
  .k-appointment { --k: var(--blue); }
  .k-birthday { --k: #db2777; }
  .k-exam { --k: #d97706; }
  .k-interview { --k: #0d9488; }
  .k-deadline { --k: #dc2626; }
  .k-reminder { --k: #16a34a; }
  .help { font-size: .85rem; line-height: 1.4; color: var(--muted); }
  .wide { display: flex; align-items: center; justify-content: center; gap: .5rem; min-height: 3rem; border: 1px solid var(--line); border-radius: .6rem; font-weight: 600; }
  .danger { color: var(--red); }
</style>
