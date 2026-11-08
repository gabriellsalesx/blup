// The billing rules approved by the owner: 14 days of grace on a failed
// payment, immediate cancellation on full refund/dispute/fraud warning,
// devices over the new limit disconnected, and a restored account that
// keeps its subscription canceled. Real SQLite, Stripe answered locally.
import {afterEach,beforeEach,describe,expect,it,vi} from "vitest";
import {drizzle} from "drizzle-orm/d1";
import {TestD1,seedUser} from "./security-test-support";
import * as schema from "./db/schema";

const state=vi.hoisted(()=>({db:undefined as unknown as ReturnType<typeof drizzle>,env:{} as Cloudflare.Env}));
vi.mock("cloudflare:workers",()=>({env:state.env,DurableObject:class{},waitUntil:()=>{}}));
vi.mock("./auth",()=>({db:new Proxy({},{get:(_,k)=>{const v=Reflect.get(state.db,k);return typeof v==="function"?v.bind(state.db):v;}})}));

import {stripeWebhook} from "./billing";
import {createStripe} from "./stripe-client";
import {DAY,PAST_DUE_GRACE,accountPlan,expirePastDue,settlePlan,softDelete,restoreAccount,cancelDeletedBilling} from "./lifecycle";
import {activeSubscription} from "./account";
import {disconnectExcessDevices} from "./devices";

let d1:TestD1;
let env:Cloudflare.Env;
let calls:string[];
let revoked:string[];
const answers=new Map<string,unknown>();
const stripe=createStripe("sk_test_fixture");
const subscription=(status:string,plan="max",id="sub_audit")=>({id,object:"subscription",customer:"cus_audit",metadata:{userId:"alice"},status,cancel_at_period_end:false,cancel_at:null,canceled_at:status==="canceled"?123:null,ended_at:status==="canceled"?123:null,trial_start:null,trial_end:null,schedule:null,items:{has_more:false,data:[{current_period_start:1,current_period_end:2,quantity:1,price:{id:`price_${plan}`,lookup_key:`blup_${plan}_monthly`,metadata:{},product:{id:"prod_x",metadata:{blup_plan:plan}},recurring:{interval:"month",interval_count:1}}}]}});
const event=(id:string,type:string,objectId:string)=>({id,type,livemode:false,data:{object:{id:objectId}}});
async function deliver(value:unknown){
 const body=JSON.stringify(value);
 const signature=await stripe.webhooks.generateTestHeaderStringAsync({payload:body,secret:"whsec_fixture"});
 return stripeWebhook(new Request("https://cloud.audit.invalid/api/auth/stripe/webhook",{method:"POST",headers:{"stripe-signature":signature},body}),env);
}
const status=()=>d1.sqlite.prepare("SELECT status FROM subscription WHERE reference_id='alice' AND stripe_subscription_id='sub_audit'").get()?.status;
const grace=()=>{const since=d1.sqlite.prepare("SELECT past_due_since AS since FROM subscription_grace").get()?.since;return typeof since==="number"?since:undefined;};
const devices=()=>d1.sqlite.prepare("SELECT id FROM session WHERE user_id='alice' AND studio_device=1 ORDER BY id").all().map(r=>r.id);
function seedSessions(count:number,now:number){
 for(let i=1;i<=count;i++)d1.sqlite.prepare("INSERT INTO session(id,expires_at,token,created_at,updated_at,user_id,studio_device) VALUES(?,?,?,?,?,'alice',1)").run(`dev-${i}`,now+30*DAY,`tok-${i}`,now-10*DAY,now-(10-i)*60_000);
 d1.sqlite.prepare("INSERT INTO session(id,expires_at,token,created_at,updated_at,user_id,studio_device) VALUES('browser',?,'tok-browser',?,?,'alice',0)").run(now+30*DAY,now-10*DAY,now-20*DAY);
}

beforeEach(()=>{
 d1=new TestD1();seedUser(d1,"alice");seedUser(d1,"bob");
 d1.sqlite.exec("UPDATE user SET stripe_customer_id='cus_audit' WHERE id='alice'; UPDATE user SET stripe_customer_id='cus_bob' WHERE id='bob'");
 state.db=drizzle(d1.asD1(),{schema});revoked=[];
 env=state.env;
 Object.assign(env,{DB:d1.asD1(),STRIPE_SECRET_KEY:"sk_test_fixture",STRIPE_WEBHOOK_SECRET:"whsec_fixture",STAFF_OWNER_EMAIL:"",SYNC_LIVE:{getByName:(user:string)=>({revoke:async()=>{revoked.push(user);}})}});
 calls=[];answers.clear();answers.set("GET /v1/subscriptions/sub_audit",subscription("active"));
 vi.stubGlobal("fetch",vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
  const request=input instanceof Request?input:new Request(String(input),init);
  const url=new URL(request.url);expect(url.origin).toBe("https://api.stripe.com");
  const key=`${request.method} ${url.pathname}`;calls.push(key+url.search);
  if(key.startsWith("DELETE ")||key.startsWith("POST "))expect(request.headers.get("idempotency-key")??"").not.toBe("");
  if(!answers.has(key))throw new Error(`Unmocked provider request ${key}`);
  return Response.json(answers.get(key));
 }));
 vi.spyOn(console,"error").mockImplementation(()=>{});
});
afterEach(()=>{d1.close();vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});

