import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { betterAuth } from "better-auth";
import { memoryAdapter } from "better-auth/adapters/memory";
import { authOptions, NO_EMAIL_DOMAIN } from "./auth-options";
import { isoCBOR } from "@simplewebauthn/server/helpers";

const runtime=vi.hoisted(()=>({service:undefined as unknown as {api:{getSession:(v:unknown)=>Promise<unknown>};handler:(r:Request)=>Promise<Response>}}));
vi.mock("cloudflare:workers",()=>({env:{},DurableObject:class{},waitUntil:()=>{}}));
vi.mock("./auth",()=>({db:{},auth:{api:{getSession:(v:unknown)=>runtime.service.api.getSession(v)},handler:(r:Request)=>runtime.service.handler(r)}}));
import worker from "./index";

const ORIGIN="https://cloud.audit.invalid";
let codes:Map<string,string>;
let tables:Record<string,Record<string,unknown>[]>;
let service:ReturnType<typeof makeAuth>;
function makeAuth(){
	const options=authOptions({} as never,{}, {
		appUrl:ORIGIN,rpID:"audit.invalid",secret:"local-audit-placeholder-not-a-real-secret-123456789",
		turnstileSecret:"audit-only-placeholder",stripeSecret:"sk_test_audit_only_placeholder",stripeWebhookSecret:"audit-only-placeholder",
		isBanned:async()=>false,sendCode:(email,code)=>{codes.set(email,code);},
	});
	return betterAuth({...options,advanced:{...options.advanced,disableOriginCheck:false,disableCSRFCheck:false},database:memoryAdapter(tables),rateLimit:{enabled:false},logger:{disabled:true}});
}
beforeEach(()=>{
	codes=new Map();tables={user:[],session:[],account:[],verification:[],passkey:[],subscription:[],deviceCode:[]};
	service=makeAuth();
	runtime.service=service as unknown as typeof runtime.service;
	vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL)=>{
		if(String(input).startsWith("https://challenges.cloudflare.com/turnstile/v0/siteverify"))return Response.json({success:true});
		throw new Error("External network forbidden");
	}));
});
afterEach(()=>vi.unstubAllGlobals());

async function call(path:string,body?:unknown,cookie="",origin=ORIGIN,extra:Record<string,string>={}){
	return service.handler(new Request(ORIGIN+"/api/auth"+path,{method:body===undefined?"GET":"POST",headers:{origin,...(cookie?{cookie}:{}),...(body===undefined?{}:{"content-type":"application/json"}),...extra},body:body===undefined?undefined:JSON.stringify(body)}));
}
async function login(email="alice@audit.invalid"){
	const sent=await call("/email-otp/send-verification-otp",{email,type:"sign-in"},"",ORIGIN,{"x-captcha-response":"audit-only"});
	expect(sent.status).toBe(200);
	const otp=codes.get(email)!;
	const res=await call("/sign-in/email-otp",{email,otp});
	expect(res.status).toBe(200);
	const cookie=res.headers.getSetCookie().map(c=>c.split(";")[0]).join("; ");
	return {res,cookie,otp};
}
const bytes64=(s:string)=>Uint8Array.from(atob(s.replace(/-/g,"+").replace(/_/g,"/")),c=>c.charCodeAt(0));
const b64=(b:Uint8Array)=>btoa(String.fromCharCode(...b)).replace(/\+/g,"-").replace(/\//g,"_").replace(/=+$/,"");
const join=(...parts:Uint8Array[])=>{const out=new Uint8Array(parts.reduce((n,p)=>n+p.length,0));let at=0;for(const p of parts){out.set(p,at);at+=p.length;}return out;};
const cookieOf=(r:Response)=>r.headers.getSetCookie().map(c=>c.split(";")[0]).join("; ");

async function registerPasskey(language="en"){
	const optionsResponse=await call("/passkey/generate-register-options",undefined,"",ORIGIN,{"x-blup-language":language});
	expect(optionsResponse.status).toBe(200);
	const options=await optionsResponse.json() as {challenge:string};
	expect(tables.user).toHaveLength(0);
	const key=await crypto.subtle.generateKey({name:"ECDSA",namedCurve:"P-256"},true,["sign","verify"]) as CryptoKeyPair;
	const pub=await crypto.subtle.exportKey("jwk",key.publicKey) as JsonWebKey;
	const cose=isoCBOR.encode(new Map<number,unknown>([[1,2],[3,-7],[-1,1],[-2,bytes64(pub.x!)],[-3,bytes64(pub.y!)]]) as never);
	const id=crypto.getRandomValues(new Uint8Array(16));
	const rp=new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode("audit.invalid")));
	const authData=join(rp,new Uint8Array([0x45,0,0,0,0]),new Uint8Array(16),new Uint8Array([0,id.length]),id,cose);
	const att=isoCBOR.encode(new Map<string,unknown>([["fmt","none"],["attStmt",new Map()],["authData",authData]]) as never);
	const response={id:b64(id),rawId:b64(id),type:"public-key",response:{clientDataJSON:b64(new TextEncoder().encode(JSON.stringify({type:"webauthn.create",challenge:options.challenge,origin:ORIGIN}))),attestationObject:b64(att)},clientExtensionResults:{},authenticatorAttachment:"platform"};
	const res=await call("/passkey/verify-registration",{response,name:"local audit",createSession:true},cookieOf(optionsResponse),ORIGIN,{"x-blup-language":language});
	expect(res.status).toBe(200);
	return {key,id,rp,res};
}

