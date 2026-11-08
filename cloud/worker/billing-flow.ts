import {z} from "zod";
import {keepSchema} from "../shared/pulse-selection";
import type Stripe from "stripe";
import {purchaseAvailable} from "./pricing";
import {createStripe} from "./stripe-client";
import {PLANS,ownerMatches,type Plan} from "./plans";
import {siteLock,unlockSite} from "./quota";
import {hasEmail} from "./auth-options";
import {sha256} from "./pulse/limits";
import {stripeLocale,languageOf} from "../shared/language";
import {planNameSchema} from "../shared/pulse-plans";
import {cloudPaths,checkoutParam} from "../shared/panel-paths";
export const stripeOptions={timeout:5000,maxNetworkRetries:0} as const;
const id=z.string().regex(/^[A-Za-z][A-Za-z0-9_]*$/).max(255),meta=z.record(z.string(),z.string()).default({}),money=z.number().int().safe(),seconds=z.number().int().nonnegative().max(8_640_000_000_000);
const product=z.object({id:id.startsWith("prod_"),metadata:meta,active:z.boolean().optional()});
export const flowPrice=z.object({id:id.startsWith("price_"),lookup_key:z.string().nullable(),metadata:meta,product,active:z.boolean(),unit_amount:money.nonnegative(),currency:z.string().regex(/^[a-z]{3}$/),type:z.literal("recurring"),billing_scheme:z.literal("per_unit"),recurring:z.object({interval:z.enum(["month","year"]),interval_count:z.literal(1),usage_type:z.literal("licensed")})});
export const flowSubscription=z.object({id:id.startsWith("sub_"),customer:z.string(),status:z.string(),cancel_at_period_end:z.boolean(),cancel_at:seconds.nullable(),schedule:z.string().nullable(),pending_update:z.object({expires_at:seconds}).nullable(),collection_method:z.literal("charge_automatically"),discounts:z.array(z.unknown()).max(0),default_tax_rates:z.array(z.unknown()).max(0),automatic_tax:z.object({enabled:z.literal(false)}),items:z.object({has_more:z.literal(false),data:z.array(z.object({id:id.startsWith("si_"),quantity:z.literal(1),current_period_start:seconds,current_period_end:seconds,price:flowPrice})).length(1)}),latest_invoice:z.union([z.string(),z.object({id,hosted_invoice_url:z.string().nullable(),status:z.string().nullable()})]).nullable()});
export type FlowSubscription=z.infer<typeof flowSubscription>;
export const quoteSchema=z.object({mode:z.enum(["immediate","scheduled","cancel","restore"]),target:planNameSchema,annual:z.boolean(),currency:z.string().regex(/^[a-z]{3}$/),amountDue:money.nonnegative(),total:money,credit:money.nonnegative(),debit:money.nonnegative(),recurringAmount:money.nonnegative(),effectiveAt:z.iso.datetime(),periodStart:seconds,periodEnd:seconds,itemId:id,currentPrice:id,targetPrice:id.nullable(),prorationDate:seconds,selectionRequired:z.boolean(),resources:z.object({projects:z.array(z.object({id:z.string(),name:z.string()})).max(10),monitors:z.array(z.object({id:z.string(),name:z.string(),projectId:z.string()})).max(50),pages:z.array(z.object({id:z.string(),name:z.string()})).max(3)}),limits:z.object({projects:z.number().int().nonnegative(),monitors:z.number().int().nonnegative(),pages:z.number().int().nonnegative()}).nullable()});
export type BillingQuote=z.infer<typeof quoteSchema>;
export {keepSchema} from "../shared/pulse-selection";
export class BillingFlowError extends Error{constructor(readonly code:string,readonly status:400|403|404|409|502|503=409){super(code);}}
export function stripeURL(raw:string,hosts:readonly string[]){const url=new URL(raw);if(url.protocol!=="https:"||url.username||url.password||url.port||!hosts.includes(url.hostname))throw new BillingFlowError("BILLING_UNAVAILABLE",502);return url.href;}
export function pricePlan(price:Pick<z.infer<typeof flowPrice>,"lookup_key"|"metadata"|"product">){return PLANS.find(p=>p.lookupKey&&(price.lookup_key===p.lookupKey||price.lookup_key===p.yearlyLookupKey))??PLANS.find(p=>p.lookupKey&&p.name===(price.metadata.blup_plan??price.product.metadata.blup_plan));}
export async function selectedPrice(stripe:Stripe,plan:Plan,annual:boolean){
 const key=annual?plan.yearlyLookupKey:plan.lookupKey;if(!key)throw new BillingFlowError("PRICING_UNAVAILABLE",503);
 const list=z.object({has_more:z.literal(false),data:z.array(flowPrice).length(1)}).safeParse(await stripe.prices.list({active:true,type:"recurring",lookup_keys:[key],limit:2,expand:["data.product"]},stripeOptions));
 if(!list.success)throw new BillingFlowError("PRICING_UNAVAILABLE",503);const price=list.data.data[0];if(price.lookup_key!==key||!price.active||price.product.active===false||price.recurring.interval!==(annual?"year":"month")||pricePlan(price)?.name!==plan.name)throw new BillingFlowError("PRICING_UNAVAILABLE",503);return price;
}
export async function billingOwner(env:Cloudflare.Env,userId:string){
 const owner=await env.DB.prepare("SELECT id,email,email_verified AS verified,stripe_customer_id AS customer,language FROM user WHERE id=? AND banned=0 AND NOT EXISTS(SELECT 1 FROM resource_quota WHERE user_id=user.id AND deleted_at IS NOT NULL)").bind(userId).first<{id:string;email:string;verified:number;customer:string|null;language:string}>();if(!owner)throw new BillingFlowError("UNAUTHORIZED",403);if(ownerMatches(env.STAFF_OWNER_EMAIL,owner.email,!!owner.verified))throw new BillingFlowError("BILLING_GIFT",403);return owner;
}
type BillingContext={stripe:Stripe;sub:FlowSubscription;owner:Awaited<ReturnType<typeof billingOwner>>;locks:{id:string;token:string}[]};
export async function withSubscription<T>(env:Cloudflare.Env,userId:string,fn:(context:BillingContext)=>Promise<T>):Promise<T>{
 const owner=await billingOwner(env,userId),saved=await env.DB.prepare("SELECT stripe_subscription_id AS id FROM subscription WHERE reference_id=? AND stripe_subscription_id IS NOT NULL AND status IN('active','trialing','past_due') LIMIT 1").bind(userId).first<{id:string}>();if(!saved||!owner.customer)throw new BillingFlowError("BILLING_NO_SUBSCRIPTION");
 const locks:{id:string;token:string}[]=[];
 try{
  // Same order as webhook reconciliation: subscription, then account.
  for(const key of [`billing:${saved.id}`,`billing-account:${userId}`]){const token=await siteLock(env.DB,key,60_000);if(!token)throw new BillingFlowError("BILLING_BUSY",503);locks.push({id:key,token});}
  const stripe=createStripe(env.STRIPE_SECRET_KEY),parsed=flowSubscription.safeParse(await stripe.subscriptions.retrieve(saved.id,{expand:["items.data.price.product","latest_invoice"]},stripeOptions));if(!parsed.success)throw new BillingFlowError("BILLING_UNSUPPORTED");const sub=parsed.data;
  if(sub.id!==saved.id||sub.customer!==owner.customer||!["active","trialing","past_due"].includes(sub.status))throw new BillingFlowError("BILLING_NO_SUBSCRIPTION");
  if(sub.schedule){
   const ticket=await env.DB.prepare("SELECT id,quote FROM pulse_billing_ticket WHERE user_id=? AND subscription_id=? AND schedule_id=? AND state='done' LIMIT 1").bind(userId,sub.id,sub.schedule).first<{id:string;quote:string}>();
   const quote=ticket?quoteSchema.safeParse(JSON.parse(ticket.quote)):null;
   // Release only our schedule after its requested phase has actually
   // started. Releasing preserves the current subscription and price.
   if(ticket&&quote?.success&&Date.parse(quote.data.effectiveAt)<=Date.now()&&sub.items.data[0].price.id===quote.data.targetPrice){
    await stripe.subscriptionSchedules.release(sub.schedule,{}, {...stripeOptions,idempotencyKey:`blup-change:${ticket.id}:applied`});sub.schedule=null;
   }
  }
  return await fn({stripe,sub,owner,locks});
 }finally{for(const lock of locks.reverse())await unlockSite(env.DB,lock.id,lock.token);}
}
export async function subscriptionFingerprint(sub:FlowSubscription){return sha256(JSON.stringify([sub.id,sub.customer,sub.status,sub.cancel_at_period_end,sub.cancel_at,sub.schedule,sub.pending_update,sub.items.data.map(i=>[i.id,i.price.id,i.quantity,i.current_period_start,i.current_period_end])]));}
export async function resourceChoices(db:D1Database,userId:string){
 const projects=await db.prepare("SELECT id,name FROM pulse_project WHERE user_id=? AND deleted_at IS NULL AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 10").bind(userId,Date.now()).all<{id:string;name:string}>();
 const monitors=await db.prepare("SELECT id,name,project_id AS projectId FROM pulse_monitor WHERE user_id=? AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 50").bind(userId,Date.now()).all<{id:string;name:string;projectId:string}>();
 const pages=await db.prepare("SELECT id,title AS name FROM pulse_page WHERE user_id=? AND (retire_at IS NULL OR retire_at>?) ORDER BY created_at,id LIMIT 3").bind(userId,Date.now()).all<{id:string;name:string}>();return {projects:projects.results,monitors:monitors.results,pages:pages.results};
}
export async function buildQuote(db:D1Database,stripe:Stripe,sub:FlowSubscription,userId:string,target:Plan,annual:boolean,prorationDate:number):Promise<BillingQuote>{
 const item=sub.items.data[0],current=pricePlan(item.price);if(!current)throw new BillingFlowError("BILLING_UNSUPPORTED");
 if(sub.pending_update)throw new BillingFlowError("BILLING_PENDING");
 const resources=await resourceChoices(db,userId),rank={free:0,pro:1,max:2,ultra:3};
 const same=target.name===current.name&&annual===(item.price.recurring.interval==="year");
 const mode=target.name==="free"?"cancel":same&&(sub.cancel_at_period_end||sub.cancel_at)?"restore":rank[target.basePlan]<rank[current.basePlan]||current.pulse&&!target.pulse||current.deals&&!target.deals||item.price.recurring.interval==="year"&&!annual?"scheduled":"immediate";
 // Never credit an unpaid period into a plan change. Cancellation and
 // restoring the existing price do not create a proration or new credit.
 if((mode==="immediate"||mode==="scheduled")&&(sub.status==="past_due"||sub.latest_invoice!==null&&(typeof sub.latest_invoice==="string"||sub.latest_invoice.status!=="paid")))throw new BillingFlowError("BILLING_PAYMENT_REQUIRED");
 const price=mode==="restore"?item.price:target.name==="free"?null:await selectedPrice(stripe,target,annual);
 if(price&&price.currency!==item.price.currency)throw new BillingFlowError("BILLING_CURRENCY");
 if(same&&mode!=="restore")throw new BillingFlowError("BILLING_NO_CHANGE");
 const limits=target.pulse?{projects:target.pulse.projects,monitors:target.pulse.monitors,pages:target.pulse.pages}:null;
 const selectionRequired=!!limits&&(resources.projects.length>limits.projects||resources.monitors.length>limits.monitors||resources.pages.length>limits.pages);
 let amountDue=0,total=0,credit=0,debit=0;
 if(mode==="immediate"){
  if(!price)throw new BillingFlowError("PRICING_UNAVAILABLE",503);
  const invoice=z.object({amount_due:money.nonnegative(),total:money,currency:z.string(),lines:z.object({has_more:z.literal(false),data:z.array(z.object({amount:money,parent:z.object({subscription_item_details:z.object({proration:z.boolean()}).nullable().optional()}).nullable()})).max(100)})}).parse(await stripe.invoices.createPreview({customer:sub.customer,subscription:sub.id,subscription_details:{items:[{id:item.id,price:price.id,quantity:1}],proration_date:prorationDate,proration_behavior:"always_invoice"}},stripeOptions));
  if(invoice.currency!==price.currency)throw new BillingFlowError("BILLING_CURRENCY");
  amountDue=invoice.amount_due;total=invoice.total;
  for(const line of invoice.lines.data.filter(l=>l.parent?.subscription_item_details?.proration)){if(line.amount<0)credit-=line.amount;else debit+=line.amount;}
 }
 return quoteSchema.parse({mode,target:target.name,annual,currency:price?.currency??item.price.currency,amountDue,total,credit,debit,recurringAmount:price?.unit_amount??0,effectiveAt:new Date((mode==="scheduled"||mode==="cancel"?item.current_period_end:prorationDate)*1000).toISOString(),periodStart:item.current_period_start,periodEnd:item.current_period_end,itemId:item.id,currentPrice:item.price.id,targetPrice:price?.id??null,prorationDate,selectionRequired,resources,limits});
}
export function selectionFor(quote:BillingQuote,selected:z.infer<typeof keepSchema>|undefined){
 const all={projects:quote.resources.projects.map(p=>p.id),monitors:quote.resources.monitors.map(p=>p.id),pages:quote.resources.pages.map(p=>p.id)};
 if(quote.selectionRequired&&!selected)throw new BillingFlowError("BILLING_SELECTION_REQUIRED");
 const keep=selected??all;
 for(const field of ["projects","monitors","pages"] as const){if(keep[field].some(id=>!all[field].includes(id))||quote.limits&&keep[field].length>quote.limits[field])throw new BillingFlowError("BILLING_SELECTION_REQUIRED");}
 if(quote.limits&&all.projects.length&&keep.projects.length===0)throw new BillingFlowError("BILLING_SELECTION_REQUIRED");
 if(quote.resources.monitors.some(m=>keep.monitors.includes(m.id)&&!keep.projects.includes(m.projectId)))throw new BillingFlowError("BILLING_SELECTION_REQUIRED");return keep;
}
// Stripe accepts expires_at from 30 minutes to 24 hours after the session is
// created. The saved params are replayed after a lost or failed create, so a
// one hour expiry leaves a half hour window to replay them; past that, a row
// without a session can no longer be created and is dropped.
export const CHECKOUT_TTL_SECONDS=3600;
export const checkoutReplayable=(saved:{sessionId:string|null;expiresAt:number},now=Date.now())=>!!saved.sessionId||saved.expiresAt-now>31*60_000;
export const checkoutParamsSchema=z.object({mode:z.literal("subscription"),customer:z.string(),client_reference_id:z.string(),line_items:z.array(z.object({price:z.string(),quantity:z.literal(1)})).length(1),success_url:z.url(),cancel_url:z.url(),locale:z.enum(["pt-BR","en","es"]),expires_at:seconds,metadata:meta,subscription_data:z.object({metadata:meta,billing_mode:z.object({type:z.literal("flexible")})})});
export async function checkout(env:Cloudflare.Env,userId:string,plan:Plan,annual:boolean){
 const token=await siteLock(env.DB,`billing-account:${userId}`,60_000);if(!token)throw new BillingFlowError("BILLING_BUSY",503);
 try{
  const owner=await billingOwner(env,userId);if(!hasEmail(owner.email)||!owner.verified)throw new BillingFlowError("EMAIL_REQUIRED",403);if(!await purchaseAvailable(env,plan.name,annual))throw new BillingFlowError("PRICING_UNAVAILABLE",503);const stripe=createStripe(env.STRIPE_SECRET_KEY);
  if(!owner.customer){const customer=z.object({id:id.startsWith("cus_")}).parse(await stripe.customers.create({email:owner.email,metadata:{blup_user:userId}},{...stripeOptions,idempotencyKey:`blup-customer:${userId}`}));await env.DB.prepare("UPDATE user SET stripe_customer_id=? WHERE id=? AND stripe_customer_id IS NULL").bind(customer.id,userId).run();owner.customer=customer.id;}
  const subscriptions=z.object({has_more:z.literal(false),data:z.array(z.object({id,status:z.string()})).max(100)}).parse(await stripe.subscriptions.list({customer:owner.customer,status:"all",limit:100},stripeOptions));
  if(subscriptions.data.some(s=>!["canceled","incomplete_expired"].includes(s.status)))throw new BillingFlowError("BILLING_USE_PREVIEW");
  let saved=await env.DB.prepare("SELECT id,plan,annual,params,session_id AS sessionId,url,expires_at AS expiresAt FROM billing_checkout WHERE user_id=?").bind(userId).first<{id:string;plan:string;annual:number;params:string;sessionId:string|null;url:string|null;expiresAt:number}>();
  if(saved&&(saved.expiresAt<=Date.now()||!checkoutReplayable(saved))){await env.DB.prepare("DELETE FROM billing_checkout WHERE user_id=? AND id=?").bind(userId,saved.id).run();saved=null;}
  if(saved&&(saved.plan!==plan.name||!!saved.annual!==annual))throw new BillingFlowError("BILLING_CHECKOUT_PENDING");
  if(!saved){
   const open=z.object({has_more:z.literal(false),data:z.array(z.object({id,mode:z.string(),url:z.string().nullable(),expires_at:seconds})).max(10)}).parse(await stripe.checkout.sessions.list({customer:owner.customer,status:"open",limit:10},stripeOptions));if(open.data.some(s=>s.mode==="subscription"))throw new BillingFlowError("BILLING_CHECKOUT_PENDING");
   const price=await selectedPrice(stripe,plan,annual),intent=crypto.randomUUID(),expires=Math.floor(Date.now()/1000)+CHECKOUT_TTL_SECONDS;
   const params=checkoutParamsSchema.parse({mode:"subscription",customer:owner.customer,client_reference_id:userId,line_items:[{price:price.id,quantity:1}],success_url:`${env.APP_URL}${cloudPaths.plan}?${checkoutParam}=success`,cancel_url:`${env.APP_URL}${cloudPaths.plan}`,locale:stripeLocale[languageOf(owner.language)??"en"],expires_at:expires,metadata:{blup_plan:plan.name},subscription_data:{metadata:{userId,referenceId:userId,blup_plan:plan.name},billing_mode:{type:"flexible"}}});
   await env.DB.prepare("INSERT INTO billing_checkout(user_id,id,plan,annual,params,expires_at) VALUES(?,?,?,?,?,?)").bind(userId,intent,plan.name,annual?1:0,JSON.stringify(params),expires*1000).run();saved={id:intent,plan:plan.name,annual:annual?1:0,params:JSON.stringify(params),sessionId:null,url:null,expiresAt:expires*1000};
  }
  const session=z.object({id:id.startsWith("cs_"),url:z.string().nullable(),status:z.string(),customer:z.string(),mode:z.literal("subscription")}).parse(await stripe.checkout.sessions.create(checkoutParamsSchema.parse(JSON.parse(saved.params)),{...stripeOptions,idempotencyKey:`blup-checkout:${saved.id}`}));
  if(session.customer!==owner.customer||session.status!=="open"||!session.url)throw new BillingFlowError("BILLING_CHECKOUT_PENDING");const url=stripeURL(session.url,["checkout.stripe.com"]);
  await env.DB.prepare("UPDATE billing_checkout SET session_id=?,url=? WHERE user_id=? AND id=?").bind(session.id,url,userId,saved.id).run();return {url};
 }finally{await unlockSite(env.DB,`billing-account:${userId}`,token);}
}
export async function portal(env:Cloudflare.Env,userId:string){
 const owner=await billingOwner(env,userId);if(!owner.customer)throw new BillingFlowError("BILLING_NO_SUBSCRIPTION");const stripe=createStripe(env.STRIPE_SECRET_KEY);
 const config=z.object({has_more:z.literal(false),data:z.array(z.object({id,active:z.literal(true),is_default:z.literal(true),features:z.object({subscription_update:z.object({enabled:z.literal(false)}),subscription_cancel:z.object({enabled:z.boolean(),mode:z.string()})})})).length(1)}).parse(await stripe.billingPortal.configurations.list({active:true,is_default:true,limit:2},stripeOptions));
 const current=config.data[0];if(current.features.subscription_cancel.enabled&&current.features.subscription_cancel.mode!=="at_period_end")throw new BillingFlowError("BILLING_PORTAL_UNSAFE");
 const session=z.object({url:z.string()}).parse(await stripe.billingPortal.sessions.create({customer:owner.customer,configuration:current.id,return_url:`${env.APP_URL}${cloudPaths.plan}`,locale:stripeLocale[languageOf(owner.language)??"en"]},stripeOptions));return {url:stripeURL(session.url,["billing.stripe.com"])};
}