describe("past_due grace of 14 days",()=>{
 it("keeps the paid plan for 14 days, then Grátis everywhere, and returns it when paid",async()=>{
  const start=Date.now();
  answers.set("GET /v1/subscriptions/sub_audit",subscription("past_due"));
  expect((await deliver(event("evt_due","customer.subscription.updated","sub_audit"))).status).toBe(200);
  const since=grace();if(since===undefined)throw new Error("Missing grace");expect(since).toBeGreaterThanOrEqual(start);expect(status()).toBe("past_due");
  // A later failure keeps the first moment.
  expect((await deliver(event("evt_due_again","customer.subscription.updated","sub_audit"))).status).toBe(200);
  expect(grace()).toBe(since);
  expect((await accountPlan(env,"alice",since+13*DAY))?.name).toBe("max");
  expect((await activeSubscription("alice",since+13*DAY))?.plan).toBe("max");
  // Past the grace, before any cron: the central readers already say Grátis.
  expect((await accountPlan(env,"alice",since+PAST_DUE_GRACE+1))?.name).toBe("free");
  expect(await activeSubscription("alice",since+PAST_DUE_GRACE+1)).toBeUndefined();
  // The cron makes every other reader of the table see it too.
  await expirePastDue(env,since+PAST_DUE_GRACE+1);
  expect(status()).toBe("unpaid");
  expect(d1.sqlite.prepare("SELECT last_plan FROM resource_quota WHERE user_id='alice'").get()?.last_plan).toBe("free");
  // Stripe still retrying (past_due) does not bring the plan back.
  vi.useFakeTimers({toFake:["Date"]});vi.setSystemTime(since+PAST_DUE_GRACE+DAY);
  answers.set("GET /v1/invoices/in_retry",{id:"in_retry",customer:"cus_audit",amount_paid:0,amount_due:4990,currency:"brl",parent:{type:"subscription_details",subscription_details:{subscription:"sub_audit"}}});
  expect((await deliver(event("evt_retry_failed","invoice.payment_failed","in_retry"))).status).toBe(200);
  expect(status()).toBe("unpaid");expect((await accountPlan(env,"alice"))?.name).toBe("free");
  // Recovered payment: the webhook restores the plan and clears the grace.
  answers.set("GET /v1/subscriptions/sub_audit",subscription("active"));
  expect((await deliver(event("evt_paid","customer.subscription.updated","sub_audit"))).status).toBe(200);
  expect(status()).toBe("active");expect(grace()).toBeUndefined();
  expect((await accountPlan(env,"alice"))?.name).toBe("max");
 });
 it("does not let the cron overwrite a payment the webhook already recovered",async()=>{
  answers.set("GET /v1/subscriptions/sub_audit",subscription("past_due"));
  await deliver(event("evt_due","customer.subscription.updated","sub_audit"));
  answers.set("GET /v1/subscriptions/sub_audit",subscription("active"));
  await deliver(event("evt_paid","customer.subscription.updated","sub_audit"));
  await expirePastDue(env,Date.now()+PAST_DUE_GRACE+DAY);
  expect(status()).toBe("active");
 });
 it("starts the grace of subscriptions already past_due at the migration",()=>{
  const db=new TestD1("0044_nonce_ponte_deals.sql");
  db.sqlite.exec("INSERT INTO user(id,name,email) VALUES('u','u','u@audit.invalid'); INSERT INTO subscription(id,plan,reference_id,status) VALUES('s1','max','u','past_due'),('s2','max','u','active')");
  db.applyMigration("0045_carencia_pagamento.sql");
  expect(db.sqlite.prepare("SELECT subscription_id AS id FROM subscription_grace").all()).toEqual([{id:"s1"}]);
  db.close();
 });
});

