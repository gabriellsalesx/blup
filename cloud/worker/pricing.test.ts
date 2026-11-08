import { afterEach,beforeEach,describe,expect,it,vi } from "vitest";
import { PriceCatalog,publicPlans,purchaseAvailable,validatePrices } from "./pricing";
import { createStripe } from "./stripe-client";
import { resetRatesForTest } from "./fx";

function price(key="blup_pro_mensal",amount=1750,currency="brl"){
	return {id:`price_${key}`,object:"price",active:true,lookup_key:key,unit_amount:amount,currency,type:"recurring",billing_scheme:"per_unit",recurring:{interval:key.endsWith("anual")?"year":"month",interval_count:1,usage_type:"licensed"},product:{id:"prod_fixture",active:true}};
}
const list=(...data:unknown[])=>({object:"list",has_more:false,data});
beforeEach(()=>{vi.useFakeTimers();vi.setSystemTime(new Date("2026-11-02T12:00:00Z"));vi.stubGlobal("fetch",vi.fn(async()=>{throw new Error("External network forbidden");}));});
afterEach(()=>{vi.useRealTimers();vi.unstubAllGlobals();resetRatesForTest();});
describe("Stripe display catalog",()=>{
	it("reads monthly and annual amounts/currencies from the actual lookup response",async()=>{
		const urls:string[]=[];
		vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
			if(String(input).startsWith("https://api.frankfurter.dev/"))return Response.json({amount:1,base:"BRL",date:"2026-11-02",rates:{USD:0.2,EUR:0.1}});
			urls.push(String(input));expect(new Headers(init?.headers).get("stripe-version")).toBe("2026-10-28.endive");
			const asked=[...new URL(String(input)).searchParams].filter(([k])=>k.startsWith("lookup_keys")).map(([,v])=>v);
			return Response.json(list(...[price(),price("blup_pro_anual",18900,"usd")].filter(p=>asked.includes(p.lookup_key))));
		}));
		const env={STRIPE_SECRET_KEY:"sk_test_audit_catalog_reads"};
		const plans=await publicPlans(env),pro=plans.find(p=>p.name==="pro");
		expect(pro).toMatchObject({prices:{monthly:{unitAmount:1750,currency:"brl",approx:{usd:350,eur:175}},yearly:{unitAmount:18900,currency:"usd",approx:null}}});
		expect(pro).not.toHaveProperty("priceCents");expect(plans.find(p=>p.name==="max")?.prices.monthly).toBeNull();
		expect(await purchaseAvailable(env,"pro",true)).toBe(true);expect(await purchaseAvailable(env,"max",false)).toBe(false);
		// Stripe accepts at most 10 lookup_keys per list call.
		expect(urls).toHaveLength(2);for(const url of urls){expect(url).toContain("active=true");expect(url).toContain("data.product");expect([...new URL(url).searchParams.keys()].filter(k=>k.startsWith("lookup_keys")).length).toBeLessThanOrEqual(10);}
		expect(urls.join(" ")).toContain("blup_ultra_anual");expect(urls.join(" ")).toContain("blup_ultra_pulse_deals_anual");
	});
	it("coalesces concurrent refreshes and refreshes only after ten minutes",async()=>{
		const load=vi.fn(async()=>list(price()));const catalog=new PriceCatalog(load);
		await Promise.all(Array.from({length:20},()=>catalog.prices()));expect(load).toHaveBeenCalledTimes(1);
		vi.setSystemTime(Date.now()+599_999);await catalog.prices();expect(load).toHaveBeenCalledTimes(1);
		load.mockResolvedValueOnce(list(price("blup_pro_mensal",2100)));vi.setSystemTime(Date.now()+2);
		expect((await catalog.prices()).get("blup_pro_mensal")?.unitAmount).toBe(2100);expect(load).toHaveBeenCalledTimes(2);
	});
	it("retains last good values after failure and backs off retries",async()=>{
		const load=vi.fn<()=>Promise<unknown>>().mockResolvedValueOnce(list(price())).mockRejectedValue(new Error("fixture outage"));const catalog=new PriceCatalog(load);
		await catalog.prices();vi.setSystemTime(Date.now()+600_001);expect((await catalog.prices()).get("blup_pro_mensal")?.unitAmount).toBe(1750);
		await catalog.prices();expect(load).toHaveBeenCalledTimes(2);vi.setSystemTime(Date.now()+30_001);await catalog.prices();expect(load).toHaveBeenCalledTimes(3);
	});
	it("has no invented amount on a cold failure or malformed provider list",async()=>{
		const env={STRIPE_SECRET_KEY:"sk_test_audit_cold_outage"};const plans=await publicPlans(env);
		expect(plans.filter(p=>p.name!=="free").every(p=>p.prices.monthly===null&&p.prices.yearly===null)).toBe(true);
		expect(await purchaseAvailable(env,"pro",false)).toBe(false);
		const bad=new PriceCatalog(async()=>({object:"list",has_more:true,data:[price()]}));expect((await bad.prices()).size).toBe(0);
	});
	it.each([NaN,Infinity,-1,1.5,"1490",null])("does not display invalid unit_amount %s",value=>{
		expect(validatePrices(list({...price(),unit_amount:value})).get("blup_pro_mensal")).toBeNull();
	});
	it("disables archived products, inactive/wrong-interval prices, duplicates and missing keys",()=>{
		for(const invalid of [{...price(),active:false},{...price(),product:{id:"prod_fixture",active:false}},{...price(),recurring:{interval:"year",interval_count:1,usage_type:"licensed"}},{...price(),transform_quantity:{divide_by:10,round:"up"}}])expect(validatePrices(list(invalid)).get("blup_pro_mensal")).toBeNull();
		expect(validatePrices(list(price(),price())).get("blup_pro_mensal")).toBeNull();
		expect(validatePrices(list()).get("blup_pro_mensal")).toBeNull();
	});
	it("does not revive a deliberately removed price from the old snapshot",async()=>{
		const load=vi.fn(async()=>list(price()));const catalog=new PriceCatalog(load);await catalog.prices();
		load.mockResolvedValueOnce(list());vi.setSystemTime(Date.now()+600_001);expect((await catalog.prices()).get("blup_pro_mensal")).toBeNull();
	});
	it("supports a deliberately zero-amount recurring price",()=>{
		expect(validatePrices(list(price("blup_pro_mensal",0))).get("blup_pro_mensal")).toEqual({unitAmount:0,currency:"brl"});
	});
	it("pins the Stripe SDK to the supported stable API",()=>{
		expect(createStripe("sk_test_audit_fixture").getApiField("version")).toBe("2026-10-28.endive");
	});
});
