import {describe,it,expect,vi} from "vitest";
import {betterAuth} from "better-auth";
import {drizzle} from "drizzle-orm/d1";
import {authOptions} from "./auth-options";
import {approveDevice,deviceMark,finishDevice} from "./devices";
import {accountAllowed} from "./lifecycle";
import {TestD1} from "./security-test-support";
import * as schema from "./db/schema";

describe("real Better Auth device redemption with D1 reservations",()=>{
	it("marks only the redeemed session and cannot replay a code",async()=>{
		const d1=new TestD1(),codes=new Map<string,string>();
		const cfg=authOptions(drizzle(d1.asD1(),{schema}),schema,{
			appUrl:"https://cloud.audit.invalid",rpID:"audit.invalid",secret:"audit-placeholder-only-12345678901234567890123456789",
			stripeSecret:"sk_test_audit_placeholder",stripeWebhookSecret:"audit-placeholder",turnstileSecret:"audit-placeholder",
			isBanned:async id=>!await accountAllowed(d1.asD1(),id),sendCode:(email,code)=>{codes.set(email,code);},
			deviceMark:(id,body)=>deviceMark(d1.asD1(),id,body),finishDevice:(id,body)=>finishDevice(d1.asD1(),id,body),
		});
		const service=betterAuth({...cfg,logger:{disabled:true},rateLimit:{enabled:false}});
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({success:true})));
		const call=(path:string,body?:unknown,cookie="",agent="forged browser agent")=>service.handler(new Request(`https://cloud.audit.invalid/api/auth${path}`,{method:body===undefined?"GET":"POST",headers:{origin:"https://cloud.audit.invalid","user-agent":agent,...(cookie?{cookie}:{}),...(body!==undefined?{"content-type":"application/json","x-captcha-response":"fixture"}:{})},body:body===undefined?undefined:JSON.stringify(body)}));
		try{
			expect((await call("/email-otp/send-verification-otp",{email:"device@audit.invalid",type:"sign-in"})).status).toBe(200);
			const login=await call("/sign-in/email-otp",{email:"device@audit.invalid",otp:codes.get("device@audit.invalid")},"","Blup Studio forged");
			expect(login.status).toBe(200);
			const cookie=login.headers.getSetCookie().map(c=>c.split(";")[0]).join("; ");
			const user=await service.api.getSession({headers:new Headers({cookie})});if(!user)throw new Error("session");
			expect(d1.sqlite.prepare("SELECT studio_device FROM session").get()?.studio_device).toBe(0);
			const code=await(await call("/device/code",{client_id:"blup-studio"})).json() as {user_code:string;device_code:string};
			expect((await call(`/device?user_code=${code.user_code}`,undefined,cookie)).status).toBe(200);
			expect(await approveDevice(d1.asD1(),user.user.id,code.user_code,1)).toBe("OK");
			const body={device_code:code.device_code,client_id:"blup-studio",grant_type:"urn:ietf:params:oauth:grant-type:device_code"};
			expect((await call("/device/token",body)).status).toBe(200);
			expect(d1.sqlite.prepare("SELECT count(*) AS n FROM session WHERE studio_device=1").get()?.n).toBe(1);
			expect(d1.sqlite.prepare("SELECT devices FROM resource_quota").get()?.devices).toBe(0);
			expect((await call("/device/token",body)).ok).toBe(false);
		}finally{vi.unstubAllGlobals();d1.close();}
	});
});