describe("full refund, dispute and fraud warning cancel at once",()=>{
 const charge=(over:Record<string,unknown>={})=>({id:"ch_audit",customer:"cus_audit",payment_intent:"pi_audit",amount:4990,amount_refunded:4990,refunded:true,currency:"brl",...over});
 beforeEach(()=>{
  answers.set("GET /v1/invoice_payments",{has_more:false,data:[{invoice:"in_audit"}]});
  answers.set("GET /v1/invoices/in_audit",{id:"in_audit",customer:"cus_audit",amount_paid:4990,amount_due:4990,currency:"brl",parent:{type:"subscription_details",subscription_details:{subscription:"sub_audit"}}});
  answers.set("DELETE /v1/subscriptions/sub_audit",subscription("canceled"));
 });
 it.each([
  ["charge.refunded","ch_audit",()=>answers.set("GET /v1/charges/ch_audit",charge())],
  ["charge.dispute.created","dp_audit",()=>{answers.set("GET /v1/charges/ch_audit",charge({refunded:false,amount_refunded:0}));answers.set("GET /v1/disputes/dp_audit",{id:"dp_audit",charge:"ch_audit",amount:4990,currency:"brl"});}],
  ["radar.early_fraud_warning.created","issfr_audit",()=>{answers.set("GET /v1/charges/ch_audit",charge({refunded:false,amount_refunded:0}));answers.set("GET /v1/radar/early_fraud_warnings/issfr_audit",{id:"issfr_audit",charge:"ch_audit"});}],
 ])("%s cancels the subscription found through the invoice",async(type,objectId,setup)=>{
  setup();
  expect((await deliver(event("evt_risk",type,objectId))).status).toBe(200);
  const cancel=calls.filter(c=>c.startsWith("DELETE "));
  expect(cancel).toHaveLength(1);expect(cancel[0]).toContain("/v1/subscriptions/sub_audit");
  expect(cancel[0]).toContain("prorate=false");expect(cancel[0]).toContain("invoice_now=false");
  expect(calls.some(c=>c.startsWith("GET /v1/invoice_payments")&&c.includes("pi_audit"))).toBe(true);
  expect(d1.sqlite.prepare("SELECT type,user_id AS userId FROM billing_event").all()).toEqual([{type,userId:"alice"}]);
  // The plan drops with the deleted event that follows.
  answers.set("GET /v1/subscriptions/sub_audit",subscription("canceled"));
  expect((await deliver(event("evt_deleted","customer.subscription.deleted","sub_audit"))).status).toBe(200);
  expect((await accountPlan(env,"alice"))?.name).toBe("free");
 });
 it("uses the charge customer when no invoice is linked, and skips ended ones",async()=>{
  answers.set("GET /v1/charges/ch_audit",charge({payment_intent:null}));
  answers.set("GET /v1/subscriptions",{has_more:false,data:[{id:"sub_old",customer:"cus_audit",status:"canceled"},{id:"sub_audit",customer:"cus_audit",status:"active"}]});
  expect((await deliver(event("evt_refund","charge.refunded","ch_audit"))).status).toBe(200);
  expect(calls.filter(c=>c.startsWith("DELETE "))).toEqual([expect.stringContaining("/v1/subscriptions/sub_audit")]);
 });
 it("only records a partial refund",async()=>{
  answers.set("GET /v1/charges/ch_audit",charge({refunded:false,amount_refunded:1000}));
  expect((await deliver(event("evt_partial","charge.refunded","ch_audit"))).status).toBe(200);
  expect(calls.filter(c=>c.startsWith("DELETE ")||c.includes("invoice_payments"))).toEqual([]);
  expect(d1.sqlite.prepare("SELECT amount FROM billing_event").get()?.amount).toBe(1000);
 });
 it("does not cancel twice on redelivery or when already canceled",async()=>{
  answers.set("GET /v1/charges/ch_audit",charge());
  await deliver(event("evt_refund","charge.refunded","ch_audit"));await deliver(event("evt_refund","charge.refunded","ch_audit"));
  answers.set("GET /v1/subscriptions/sub_audit",subscription("canceled"));
  await deliver(event("evt_dispute_later","charge.refunded","ch_audit"));
  expect(calls.filter(c=>c.startsWith("DELETE "))).toHaveLength(1);
 });
 it("never cancels a subscription of another customer",async()=>{
  answers.set("GET /v1/charges/ch_audit",charge());
  answers.set("GET /v1/invoices/in_audit",{id:"in_audit",customer:"cus_bob",amount_paid:4990,amount_due:4990,currency:"brl",parent:{type:"subscription_details",subscription_details:{subscription:"sub_audit"}}});
  expect((await deliver(event("evt_foreign","charge.refunded","ch_audit"))).status).toBe(503);
  expect(calls.filter(c=>c.startsWith("DELETE "))).toEqual([]);
  expect(d1.sqlite.prepare("SELECT count(*) AS n FROM billing_event").get()?.n).toBe(0);
 });
 it("retries without a receipt when the cancellation fails",async()=>{
  answers.set("GET /v1/charges/ch_audit",charge());answers.delete("DELETE /v1/subscriptions/sub_audit");
  expect((await deliver(event("evt_refund","charge.refunded","ch_audit"))).status).toBe(503);
  expect(d1.sqlite.prepare("SELECT count(*) AS n FROM stripe_webhook_event").get()?.n).toBe(0);
 });
});

