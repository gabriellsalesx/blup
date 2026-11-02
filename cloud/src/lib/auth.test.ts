import { afterEach, describe, expect, it, vi } from "vitest";
import { afterSignIn } from "./auth";

afterEach(()=>vi.unstubAllGlobals());
describe("login return paths",()=>{
	it.each(["//evil.invalid/","/\\evil.invalid/","/\n/evil.invalid/","/\t\\evil.invalid/","https://evil.invalid/","javascript:alert(1)"])("refuses browser-normalized external targets %s",back=>{
		vi.stubGlobal("location",new URL("https://cloud.audit.invalid/entrar?volta="+encodeURIComponent(back)));
		expect(afterSignIn()).toBe("/conta");
	});
	it("keeps an internal path and its query",()=>{
		vi.stubGlobal("location",new URL("https://cloud.audit.invalid/entrar?volta="+encodeURIComponent("/aparelho?user_code=ABCDEFGH")));
		expect(afterSignIn()).toBe("/aparelho?user_code=ABCDEFGH");
	});
});
