import { describe, expect, it } from "vitest";
import { thin } from "./history";

const HOUR = 3_600_000;
const DAY = 24 * HOUR;
const now = Date.UTC(2026, 9, 31, 12);
// One version every 20 minutes for 40 days, newest first.
const versions = Array.from({ length: 40 * 72 }, (_, i) => ({ n: 40 * 72 - i, key: "", size: 1, createdAt: new Date(now - i * 20 * 60_000) }));

describe("thin", () => {
	it("keeps only the head without history", () => {
		expect([...thin(versions, 0, now)]).toEqual([versions[0]?.n]);
	});
	it("keeps the last day, then one per hour, then one per day, up to the plan's history", () => {
		const keep = thin(versions, 30, now);
		const kept = versions.filter((v) => keep.has(v.n));
		const age = (v: (typeof versions)[number]) => now - v.createdAt.getTime();
		expect(kept.filter((v) => age(v) <= DAY)).toHaveLength(versions.filter((v) => age(v) <= DAY).length);
		const week = kept.filter((v) => age(v) > DAY && age(v) <= 7 * DAY);
		expect(week.length).toBeGreaterThanOrEqual(6 * 24 - 1);
		expect(week.length).toBeLessThanOrEqual(6 * 24 + 1);
		const older = kept.filter((v) => age(v) > 7 * DAY);
		expect(older.length).toBeGreaterThanOrEqual(22);
		expect(older.length).toBeLessThanOrEqual(24);
		expect(kept.every((v) => age(v) <= 30 * DAY)).toBe(true);
	});
	it("always keeps the head, however old", () => {
		expect([...thin([{ n: 7, key: "", size: 1, createdAt: new Date(now - 400 * DAY) }], 30, now)]).toEqual([7]);
	});
});
