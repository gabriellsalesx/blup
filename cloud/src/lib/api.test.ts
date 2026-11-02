import {afterEach,describe,expect,it,vi} from "vitest";
import {api} from "./api";
import {money} from "./money";
import {money as staffMoney} from "./staff";
afterEach(()=>vi.unstubAllGlobals());
describe("public configuration RPC",()=>{
	it("rejects a failed config request instead of accepting its JSON as configuration",async()=>{
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({code:"INTERNAL"},{status:500})));
		await expect(api.config()).rejects.toBeDefined();
	});
	it("reads a successful configuration through the inferred endpoint",async()=>{
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({turnstileSiteKey:"public-fixture"})));
		expect(await api.config()).toEqual({turnstileSiteKey:"public-fixture"});
	});
});
describe("Stripe minor-unit display",()=>{
	it("formats BRL, JPY and Stripe's ISK exception without inventing decimals",()=>{
		expect(money(1750,"brl")).toContain("17,50");expect(money(500,"jpy")).toContain("500");expect(money(500,"isk")).toMatch(/\b5\b/);
	});
	it("uses the same units for staff payments and does not guess a missing currency",()=>{
		expect(staffMoney(500,"jpy")).toBe(money(500,"jpy"));
		expect(staffMoney(500,null)).toBe("");expect(staffMoney(null,"brl")).toBe("");
	});
});
