import {describe,expect,it,vi} from "vitest";
import {auth} from "./auth.cli";
import {authOptions} from "./auth-options";
import {drizzle} from "drizzle-orm/sqlite-proxy";
describe("auth configuration and logging",()=>{
	it("initializes schema tooling without an asserted fake D1 binding",async()=>{
		const context=await auth.$context;
		expect(context.options.plugins?.some(p=>p.id==="stripe")).toBe(true);
		expect(context.options.plugins?.some(p=>p.id==="passkey")).toBe(true);
	});
	it("disables the plugin webhook so an old payload cannot overwrite reconciliation",async()=>{
		const context=await auth.$context;
		expect(context.options.disabledPaths).toContain("/stripe/webhook");
		const response=await auth.handler(new Request("http://localhost/api/auth/stripe/webhook",{method:"POST",body:"{}",headers:{"content-type":"application/json"}}));
		expect(response.status).toBe(404);
	});
	it("keeps SDK errors from logging customer or credential fields",()=>{
		const options=authOptions(drizzle(async()=>{throw new Error("No SQL");}),{}, {appUrl:"https://cloud.audit.invalid",rpID:"audit.invalid",secret:"audit-placeholder-long-secret",turnstileSecret:"fixture",stripeSecret:"sk_test_fixture",stripeWebhookSecret:"whsec_fixture",isBanned:async()=>false,sendCode:()=>{}});
		const log=vi.spyOn(console,"error").mockImplementation(()=>{});
		try{
			options.logger.log("error","customer@audit.invalid",{token:"private-fixture",query:"private-query"});
			expect(log.mock.calls).toEqual([["[blup] auth_error"]]);
		}finally{log.mockRestore();}
	});
});