describe("a dropped plan disconnects the devices over its limit",()=>{
 it("Ultra with 5 devices falling to Grátis keeps only the most recent one",async()=>{
  const now=Date.now();seedSessions(5,now);
  answers.set("GET /v1/subscriptions/sub_audit",subscription("active","ultra"));
  expect((await deliver(event("evt_ultra","customer.subscription.updated","sub_audit"))).status).toBe(200);
  expect(devices()).toHaveLength(5);expect(revoked).toEqual([]);
  answers.set("GET /v1/subscriptions/sub_audit",subscription("canceled","ultra"));
  expect((await deliver(event("evt_end","customer.subscription.deleted","sub_audit"))).status).toBe(200);
  expect(devices()).toEqual(["dev-5"]);
  expect(d1.sqlite.prepare("SELECT id FROM session WHERE id='browser'").get()).toBeTruthy();
  expect(revoked).toEqual(["alice"]);
 });
 it("disconnects when the grace ends in the cron",async()=>{
  const now=Date.now();seedSessions(3,now);
  answers.set("GET /v1/subscriptions/sub_audit",subscription("past_due","max"));
  await deliver(event("evt_due","customer.subscription.updated","sub_audit"));
  expect(devices()).toHaveLength(3);
  await expirePastDue(env,now+PAST_DUE_GRACE+DAY);
  expect(devices()).toEqual(["dev-3"]);expect(revoked).toEqual(["alice"]);
 });
 it("cancels approvals waiting for their token past the new limit",async()=>{
  const now=Date.now();seedSessions(1,now);
  d1.sqlite.exec(`INSERT INTO resource_quota(user_id,devices) VALUES('alice',1) ON CONFLICT(user_id) DO UPDATE SET devices=1; INSERT INTO resource_reservation(id,user_id,kind,amount,expires_at) VALUES('hold','alice','devices',1,${now+60_000}); INSERT INTO device_approval(code_hash,user_id,reservation_id,expires_at) VALUES('hash','alice','hold',${now+60_000})`);
  expect(await disconnectExcessDevices(env.DB,"alice",1,now)).toBe(0);
  expect(d1.sqlite.prepare("SELECT count(*) AS n FROM device_approval").get()?.n).toBe(0);
  expect(d1.sqlite.prepare("SELECT devices FROM resource_quota WHERE user_id='alice'").get()?.devices).toBe(0);
 });
 it("leaves unlimited plans and accounts within the limit alone",async()=>{
  const now=Date.now();seedSessions(2,now);
  d1.sqlite.exec("INSERT INTO subscription(id,plan,reference_id,status) VALUES('local','ultra','alice','active')");
  await settlePlan(env,"alice",now);expect(devices()).toHaveLength(2);
  d1.sqlite.exec("UPDATE subscription SET plan='pro' WHERE id='local'");
  await settlePlan(env,"alice",now);expect(devices()).toHaveLength(2);expect(revoked).toEqual([]);
 });
});

describe("restoring a deleted account",()=>{
 it("keeps the subscription canceled",async()=>{
  answers.set("GET /v1/subscriptions/sub_audit",subscription("active"));
  await deliver(event("evt_active","customer.subscription.updated","sub_audit"));
  const now=Date.now();
  await softDelete(env,"alice",now);
  answers.set("GET /v1/checkout/sessions",{object:"list",has_more:false,data:[]});
  answers.set("GET /v1/subscriptions",{object:"list",has_more:false,data:[subscription("active")]});
  answers.set("DELETE /v1/subscriptions/sub_audit",subscription("canceled"));
  await cancelDeletedBilling(env);
  expect(calls.filter(c=>c.startsWith("DELETE "))).toHaveLength(1);
  expect(await restoreAccount(env.DB,"alice",now+DAY)).toBe(true);
  expect(status()).toBe("canceled");
  expect((await accountPlan(env,"alice"))?.name).toBe("free");
  // Nothing asks Stripe to bring it back.
  expect(calls.filter(c=>c.startsWith("POST "))).toEqual([]);
  answers.set("GET /v1/subscriptions/sub_audit",subscription("canceled"));
  await deliver(event("evt_deleted","customer.subscription.deleted","sub_audit"));
  expect(status()).toBe("canceled");
 });
});
