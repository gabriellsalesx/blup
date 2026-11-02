import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
const state=vi.hoisted(()=>({env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env}));
vi.mock("../auth",()=>({auth:{api:{getSession:async({headers}:{headers:Headers})=>{const id=headers.get("authorization")?.replace("Bearer ","");return id?{user:{id}}:null;}}}}));
import {billingApi} from "../billing-api";
import {buildQuote,flowSubscription,selectionFor} from "../billing-flow";
import {createStripe} from "../stripe-client";
import {planNamed} from "../plans";
import {observePulseEntitlements} from "./entitlements";
import {pulseAccess} from "./limits";
import {claimHostname} from "../hostname-claim";
let db:TestD1,calls:{path:string;method:string;body:URLSearchParams;key:string|null}[];
const price=(name="max_pulse",annual=false)=>({object:"price",id:`price_${name}${annual?"_annual":""}`,lookup_key:`blup_${name}_${annual?"anual":"mensal"}`,metadata:{blup_plan:name},product:{id:`prod_${name}`,metadata:{blup_plan:name},active:true},active:true,unit_amount:name==="max_pulse"?4990:12990,currency:"brl",type:"recurring",billing_scheme:"per_unit",recurring:{interval:annual?"year":"month",interval_count:1,usage_type:"licensed"}});
const subscription=(name="max_pulse")=>({id:"sub_fixture",customer:"cus_fixture",status:"active",cancel_at_period_end:false,cancel_at:null,schedule:null as string|null,pending_update:null as {expires_at:number}|null,collection_method:"charge_automatically",discounts:[],default_tax_rates:[],automatic_tax:{enabled:false},items:{has_more:false,data:[{id:"si_fixture",quantity:1,current_period_start:Math.floor(Date.now()/1000)-100,current_period_end:Math.floor(Date.now()/1000)+86400,price:price(name)}]},latest_invoice:null as {id:string;status:string;hosted_invoice_url:string}|null});
let current:ReturnType<typeof subscription>,pendingPayment:boolean;
const request=(path:string,body?:unknown,user="alice",method="POST")=>billingApi.fetch(new Request(`https://cloud.audit.invalid${path}`,{method,headers:{authorization:`Bearer ${user}`,"content-type":"application/json"},body:body===undefined?undefined:JSON.stringify(body)}),state.env);
beforeEach(()=>{
 vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
 db=new TestD1();seedUser(db,"alice","max_pulse");seedUser(db,"bob","max_pulse");
 db.sqlite.exec("UPDATE user SET stripe_customer_id='cus_fixture' WHERE id='alice'; UPDATE subscription SET stripe_subscription_id='sub_fixture' WHERE reference_id='alice'");
 db.sqlite.prepare("UPDATE subscription SET period_end=? WHERE reference_id='alice'").run(Date.now()+86400000);
 Object.assign(state.env,{DB:db.asD1(),STAFF_OWNER_EMAIL:"",STRIPE_SECRET_KEY:"sk_test_pulse_fixture",APP_URL:"https://cloud.audit.invalid"});current=subscription();pendingPayment=false;calls=[];
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
  const req=input instanceof Request?input:new Request(input,init),url=new URL(req.url);expect(url.origin).toBe("https://api.stripe.com");const body=new URLSearchParams(await req.text());calls.push({path:url.pathname,method:req.method,body,key:req.headers.get("idempotency-key")});
  if(url.pathname==="/v1/subscriptions/sub_fixture")return Response.json(req.method==="POST"&&pendingPayment?{...current,pending_update:{expires_at:Math.floor(Date.now()/1000)+100},latest_invoice:{id:"in_fixture",status:"open",hosted_invoice_url:"https://invoice.stripe.com/i/fixture"}}:current);
  if(url.pathname==="/v1/prices"){const keys=[...url.searchParams].filter(([key])=>key.startsWith("lookup_keys[")).map(([,value])=>value);return Response.json({object:"list",has_more:false,data:keys.map(key=>{const name=key.replace(/^blup_/,"").replace(/_(mensal|anual)$/,"");return price(name,key.endsWith("anual"));})});}
  if(url.pathname==="/v1/invoices/create_preview")return Response.json({amount_due:5000,total:5000,currency:"brl",lines:{has_more:false,data:[{amount:-2000,parent:{subscription_item_details:{proration:true}}},{amount:7000,parent:{subscription_item_details:{proration:true}}}]}});
  if(url.pathname==="/v1/subscriptions")return Response.json({has_more:false,data:[]});
  if(url.pathname==="/v1/checkout/sessions")return req.method==="GET"?Response.json({has_more:false,data:[]}):Response.json({id:"cs_fixture",customer:"cus_fixture",mode:"subscription",status:"open",url:"https://checkout.stripe.com/c/fixture"});
  if(url.pathname==="/v1/subscription_schedules")return Response.json({id:"sub_sched_fixture",subscription:current.id});
  if(url.pathname==="/v1/subscription_schedules/sub_sched_fixture")return Response.json({id:"sub_sched_fixture",subscription:current.id});
  if(url.pathname.endsWith("/release"))return Response.json({id:"sub_sched_fixture"});
  throw new Error("Unmocked Stripe request");
 }));vi.spyOn(console,"error").mockImplementation(()=>{});
});
afterEach(()=>{db.close();vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});
it("previews a real proration and confirms on the same single item/date, without granting before webhook",async()=>{
 const preview=await request("/preview",{plan:"ultra_pulse",annual:false});expect(preview.status).toBe(200);const value=await preview.json() as {ticket:string;quote:{credit:number;debit:number;amountDue:number;prorationDate:number}};expect(value.quote).toMatchObject({credit:2000,debit:7000,amountDue:5000});
 current.pending_update={expires_at:Math.floor(Date.now()/1000)+100};current.latest_invoice={id:"in_fixture",status:"open",hosted_invoice_url:"https://invoice.stripe.com/i/fixture"};
 // A provider-side pending update before confirmation invalidates the quote.
 expect((await request("/confirm",{ticket:value.ticket})).status).toBe(409);current.pending_update=null;
 expect((await request("/confirm",{ticket:value.ticket})).status).toBe(409);current.latest_invoice=null;pendingPayment=true;
 const confirmation=await request("/confirm",{ticket:value.ticket});expect(confirmation.status).toBe(200);expect(await confirmation.json()).toMatchObject({state:"pending",paymentUrl:"https://invoice.stripe.com/i/fixture"});
 const update=calls.find(c=>c.path==="/v1/subscriptions/sub_fixture"&&c.method==="POST");expect(update?.body.get("items[0][id]")).toBe("si_fixture");expect(update?.body.get("items[0][price]")).toBe("price_ultra_pulse");expect(update?.body.get("proration_date")).toBe(String(value.quote.prorationDate));expect(update?.body.get("payment_behavior")).toBe("pending_if_incomplete");expect(db.sqlite.prepare("SELECT plan FROM subscription WHERE reference_id='alice'").get()?.plan).toBe("max_pulse");
 const count=calls.length;expect((await request("/confirm",{ticket:value.ticket})).status).toBe(200);expect(calls).toHaveLength(count);expect(await(await request("/state",undefined,"alice","GET")).json()).toMatchObject({paymentUrl:"https://invoice.stripe.com/i/fixture"});
 db.sqlite.prepare("UPDATE pulse_billing_ticket SET expires_at=? WHERE id=?").run(Date.now()-24*60*60_000,value.ticket);expect(await(await request("/state",undefined,"alice","GET")).json()).toMatchObject({change:null,paymentUrl:null});expect((await request("/confirm",{ticket:value.ticket},"bob")).status).toBe(404);
});
it("schedules Pulse removal at the paid end with no immediate price update",async()=>{
 const preview=await request("/preview",{plan:"max",annual:false}),value=await preview.json() as {ticket:string;quote:{mode:string;effectiveAt:string}};expect(value.quote.mode).toBe("scheduled");expect(Date.parse(value.quote.effectiveAt)).toBe(current.items.data[0].current_period_end*1000);expect((await request("/confirm",{ticket:value.ticket})).status).toBe(200);
 expect(calls.some(c=>c.path==="/v1/subscriptions/sub_fixture"&&c.method==="POST")).toBe(false);const update=calls.find(c=>c.path==="/v1/subscription_schedules/sub_sched_fixture");expect(update?.body.get("phases[0][items][0][price]")).toBe("price_max_pulse");expect(update?.body.get("phases[1][items][0][price]")).toBe("price_max");expect(update?.body.get("proration_behavior")).toBe("none");
});
it("restores a canceled legacy price after its lookup key was transferred",async()=>{
 current.cancel_at_period_end=true;current.items.data[0].price.lookup_key="";const quote=await buildQuote(db.asD1(),createStripe("sk_test_fixture"),flowSubscription.parse(current),"alice",planNamed("max_pulse"),false,Math.floor(Date.now()/1000));expect(quote).toMatchObject({mode:"restore",targetPrice:"price_max_pulse",recurringAmount:4990});expect(calls).toHaveLength(0);
});
it("rejects stale quotes and a simultaneous mutation with a short retry",async()=>{
 const value=await(await request("/preview",{plan:"ultra_pulse",annual:false})).json() as {ticket:string};current.items.data[0].price.id="price_changed";expect((await request("/confirm",{ticket:value.ticket})).status).toBe(409);
 db.sqlite.prepare("INSERT INTO site_lease VALUES(?,?,?)").run("billing:sub_fixture","held",Date.now()+60000);const busy=await request("/preview",{plan:"ultra_pulse",annual:false});expect(busy.status).toBe(503);expect(busy.headers.get("retry-after")).toBe("1");
});
it("never grants more Pulse resources than a downgraded account selected",async()=>{
 db.sqlite.exec("UPDATE subscription SET plan='ultra_pulse' WHERE reference_id='alice'");for(let i=1;i<=4;i++)db.sqlite.prepare("INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at) VALUES(?,'alice','project','go',?,?)").run(String(i),`object-${i}`,i);
 await pulseAccess(state.env,"alice");db.sqlite.prepare("INSERT INTO pulse_selection(user_id,target_plan,effective_at,keep) VALUES('alice','max_pulse',?,?)").run(Date.now()-1,JSON.stringify({projects:["2","3","4"],monitors:[],pages:[]}));db.sqlite.exec("UPDATE subscription SET plan='max_pulse' WHERE reference_id='alice'");await observePulseEntitlements(state.env,"alice");expect(db.sqlite.prepare("SELECT id FROM pulse_project WHERE plan_disabled=0 ORDER BY id").all()).toEqual([{id:"2"},{id:"3"},{id:"4"}]);expect(db.sqlite.prepare("SELECT retire_at FROM pulse_project WHERE id='1'").get()?.retire_at).toBeGreaterThan(Date.now());await observePulseEntitlements(state.env,"alice");expect(db.sqlite.prepare("SELECT plan_disabled FROM pulse_project WHERE id='1'").get()?.plan_disabled).toBe(1);
});
it("starts read-only at the actual paid end even on a late first access",async()=>{
 const now=Date.now();await pulseAccess(state.env,"alice",now);db.sqlite.prepare("UPDATE pulse_account SET paid_until=? WHERE user_id='alice'").run(now-20*86400000);db.sqlite.exec("UPDATE subscription SET status='canceled' WHERE reference_id='alice'");const access=await pulseAccess(state.env,"alice",now);expect(access.readable).toBe(false);expect(access.readOnlyUntil).toBe(now-6*86400000);
});
it("does not let a stale entitlement read overwrite a concurrent paid upgrade",async()=>{
 await pulseAccess(state.env,"alice");const original=db.prepare.bind(db);let switched=false;
 vi.spyOn(db,"prepare").mockImplementation(sql=>{
  if(!switched&&sql.startsWith("INSERT INTO pulse_account")){switched=true;db.sqlite.exec("UPDATE subscription SET plan='ultra_pulse' WHERE reference_id='alice'; UPDATE pulse_account SET last_plan='ultra_pulse',resource_plan='ultra_pulse' WHERE user_id='alice'");}
  return original(sql);
 });
 expect((await pulseAccess(state.env,"alice")).writable).toBe(false);expect(db.sqlite.prepare("SELECT last_plan FROM pulse_account WHERE user_id='alice'").get()?.last_plan).toBe("ultra_pulse");
});
it("does not restore retired resources from an upgrade observation that lost a downgrade race",async()=>{
 db.sqlite.exec("UPDATE subscription SET plan='ultra_pulse' WHERE reference_id='alice'; INSERT INTO pulse_project(id,user_id,name,platform,object_name,created_at,plan_disabled,retire_at) VALUES('1','alice','old','go','old',0,1,9999999999999)");await pulseAccess(state.env,"alice");const original=db.prepare.bind(db);let switched=false;
 vi.spyOn(db,"prepare").mockImplementation(sql=>{
  if(!switched&&sql.startsWith("UPDATE pulse_project SET plan_disabled=0")){switched=true;db.sqlite.exec("UPDATE subscription SET plan='max_pulse' WHERE reference_id='alice'; UPDATE pulse_account SET last_plan='max_pulse',resource_plan='max_pulse' WHERE user_id='alice'");}
  return original(sql);
 });
 await observePulseEntitlements(state.env,"alice");expect(switched).toBe(true);expect(db.sqlite.prepare("SELECT plan_disabled FROM pulse_project WHERE id='1'").get()?.plan_disabled).toBe(1);
});
it("shares an atomic verified hostname namespace across Cloud and Pulse",async()=>{
 const claims=await Promise.all([claimHostname(db.asD1(),"status.audit.invalid","site","s1"),claimHostname(db.asD1(),"status.audit.invalid","pulse","p1")]);expect(claims.filter(Boolean)).toHaveLength(1);expect(await claimHostname(db.asD1(),"status.audit.invalid","site","s1")).toBe(true);
});
it("inherits Cloud limits in both Pulse variants and rejects foreign resource selection",async()=>{
 for(const name of ["max","ultra"] as const)expect(planNamed(`${name}_pulse`).limits).toEqual(planNamed(name).limits);
 const quote=await buildQuote(db.asD1(),createStripe("sk_test_fixture"),flowSubscription.parse(current),"alice",planNamed("ultra_pulse"),false,Math.floor(Date.now()/1000));expect(()=>selectionFor(quote,{projects:["999"],monitors:[],pages:[]})).toThrow();
});

