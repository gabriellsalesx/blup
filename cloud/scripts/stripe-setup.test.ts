import {afterEach,describe,expect,it,vi} from "vitest";
import {checkCatalog} from "./stripe-setup";
afterEach(()=>vi.unstubAllGlobals());
describe("read-only Stripe setup",()=>{
	it("supports restricted sandbox keys and reports missing prices without writing",async()=>{
		const requests:{url:string;method:string}[]=[];
		vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{requests.push({url:String(input),method:init?.method??"GET"});return Response.json({object:"list",has_more:false,data:[]});}));
		const messages:string[]=[];
		expect(await checkCatalog("rk_test_audit_fixture",message=>messages.push(message))).toBe(false);
		expect(requests).toHaveLength(1);expect(requests[0]?.method).toBe("GET");expect(messages.join(" ")).toContain("blup_pro_anual");expect(messages.join(" ")).not.toContain("rk_test_");
	});
	it("does not send a missing or placeholder credential to the network",async()=>{
		const fetchMock=vi.fn();vi.stubGlobal("fetch",fetchMock);
		expect(await checkCatalog("",()=>{})).toBe(false);expect(await checkCatalog("rk_test_coloque_a_sua",()=>{})).toBe(false);expect(fetchMock).not.toHaveBeenCalled();
	});
});
