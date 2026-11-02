import { describe, expect, it } from "vitest";
import { AHEAD, LATE, keep, listSchema, monthOf } from "./reminder-rules";
import { planNamed } from "./plans";

const id = (n: number) => n.toString(16).padStart(64, "0");

describe("reminders by e-mail", () => {
	it("are in Max (50) and Ultra (100) only", () => {
		expect(["free", "pro", "max", "ultra"].map((p) => planNamed(p).limits.reminderEmails)).toEqual([0, 0, 50, 100]);
	});
	it("take only an opaque id and a time", () => {
		expect(listSchema.safeParse({ reminders: [{ id: id(1), at: 1 }] }).success).toBe(true);
		expect(listSchema.safeParse({ reminders: [{ id: "Prova de cálculo", at: 1 }] }).success).toBe(false);
		expect(listSchema.safeParse({ reminders: [{ id: id(1), at: 1, title: "x" }] }).data).toEqual({ reminders: [{ id: id(1), at: 1 }] });
		expect(listSchema.safeParse({ reminders: [{ id: id(1), at: 1 }, { id: id(1), at: 2 }] }).success).toBe(false);
	});
	it("keep what is due soon, not the past nor too far", () => {
		const now = Date.UTC(2026, 10, 2, 12);
		const list = [{ id: id(1), at: now - LATE - 1 }, { id: id(2), at: now - 1000 }, { id: id(3), at: now + AHEAD }, { id: id(4), at: now + AHEAD + 1 }];
		expect(keep(list, now).map((r) => r.id)).toEqual([id(2), id(3)]);
	});
	it("count by month", () => {
		expect(monthOf(Date.UTC(2026, 10, 30, 23, 59))).toBe("2026-11");
		expect(monthOf(Date.UTC(2026, 11, 1))).toBe("2026-12");
	});
});
