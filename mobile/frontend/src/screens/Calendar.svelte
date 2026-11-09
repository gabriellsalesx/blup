<script lang="ts">
  import CalendarIcon from "@lucide/svelte/icons/calendar";
  import ChevronLeft from "@lucide/svelte/icons/chevron-left";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import Plus from "@lucide/svelte/icons/plus";
  import NotebookPen from "@lucide/svelte/icons/notebook-pen";
  import { CalendarService, NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
  import { app, message } from "../lib/app.svelte";
  import { holidays } from "../lib/holidays";
  import { KINDS, KIND_ICON, iso, parse, addDays, byDay, type Event, type Kind } from "../lib/calendar";
  import { fmt, shown, t } from "../desk/i18n";
  import EventForm from "../ui/EventForm.svelte";
  import Sheet from "../ui/Sheet.svelte";

  // The Studio's calendar for a phone: the month fills the screen, with what
  // is on each day written in it. The chosen day, the options to add and what
  // is coming up open in sheets. Each event is a note in Agenda.
  const locale = $derived(t.locale);
  // Today follows the clock (app.today changes at midnight and on coming back).
  const today = $derived(app.today);
  const first = parse(app.day);
  let month = $state(first.getMonth());
  let year = $state(first.getFullYear());
  let pickYear = $state(0);
  // Only the weeks the month needs, so each day gets as much room as it can.
  const days = $derived.by(() => {
    const start = new Date(year, month, 1);
    const from = addDays(start, -start.getDay());
    const weeks = Math.ceil((start.getDay() + new Date(year, month + 1, 0).getDate()) / 7);
    return Array.from({ length: weeks * 7 }, (_, i) => addDays(from, i));
  });
  const onDays = $derived(byDay(app.events, days[0], days[days.length - 1]));
  const holidayOf = $derived.by(() => {
    if (!app.showHolidays) return {} as Record<string, string>;
    const lang = shown.lang;
    return { ...holidays(year - 1, lang), ...holidays(year, lang), ...holidays(year + 1, lang) };
  });
  const nowMonth = $derived(parse(today).getMonth());
  const nowYear = $derived(parse(today).getFullYear());
  const title = $derived(new Intl.DateTimeFormat(locale, parse(app.day).getFullYear() === nowYear ? { day: "numeric", month: "long" } : { day: "numeric", month: "long", year: "numeric" }).format(parse(app.day)));
  // How much of this year has gone by.
  const yearDone = $derived(Math.round(((parse(today).getTime() - new Date(nowYear, 0, 1).getTime()) / 864e5 + 1) / ((new Date(nowYear + 1, 0, 1).getTime() - new Date(nowYear, 0, 1).getTime()) / 864e5) * 100));
  const longDay = (s: string) => new Intl.DateTimeFormat(locale, { weekday: "long", day: "numeric", month: "long" }).format(parse(s));
  const monthNames = $derived(Array.from({ length: 12 }, (_, i) => new Intl.DateTimeFormat(locale, { month: "short" }).format(new Date(2026, i, 1)).replace(".", "")));
  // The week's initials, Sunday first (4 January 2026 is a Sunday).
  const weekDays = $derived(Array.from({ length: 7 }, (_, i) => new Intl.DateTimeFormat(locale, { weekday: "narrow" }).format(new Date(2026, 0, 4 + i)).toUpperCase()));
  const capital = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
  const addLabel = $derived(new Intl.DateTimeFormat(locale, { day: "numeric", month: "short" }).format(parse(app.day)));

  // Showing another month moves the chosen day there, so adding never lands on
  // a day that is out of sight.
  function show(y: number, m: number) {
    year = y;
    month = m;
    app.day = y === nowYear && m === nowMonth ? today : iso(new Date(y, m, 1));
  }
  function go(months: number) {
    const d = new Date(year, month + months, 1);
    show(d.getFullYear(), d.getMonth());
  }
  function pick(day: string) {
    app.day = day;
    const d = parse(day);
    if (d.getMonth() !== month || d.getFullYear() !== year) { month = d.getMonth(); year = d.getFullYear(); }
  }

  function tap(day: string) {
    if (day === app.day) app.go({ panel: "day" });
    else pick(day);
  }

  let touch = { x: 0, y: 0 };
  function swipeStart(e: TouchEvent) {
    touch = { x: e.touches[0].clientX, y: e.touches[0].clientY };
  }
  function swipeEnd(e: TouchEvent) {
    const dx = e.changedTouches[0].clientX - touch.x, dy = e.changedTouches[0].clientY - touch.y;
    if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.5) go(dx < 0 ? 1 : -1);
  }

  const dayEvents = $derived(byDay(app.events, parse(app.day), parse(app.day)).get(app.day) ?? []);
  const dailyPath = (day: string) => `${t.daily.folder}/${day}.md`;
  const hasNote = (id: string) => app.notes.some((n) => n.id === id);

  const noteOpens = $derived(app.day === today || hasNote(dailyPath(app.day)));

  // These run from a sheet, which the next place takes over: the back button
  // then returns to the month.
  function add(kind: Kind) {
    app.swap({ isNew: true, event: { path: "", title: "", date: app.day, time: "", kind, repeat: kind === "birthday" ? "yearly" : "", remind: "1d", alarm: false } as Event });
  }
  const edit = (ev: Event) => app.swap({ isNew: false, event: { ...ev } as Event });

  async function save(ev: Event) {
    try {
      await CalendarService.Save(ev);
      app.back();
      pick(ev.date);
      await app.refresh();
      app.sync();
    } catch (e) { app.say({ text: message(e) }); }
  }
  async function remove(ev: Event) {
    try {
      const where = await CalendarService.Delete(ev.path);
      app.back();
      await app.refresh();
      app.sync();
      app.say({ text: t.calx.deleted, action: { label: t.common.undo, run: async () => { await CalendarService.Restore(where); await app.refresh(); app.sync(); } } });
    } catch (e) { app.say({ text: message(e) }); }
  }
  const notes = (ev: Event) => app.swap({ note: ev.path });
  async function dailyNote() {
    if (app.day !== today) return app.swap({ note: dailyPath(app.day) });
    try {
      const id = await CalendarService.DailyNote();
      await app.refresh();
      app.swap({ note: id, edit: true });
    } catch (e) { app.say({ text: message(e) }); }
  }
  const timeOf = (ev: Event) => ev.time || t.cal.allDay;
