import {Hono} from "hono";
import {validate} from "../validation";
import {dealsWrite,type DealsApp} from "./access";
import {settingsBody,settingsOf,respace,DAY,type DealsSettings} from "./store";

// /api/deals/settings: one setting for all the account's channels. behind
// says a hunt found more than the pace could send in the last day.
const view=(s:DealsSettings,now:number)=>({pace:s.pace,headlines:s.headlines,extra:s.extra,socials:s.socials,behind:s.droppedAt!==null&&s.droppedAt>now-DAY});
export const settingsApi=new Hono<DealsApp>()
 .get("/",async c=>c.json(view(await settingsOf(c.env.DB,c.get("session").user.id),Date.now())))
 .put("/",dealsWrite,validate("json",settingsBody),async c=>{
  const owner=c.get("session").user.id,body=c.req.valid("json"),now=Date.now(),before=await settingsOf(c.env.DB,owner);
  await c.env.DB.prepare(`INSERT INTO deals_setting(user_id,pace,headlines,extra,extra_on,socials,socials_on,updated_at) VALUES(?,?,?,?,?,?,?,?)
   ON CONFLICT(user_id) DO UPDATE SET pace=excluded.pace,headlines=excluded.headlines,extra=excluded.extra,extra_on=excluded.extra_on,socials=excluded.socials,socials_on=excluded.socials_on,updated_at=excluded.updated_at`)
   .bind(owner,body.pace,JSON.stringify(body.headlines),body.extra.text||null,body.extra.on?1:0,JSON.stringify(body.socials.links),body.socials.on?1:0,now).run();
  // A new pace spaces again what already waits.
  if(body.pace!==before.pace)await respace(c.env.DB,owner,body.pace,now);
  return c.json(view(await settingsOf(c.env.DB,owner),now));
 });