describe("Better Auth 1.7.14, actual plugins and endpoints",()=>{
 it("reads a saved account language again instead of using an old session snapshot",async()=>{
  const {cookie}=await login();
  if(!tables.user[0])throw new Error("Missing fixture account");tables.user[0].language="es";
  const response=await call("/get-session",undefined,cookie);
  expect(response.status).toBe(200);expect(await response.json()).toMatchObject({user:{language:"es"}});
 });

 it("stores the selected language for a new passkey-only account",async()=>{
  await registerPasskey("es");expect(tables.user[0]?.language).toBe("es");
 });

 it.each(["pt","en","es"] as const)("stores negotiated %s when creating an email account",async language=>{
  const email="localized@audit.invalid";
  const send=await service.handler(new Request(ORIGIN+"/api/auth/email-otp/send-verification-otp",{method:"POST",headers:{origin:ORIGIN,"content-type":"application/json","x-captcha-response":"audit-only","x-blup-language":language},body:JSON.stringify({email,type:"sign-in"})}));
  expect(send.status).toBe(200);
  const response=await service.handler(new Request(ORIGIN+"/api/auth/sign-in/email-otp",{method:"POST",headers:{origin:ORIGIN,"content-type":"application/json","x-blup-language":language},body:JSON.stringify({email,otp:codes.get(email)})}));
  expect(response.status).toBe(200);expect(tables.user[0]?.language).toBe(language);
  const changed=await call("/update-user",{language:language==="pt"?"es":"pt",name:"Localized"},cookieOf(response));
  expect([200,400]).toContain(changed.status);expect(tables.user[0]?.language).toBe(language);
 });

	it("creates a passkey-only account and verifies a cryptographically signed assertion",async()=>{
		const {key,id,rp,res}=await registerPasskey();
		expect(tables.user).toHaveLength(1);expect(tables.passkey).toHaveLength(1);
		expect(String(tables.user[0].email)).toMatch(new RegExp(`@${NO_EMAIL_DOMAIN.replace(/\./g,"\\.")}$`));
		expect(tables.user[0].emailVerified).toBe(false);expect(cookieOf(res)).toContain("session_token");
		const start=await call("/passkey/generate-authenticate-options");
		const {challenge}=await start.json() as {challenge:string};
		const client=new TextEncoder().encode(JSON.stringify({type:"webauthn.get",challenge,origin:ORIGIN}));
		const authData=join(rp,new Uint8Array([0x05,0,0,0,1]));
		const signature=new Uint8Array(await crypto.subtle.sign({name:"ECDSA",hash:"SHA-256"},key.privateKey,join(authData,new Uint8Array(await crypto.subtle.digest("SHA-256",client)))));
		const integer=(raw:Uint8Array)=>{let v=raw;while(v.length>1&&v[0]===0)v=v.slice(1);if(v[0]&128)v=join(new Uint8Array([0]),v);return join(new Uint8Array([2,v.length]),v);};
		const der=join(integer(signature.slice(0,32)),integer(signature.slice(32)));
		const response={id:b64(id),rawId:b64(id),type:"public-key",response:{clientDataJSON:b64(client),authenticatorData:b64(authData),signature:b64(join(new Uint8Array([0x30,der.length]),der))},clientExtensionResults:{}};
		expect((await call("/passkey/verify-authentication",{response},cookieOf(start))).status).toBe(200);
		expect((await call("/passkey/verify-authentication",{response},cookieOf(start))).ok).toBe(false);
	});
	it("creates a verified email account with secure cookies and hashed single-use OTP",async()=>{
		const {res,cookie,otp}=await login();
		const data=await res.json() as {user:{emailVerified:boolean}};
		expect(data.user.emailVerified).toBe(true);
		const cookies=res.headers.getSetCookie().join("; ");
		expect(cookies).toMatch(/HttpOnly/i);expect(cookies).toMatch(/Secure/i);expect(cookies).toMatch(/SameSite=Lax/i);
		expect(JSON.stringify(tables.verification??[])).not.toContain(otp);
		expect((await call("/get-session",undefined,cookie)).status).toBe(200);
		expect((await call("/sign-in/email-otp",{email:"alice@audit.invalid",otp})).status).toBe(400);
	});
	it("requires Turnstile for email sends and stops OTP guessing",async()=>{
		expect((await call("/email-otp/send-verification-otp",{email:"alice@audit.invalid",type:"sign-in"})).status).toBe(400);
		await call("/email-otp/send-verification-otp",{email:"alice@audit.invalid",type:"sign-in"},"",ORIGIN,{"x-captcha-response":"audit-only"});
		const correct=codes.get("alice@audit.invalid")!;
		for(let i=0;i<6;i++)expect((await call("/sign-in/email-otp",{email:"alice@audit.invalid",otp:correct==="000000"?"111111":"000000"})).ok).toBe(false);
		expect((await call("/sign-in/email-otp",{email:"alice@audit.invalid",otp:correct})).ok).toBe(false);
	});
	it("rejects forged origins, passwords and unapproved device clients",async()=>{
		const {cookie}=await login();
		expect((await call("/update-user",{name:"forged"},cookie,"https://evil.audit.invalid")).status).toBe(403);
		expect((await call("/sign-in/email",{email:"alice@audit.invalid",password:"anything"})).ok).toBe(false);
		expect((await call("/device/code",{client_id:"evil"})).status).toBe(400);
	});
	it("does not let a different account approve a claimed device, and consumes it once",async()=>{
		const alice=await login();const bob=await login("bob@audit.invalid");
		const code=await (await call("/device/code",{client_id:"blup-studio"})).json() as {user_code:string;device_code:string};
		expect((await call(`/device?user_code=${code.user_code}`,undefined,alice.cookie)).status).toBe(200);
		expect((await call("/device/approve",{userCode:code.user_code},bob.cookie)).status).toBe(403);
		expect((await call("/device/approve",{userCode:code.user_code},alice.cookie)).status).toBe(200);
		const body={grant_type:"urn:ietf:params:oauth:grant-type:device_code",device_code:code.device_code,client_id:"blup-studio"};
		const token=await call("/device/token",body,"",ORIGIN,{"user-agent":"Blup Studio (Audit; fixture)"});
		expect(token.status).toBe(200);
		expect((await call("/device/token",body)).ok).toBe(false);
	});
	it("does not create an account from a forged passkey assertion",async()=>{
		const before=(tables.user??[]).length;
		const res=await call("/passkey/verify-registration",{response:{id:"forged",rawId:"forged",type:"public-key",response:{clientDataJSON:"bad",attestationObject:"bad"}},createSession:true});
		expect(res.ok).toBe(false);expect((tables.user??[]).length).toBe(before);
	});
	it("links an email to a passkey-only account only after verifying the new address",async()=>{
		const {cookie}=await login();
		tables.user[0].email=`fixture@${NO_EMAIL_DOMAIN}`; tables.user[0].emailVerified=false;
		const newEmail="linked@audit.invalid";
		expect((await call("/email-otp/request-email-change",{newEmail},cookie)).status).toBe(200);
		expect((await call("/email-otp/change-email",{newEmail,otp:"wrong"},cookie)).ok).toBe(false);
		expect(tables.user[0].emailVerified).toBe(false);
		expect((await call("/email-otp/change-email",{newEmail,otp:codes.get(newEmail)},cookie)).status).toBe(200);
		expect(tables.user[0].email).toBe(newEmail);expect(tables.user[0].emailVerified).toBe(true);
	});
	it("rejects unsigned Stripe events and reference IDs of another account",async()=>{
		const {cookie}=await login();
		expect((await call("/stripe/webhook",{type:"customer.subscription.updated",data:{object:{status:"active"}}})).ok).toBe(false);
		expect((await call("/subscription/upgrade",{plan:"ultra",referenceId:"another-account",successUrl:ORIGIN+"/account",cancelUrl:ORIGIN+"/account"},cookie)).ok).toBe(false);
		expect(fetch).toHaveBeenCalledTimes(1); // Turnstile only
	});
	it.each(["/passkey/verify-registration/","/passkey//verify-registration","/passkey/%76erify-registration"])("cannot bypass registration captcha using an alternate path %s",async path=>{
		const res=await worker.fetch(new Request(ORIGIN+"/api/auth"+path,{method:"POST",headers:{origin:ORIGIN,"content-type":"application/json"},body:JSON.stringify({response:{},createSession:true})}),{APP_URL:ORIGIN,STAFF_HOST:"",OTP_LIMIT:{limit:async()=>({success:true})}} as unknown as Cloudflare.Env,{} as ExecutionContext);
		expect([403,404]).toContain(res.status);
	});
	it("cannot bypass the support-only email change restriction with a trailing slash",async()=>{
		const {cookie}=await login();
		const res=await worker.fetch(new Request(ORIGIN+"/api/auth/email-otp/request-email-change/",{method:"POST",headers:{origin:ORIGIN,cookie,"content-type":"application/json"},body:JSON.stringify({newEmail:"attacker@audit.invalid"})}),{APP_URL:ORIGIN,STAFF_HOST:""} as unknown as Cloudflare.Env,{} as ExecutionContext);
		expect([403,404]).toContain(res.status);
		expect(codes.has("attacker@audit.invalid")).toBe(false);
	});
});
