import {describe,expect,it} from "vitest";
import {closeCodeToSend} from "./live-close";
describe("live sync close",()=>{
	it("never answers with a code only a receiver may see",()=>{
		for(const code of [1005,1006,1015,1004,999,5000])expect(closeCodeToSend(code)).toBe(1000);
		for(const code of [1000,1001,1008,1011,3000,4999])expect(closeCodeToSend(code)).toBe(code);
	});
});
