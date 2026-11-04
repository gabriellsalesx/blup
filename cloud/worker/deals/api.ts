import {Hono} from "hono";
import {dealsSession,type DealsApp} from "./access";

const DAY=86_400_000;
export const dayOf=(now=Date.now())=>new Date(now).toISOString().slice(0,10);

// /api/deals: the Deals panel's API (phase F0: who may use it and how much
// is used). Every route needs a session; changes go behind dealsWrite.
export const dealsApi=new Hono<DealsApp>().use("*",dealsSession)
 .get("/usage",async c=>{
  const owner=c.get("session").user.id,access=c.get("access"),now=Date.now();
  const row=await c.env.DB.prepare(`SELECT
   (SELECT count(*) FROM deals_channel WHERE user_id=?1 AND (retire_at IS NULL OR retire_at>?2)) AS channels,
   (SELECT count(*) FROM deals_hunt WHERE user_id=?1 AND kind<>'mirror' AND (retire_at IS NULL OR retire_at>?2)) AS hunts,
   (SELECT count(*) FROM deals_hunt WHERE user_id=?1 AND kind='mirror' AND (retire_at IS NULL OR retire_at>?2)) AS mirrors,
   (SELECT count(*) FROM deals_post WHERE user_id=?1 AND day=?3) AS postsToday`).bind(owner,now,dayOf(now)).first<{channels:number;hunts:number;mirrors:number;postsToday:number}>();
  return c.json({plan:access.plan,limits:access.limits,readOnly:access.readOnly,used:{channels:row?.channels??0,hunts:row?.hunts??0,mirrors:row?.mirrors??0,postsToday:row?.postsToday??0},resetsAt:new Date(Math.floor(now/DAY)*DAY+DAY).toISOString()});
 });
