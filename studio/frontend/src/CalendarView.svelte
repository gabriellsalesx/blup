<script lang="ts">
  import { CalendarEvents, SaveEvent, DeleteItem, RestoreSection } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import ChevronLeft from "@lucide/svelte/icons/chevron-left";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import Plus from "@lucide/svelte/icons/plus";
  import NotebookPen from "@lucide/svelte/icons/notebook-pen";
  import { t } from "./i18n";
  import { holidays } from "./holidays";
  import { KIND_ICON, iso, parse, addDays, byDay, type Event, type Kind } from "./calendar";
  import EventForm from "./EventForm.svelte";
  import { toast, toastError } from "./toast.svelte";
  import { onNotesChanged } from "./changes";

  // The calendar of a vault: the month on the left, the chosen day and
  // what is coming up on the right. A click on a day picks it; "Add on this
  // day" in its panel (or a double click on a day) opens the form. Each event is a note in the
  // Agenda folder, opened from its form to write in it.
  let { start = "", hasNote, onOpenNote, onDailyNote, onChanged }: {
    start?: string; // a day to show first (a reminder was clicked)
    hasNote: (rel: string) => boolean;
    onOpenNote: (rel: string) => void;
    onDailyNote: () => void;
    onChanged: () => void; // a note was made, renamed or deleted
  } = $props();
  const c = t.cal;

  const today = iso(new Date());
  let events = $state<Event[]>([]);
  let loaded = $state(false);
  // svelte-ignore state_referenced_locally
  const first = parse(start || today);
  let selected = $state(iso(first));
  let month = $state(first.getMonth());
  let year = $state(first.getFullYear());
  let form = $state<{ ev: Event; isNew: boolean } | null>(null);

  // The holidays, on unless turned off (kept on this computer).
  let showHolidays = $state(true);
  try { showHolidays = localStorage.getItem("blup.cal.holidays") !== "0"; } catch {}
  function toggleHolidays() {
    showHolidays = !showHolidays;
    try { localStorage.setItem("blup.cal.holidays", showHolidays ? "1" : "0"); } catch {}
  }

  const load = () => CalendarEvents().then((list) => { events = list ?? []; loaded = true; }).catch(toastError);
  load();
  // An event changed on disk (the sync, an agent, another editor): read
  // again, quietly.
  $effect(() => onNotesChanged(() => { CalendarEvents().then((list) => (events = list ?? [])).catch(() => {}); }));

  // The weeks shown: six, from the week of the month's first day (Sunday
  // first; Monday in Spanish).
  const weekStart = $derived(t.lang === "es" ? 1 : 0);
  const days = $derived.by(() => {
    const first = new Date(year, month, 1);
    const from = addDays(first, -((first.getDay() - weekStart + 7) % 7));
    return Array.from({ length: 42 }, (_, i) => addDays(from, i));
  });
  const onDays = $derived(byDay(events, days[0], days[41]));
  const holidayOf = $derived.by(() => {
    if (!showHolidays) return {} as Record<string, string>;
    return { ...holidays(year - 1, t.lang), ...holidays(year, t.lang), ...holidays(year + 1, t.lang) };
  });
  const weekdays = $derived(Array.from({ length: 7 }, (_, i) => new Intl.DateTimeFormat(t.locale, { weekday: "short" }).format(new Date(2026, 1, 1 + ((i + weekStart) % 7)))));
  const title = $derived(new Intl.DateTimeFormat(t.locale, { month: "long", year: "numeric" }).format(new Date(year, month, 1)));
  const longDay = (s: string) => new Intl.DateTimeFormat(t.locale, { weekday: "long", day: "numeric", month: "long" }).format(parse(s));
  const shortDay = (s: string) => new Intl.DateTimeFormat(t.locale, { weekday: "short", day: "numeric", month: "short" }).format(parse(s));

  // The month picker (a click on the month's name): any month of any year
  // at once, the arrows stay for the next and the last.
  let picker = $state(false);
  let pickYear = $state(0);
  const nowMonth = new Date().getMonth(), nowYear = new Date().getFullYear();
  const monthNames = $derived(Array.from({ length: 12 }, (_, i) => new Intl.DateTimeFormat(t.locale, { month: "short" }).format(new Date(2026, i, 1)).replace(".", "")));

  function go(months: number) {
    const d = new Date(year, month + months, 1);
    year = d.getFullYear();
    month = d.getMonth();
  }
  function pick(day: string) {
    selected = day;
    const d = parse(day);
    if (d.getMonth() !== month || d.getFullYear() !== year) { month = d.getMonth(); year = d.getFullYear(); }
  }
  const goToday = () => pick(today);

  // The chosen day's events, and the next 30 days' (today's included).
  const dayEvents = $derived(byDay(events, parse(selected), parse(selected)).get(selected) ?? []);
  const upcoming = $derived.by(() => {
    const from = parse(today);
    return [...byDay(events, from, addDays(from, 30)).entries()].sort(([a], [b]) => a.localeCompare(b));
  });
  function whenLabel(day: string) {
    const n = Math.round((parse(day).getTime() - parse(today).getTime()) / 864e5);
    return n === 0 ? c.today : n === 1 ? c.tomorrow : c.inDays.replace("{n}", String(n));
  }
  const dailyPath = (day: string) => `${t.daily.folder}/${day}.md`;

  // The form: a new event (on a day, maybe of a kind) or one to change.
  function add(day = selected, kind: Kind = "appointment") {
    form = { isNew: true, ev: { path: "", title: "", date: day, time: "", kind, repeat: kind === "birthday" ? "yearly" : "", remind: "1d", alarm: false } };
  }
  const edit = (ev: Event) => (form = { isNew: false, ev: { ...ev } });
  async function save(ev: Event) {
    try {
      await SaveEvent(ev);
      form = null;
      pick(ev.date);
      await load();
      onChanged();
    } catch (e) { toastError(e); }
  }
  async function remove(ev: Event) {
    try {
      const where = await DeleteItem(ev.path);
      form = null;
      await load();
      onChanged();
      toast(c.deleted, { action: { label: t.toast.undo, run: () => RestoreSection(where).then(() => { load(); onChanged(); }).catch(toastError) } });
    } catch (e) { toastError(e); }
  }
  function openNotes(ev: Event) {
    form = null;
    onOpenNote(ev.path);
  }
  const timeOf = (ev: Event) => ev.time || c.allDay;
