import {Hono} from "hono";
import {z} from "zod";
import {HTTPException} from "hono/http-exception";
import {validate,opaqueId} from "./validation";
import {pulseSession,type PulseApp} from "./pulse/access";
import {planNamed} from "./plans";
import {planNameSchema,paidPlanNameSchema} from "../shared/pulse-plans";
import {BillingFlowError,withSubscription,buildQuote,subscriptionFingerprint,quoteSchema,keepSchema,selectionFor,stripeOptions,stripeURL,flowSubscription,checkout,portal,checkoutParamsSchema,checkoutReplayable} from "./billing-flow";
import {createStripe} from "./stripe-client";
import {siteLock,unlockSite} from "./quota";

const resultSchema=z.object({state:z.enum(["pending","scheduled","restored","awaiting_webhook"]),effectiveAt:z.iso.datetime(),paymentUrl:z.string().nullable()});
const ticketBody=z.strictObject({ticket:opaqueId,keep:keepSchema.optional()});
const scheduleSchema=z.object({id:z.string().regex(/^sub_sched_[A-Za-z0-9_]+$/),subscription:z.string()});
type Ticket={id:string;userId:string;subscriptionId:string;targetPlan:string;annual:number;prorationDate:number;fingerprint:string;quote:string;targetPrice:string|null;expiresAt:number;state:string;result:string|null;scheduleId:string|null};
async function ticketOf(db:D1Database,owner:string,id:string){return db.prepare("SELECT id,user_id AS userId,subscription_id AS subscriptionId,target_plan AS targetPlan,annual,proration_date AS prorationDate,fingerprint,quote,target_price AS targetPrice,expires_at AS expiresAt,state,result,schedule_id AS scheduleId FROM pulse_billing_ticket WHERE id=? AND user_id=?").bind(id,owner).first<Ticket>();}
export const billingApi=new Hono<PulseApp>().use("*",pulseSession)
 .onError((error,c)=>{
  if(error instanceof HTTPException)return error.getResponse().headers.get("content-type")?.includes("application/json")?error.getResponse():c.json({code:error.status===413?"TOO_LARGE":"INVALID"},error.status);
  if(error instanceof BillingFlowError){if(error.code==="BILLING_BUSY")c.header("Retry-After","1");return c.json({code:error.code},error.status);}
  console.error("[blup] billing_action_failed");return c.json({code:"BILLING_UNAVAILABLE"},502);
 })
 .post("/checkout",validate("json",z.strictObject({plan:paidPlanNameSchema,annual:z.boolean()})),async c=>{const input=c.req.valid("json");return c.json(await checkout(c.env,c.get("session").user.id,planNamed(input.plan),input.annual));})
 .post("/portal",validate("json",z.strictObject({})),async c=>c.json(await portal(c.env,c.get("session").user.id)))
 .post("/preview",validate("json",z.strictObject({plan:planNameSchema,annual:z.boolean()})),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json");
  return withSubscription(c.env,owner,async({stripe,sub})=>{
   if(sub.schedule)throw new BillingFlowError("BILLING_PENDING");
   const prorationDate=Math.floor(Date.now()/1000),quote=await buildQuote(c.env.DB,stripe,sub,owner,planNamed(input.plan),input.annual,prorationDate),id=crypto.randomUUID(),expiresAt=Date.now()+10*60_000;
   await c.env.DB.prepare("INSERT INTO pulse_billing_ticket(id,user_id,subscription_id,target_plan,annual,proration_date,fingerprint,quote,target_price,expires_at) VALUES(?,?,?,?,?,?,?,?,?,?)").bind(id,owner,sub.id,input.plan,input.annual?1:0,prorationDate,await subscriptionFingerprint(sub),JSON.stringify(quote),quote.targetPrice,expiresAt).run();
   return c.json({ticket:id,expiresAt:new Date(expiresAt).toISOString(),quote});
  });
 })
 .post("/confirm",validate("json",ticketBody),async c=>{
  const owner=c.get("session").user.id,input=c.req.valid("json");let ticket=await ticketOf(c.env.DB,owner,input.ticket);if(!ticket)throw new BillingFlowError("NOT_FOUND",404);
  if(ticket.state==="done"&&ticket.result)return c.json(resultSchema.parse(JSON.parse(ticket.result)));
  if(ticket.expiresAt<=Date.now()&&ticket.state==="preview"||ticket.expiresAt+23*60*60_000<=Date.now())throw new BillingFlowError("BILLING_QUOTE_EXPIRED");
  return withSubscription(c.env,owner,async({stripe,sub,locks})=>{
   ticket=await ticketOf(c.env.DB,owner,input.ticket);if(!ticket)throw new BillingFlowError("NOT_FOUND",404);
   if(ticket.state==="done"&&ticket.result)return c.json(resultSchema.parse(JSON.parse(ticket.result)));
   if(ticket.expiresAt<=Date.now()&&ticket.state==="preview")throw new BillingFlowError("BILLING_QUOTE_EXPIRED");
   if(sub.id!==ticket.subscriptionId)throw new BillingFlowError("BILLING_QUOTE_EXPIRED");
   const quote=quoteSchema.parse(JSON.parse(ticket.quote)),keep=ticket.state==="processing"?z.object({keep:keepSchema}).parse(JSON.parse(ticket.result??"null")).keep:selectionFor(quote,input.keep);
   if(ticket.state==="preview"){
    if(await subscriptionFingerprint(sub)!==ticket.fingerprint)throw new BillingFlowError("BILLING_QUOTE_EXPIRED");
    const current=await buildQuote(c.env.DB,stripe,sub,owner,planNamed(ticket.targetPlan),!!ticket.annual,ticket.prorationDate);
    if(JSON.stringify(current)!==JSON.stringify(quote))throw new BillingFlowError("BILLING_QUOTE_EXPIRED");
    await c.env.DB.prepare("UPDATE pulse_billing_ticket SET state='processing',result=? WHERE id=? AND state='preview'").bind(JSON.stringify({keep}),ticket.id).run();
   }
   let result:z.infer<typeof resultSchema>;
   if(quote.mode==="scheduled"){
    // This MVP supports one fixed-price item without discounts/tax
    // overrides (validated in flowSubscription). Preserve its paid phase.
    const schedule=scheduleSchema.parse(await stripe.subscriptionSchedules.create({from_subscription:sub.id},{...stripeOptions,idempotencyKey:`blup-change:${ticket.id}:schedule`}));if(schedule.subscription!==sub.id)throw new BillingFlowError("BILLING_UNSUPPORTED");
    await c.env.DB.prepare("UPDATE pulse_billing_ticket SET schedule_id=? WHERE id=?").bind(schedule.id,ticket.id).run();
    if(!quote.targetPrice)throw new BillingFlowError("PRICING_UNAVAILABLE",503);
    const changed=scheduleSchema.parse(await stripe.subscriptionSchedules.update(schedule.id,{end_behavior:"release",metadata:{blup_user:owner,blup_change:ticket.id},proration_behavior:"none",phases:[{items:[{price:quote.currentPrice,quantity:1}],start_date:quote.periodStart,end_date:quote.periodEnd,proration_behavior:"none"},{items:[{price:quote.targetPrice,quantity:1}],start_date:quote.periodEnd,duration:{interval:quote.annual?"year":"month",interval_count:1},proration_behavior:"none"}]},{...stripeOptions,idempotencyKey:`blup-change:${ticket.id}:phases`}));if(changed.id!==schedule.id)throw new BillingFlowError("BILLING_UNSUPPORTED");
    result={state:"scheduled",effectiveAt:quote.effectiveAt,paymentUrl:null};
   }else if(quote.mode==="cancel"||quote.mode==="restore"){
    const response=flowSubscription.parse(await stripe.subscriptions.update(sub.id,{cancel_at_period_end:quote.mode==="cancel",expand:["items.data.price.product","latest_invoice"]},{...stripeOptions,idempotencyKey:`blup-change:${ticket.id}:cancel`}));if(response.id!==sub.id)throw new BillingFlowError("BILLING_UNSUPPORTED");result={state:quote.mode==="cancel"?"scheduled":"restored",effectiveAt:quote.effectiveAt,paymentUrl:null};
   }else{
    if(!quote.targetPrice)throw new BillingFlowError("PRICING_UNAVAILABLE",503);
    const updated=flowSubscription.parse(await stripe.subscriptions.update(sub.id,{items:[{id:quote.itemId,price:quote.targetPrice,quantity:1}],proration_date:quote.prorationDate,proration_behavior:"always_invoice",payment_behavior:"pending_if_incomplete",metadata:{blup_change:ticket.id},expand:["items.data.price.product","latest_invoice"]},{...stripeOptions,idempotencyKey:`blup-change:${ticket.id}:update`}));if(updated.id!==sub.id)throw new BillingFlowError("BILLING_UNSUPPORTED");
    const raw=typeof updated.latest_invoice==="object"?updated.latest_invoice?.hosted_invoice_url:null;
    result={state:updated.pending_update?"pending":"awaiting_webhook",effectiveAt:quote.effectiveAt,paymentUrl:raw?stripeURL(raw,["invoice.stripe.com","pay.stripe.com"]):null};
   }
   const guard=locks.map(()=>"EXISTS(SELECT 1 FROM site_lease WHERE site_id=? AND token=? AND expires_at>?)").join(" AND ");
   await c.env.DB.batch([
    c.env.DB.prepare("INSERT INTO pulse_selection(user_id,target_plan,effective_at,keep) VALUES(?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET target_plan=excluded.target_plan,effective_at=excluded.effective_at,keep=excluded.keep,applied_at=NULL").bind(owner,quote.target,Date.parse(quote.effectiveAt),JSON.stringify(keep)),
    c.env.DB.prepare(`UPDATE pulse_billing_ticket SET state=CASE WHEN ${guard} THEN 'done' ELSE NULL END,result=? WHERE id=?`).bind(...locks.flatMap(lock=>[lock.id,lock.token,Date.now()]),JSON.stringify(result),ticket.id),
   ]);
   // No entitlement write here. Only signed webhook/current Stripe state
   // can change the plan, including when a payment is still pending.
   return c.json(result);
  });
 })
 .get("/state",async c=>{
  const owner=c.get("session").user.id;let change=await c.env.DB.prepare("SELECT target_plan AS plan,effective_at AS effectiveAt,applied_at AS appliedAt FROM pulse_selection WHERE user_id=?").bind(owner).first<{plan:string;effectiveAt:number;appliedAt:number|null}>();
  const latest=change?await c.env.DB.prepare("SELECT result,expires_at AS expiresAt FROM pulse_billing_ticket WHERE user_id=? AND target_plan=? AND state='done' AND schedule_id IS NULL AND json_extract(quote,'$.effectiveAt')=? ORDER BY expires_at DESC LIMIT 1").bind(owner,change.plan,new Date(change.effectiveAt).toISOString()).first<{result:string|null;expiresAt:number}>():null;
  const completed=latest?.result?resultSchema.safeParse(JSON.parse(latest.result)):null;
  if(change&&change.appliedAt===null&&completed?.success&&completed.data.state==="pending"&&latest&&latest.expiresAt+23*60*60_000<=Date.now()){
   await c.env.DB.prepare("DELETE FROM pulse_selection WHERE user_id=? AND target_plan=? AND effective_at=? AND applied_at IS NULL").bind(owner,change.plan,change.effectiveAt).run();change=null;
  }
  const checkout=await c.env.DB.prepare("SELECT plan,url,expires_at AS expiresAt FROM billing_checkout WHERE user_id=? AND expires_at>?").bind(owner,Date.now()).first<{plan:string;url:string|null;expiresAt:number}>();
  return c.json({paymentUrl:change&&change.appliedAt===null&&completed?.success&&completed.data.state==="pending"?completed.data.paymentUrl:null,change:change&&change.appliedAt===null?{plan:planNameSchema.parse(change.plan),effectiveAt:new Date(change.effectiveAt).toISOString()}:null,checkout:checkout?.url?{plan:planNameSchema.parse(checkout.plan),url:stripeURL(checkout.url,["checkout.stripe.com"]),expiresAt:new Date(checkout.expiresAt).toISOString()}:null});
 })
 .delete("/pending",async c=>{
  const owner=c.get("session").user.id;
  return withSubscription(c.env,owner,async({stripe,sub})=>{
   if(!sub.schedule){
    if(!sub.cancel_at_period_end)throw new BillingFlowError("BILLING_NO_CHANGE");
    await stripe.subscriptions.update(sub.id,{cancel_at_period_end:false},stripeOptions);
    await c.env.DB.prepare("DELETE FROM pulse_selection WHERE user_id=? AND applied_at IS NULL").bind(owner).run();return c.json({ok:true});
   }
   const ours=await c.env.DB.prepare("SELECT id,quote FROM pulse_billing_ticket WHERE user_id=? AND schedule_id=? AND state='done' LIMIT 1").bind(owner,sub.schedule).first<{id:string;quote:string}>();if(!ours)throw new BillingFlowError("BILLING_UNSUPPORTED");const quote=quoteSchema.parse(JSON.parse(ours.quote));if(Date.parse(quote.effectiveAt)<=Date.now())throw new BillingFlowError("BILLING_NO_CHANGE");
   await stripe.subscriptionSchedules.release(sub.schedule,{}, {...stripeOptions,idempotencyKey:`blup-change:${ours.id}:release`});await c.env.DB.prepare("DELETE FROM pulse_selection WHERE user_id=? AND applied_at IS NULL").bind(owner).run();return c.json({ok:true});
  });
 })
 .delete("/checkout",async c=>{
  const owner=c.get("session").user.id,lock=await siteLock(c.env.DB,`billing-account:${owner}`,60_000);if(!lock)throw new BillingFlowError("BILLING_BUSY",503);
  try{
   const saved=await c.env.DB.prepare("SELECT id,params,session_id AS sessionId,expires_at AS expiresAt FROM billing_checkout WHERE user_id=?").bind(owner).first<{id:string;params:string;sessionId:string|null;expiresAt:number}>();if(saved&&!saved.sessionId&&checkoutReplayable(saved)){const session=z.object({id:z.string()}).parse(await createStripe(c.env.STRIPE_SECRET_KEY).checkout.sessions.create(checkoutParamsSchema.parse(JSON.parse(saved.params)),{...stripeOptions,idempotencyKey:`blup-checkout:${saved.id}`}));saved.sessionId=session.id;}if(saved?.sessionId)await createStripe(c.env.STRIPE_SECRET_KEY).checkout.sessions.expire(saved.sessionId,{},stripeOptions);else if(saved&&checkoutReplayable(saved))throw new BillingFlowError("BILLING_CHECKOUT_PENDING");
   await c.env.DB.prepare("DELETE FROM billing_checkout WHERE user_id=?").bind(owner).run();return c.json({ok:true});
  }finally{await unlockSite(c.env.DB,`billing-account:${owner}`,lock);}
 });
