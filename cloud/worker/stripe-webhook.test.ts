import {afterEach,beforeEach,describe,expect,it,vi} from "vitest";
import {TestD1,seedUser} from "./security-test-support";
import {stripeWebhook,cleanWebhookReceipts} from "./billing";
import {createStripe} from "./stripe-client";

let d1:TestD1;
let env:Cloudflare.Env;
let calls:string[];
const objects=new Map<string,unknown>();
const stripe=createStripe("sk_test_fixture");
const current=(subId="sub_audit",status="active",plan="max")=>({id:subId,object:"subscription",customer:"cus_audit",metadata:{userId:"alice",referenceId:"alice",subscriptionId:"pending_alice"},status,cancel_at_period_end:false,cancel_at:null,canceled_at:status==="canceled"?123:null,ended_at:status==="canceled"?123:null,trial_start:null,trial_end:null,schedule:null,items:{has_more:false,data:[{current_period_start:100,current_period_end:200,quantity:1,price:{id:"price_audit",lookup_key:`blup_${plan}_mensal`,metadata:{},product:{id:"prod_audit",metadata:{blup_plan:plan}},recurring:{interval:"month",interval_count:1}}}]}});
const hookEvent=(eventId:string,type="customer.subscription.updated",objectId="sub_audit",snapshot:Record<string,unknown>={})=>({id:eventId,type,livemode:false,data:{object:{id:objectId,...snapshot}}});
async function signed(value:unknown){
 const body=JSON.stringify(value);
 const signature=await stripe.webhooks.generateTestHeaderStringAsync({payload:body,secret:"whsec_fixture"});
 return new Request("https://cloud.audit.invalid/api/auth/stripe/webhook",{method:"POST",headers:{"stripe-signature":signature,"content-type":"application/json"},body});
}
async function deliver(value:unknown){return stripeWebhook(await signed(value),env);}
function rows(){return d1.sqlite.prepare("SELECT stripe_subscription_id AS stripeId,status,plan FROM subscription ORDER BY stripe_subscription_id").all();}
function receipts(){return d1.sqlite.prepare("SELECT count(*) AS n FROM stripe_webhook_event").get();}
beforeEach(()=>{
 d1=new TestD1();seedUser(d1,"alice");seedUser(d1,"bob");
 d1.sqlite.exec("UPDATE user SET stripe_customer_id='cus_audit' WHERE id='alice'; INSERT INTO subscription (id,plan,reference_id,stripe_customer_id,status) VALUES ('pending_alice','max','alice','cus_audit','incomplete')");
 env={DB:d1.asD1(),STRIPE_SECRET_KEY:"sk_test_fixture",STRIPE_WEBHOOK_SECRET:"whsec_fixture",STAFF_OWNER_EMAIL:""} as Cloudflare.Env;
 calls=[];objects.clear();objects.set("/v1/subscriptions/sub_audit",current());
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL)=>{
  const url=new URL(input instanceof Request?input.url:String(input));
  expect(url.origin).toBe("https://api.stripe.com");calls.push(url.pathname+url.search);
  if(!objects.has(url.pathname))throw new Error("Unmocked provider request");
  return Response.json(objects.get(url.pathname));
 }));
 vi.spyOn(console,"error").mockImplementation(()=>{});
});
afterEach(()=>{d1.close();vi.unstubAllGlobals();vi.restoreAllMocks();});

