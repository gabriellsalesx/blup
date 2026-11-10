import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {TestD1,seedUser} from "./security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env}));
vi.mock("./auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {billingApi} from "./billing-api";
import {buildQuote,BillingFlowError,type FlowSubscription} from "./billing-flow";
import {planNamed} from "./plans";
import {supersetOf} from "../shared/pulse-plans";

// Stripe: expires_at "can be anywhere from 30 minutes to 24 hours after
// Checkout Session creation" (API reference, Create a Checkout Session).
let db:TestD1,posts:{at:number;expiresAt:number;key:string|null}[],failCreate:boolean;
const price=(key:string)=>({object:"price",id:`price_${key}`,lookup_key:key,metadata:{},product:{id:"prod_fixture",metadata:{},active:true},active:true,unit_amount:4990,currency:"brl",type:"recurring",billing_scheme:"per_unit",recurring:{interval:key.endsWith("anual")?"year":"month",interval_count:1,usage_type:"licensed"}});
const request=(path:string,method="POST",body?:unknown)=>billingApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method,headers:{authorization:"Bearer alice","content-type":"application/json"},body:body===undefined?undefined:JSON.stringify(body)}),state.env);
beforeEach(()=>{
 vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
 db=new TestD1();seedUser(db,"alice");db.sqlite.exec("UPDATE user SET stripe_customer_id='cus_fixture' WHERE id='alice'");
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",STRIPE_SECRET_KEY:"sk_test_checkout_fixture",APP_URL:"https://cloud.audit.invalid"});posts=[];failCreate=false;
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
  const req=input instanceof Request?input:new Request(input,init),url=new URL(req.url);expect(url.origin).toBe("https://api.stripe.com");
  if(url.pathname==="/v1/prices"){const keys=[...url.searchParams].filter(([k])=>k.startsWith("lookup_keys")).map(([,v])=>v);expect(keys.length).toBeLessThanOrEqual(10);return Response.json({object:"list",has_more:false,data:keys.map(price)});}
  if(url.pathname==="/v1/subscriptions")return Response.json({has_more:false,data:[]});
  if(url.pathname==="/v1/checkout/sessions"&&req.method==="GET")return Response.json({has_more:false,data:[]});
  if(url.pathname==="/v1/checkout/sessions"){
   const body=new URLSearchParams(await req.text()),now=Math.floor(Date.now()/1000),expiresAt=Number(body.get("expires_at"));posts.push({at:now,expiresAt,key:req.headers.get("idempotency-key")});
   if(expiresAt-now<1800)return Response.json({error:{type:"invalid_request_error",message:"expires_at must be at least 30 minutes after creation"}},{status:400});
   if(failCreate)return Response.json({error:{type:"api_error",message:"fixture outage"}},{status:500});
   return Response.json({id:"cs_fixture",customer:"cus_fixture",mode:"subscription",status:"open",url:"https://checkout.stripe.com/c/fixture"});
  }
  if(url.pathname==="/v1/checkout/sessions/cs_fixture/expire")return Response.json({id:"cs_fixture",status:"expired"});
  throw new Error("Unmocked Stripe request");
 }));vi.spyOn(console,"error").mockImplementation(()=>{});
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});

it("replays a failed Checkout create minutes later with an expiry Stripe still accepts",async()=>{
 failCreate=true;expect((await request("/checkout","POST",{plan:"max",annual:false})).status).toBe(502);
 failCreate=false;vi.setSystemTime(Date.now()+10*60_000);
 expect((await request("/checkout","POST",{plan:"max",annual:false})).status).toBe(200);
 expect(posts).toHaveLength(2);{const [a,b]=posts;if(!a||!b)throw new Error("missing post");expect(a.key).toBe(b.key);expect(b.expiresAt-b.at).toBeGreaterThanOrEqual(1800);}
});
it("drops a never created Checkout row once Stripe could no longer accept its expiry",async()=>{
 failCreate=true;expect((await request("/checkout","POST",{plan:"max",annual:false})).status).toBe(502);
 failCreate=false;vi.setSystemTime(Date.now()+40*60_000);
 expect((await request("/checkout","POST",{plan:"max",annual:false})).status).toBe(200);
 expect(posts).toHaveLength(2);{const [a,b]=posts;if(!a||!b)throw new Error("missing post");expect(b.key).not.toBe(a.key);expect(b.expiresAt-b.at).toBeGreaterThanOrEqual(1800);}
});
it("cancels a stale never created Checkout without calling Stripe again",async()=>{
 failCreate=true;expect((await request("/checkout","POST",{plan:"max",annual:false})).status).toBe(502);
 vi.setSystemTime(Date.now()+40*60_000);
 expect((await request("/checkout","DELETE")).status).toBe(200);expect(posts).toHaveLength(1);
 expect(db.sqlite.prepare("SELECT count(*) AS n FROM billing_checkout").get()).toEqual({n:0});
});

// An AppSumo lifetime plan is never lost to a Stripe one: only a plan with
// all of it and more is sold next to it.
const lifetime=(plan:string)=>db.sqlite.prepare("INSERT INTO subscription(id,plan,reference_id,status,period_start) VALUES ('appsumo-alice',?,'alice','active',1)").run(plan);
const codeOf=async(res:Response)=>({status:res.status,code:(await res.json() as {code?:string}).code});
it("refuses a Checkout the lifetime Ultra already covers, and sells Ultra with Pulse",async()=>{
 lifetime("ultra");
 for(const plan of ["pro","max","max_pulse","ultra"])expect(await codeOf(await request("/checkout","POST",{plan,annual:false}))).toEqual({status:409,code:"LIFETIME_COVERS"});
 expect(posts).toHaveLength(0);
 expect((await request("/checkout","POST",{plan:"ultra_pulse",annual:false})).status).toBe(200);
});
it("never sells a lifetime Max with Pulse a plan without Pulse",async()=>{
 lifetime("max_pulse");
 expect(await codeOf(await request("/checkout","POST",{plan:"ultra",annual:false}))).toEqual({status:409,code:"LIFETIME_COVERS"});
 expect((await request("/checkout","POST",{plan:"ultra_pulse",annual:false})).status).toBe(200);
});
it("refuses the same plan changes in the preview and its confirmation (buildQuote), but not a cancellation",async()=>{
 lifetime("ultra");
 const sub={} as FlowSubscription,quote=(plan:"max"|"ultra"|"free")=>buildQuote(db.asD1(),{} as never,sub,"alice",planNamed(plan),false,0);
 for(const plan of ["max","ultra"] as const)await expect(quote(plan)).rejects.toMatchObject({code:"LIFETIME_COVERS"});
 await expect(quote("free")).rejects.not.toBeInstanceOf(BillingFlowError);
});
it("compares plans by base and add-ons",()=>{
 expect(supersetOf("ultra_pulse","ultra")).toBe(true);expect(supersetOf("ultra_deals","ultra")).toBe(true);expect(supersetOf("ultra","ultra")).toBe(false);
 expect(supersetOf("ultra","max_pulse")).toBe(false);expect(supersetOf("ultra_pulse","max_pulse")).toBe(true);expect(supersetOf("max","pro")).toBe(true);
 expect(supersetOf("ultra_pulse","ultra_pulse_deals")).toBe(false);expect(supersetOf("ultra_pulse_deals","ultra_pulse")).toBe(true);
});
