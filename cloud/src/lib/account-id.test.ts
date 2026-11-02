import {describe,expect,it,vi} from "vitest";
import {accountIdHelp,copyAccountId,shortAccountId} from "./account-id";
describe("account ID support",()=>{
	it("copies the complete ID even when the list displays its short form",async()=>{
		const id="01900000-0000-7000-8000-000000000001",writeText=vi.fn(async()=>{});
		expect(shortAccountId(id)).toBe("01900000…0001");expect(await copyAccountId(id,{writeText})).toBe(true);expect(writeText).toHaveBeenCalledWith(id);
		expect(accountIdHelp()).toBe("Em caso de ajuda, informe seu ID.");
	});
	it("uses the approved help text in each language",()=>{expect(accountIdHelp("en")).toBe("If you need help, share your ID.");expect(accountIdHelp("es")).toBe("Si necesitas ayuda, comparte tu ID.");});
	it("does not report success if the clipboard is unavailable, denied or the ID is empty",async()=>{
		expect(await copyAccountId("fixture",undefined)).toBe(false);expect(await copyAccountId("fixture",{writeText:async()=>{throw new Error("denied");}})).toBe(false);expect(await copyAccountId("",{writeText:async()=>{}})).toBe(false);
	});
});
