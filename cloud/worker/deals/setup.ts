import {Hono} from "hono";
import type {DealsApp} from "./access";
// The first steps of an account (the guide under the panel's main pages),
// read from what it already has, as Pulse's: a plan with Deals, a channel,
// an affiliate ID, a hunt and a deal sent. Nothing is marked by hand.
export const setupSteps=["plan","channel","ids","hunt","post"] as const;
export type SetupStep=(typeof setupSteps)[number];
type Found={channel:number;ids:number;hunt:number;post:number};
export const setupApi=new Hono<DealsApp>()
 .get("/",async c=>{
  const owner=c.get("session").user.id,now=Date.now();
  const found=await c.env.DB.prepare(`SELECT
   EXISTS(SELECT 1 FROM deals_channel WHERE user_id=?1 AND (retire_at IS NULL OR retire_at>?2)) AS channel,
   EXISTS(SELECT 1 FROM deals_affiliate WHERE user_id=?1 AND credentials IS NOT NULL) AS ids,
   EXISTS(SELECT 1 FROM deals_hunt WHERE user_id=?1 AND (retire_at IS NULL OR retire_at>?2)) AS hunt,
   EXISTS(SELECT 1 FROM deals_post WHERE user_id=?1 AND state='sent') AS post`).bind(owner,now).first<Found>();
  const done:Record<SetupStep,boolean>={plan:!!c.get("access").plan,channel:!!found?.channel,ids:!!found?.ids,hunt:!!found?.hunt,post:!!found?.post};
  return c.json({steps:setupSteps.map(id=>({id,done:done[id]}))});
 });
