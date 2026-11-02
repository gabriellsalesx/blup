import {observePulseEntitlements} from "./pulse/entitlements";
import { z } from "zod";
import type Stripe from "stripe";
import { createStripe } from "./stripe-client";
import { PLANS } from "./plans";
import { siteLock,unlockSite } from "./quota";
import { accountPlan,observePlan } from "./lifecycle";

const id=z.string().regex(/^[A-Za-z][A-Za-z0-9_]*$/).max(255);
const reference=z.union([id,z.object({id})]).transform(v=>typeof v==="string"?v:v.id);
const seconds=z.number().int().nonnegative().max(8_640_000_000_000);
const metadata=z.record(z.string(),z.string()).default({});
const eventSchema=z.object({id:id.startsWith("evt_"),type:z.string().max(160),livemode:z.boolean(),data:z.object({object:z.object({id})})});
const subscriptionSchema=z.object({
 id:id.startsWith("sub_"),object:z.literal("subscription"),customer:reference,metadata,
 status:z.enum(["incomplete","incomplete_expired","trialing","active","past_due","canceled","unpaid","paused"]),
 cancel_at_period_end:z.boolean(),cancel_at:seconds.nullable(),canceled_at:seconds.nullable(),ended_at:seconds.nullable(),
 trial_start:seconds.nullable(),trial_end:seconds.nullable(),schedule:reference.nullable(),
 items:z.object({has_more:z.literal(false),data:z.array(z.object({
  current_period_start:seconds,current_period_end:seconds,quantity:z.number().int().nonnegative().safe().nullable(),
  price:z.object({id,lookup_key:z.string().nullable(),metadata,product:z.object({id,metadata}),recurring:z.object({interval:z.enum(["day","week","month","year"]),interval_count:z.number().int().positive()})}),
 })).min(1).max(100)}),
});
const invoiceSchema=z.object({id,customer:reference.nullable(),amount_paid:z.number().int().nonnegative().safe(),amount_due:z.number().int().nonnegative().safe(),currency:z.string().regex(/^[a-z]{3}$/),parent:z.object({type:z.string(),subscription_details:z.object({subscription:reference}).nullable().optional()}).nullable()});
const checkoutSchema=z.object({id,mode:z.enum(["subscription","payment","setup"]),subscription:reference.nullable()});
const chargeSchema=z.object({id,customer:reference.nullable(),amount:z.number().int().nonnegative().safe(),amount_refunded:z.number().int().nonnegative().safe(),currency:z.string().regex(/^[a-z]{3}$/)});
const disputeSchema=z.object({id,charge:reference,amount:z.number().int().nonnegative().safe(),currency:z.string().regex(/^[a-z]{3}$/)});
const warningSchema=z.object({id,charge:reference});
const requestOptions={timeout:5000,maxNetworkRetries:0} as const;
const moneyTypes=new Set(["invoice.paid","invoice.payment_failed","charge.refunded","charge.dispute.created","radar.early_fraud_warning.created"]);
type Event=z.infer<typeof eventSchema>;
type Money={customer:string|null;amount:number;currency:string};
type Receipt={userId:string|null};
type Existing={id:string;plan:string;userId:string;customer:string|null;stripeId:string|null};
type Lease={id:string;token:string};
class BillingError extends Error {}

async function receipt(db:D1Database,eventId:string){return db.prepare("SELECT user_id AS userId FROM stripe_webhook_event WHERE id=?").bind(eventId).first<Receipt>();}
async function refreshPlan(env:Cloudflare.Env,userId:string|null){if(userId){const plan=await accountPlan(env,userId);if(plan)await observePlan(env.DB,userId,plan.name);await observePulseEntitlements(env,userId);}}

// Only identifiers are read from the signed event. All mutable state and
// amounts come from current API objects, never from the delivered snapshot.
async function resolveEvent(stripe:Stripe,event:Event):Promise<{subscription:string|null;money:Money|null}> {
 const objectId=event.data.object.id;
 if(event.type.startsWith("customer.subscription."))return {subscription:id.startsWith("sub_").parse(objectId),money:null};
 if(event.type.startsWith("checkout.session.")){
  const current=checkoutSchema.parse(await stripe.checkout.sessions.retrieve(objectId,{},requestOptions));
  if(current.id!==objectId)throw new BillingError();
  return {subscription:current.mode==="subscription"?current.subscription:null,money:null};
 }
 if(event.type.startsWith("invoice.")){
  const current=invoiceSchema.parse(await stripe.invoices.retrieve(objectId,{},requestOptions));
  if(current.id!==objectId)throw new BillingError();
  return {subscription:current.parent?.type==="subscription_details"?current.parent.subscription_details?.subscription??null:null,money:moneyTypes.has(event.type)?{customer:current.customer,amount:event.type==="invoice.paid"?current.amount_paid:current.amount_due,currency:current.currency}:null};
 }
 if(event.type==="charge.refunded"){
  const current=chargeSchema.parse(await stripe.charges.retrieve(objectId,{},requestOptions));
  if(current.id!==objectId)throw new BillingError();
  return {subscription:null,money:{customer:current.customer,amount:current.amount_refunded,currency:current.currency}};
 }
 if(event.type==="charge.dispute.created"){
  const current=disputeSchema.parse(await stripe.disputes.retrieve(objectId,{},requestOptions));
  if(current.id!==objectId)throw new BillingError();
  const charge=chargeSchema.parse(await stripe.charges.retrieve(current.charge,{},requestOptions));
  return {subscription:null,money:{customer:charge.customer,amount:current.amount,currency:current.currency}};
 }
 if(event.type==="radar.early_fraud_warning.created"){
  const current=warningSchema.parse(await stripe.radar.earlyFraudWarnings.retrieve(objectId,{},requestOptions));
  if(current.id!==objectId)throw new BillingError();
  const charge=chargeSchema.parse(await stripe.charges.retrieve(current.charge,{},requestOptions));
  return {subscription:null,money:{customer:charge.customer,amount:charge.amount,currency:charge.currency}};
 }
 return {subscription:null,money:null};
}