</script>

{#snippet row(ev: Event)}
  {@const Icon = KIND_ICON[ev.kind as Kind] ?? KIND_ICON.reminder}
  <button type="button" class="row k-{ev.kind}" onclick={() => edit(ev)}>
    <span class="dot"><Icon size={17} /></span>
    <span class="text"><b>{ev.title}</b><small>{t.cal.kinds[ev.kind as Kind] ?? ""} · {timeOf(ev)}</small></span>
  </button>
{/snippet}

{#snippet noteButton()}
  {#if noteOpens}<button type="button" class="plain" onclick={dailyNote}><NotebookPen size={18} />{t.cal.dayNote}</button>{/if}
{/snippet}

<header>
  <button type="button" class="title" aria-label={t.cal.pickMonth} onclick={() => { pickYear = year; app.go({ panel: "month" }); }}><CalendarIcon size={26} /><h1>{title}</h1><ChevronDown size={20} /></button>
  <button type="button" class="today" aria-label={t.calx.goToday} onclick={() => pick(today)}>{parse(today).getDate()}</button>
  <div class="year" role="img" aria-label={fmt(t.calx.yearDone, { p: yearDone, year: nowYear })}>
    <span>{nowYear}</span><i><b style:width={`${yearDone}%`}></b></i><span>{yearDone}%</span>
  </div>
</header>

<div class="grid week" aria-hidden="true">
  {#each weekDays as w, i (i)}<span class={{ sun: i === 0 }}>{w}</span>{/each}
</div>
<div class="grid month" role="group" aria-label={title} style:--weeks={days.length / 7} ontouchstart={swipeStart} ontouchend={swipeEnd}>
  {#each days as d (iso(d))}
    {@const day = iso(d)}
    {@const list = onDays.get(day) ?? []}
    {@const holiday = holidayOf[day]}
    {@const total = list.length + (holiday ? 1 : 0)}
    {@const room = total > 3 ? 2 : 3}
    <button type="button" aria-pressed={day === app.day} aria-label={longDay(day)} class={["day", { out: d.getMonth() !== month, now: day === today, sun: d.getDay() === 0, holiday: !!holiday }]} onclick={() => tap(day)}>
      <span class="num">{d.getDate()}</span>
      {#if holiday}<span class="chip k-holiday">{holiday}</span>{/if}
      {#each list.slice(0, room - (holiday ? 1 : 0)) as ev (ev.path)}<span class="chip k-{ev.kind}">{ev.title}</span>{/each}
      {#if total > room}<em>+{total - room}</em>{/if}
    </button>
  {/each}
</div>
<div class="bar">
  <button type="button" class="add" onclick={() => { NotesService.Haptic(); app.go({ panel: "add" }); }}><span>{fmt(t.calx.addOn, { date: addLabel })}</span><Plus size={24} /></button>
</div>

{#if app.place.panel === "add"}
  <Sheet title={capital(longDay(app.day))} onclose={() => app.back()}>
    <div class="kinds">
      {#each KINDS as k (k)}
        {@const Icon = KIND_ICON[k]}
        <button type="button" class="kind k-{k}" onclick={() => add(k)}><span class="dot"><Icon size={20} /></span>{t.cal.kinds[k]}</button>
      {/each}
    </div>
    {@render noteButton()}
  </Sheet>
{:else if app.place.panel === "day"}
  <Sheet title={capital(longDay(app.day))} onclose={() => app.back()}>
    {#if holidayOf[app.day]}<p class="hol">{t.cal.holiday}: {holidayOf[app.day]}</p>{/if}
    {#each dayEvents as ev (ev.path)}{@render row(ev)}{:else}<p class="muted">{t.cal.nothingDay}</p>{/each}
    <button type="button" class="main" onclick={() => app.swap({ panel: "add" })}><Plus size={19} />{t.calx.addDay}</button>
    {@render noteButton()}
  </Sheet>
{:else if app.place.panel === "month"}
  <Sheet title={t.cal.pickMonth} onclose={() => app.back()}>
    <div class="pick-year">
      <button type="button" aria-label={t.cal.prevYear} onclick={() => pickYear--}><ChevronLeft size={24} /></button>
      <b>{pickYear}</b>
      <button type="button" aria-label={t.cal.nextYear} onclick={() => pickYear++}><ChevronRight size={24} /></button>
    </div>
    <div class="mgrid">
      {#each monthNames as name, i (i)}
        <button type="button" class={{ on: i === month && pickYear === year, now: i === nowMonth && pickYear === nowYear }} onclick={() => { show(pickYear, i); app.back(); }}>{name}</button>
      {/each}
    </div>
  </Sheet>
{/if}

{#if app.place.event}
  <EventForm ev={app.place.event} isNew={!!app.place.isNew} onSave={save} onDelete={remove} onNotes={notes} onClose={() => app.back()} />
{/if}

<style>
  header { position: relative; flex: none; display: grid; gap: .55rem; padding: calc(.5rem + var(--sat)) 1rem .5rem; }
  .title { display: flex; align-items: center; justify-content: center; gap: .5rem; min-width: 0; min-height: 2.75rem; padding-inline: 2.5rem; color: var(--blue-text); }
  .title :global(svg) { flex: none; }
  h1 { font-size: 1.6rem; font-weight: 700; letter-spacing: -.02em; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .today { position: absolute; inset-block-start: calc(.8rem + var(--sat)); inset-inline-end: .75rem; display: grid; place-items: center; width: 2.1rem; height: 2.1rem; border: 2px solid var(--text); border-radius: .5rem; font-size: .9rem; font-weight: 700; }
  .year { display: flex; align-items: center; gap: .6rem; font-size: .8rem; font-weight: 600; font-variant-numeric: tabular-nums; color: var(--muted); }
  .year i { flex: 1; height: .4rem; border-radius: 999px; overflow: hidden; background: var(--soft); }
  .year b { display: block; height: 100%; border-radius: 999px; background: var(--blue); }
  .grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: .2rem; padding-inline: .3rem; }
  .week { flex: none; padding-block: .25rem .35rem; font-size: .8rem; font-weight: 600; text-align: center; }
  .month { flex: 1; min-height: 0; grid-template-rows: repeat(var(--weeks), minmax(0, 1fr)); touch-action: pan-y; }
  .sun { color: var(--red); }
  .day { display: flex; flex-direction: column; align-items: stretch; gap: .1rem; min-width: 0; min-height: 0; overflow: hidden; padding: .2rem .1rem; border: 1.5px solid transparent; border-radius: .45rem; background: var(--panel); font-size: .9rem; font-weight: 600; }
  .day.holiday { background: color-mix(in srgb, var(--holiday) 22%, var(--panel)); }
  .day[aria-pressed="true"] { border-color: var(--blue); }
  .out { opacity: .45; }
  .num { align-self: center; flex: none; display: grid; place-items: center; min-width: 1.4rem; height: 1.4rem; border-radius: .35rem; }
  .now .num { background: var(--blue); color: #fff; }
  .chip { flex: none; padding: .05rem .15rem; border-radius: .2rem; overflow: hidden; white-space: nowrap; text-align: start; font-size: .66rem; font-weight: 500; line-height: 1.25; color: #fff; background: color-mix(in srgb, var(--k) 85%, #000); }
  .day em { font-size: .66rem; font-style: normal; font-weight: 700; line-height: 1.2; text-align: center; color: var(--muted); }
  .bar { flex: none; display: flex; justify-content: center; padding: .5rem 1rem; }
  .add { display: flex; align-items: center; justify-content: space-between; gap: 1rem; width: min(100%, 17rem); height: 3.1rem; padding-inline: 1.25rem 1rem; border: 1px solid var(--line); border-radius: 999px; background: var(--panel); color: var(--muted); box-shadow: 0 4px 12px rgb(0 0 0 / .2); }
  .add:active { transform: scale(.98); }
  .add :global(svg) { color: var(--text); }
  .kinds { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .5rem; }
  .kind { display: grid; justify-items: center; gap: .4rem; padding-block: .7rem; border: 1px solid var(--line); border-radius: .75rem; font-size: .85rem; font-weight: 600; }
  .kind .dot { width: 2.6rem; height: 2.6rem; }
  .muted { font-size: .9rem; line-height: 1.4; color: var(--muted); }
  .hol { font-size: .875rem; font-weight: 600; color: var(--red); }
  .row { display: flex; align-items: center; gap: .7rem; width: 100%; min-height: 3rem; text-align: start; }
  .dot { flex: none; display: grid; place-items: center; width: 2.2rem; height: 2.2rem; border-radius: .6rem; color: var(--k); background: color-mix(in srgb, var(--k) 14%, transparent); }
  .text { display: grid; min-width: 0; }
  .text b { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .text small { font-size: .8rem; color: var(--muted); }
  .main, .plain { display: flex; align-items: center; justify-content: center; gap: .5rem; min-height: 2.9rem; border-radius: .6rem; font-weight: 600; }
  .main { background: var(--blue); color: #fff; }
  .plain { border: 1px solid var(--line); }
  .k-appointment { --k: var(--blue); }
  .k-birthday { --k: #db2777; }
  .k-exam { --k: #d97706; }
  .k-interview { --k: #0d9488; }
  .k-deadline { --k: #dc2626; }
  .k-reminder { --k: #16a34a; }
  .k-holiday { --k: var(--holiday); }
  .month { --holiday: #15803d; }
  .pick-year { display: flex; align-items: center; justify-content: space-between; }
  .pick-year button { display: grid; place-items: center; width: 2.9rem; height: 2.9rem; color: var(--muted); }
  .mgrid { display: grid; grid-template-columns: repeat(4, 1fr); gap: .35rem; }
  .mgrid button { min-height: 3rem; border-radius: .6rem; font-size: .95rem; text-transform: capitalize; }
  .mgrid .now { color: var(--blue-text); font-weight: 700; }
  .mgrid .on { background: var(--blue); color: #fff; font-weight: 600; }
</style>