it("serializes initial checkout and reuses one persisted idempotency key instead of opening another subscription",async()=>{
 db.sqlite.exec("DELETE FROM subscription WHERE reference_id='alice'");
 const results=await Promise.all([request("/checkout",{plan:"max_pulse",annual:false}),request("/checkout",{plan:"max_pulse",annual:false})]);expect(results.map(r=>r.status).sort()).toEqual([200,503]);expect((await request("/checkout",{plan:"max_pulse",annual:false})).status).toBe(200);
 const checkouts=calls.filter(c=>c.path==="/v1/checkout/sessions"&&c.method==="POST");expect(checkouts).toHaveLength(2);expect(checkouts[0].key).toBe(checkouts[1].key);expect(checkouts[0].body.get("line_items[0][price]")).toBe("price_max_pulse");expect(checkouts[0].body.get("subscription_data[metadata][referenceId]")).toBe("alice");expect((await request("/checkout",{plan:"ultra_pulse",annual:false})).status).toBe(409);
});
it("does not charge after account deletion or allow caller-provided customer/price",async()=>{
 expect((await request("/checkout",{plan:"max_pulse",annual:false,customer:"cus_other"})).status).toBe(400);db.sqlite.exec("INSERT INTO resource_quota(user_id,deleted_at) VALUES('alice',1)");expect((await request("/preview",{plan:"ultra_pulse",annual:false})).status).toBe(401);expect(calls).toHaveLength(0);
});