async function subscriptionWrites(db:D1Database,stripe:Stripe,subscriptionId:string,locks:Lease[]){
 // This retrieval must happen AFTER acquiring the subscription lease.
 const current=subscriptionSchema.parse(await stripe.subscriptions.retrieve(subscriptionId,{expand:["items.data.price.product"]},requestOptions));
 if(current.id!==subscriptionId)throw new BillingError();
 const owner=await db.prepare("SELECT u.id,q.deleted_at AS deletedAt FROM user u LEFT JOIN resource_quota q ON q.user_id=u.id WHERE u.stripe_customer_id=? LIMIT 1").bind(current.customer).first<{id:string;deletedAt:number|null}>();
 // A deleted account or another application's customer must not be recreated.
 if(!owner)return {userId:null,writes:[]};
 if(owner.deletedAt!==null&&owner.deletedAt!==undefined)return {userId:owner.id,writes:[db.prepare("UPDATE resource_quota SET billing_cancel_pending=1 WHERE user_id=? AND deleted_at IS NOT NULL").bind(owner.id)]};
 if([current.metadata.userId,current.metadata.referenceId].some(value=>value&&value!==owner.id))throw new BillingError();
 // Two distinct subscriptions may reference the same pending Checkout
 // row. Serialize its adoption as well as this account's plan transition.
 const accountLock=`billing-account:${owner.id}`,token=await siteLock(db,accountLock);
 if(!token)throw new BillingError();
 locks.push({id:accountLock,token});
 let existing=await db.prepare("SELECT id,plan,reference_id AS userId,stripe_customer_id AS customer,stripe_subscription_id AS stripeId FROM subscription WHERE stripe_subscription_id=? LIMIT 1").bind(current.id).first<Existing>();
 if(!existing&&current.metadata.subscriptionId){
  existing=await db.prepare("SELECT id,plan,reference_id AS userId,stripe_customer_id AS customer,stripe_subscription_id AS stripeId FROM subscription WHERE id=?").bind(current.metadata.subscriptionId).first<Existing>();
 }
 if(existing&&(existing.userId!==owner.id||existing.customer&&existing.customer!==current.customer))throw new BillingError();
 // Preserve another real subscription instead of overwriting it through
 // reused Checkout metadata. The provider subscription ID is authoritative.
 if(existing?.stripeId&&existing.stripeId!==current.id)existing=null;
 const candidates=current.items.data.flatMap(item=>{
  // Transferring a lookup key does not change an existing subscriber's Price.
  // Product metadata identifies that legacy Price without granting a new plan.
  const plan=PLANS.find(p=>p.lookupKey&&(item.price.lookup_key===p.lookupKey||item.price.lookup_key===p.yearlyLookupKey))??PLANS.find(p=>p.lookupKey&&p.name===(item.price.metadata.blup_plan??item.price.product.metadata.blup_plan));
  return plan?[{item,plan:plan.name}]:[];
 });
 const matched=candidates.length===1?candidates[0]:null;
 // Unknown active products fail closed. A revoked legacy subscription may
 // keep its old label while its canceled/unpaid status revokes entitlement.
 if(!matched&&(!existing||!["canceled","unpaid","incomplete_expired"].includes(current.status)))throw new BillingError();
 const item=matched?.item??current.items.data[0];
 if(!item||matched&&(item.price.recurring.interval_count!==1||!["month","year"].includes(item.price.recurring.interval)))throw new BillingError();
 const millis=(n:number|null)=>n===null?null:n*1000;
 const rowId=existing?.id??crypto.randomUUID();
 const writes=[db.prepare(`INSERT INTO subscription (id,plan,reference_id,stripe_customer_id,stripe_subscription_id,status,period_start,period_end,trial_start,trial_end,cancel_at_period_end,cancel_at,canceled_at,ended_at,seats,billing_interval,stripe_schedule_id)
 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET plan=excluded.plan,stripe_customer_id=excluded.stripe_customer_id,stripe_subscription_id=excluded.stripe_subscription_id,status=excluded.status,period_start=excluded.period_start,period_end=excluded.period_end,trial_start=excluded.trial_start,trial_end=excluded.trial_end,cancel_at_period_end=excluded.cancel_at_period_end,cancel_at=excluded.cancel_at,canceled_at=excluded.canceled_at,ended_at=excluded.ended_at,seats=excluded.seats,billing_interval=excluded.billing_interval,stripe_schedule_id=excluded.stripe_schedule_id`).bind(rowId,matched?.plan??existing?.plan??"free",owner.id,current.customer,current.id,current.status,millis(item.current_period_start),millis(item.current_period_end),millis(current.trial_start),millis(current.trial_end),current.cancel_at_period_end?1:0,millis(current.cancel_at),millis(current.canceled_at),millis(current.ended_at),item.quantity,item.price.recurring.interval,current.schedule)];
 return {userId:owner.id,writes};
}

