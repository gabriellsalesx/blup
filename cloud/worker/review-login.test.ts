import { beforeEach, describe, expect, it, vi } from "vitest";
import { betterAuth } from "better-auth";
import { memoryAdapter } from "better-auth/adapters/memory";
import { authOptions, reviewLogin, type ReviewLogin } from "./auth-options";

vi.mock("cloudflare:workers",()=>({env:{},DurableObject:class{},waitUntil:()=>{}}));

const ORIGIN="https://cloud.audit.invalid";
const REVIEW:ReviewLogin={email:"review@audit.invalid",code:"428193"};
type Row=Record<string,unknown>;
let sent:{email:string;code:string;kind:string}[];
let service:ReturnType<typeof makeAuth>;

function makeAuth(review:ReviewLogin|null){
	const tables:Record<string,Row[]>={user:[],session:[],account:[],verification:[],passkey:[],subscription:[],deviceCode:[]};
	const options=authOptions({} as never,{}, {
		appUrl:ORIGIN,rpID:"audit.invalid",secret:"local-audit-placeholder-not-a-real-secret-123456789",
		turnstileSecret:"audit-only-placeholder",stripeSecret:"sk_test_audit_only_placeholder",stripeWebhookSecret:"audit-only-placeholder",
		accountBlock:async()=>null,sendCode:(email,code,kind)=>{sent.push({email,code,kind});},review,
	});
	return betterAuth({...options,database:memoryAdapter(tables),rateLimit:{enabled:false},logger:{disabled:true}});
}
const call=(path:string,body:unknown)=>service.handler(new Request(ORIGIN+"/api/auth"+path,{method:"POST",headers:{origin:ORIGIN,"content-type":"application/json","x-captcha-response":"audit-only"},body:JSON.stringify(body)}));
const ask=(email:string)=>call("/email-otp/send-verification-otp",{email,type:"sign-in"});
const signIn=(email:string,otp:string)=>call("/sign-in/email-otp",{email,otp});

beforeEach(()=>{
	sent=[];
	vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL)=>{
		if(String(input).startsWith("https://challenges.cloudflare.com/turnstile/v0/siteverify"))return Response.json({success:true});
		throw new Error("External network forbidden");
	}));
});

describe("reviewLogin",()=>{
	it("needs a valid e-mail and a 6-digit code",()=>{
		expect(reviewLogin(" Review@Audit.invalid ","428193")).toEqual(REVIEW);
		expect(reviewLogin(undefined,undefined)).toBeNull();
		expect(reviewLogin("","")).toBeNull();
		expect(reviewLogin("review@audit.invalid","12345")).toBeNull();
		expect(reviewLogin("review@audit.invalid","abcdef")).toBeNull();
		expect(reviewLogin("not-an-email","428193")).toBeNull();
	});
});

describe("review account sign in",()=>{
	it("signs in with the fixed code, again and again, and sends no e-mail",async()=>{
		service=makeAuth(REVIEW);
		for(const email of ["review@audit.invalid","Review@Audit.invalid"]){
			expect((await ask(email)).status).toBe(200);
			expect((await signIn(email,REVIEW.code)).status).toBe(200);
		}
		expect(sent).toHaveLength(0);
	});
	it("refuses a wrong code for the review account",async()=>{
		service=makeAuth(REVIEW);
		await ask(REVIEW.email);
		expect((await signIn(REVIEW.email,"000000")).status).not.toBe(200);
	});
	it("leaves every other account on a random e-mailed code",async()=>{
		service=makeAuth(REVIEW);
		await ask("alice@audit.invalid");
		expect(sent).toHaveLength(1);
		expect(sent[0]!.email).toBe("alice@audit.invalid");
		expect((await signIn("alice@audit.invalid",REVIEW.code)).status).not.toBe(200);
		expect((await signIn("alice@audit.invalid",sent[0]!.code)).status).toBe(200);
	});
	it("without the secrets the review e-mail is an ordinary account",async()=>{
		service=makeAuth(null);
		await ask(REVIEW.email);
		expect(sent).toHaveLength(1);
		expect((await signIn(REVIEW.email,REVIEW.code)).status).not.toBe(200);
	});
});
