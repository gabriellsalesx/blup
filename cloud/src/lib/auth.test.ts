import { afterEach, describe, expect, it, vi } from "vitest";
import { afterSignIn } from "./auth";

afterEach(()=>vi.unstubAllGlobals());
describe("login return paths",()=>{
	it.each(["//evil.invalid/","/\\evil.invalid/","/\n/evil.invalid/","/\t\\evil.invalid/","https://evil.invalid/","javascript:alert(1)"])("refuses browser-normalized external targets %s",back=>{
		vi.stubGlobal("location",new URL("https://cloud.audit.invalid/sign-in?next="+encodeURIComponent(back)));
		expect(afterSignIn()).toBe("/account");
	});
	it("keeps an internal path and its query",()=>{
		vi.stubGlobal("location",new URL("https://cloud.audit.invalid/sign-in?next="+encodeURIComponent("/device?user_code=ABCDEFGH")));
		expect(afterSignIn()).toBe("/device?user_code=ABCDEFGH");
	});
});
