import {afterEach,beforeEach,expect,it} from "vitest";
import {TestD1,seedUser} from "../security-test-support";
import {respace,dayOf,DAY} from "./store";
import {dealsLimits} from "../../shared/pulse-plans";

// A new pace spreads the queue again, and never past a day's cap.
let db:TestD1;
beforeEach(()=>{db=new TestD1();seedUser(db,"alice","ultra_deals");});
afterEach(()=>db.close());
it("keeps the daily cap when the queue is spaced again",async()=>{
 const now=Date.UTC(2026,10,15,12),today=dayOf(now);
 db.sqlite.prepare("PRAGMA foreign_keys=OFF").run();
 const insert=db.sqlite.prepare("INSERT INTO deals_post(id,user_id,hunt_id,channel_id,offer_id,code,link,affiliate,price_shown,paced,state,next_at,day,created_at) VALUES(?,?,NULL,'c',?,?,'https://x.invalid/',1,1,?,?,?,?,?)");
 // 90 already sent today, then 30 waiting in the pace for later days.
 for(let i=0;i<90;i++)insert.run(`s${i}`,"alice",`os${i}`,`S${String(i).padStart(7,"0")}`,1,"sent",now-1000,today,now-1000-i);
 for(let i=0;i<30;i++)insert.run(`q${i}`,"alice",`oq${i}`,`Q${String(i).padStart(7,"0")}`,1,"queued",now+5*DAY+i,dayOf(now+5*DAY),now+i);
 await respace(db.asD1(),"alice",0,now);
 const days=db.sqlite.prepare("SELECT day,count(*) AS n FROM deals_post WHERE user_id='alice' GROUP BY day ORDER BY day").all() as {day:string;n:number}[];
 for(const d of days)expect(d.n).toBeLessThanOrEqual(dealsLimits.postsPerDay);
 expect(days.find(d=>d.day===today)?.n).toBe(dealsLimits.postsPerDay);
 expect(days.find(d=>d.day===dayOf(now+DAY))?.n).toBe(20);
});
