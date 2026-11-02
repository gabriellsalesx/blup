// The calendar's events (notes in the Agenda folder, see app/calendar.go):
// their kinds, colors and icons, and the days each one happens on, the
// same way the Go side works out the reminders.
import type { Event } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
import CalendarClock from "@lucide/svelte/icons/calendar-clock";
import Cake from "@lucide/svelte/icons/cake";
import GraduationCap from "@lucide/svelte/icons/graduation-cap";
import Briefcase from "@lucide/svelte/icons/briefcase";
import Flag from "@lucide/svelte/icons/flag";
import Bell from "@lucide/svelte/icons/bell";

export type { Event };
export type Kind = "appointment" | "birthday" | "exam" | "interview" | "deadline" | "reminder";
export const KINDS: Kind[] = ["appointment", "birthday", "exam", "interview", "deadline", "reminder"];
export const KIND_ICON = { appointment: CalendarClock, birthday: Cake, exam: GraduationCap, interview: Briefcase, deadline: Flag, reminder: Bell } as const;

// A day as 2006-01-02, in the computer's time zone.
export const iso = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
export const parse = (s: string) => { const [y, m, d] = s.split("-").map(Number); return new Date(y, m - 1, d); };
export const addDays = (d: Date, n: number) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);

// The day of a month, months after start's (its last day when shorter).
function monthDay(start: Date, months: number): Date {
  const first = new Date(start.getFullYear(), start.getMonth() + months, 1);
  const last = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate();
  return new Date(first.getFullYear(), first.getMonth(), Math.min(start.getDate(), last));
}

// occurrences(ev, from, to): the days in [from, to] the event happens on.
export function occurrences(ev: Event, from: Date, to: Date): string[] {
  const start = parse(ev.date);
  const out: string[] = [];
  const add = (d: Date) => { if (d >= from && d <= to) out.push(iso(d)); };
  if (ev.repeat === "weekly") {
    let d = start;
    if (from > d) d = addDays(d, Math.floor((from.getTime() - d.getTime()) / 864e5 / 7) * 7);
    for (; d <= to; d = addDays(d, 7)) add(d);
  } else if (ev.repeat === "monthly" || ev.repeat === "yearly") {
    const step = ev.repeat === "yearly" ? 12 : 1;
    let first = (from.getFullYear() - start.getFullYear()) * 12 + from.getMonth() - start.getMonth();
    first = Math.max(0, first - (first % step));
    for (let i = first; ; i += step) {
      const d = monthDay(start, i);
      if (d > to) break;
      add(d);
    }
  } else add(start);
  return out;
}

// byDay(events, from, to): day -> its events, the whole-day ones first,
// then by time.
export function byDay(events: Event[], from: Date, to: Date): Map<string, Event[]> {
  const map = new Map<string, Event[]>();
  for (const ev of events) for (const d of occurrences(ev, from, to)) map.set(d, [...(map.get(d) ?? []), ev]);
  for (const list of map.values()) list.sort((a, b) => (a.time || "").localeCompare(b.time || "") || a.title.localeCompare(b.title));
  return map;
}
