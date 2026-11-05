import {Hono} from "hono";
import {dealsSession,type DealsApp} from "./access";
import {channelsApi,telegramApi,telegramWebhookApi} from "./channels";
import {affiliatesApi} from "./affiliates";
import {offersApi} from "./offers";
import {huntsApi,statsApi} from "./hunts";
import {settingsApi} from "./settings";
import {whatsappApi,whatsappCallbackApi} from "./whatsapp";
import {isSampleImage,sampleImage} from "./samples";
import {DAY,dayOf} from "./store";
import {aiUsageOf} from "../ai";
import {planOf} from "../account";
export {dayOf};

// /api/deals: the Deals panel's API. The Telegram webhook and the WhatsApp
// Bridge's notices come first and outside the session (each signed with
// its own secret), and so do the example posts' pictures (public, the
// same for everyone); every other route needs a session, and changes go
// behind dealsWrite.
export const dealsApi=new Hono<DealsApp>()
 .route("/telegram",telegramWebhookApi)
 .route("/whatsapp",whatsappCallbackApi)
 .get("/samples/:name{[a-z-]{1,20}\\.jpg}",c=>{const name=c.req.param("name").slice(0,-4);if(!isSampleImage(name))return c.json({code:"NOT_FOUND"},404);return c.body(sampleImage(name),200,{"content-type":"image/jpeg","cache-control":"public, max-age=86400","x-content-type-options":"nosniff"});})
 .use("*",dealsSession)
 .get("/usage",async c=>{
  const owner=c.get("session").user.id,access=c.get("access"),now=Date.now();
  const row=await c.env.DB.prepare(`SELECT
   (SELECT count(*) FROM deals_channel WHERE user_id=?1 AND (retire_at IS NULL OR retire_at>?2)) AS channels,
   (SELECT count(*) FROM deals_hunt WHERE user_id=?1 AND kind<>'mirror' AND (retire_at IS NULL OR retire_at>?2)) AS hunts,
   (SELECT count(*) FROM deals_hunt WHERE user_id=?1 AND kind='mirror' AND (retire_at IS NULL OR retire_at>?2)) AS mirrors,
   (SELECT count(*) FROM deals_post WHERE user_id=?1 AND day=?3) AS postsToday`).bind(owner,now,dayOf(now)).first<{channels:number;hunts:number;mirrors:number;postsToday:number}>();
  return c.json({plan:access.plan,limits:access.limits,readOnly:access.readOnly,readOnlyUntil:access.readOnlyUntil===null?null:new Date(access.readOnlyUntil).toISOString(),used:{channels:row?.channels??0,hunts:row?.hunts??0,mirrors:row?.mirrors??0,postsToday:row?.postsToday??0},resetsAt:new Date(Math.floor(now/DAY)*DAY+DAY).toISOString()});
 })
 // The plan's AI allowance used this month (the Cloud's), for the "write
 // with AI" button.
 .get("/ai",async c=>{
  const owner=c.get("session").user.id,plan=await planOf(owner),ai=await aiUsageOf(owner,plan);
  return c.json({available:plan.limits.aiUsage>0,percent:ai.percent,resetsAt:ai.resetsAt});
 })
 .route("/channels",channelsApi)
 .route("/telegram",telegramApi)
 .route("/whatsapp",whatsappApi)
 .route("/affiliates",affiliatesApi)
 .route("/offers",offersApi)
 .route("/hunts",huntsApi)
 .route("/stats",statsApi)
 .route("/settings",settingsApi);