</script>

{#snippet chip(ev: Event)}
  {@const Icon = KIND_ICON[ev.kind as Kind] ?? KIND_ICON.reminder}
  <button type="button" class="chip k-{ev.kind}" title={ev.title} onclick={(e) => { e.stopPropagation(); edit(ev); }}>
    <Icon size={12} strokeWidth={2.25} /><span>{ev.time ? ev.time + " " : ""}{ev.title}</span>
  </button>
{/snippet}

{#snippet row(ev: Event)}
  {@const Icon = KIND_ICON[ev.kind as Kind] ?? KIND_ICON.reminder}
  <button type="button" class="row k-{ev.kind}" onclick={() => edit(ev)}>
    <span class="dot"><Icon size={15} /></span>
    <span class="row-text"><b>{ev.title}</b><small>{c.kinds[ev.kind as Kind] ?? ""} · {timeOf(ev)}</small></span>
  </button>
{/snippet}

<svelte:window onkeydown={(e) => picker && e.key === "Escape" && (picker = false)} />

<div class="cal">
  <section class="month">
    <header class="bar">
      <div class="pick">
        <button type="button" class="title" aria-expanded={picker} aria-label={c.pickMonth} title={c.pickMonth} onclick={() => { pickYear = year; picker = !picker; }}>
          <h1>{title}</h1><ChevronDown size={18} />
        </button>
        {#if picker}
          <!-- A click outside (or Esc) closes it. -->
          <div class="scrim" role="presentation" onclick={() => (picker = false)}></div>
          <div class="months" role="dialog" aria-label={c.pickMonth}>
            <div class="year">
              <button type="button" class="icon-btn" aria-label={c.prevYear} title={c.prevYear} onclick={() => pickYear--}><ChevronLeft size={18} /></button>
              <b>{pickYear}</b>
              <button type="button" class="icon-btn" aria-label={c.nextYear} title={c.nextYear} onclick={() => pickYear++}><ChevronRight size={18} /></button>
            </div>
            <div class="mgrid">
              {#each monthNames as name, i (i)}
                <button type="button" class={["m", { on: i === month && pickYear === year, now: i === nowMonth && pickYear === nowYear }]} onclick={() => { year = pickYear; month = i; picker = false; }}>{name}</button>
              {/each}
            </div>
          </div>
        {/if}
      </div>
      <span class="grow"></span>
      <div class="nav">
        <button type="button" class="icon-btn" aria-label={c.prev} title={c.prev} onclick={() => go(-1)}><ChevronLeft size={18} /></button>
        <button type="button" class="btn ghost" onclick={goToday}>{c.today}</button>
        <button type="button" class="icon-btn" aria-label={c.next} title={c.next} onclick={() => go(1)}><ChevronRight size={18} /></button>
      </div>
      <label class="hol"><input type="checkbox" checked={showHolidays} onchange={toggleHolidays} />{c.holidays}</label>
    </header>

    <div class="grid" role="grid" aria-label={title}>
      {#each weekdays as w (w)}<div class="wd" role="columnheader">{w}</div>{/each}
      {#each days as d (iso(d))}
        {@const day = iso(d)}
        {@const list = onDays.get(day) ?? []}
        <div role="gridcell" tabindex="0" aria-selected={day === selected}
          class={["day", { out: d.getMonth() !== month, today: day === today, on: day === selected, holiday: !!holidayOf[day] }]}
          onclick={() => pick(day)} ondblclick={() => add(day)}
          onkeydown={(e) => { if (e.key === "Enter") add(day); else if (e.key === " ") { e.preventDefault(); pick(day); } }}>
          <span class="num">{d.getDate()}</span>
          {#if holidayOf[day]}<span class="hname" title={holidayOf[day]}>{holidayOf[day]}</span>{/if}
          {#each list.slice(0, 3) as ev (ev.path)}{@render chip(ev)}{/each}
          {#if list.length > 3}<span class="more">{c.more.replace("{n}", String(list.length - 3))}</span>{/if}
        </div>
      {/each}
    </div>
  </section>

  <aside class="side">
    <section class="card">
      <h2>{longDay(selected)}</h2>
      {#if holidayOf[selected]}<p class="hol-line">{c.holiday}: {holidayOf[selected]}</p>{/if}
      {#each dayEvents as ev (ev.path)}{@render row(ev)}{:else}<p class="muted">{c.nothingDay}</p>{/each}
      <div class="day-actions">
        <button type="button" class="btn ghost" onclick={() => add(selected)}><Plus size={15} />{c.addDay}</button>
        {#if selected === today}
          <button type="button" class="btn ghost" onclick={onDailyNote}><NotebookPen size={15} />{c.dayNote}</button>
        {:else if hasNote(dailyPath(selected))}
          <button type="button" class="btn ghost" onclick={() => onOpenNote(dailyPath(selected))}><NotebookPen size={15} />{c.dayNote}</button>
        {/if}
      </div>
    </section>

    {#if loaded && events.length === 0}
      <!-- The first time: what it is for, and a way in by kind. -->
      <section class="card empty">
        <h2>{c.emptyTitle}</h2>
        <p class="muted">{c.emptyText}</p>
        {#each ["birthday", "exam", "interview"] as const as k (k)}
          {@const Icon = KIND_ICON[k]}
          <button type="button" class="example k-{k}" onclick={() => add(selected, k)}><span class="dot"><Icon size={15} /></span>{c.examples[k]}</button>
        {/each}
      </section>
    {:else if loaded}
      <section class="card">
        <h2>{c.upcoming}</h2>
        {#each upcoming as [day, list] (day)}
          <div class="up-day">
            <button type="button" class="up-when" onclick={() => pick(day)}><b>{whenLabel(day)}</b><small>{shortDay(day)}</small></button>
            {#each list as ev (ev.path)}{@render row(ev)}{/each}
          </div>
        {:else}<p class="muted">{c.noUpcoming}</p>{/each}
      </section>
    {/if}
  </aside>
</div>

{#if form}
  <EventForm ev={form.ev} isNew={form.isNew} onSave={save} onDelete={remove} onNotes={openNotes} onClose={() => (form = null)} />
{/if}

<style>
  .cal { display: grid; grid-template-columns: minmax(0, 1fr) 20rem; gap: 1.25rem; height: 100%; min-height: 0; padding: 1.25rem; }
  .month { display: grid; grid-template-rows: auto minmax(0, 1fr); gap: .9rem; min-height: 0; }
  .bar { display: flex; flex-wrap: wrap; align-items: center; gap: .75rem; }
  h1 { font-size: 1.35rem; font-weight: 700; }
  .grow { flex: 1; }
  .pick { position: relative; }
  .title { display: flex; align-items: center; gap: .35rem; padding: .2rem .5rem; margin-inline-start: -.5rem; border-radius: .5rem; color: var(--text); }
  .title :global(svg) { color: var(--muted); }
  .title:hover { background: var(--soft); }
  .scrim { position: fixed; inset: 0; z-index: 20; }
  .months { position: absolute; z-index: 21; inset-block-start: calc(100% + .4rem); inset-inline-start: -.5rem; display: grid; gap: .6rem; width: 17rem; padding: .75rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); box-shadow: 0 12px 32px rgb(0 0 0 / .18); }
  .year { display: flex; align-items: center; justify-content: space-between; }
  .year b { font-size: 1rem; }
  .mgrid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .35rem; }
  .m { padding: .5rem 0; border-radius: .5rem; font-size: .85rem; font-weight: 600; text-transform: capitalize; color: var(--text); }
  .m:hover { background: var(--soft); }
  .m.now { color: var(--blue); }
  .m.on { background: var(--blue); color: #fff; }
  h1::first-letter, .up-when small::first-letter { text-transform: uppercase; }
  .nav { display: flex; align-items: center; gap: .25rem; }
  .icon-btn { display: grid; place-items: center; width: 2rem; height: 2rem; border-radius: .45rem; color: var(--muted); }
  .icon-btn:hover { background: var(--soft); color: var(--text); }
  .hol { display: flex; align-items: center; gap: .4rem; font-size: .85rem; color: var(--muted); cursor: pointer; }
  .hol input { accent-color: var(--blue); }

  .grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); grid-template-rows: auto repeat(6, minmax(0, 1fr)); min-height: 30rem; border: 1px solid var(--line); border-radius: .75rem; overflow: hidden; background: var(--line); gap: 1px; }
  .wd { padding: .45rem .6rem; background: var(--panel); font-size: .75rem; font-weight: 600; color: var(--muted); text-transform: capitalize; }
  .day { display: flex; flex-direction: column; gap: .2rem; min-width: 0; min-height: 0; padding: .35rem .4rem; background: var(--bg); cursor: pointer; overflow: hidden; }
  .day:hover { background: color-mix(in srgb, var(--soft) 60%, var(--bg)); }
  .day:focus-visible { outline: 2px solid var(--blue); outline-offset: -2px; }
  .day.out { background: var(--panel); }
  .day.out .num { color: var(--muted); opacity: .6; }
  .day.on { box-shadow: inset 0 0 0 2px var(--blue); }
  .num { align-self: flex-start; display: grid; place-items: center; min-width: 1.6rem; height: 1.6rem; padding-inline: .25rem; border-radius: 999px; font-size: .82rem; font-weight: 600; }
  .day.today .num { background: var(--blue); color: #fff; }
  .day.holiday .num { color: var(--red); }
  .day.today.holiday .num { color: #fff; }
  .hname { overflow: hidden; font-size: .7rem; color: var(--red); white-space: nowrap; text-overflow: ellipsis; }
  .chip { display: flex; align-items: center; gap: .3rem; min-width: 0; padding: .1rem .35rem; border-radius: .3rem; background: color-mix(in srgb, var(--k) 14%, transparent); color: var(--k); font-size: .72rem; font-weight: 600; text-align: start; }
  .chip span { overflow: hidden; color: var(--text); white-space: nowrap; text-overflow: ellipsis; }
  .chip:hover { background: color-mix(in srgb, var(--k) 24%, transparent); }
  .more { font-size: .7rem; color: var(--muted); }

  .side { display: flex; flex-direction: column; gap: 1rem; min-height: 0; overflow: auto; }
  .card { display: grid; gap: .5rem; padding: 1rem; border: 1px solid var(--line); border-radius: .75rem; background: var(--panel); }
  h2 { font-size: .95rem; font-weight: 700; }
  .card > h2:first-letter { text-transform: uppercase; }
  .card p { font-size: .85rem; }
  .hol-line { color: var(--red); }
  .row, .example { display: flex; align-items: center; gap: .6rem; padding: .45rem .5rem; border-radius: .5rem; text-align: start; }
  .row:hover, .example:hover { background: var(--soft); }
  .dot { display: grid; place-items: center; flex-shrink: 0; width: 1.9rem; height: 1.9rem; border-radius: .5rem; background: color-mix(in srgb, var(--k) 14%, transparent); color: var(--k); }
  .row-text { display: grid; min-width: 0; }
  .row-text b { overflow: hidden; font-size: .88rem; white-space: nowrap; text-overflow: ellipsis; }
  .row-text small { font-size: .76rem; color: var(--muted); }
  .day-actions { display: flex; flex-wrap: wrap; gap: .4rem; margin-top: .25rem; }
  .day-actions .btn { font-size: .8rem; padding: .35rem .65rem; }
  .example { font-size: .88rem; font-weight: 600; border: 1px solid var(--line); }
  .up-day { display: grid; gap: .1rem; }
  .up-day + .up-day { padding-top: .4rem; border-top: 1px solid var(--line); }
  .up-when { display: flex; align-items: baseline; gap: .45rem; padding: .2rem .5rem 0; text-align: start; }
  .up-when b { font-size: .8rem; color: var(--blue); }
  .up-when small { display: inline-block; font-size: .75rem; color: var(--muted); }

  /* Each kind's color. */
  .k-appointment { --k: var(--blue); }
  .k-birthday { --k: #db2777; }
  .k-exam { --k: #d97706; }
  .k-interview { --k: #0d9488; }
  .k-deadline { --k: #dc2626; }
  .k-reminder { --k: #16a34a; }

  @media (width < 1100px) {
    .cal { grid-template-columns: minmax(0, 1fr); height: auto; }
    .grid { min-height: 26rem; }
  }
</style>
