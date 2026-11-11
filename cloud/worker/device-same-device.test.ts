import {describe,it,expect,vi} from "vitest";
import {betterAuth} from "better-auth";
import {drizzle} from "drizzle-orm/d1";
import {accountBlock,authOptions} from "./auth-options";
import {accountDevice,approveDevice,deviceMark,finishDevice,rememberDevice} from "./devices";
import {TestD1} from "./security-test-support";
import * as schema from "./db/schema";

const ORIGIN="https://cloud.audit.invalid";
const PHONE="a".repeat(64),OTHER="b".repeat(64);

// Blup Notes or Blup Studio signing in again on the same device (installed
// again) takes its old session's place; another device still meets the
// plan's limit; an app without the mark works as before.
describe("the same device signing in again",()=>{
	async function setup(){
		const d1=new TestD1(),codes=new Map<string,string>();
		const cfg=authOptions(drizzle(d1.asD1(),{schema}),schema,{
			appUrl:ORIGIN,rpID:"audit.invalid",secret:"audit-placeholder-only-12345678901234567890123456789",
			stripeSecret:"sk_test_audit_placeholder",stripeWebhookSecret:"audit-placeholder",turnstileSecret:"audit-placeholder",
			accountBlock:id=>accountBlock(d1.asD1(),id),sendCode:(email,code)=>{codes.set(email,code);},
			deviceMark:(id,body)=>deviceMark(d1.asD1(),id,body),
			finishDevice:(id,body,session)=>finishDevice(d1.asD1(),id,body,session),
			rememberDevice:(code,device,expiresIn)=>rememberDevice(d1.asD1(),code,device,expiresIn),
		});
		const service=betterAuth({...cfg,logger:{disabled:true},rateLimit:{enabled:false}});
		vi.stubGlobal("fetch",vi.fn(async()=>Response.json({success:true})));
		const call=(path:string,body?:unknown,headers:Record<string,string>={})=>service.handler(new Request(`${ORIGIN}/api/auth${path}`,{method:body===undefined?"GET":"POST",headers:{origin:ORIGIN,...headers,...(body!==undefined?{"content-type":"application/json","x-captcha-response":"fixture"}:{})},body:body===undefined?undefined:JSON.stringify(body)}));
		await call("/email-otp/send-verification-otp",{email:"phone@audit.invalid",type:"sign-in"});
		const login=await call("/sign-in/email-otp",{email:"phone@audit.invalid",otp:codes.get("phone@audit.invalid")});
		const cookie=login.headers.getSetCookie().map(c=>c.split(";")[0]).join("; ");
		const me=await service.api.getSession({headers:new Headers({cookie})});if(!me)throw new Error("session");
		// One device signs in: the code (with or without the mark), the
		// approval in the panel (limit), then the token.
		const signIn=async(mark:string|null,limit:number,agent:string)=>{
			const code=await(await call("/device/code",{client_id:"blup-studio"},mark?{"x-blup-device":mark}:{})).json() as {user_code:string;device_code:string};
			await call(`/device?user_code=${code.user_code}`,undefined,{cookie});
			const approved=await approveDevice(d1.asD1(),me.user.id,code.user_code,limit);
			if(approved!=="OK")return approved;
			const token=await call("/device/token",{device_code:code.device_code,client_id:"blup-studio",grant_type:"urn:ietf:params:oauth:grant-type:device_code"},{"user-agent":agent});
			return token.ok?"SIGNED_IN":`TOKEN_${token.status}`;
		};
		const sessions=()=>d1.sqlite.prepare("SELECT device_id AS device,user_agent AS agent FROM session WHERE studio_device=1 ORDER BY created_at").all() as {device:string|null;agent:string}[];
		return {d1,me,signIn,sessions};
	}

	it("replaces the old session instead of counting the phone twice",async()=>{
		const {d1,me,signIn,sessions}=await setup();
		try{
			expect(await signIn(PHONE,1,"Blup Notes (Android; Galaxy S23)")).toBe("SIGNED_IN");
			const id=await accountDevice(me.user.id,PHONE);
			expect(sessions()).toEqual([{device:id,agent:"Blup Notes (Android; Galaxy S23)"}]);
			// Installed again: the plan allows 1 device, and it is the same one.
			expect(await signIn(PHONE,1,"Blup Notes (Android; Galaxy S23) again")).toBe("SIGNED_IN");
			expect(sessions()).toEqual([{device:id,agent:"Blup Notes (Android; Galaxy S23) again"}]);
			// The account's id is not the device's mark (another account gets another).
			expect(id).not.toBe(PHONE);
			expect(await accountDevice("another-user",PHONE)).not.toBe(id);
			// Nothing left behind: the code's device is forgotten, the place released.
			expect(d1.sqlite.prepare("SELECT count(*) AS n FROM device_pending").get()?.n).toBe(0);
			expect(d1.sqlite.prepare("SELECT devices FROM resource_quota").get()?.devices).toBe(0);
		}finally{vi.unstubAllGlobals();d1.close();}
	});

	it("still refuses another device over the limit, and an app without the mark counts as before",async()=>{
		const {d1,signIn,sessions}=await setup();
		try{
			expect(await signIn(PHONE,1,"Blup Notes (Android; Galaxy S23)")).toBe("SIGNED_IN");
			expect(await signIn(OTHER,1,"Blup Studio (Linux; pc)")).toBe("DEVICES_LIMIT");
			expect(await signIn(null,1,"Blup Studio (Linux; old)")).toBe("DEVICES_LIMIT");
			expect(await signIn(null,2,"Blup Studio (Linux; old)")).toBe("SIGNED_IN");
			expect(sessions().map(s=>s.agent)).toEqual(["Blup Notes (Android; Galaxy S23)","Blup Studio (Linux; old)"]);
			expect(sessions()[1]?.device).toBeNull();
		}finally{vi.unstubAllGlobals();d1.close();}
	});

	it("ignores a mark that is not one Blup sends",async()=>{
		const {d1,signIn,sessions}=await setup();
		try{
			expect(await signIn("not-a-mark' OR 1=1 --",2,"Blup Studio (Linux; pc)")).toBe("SIGNED_IN");
			expect(sessions()[0]?.device).toBeNull();
			expect(d1.sqlite.prepare("SELECT count(*) AS n FROM device_pending").get()?.n).toBe(0);
		}finally{vi.unstubAllGlobals();d1.close();}
	});
});