export async function stripeWebhook(request:Request,env:Cloudflare.Env):Promise<Response>{
 const signature=z.string().min(1).max(4096).safeParse(request.headers.get("stripe-signature"));
 if(!signature.success)return Response.json({code:"INVALID_SIGNATURE"},{status:400});
 const stripe=createStripe(env.STRIPE_SECRET_KEY);
 let event:Event;
 try{
  const signed=await stripe.webhooks.constructEventAsync(await request.text(),signature.data,env.STRIPE_WEBHOOK_SECRET);
  event=eventSchema.parse(signed);
  const mode=/_(test|live)_/.exec(env.STRIPE_SECRET_KEY)?.[1];
  if(!mode||event.livemode!==(mode==="live"))throw new BillingError();
 }catch{return Response.json({code:"INVALID_SIGNATURE"},{status:400});}
 const locks:Lease[]=[];
 try{
  const eventLock=`billing-event:${event.id}`,token=await siteLock(env.DB,eventLock);
  if(!token)return Response.json({code:"BILLING_BUSY"},{status:503,headers:{"Retry-After":"1"}});
  locks.push({id:eventLock,token});
  const done=await receipt(env.DB,event.id);
  if(done){await refreshPlan(env,done.userId);return Response.json({received:true});}
  const resolved=await resolveEvent(stripe,event);
  let userId:string|null=null,writes:D1PreparedStatement[]=[];
  if(resolved.subscription){
   const lockId=`billing:${resolved.subscription}`,lease=await siteLock(env.DB,lockId);
   if(!lease)return Response.json({code:"BILLING_BUSY"},{status:503,headers:{"Retry-After":"1"}});
   locks.push({id:lockId,token:lease});
   const state=await subscriptionWrites(env.DB,stripe,resolved.subscription,locks);userId=state.userId;writes=state.writes;
  }
  if(resolved.money){
   const money=resolved.money;
   const owner=money.customer?await env.DB.prepare("SELECT id FROM user WHERE stripe_customer_id=? LIMIT 1").bind(money.customer).first<{id:string}>():null;
   if(userId&&owner?.id!==userId)throw new BillingError();
   userId??=owner?.id??null;
   writes.push(env.DB.prepare("INSERT INTO billing_event (stripe_event_id,user_id,type,amount,currency) VALUES (?,?,?,?,?) ON CONFLICT(stripe_event_id) DO NOTHING").bind(event.id,userId,event.type,money.amount,money.currency));
  }
  // NOT NULL enforces the leases inside the transaction. If they expired
  // while Stripe was responding, all state/log writes roll back together.
  const guard=locks.map(()=>"EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?)").join(" AND ");
  writes.push(env.DB.prepare(`INSERT INTO stripe_webhook_event (id,type,user_id,processed_at) VALUES (CASE WHEN ${guard} THEN ? ELSE NULL END,?,?,?)`).bind(...locks.flatMap(lock=>[lock.id,lock.token,Date.now()]),event.id,event.type,userId,Date.now()));
  await env.DB.batch(writes);
  await refreshPlan(env,userId);
  return Response.json({received:true});
 }catch{
  console.error("[blup] billing_reconcile_failed");
  return Response.json({code:"BILLING_RETRY"},{status:503,headers:{"Retry-After":"1"}});
 }finally{
  for(const lock of locks.reverse())await unlockSite(env.DB,lock.id,lock.token);
 }
}

// Stripe supports manual redelivery for 30 days. Ninety days retains a
// safety margin; bounded indexed batches keep the daily job predictable.
export async function cleanWebhookReceipts(db:D1Database,now=Date.now()){
 await db.prepare("DELETE FROM stripe_webhook_event WHERE id IN (SELECT id FROM stripe_webhook_event WHERE processed_at<? ORDER BY processed_at LIMIT 500)").bind(now-90*86_400_000).run();
}
