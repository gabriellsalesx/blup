import { describe, expect, it } from "vitest";
import { GIFT_PLAN, ownerMatches, planNamed } from "./plans";

describe("the owner's gift", () => {
	it("is the top plan, with 10 sites", () => {
		expect(planNamed(GIFT_PLAN).name).toBe("ultra");
		expect(planNamed(GIFT_PLAN).limits.sites).toBe(10);
	});
	it("goes only to the owner's verified e-mail", () => {
		expect(ownerMatches("Dono@Blup.io ", "dono@blup.io", true)).toBe(true);
		expect(ownerMatches("dono@blup.io", "dono@blup.io", false)).toBe(false);
		expect(ownerMatches("dono@blup.io", "outro@blup.io", true)).toBe(false);
		expect(ownerMatches("dono@blup.io", "dono@blup.io.evil", true)).toBe(false);
		expect(ownerMatches("", "", true)).toBe(false);
		expect(ownerMatches(undefined, undefined, true)).toBe(false);
	});
});
