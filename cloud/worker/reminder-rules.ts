import { z } from "zod";

// The rules of the calendar's reminders by e-mail, apart from any binding
// so the tests can reach them (worker/reminders.ts uses them).

export const DAY = 24 * 3600 * 1000;
// How far ahead the list goes, and how late a reminder may still go out.
export const AHEAD = 32 * DAY;
export const LATE = 12 * 3600 * 1000;
const MAX_LIST = 300;

// "2026-11": the month a reminder counts in (UTC).
export const monthOf = (ms: number) => new Date(ms).toISOString().slice(0, 7);

export const listSchema = z.object({
	reminders: z
		.array(z.object({ id: z.string().regex(/^[0-9a-f]{32,64}$/), at: z.number().int().positive() }))
		.max(MAX_LIST)
		.refine((l) => new Set(l.map((r) => r.id)).size === l.length, "duplicate id"),
});

// The ones worth keeping: not past (a little late is fine), not too far.
export const keep = (list: { id: string; at: number }[], now: number) => list.filter((r) => r.at > now - LATE && r.at <= now + AHEAD);