describe("signed Stripe reconciliation against current provider state",()=>{
 it("keeps canceled after deleted then an older active snapshot",async()=>{
  objects.set("/v1/subscriptions/sub_audit",current("sub_audit","canceled"));
  expect((await deliver(hookEvent("evt_deleted","customer.subscription.deleted"))).status).toBe(200);
  expect((await deliver(hookEvent("evt_old","customer.subscription.updated","sub_audit",{status:"active",created:1}))).status).toBe(200);
  expect(rows()).toEqual([{stripeId:"sub_audit",status:"canceled",plan:"max"}]);
  expect(calls).toHaveLength(2);expect(calls[0]).toContain("expand[0]=items.data.price.product");
 });
 it("does not let a deleted old subscription cancel a new subscription",async()=>{
  objects.set("/v1/subscriptions/sub_audit",current("sub_audit","canceled"));
  expect((await deliver(hookEvent("evt_deleted","customer.subscription.deleted"))).status).toBe(200);
  const next=current("sub_new","active","pro");
  objects.set("/v1/subscriptions/sub_new",next);
  expect((await deliver(hookEvent("evt_new","customer.subscription.updated","sub_new"))).status).toBe(200);
  expect((await deliver(hookEvent("evt_old_deleted","customer.subscription.deleted"))).status).toBe(200);
  expect(rows()).toEqual([{stripeId:"sub_audit",status:"canceled",plan:"max"},{stripeId:"sub_new",status:"active",plan:"pro"}]);
  expect(d1.sqlite.prepare("SELECT last_plan FROM resource_quota WHERE user_id='alice'").get()).toEqual({last_plan:"pro"});
 });
 it("serializes two events for one subscription and retries with fresh state",async()=>{
  let started!:()=>void,release!:()=>void;
  const entered=new Promise<void>(r=>{started=r;}),gate=new Promise<void>(r=>{release=r;});
  const old=current();
  vi.stubGlobal("fetch",vi.fn(async()=>{started();await gate;return Response.json(old);}));
  const first=deliver(hookEvent("evt_first"));await entered;
  const second=await deliver(hookEvent("evt_second"));
  expect(second.status).toBe(503);expect(second.headers.get("retry-after")).toBe("1");
  release();expect((await first).status).toBe(200);
  vi.stubGlobal("fetch",vi.fn(async()=>Response.json(current("sub_audit","canceled"))));
  expect((await deliver(hookEvent("evt_second"))).status).toBe(200);
  expect(rows()[0]).toMatchObject({status:"canceled"});expect(receipts()).toEqual({n:2});
 });
 it("deduplicates event.id without another provider request",async()=>{
  const event=hookEvent("evt_repeat");await deliver(event);await deliver(event);
  expect(calls).toHaveLength(1);expect(receipts()).toEqual({n:1});
 });
 it("preserves distinct subscriptions sharing pending metadata during concurrent delivery",async()=>{
  objects.set("/v1/subscriptions/sub_audit",current("sub_audit","canceled"));
  objects.set("/v1/subscriptions/sub_new",current("sub_new","active","pro"));
  const events=[hookEvent("evt_shared_a","customer.subscription.deleted"),hookEvent("evt_shared_b","customer.subscription.updated","sub_new")];
  const results=await Promise.all(events.map(deliver));
  for(let i=0;i<results.length;i++){
   expect([200,503]).toContain(results[i]?.status);
   if(results[i]?.status===503)expect((await deliver(events[i])).status).toBe(200);
  }
  expect(rows()).toEqual([{stripeId:"sub_audit",status:"canceled",plan:"max"},{stripeId:"sub_new",status:"active",plan:"pro"}]);
 });
 it("deduplicates current invoice amounts and follows its current subscription",async()=>{
  objects.set("/v1/invoices/in_audit",{id:"in_audit",customer:"cus_audit",amount_paid:2468,amount_due:2468,currency:"brl",parent:{type:"subscription_details",subscription_details:{subscription:"sub_audit"}}});
  const event=hookEvent("evt_paid","invoice.paid","in_audit",{amount_paid:999999,subscription:"sub_foreign",customer:"cus_foreign"});
  expect((await deliver(event)).status).toBe(200);expect((await deliver(event)).status).toBe(200);
  expect(d1.sqlite.prepare("SELECT user_id,amount,currency FROM billing_event").all()).toEqual([{user_id:"alice",amount:2468,currency:"brl"}]);
  expect(calls).toHaveLength(2);expect(rows()[0]).toMatchObject({stripeId:"sub_audit",status:"active"});
 });
 it("uses current checkout identifiers instead of the snapshot",async()=>{
  objects.set("/v1/checkout/sessions/cs_audit",{id:"cs_audit",mode:"subscription",subscription:"sub_audit"});
  expect((await deliver(hookEvent("evt_checkout","checkout.session.completed","cs_audit",{subscription:"sub_foreign"}))).status).toBe(200);
  expect(rows()[0]).toMatchObject({stripeId:"sub_audit",status:"active"});
 });
 it("accepts a legacy price after lookup key transfer using product metadata",async()=>{
  const value=current(),item=value.items.data[0];
  if(!item)throw new Error("Missing fixture item");
  objects.set("/v1/subscriptions/sub_audit",{...value,items:{...value.items,data:[{...item,price:{...item.price,lookup_key:null}}]}});
  expect((await deliver(hookEvent("evt_legacy"))).status).toBe(200);expect(rows()[0]).toMatchObject({plan:"max"});
 });
 it("rejects unknown active products without granting a plan",async()=>{
  objects.set("/v1/subscriptions/sub_audit",current("sub_audit","active","other"));
  expect((await deliver(hookEvent("evt_unknown"))).status).toBe(503);expect(receipts()).toEqual({n:0});
  expect(rows()[0]).toMatchObject({stripeId:null,status:"incomplete"});
 });
 it("rejects mismatched ownership without linking another account",async()=>{
  const value=current();value.metadata.referenceId="bob";objects.set("/v1/subscriptions/sub_audit",value);
  expect((await deliver(hookEvent("evt_idor"))).status).toBe(503);expect(receipts()).toEqual({n:0});
  expect(rows()[0]).toMatchObject({stripeId:null,status:"incomplete"});
 });
 it("does not resurrect a deleted customer account",async()=>{
  const value=current();value.customer="cus_deleted";objects.set("/v1/subscriptions/sub_audit",value);
  expect((await deliver(hookEvent("evt_missing_owner"))).status).toBe(200);expect(rows()[0]).toMatchObject({stripeId:null});
 });
 it("rolls back state and money if a lease expires during provider work",async()=>{
  vi.stubGlobal("fetch",vi.fn(async()=>{d1.sqlite.exec("UPDATE site_lease SET expires_at=0");return Response.json(current());}));
  expect((await deliver(hookEvent("evt_expired"))).status).toBe(503);expect(receipts()).toEqual({n:0});
  expect(rows()[0]).toMatchObject({stripeId:null,status:"incomplete"});
 });
 it.each([null,{object:"wrong"}])("retries malformed provider state without marking processed: %j",async value=>{
  objects.set("/v1/subscriptions/sub_audit",value);
  expect((await deliver(hookEvent("evt_retry"))).status).toBe(503);expect(receipts()).toEqual({n:0});
  objects.set("/v1/subscriptions/sub_audit",current());expect((await deliver(hookEvent("evt_retry"))).status).toBe(200);
 });
 it("retries a provider outage without marking processed",async()=>{
  objects.clear();expect((await deliver(hookEvent("evt_outage"))).status).toBe(503);expect(receipts()).toEqual({n:0});
 });
 it("rejects an altered signed body before fetching or writing",async()=>{
  const original=await signed(hookEvent("evt_tampered"));
  const request=new Request(original.url,{method:"POST",headers:original.headers,body:JSON.stringify(hookEvent("evt_changed"))});
  expect((await stripeWebhook(request,env)).status).toBe(400);expect(calls).toEqual([]);expect(receipts()).toEqual({n:0});
 });
 it("rejects an expired signature",async()=>{
  const body=JSON.stringify(hookEvent("evt_stale"));
  const signature=await stripe.webhooks.generateTestHeaderStringAsync({payload:body,secret:"whsec_fixture",timestamp:1});
  expect((await stripeWebhook(new Request("https://cloud.audit.invalid/api/auth/stripe/webhook",{method:"POST",headers:{"stripe-signature":signature},body}),env)).status).toBe(400);
 });
 it("does not mix live events with sandbox credentials",async()=>{
  expect((await deliver({...hookEvent("evt_live"),livemode:true})).status).toBe(400);expect(calls).toEqual([]);
 });
 it("cleans old receipts with a bounded indexed query",async()=>{
  for(let i=0;i<505;i++)d1.sqlite.prepare("INSERT INTO stripe_webhook_event VALUES (?,?,NULL,0)").run(`evt_${i}`,"test");
  await cleanWebhookReceipts(d1.asD1());expect(receipts()).toEqual({n:5});
  const query=d1.statements.at(-1)!.sql;
  expect(JSON.stringify(d1.sqlite.prepare("EXPLAIN QUERY PLAN "+query).all(Date.now()))).toContain("stripe_webhook_event_processed_idx");
 });
});
