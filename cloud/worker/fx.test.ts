import {afterEach,describe,expect,it,vi} from "vitest";
import {approximate,brlRates,resetRatesForTest} from "./fx";
import {displayCurrency,shownPrice} from "../shared/display-currency";

const ok=(rates:unknown)=>vi.fn(async()=>Response.json({amount:1,base:"BRL",date:"2026-11-06",rates})) as unknown as typeof fetch;

describe("approximate plan prices",()=>{
	afterEach(()=>resetRatesForTest());
	it("shows English in dollars and Spanish in euros, Portuguese in reais",()=>{
		expect(displayCurrency).toEqual({pt:null,en:"usd",es:"eur"});
		const price={unitAmount:2990,currency:"brl",approx:{usd:543,eur:487}};
		expect(shownPrice(price,"en")).toEqual({unitAmount:543,currency:"usd",approx:true});
		expect(shownPrice(price,"es")).toEqual({unitAmount:487,currency:"eur",approx:true});
		expect(shownPrice(price,"pt")).toEqual({unitAmount:2990,currency:"brl",approx:false});
		expect(shownPrice({...price,approx:null},"en")).toEqual({unitAmount:2990,currency:"brl",approx:false});
	});
	it("converts BRL cents with the rates",()=>{
		expect(approximate({unitAmount:2990,currency:"brl"},{usd:0.1817,eur:0.16274})).toEqual({usd:543,eur:487});
	});
	it("does nothing without rates or a BRL price",()=>{
		expect(approximate({unitAmount:2990,currency:"brl"},null)).toBeNull();
		expect(approximate({unitAmount:299,currency:"usd"},{usd:0.18,eur:0.16})).toBeNull();
		expect(approximate(null,{usd:0.18,eur:0.16})).toBeNull();
	});
	it("reads the rates once and keeps them",async()=>{
		const send=ok({USD:0.18,EUR:0.16});
		expect(await brlRates(send)).toEqual({usd:0.18,eur:0.16});
		expect(await brlRates(send)).toEqual({usd:0.18,eur:0.16});
		expect(send).toHaveBeenCalledTimes(1);
	});
	it("refuses a strange answer and a failed request",async()=>{
		expect(await brlRates(ok({USD:-1,EUR:0.16}))).toBeNull();
		expect(await brlRates(vi.fn(async()=>new Response("no",{status:503})) as unknown as typeof fetch)).toBeNull();
		expect(await brlRates(vi.fn(async()=>{throw new Error("down");}) as unknown as typeof fetch)).toBeNull();
	});
});
