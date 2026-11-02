import { accountAllowed } from "./lifecycle";
import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, asc, eq, isNull, lte } from "drizzle-orm";
import { validate } from "./validation";
import { auth, db } from "./auth";
import { reminder, reminderMonth, user } from "./db/schema";
import { planOf } from "./account";
import { sendReminder } from "./email";
import { DAY, LATE, keep, listSchema, monthOf } from "./reminder-rules";

// The calendar's reminders by e-mail (Max: 50 a month, Ultra: 100). The
// Studio sends only when each reminder is due, under an opaque id; the
// event (its name, kind, notes) never leaves the person's computer, and the
// e-mail only says there is a reminder. Turning notifications off in the
// Studio sends an empty list.

type App = { Bindings: Cloudflare.Env; Variables: { userId: string } };

const signedIn = createMiddleware<App>(async (c, next) => {
	const s = await auth.api.getSession({ headers: c.req.raw.headers });
	if (!s || !await accountAllowed(c.env.DB,s.user.id)) return c.json({ code: "UNAUTHORIZED" }, 401);
	c.set("userId", s.user.id);
	await next();
});

async function usedThisMonth(userId: string, now: number) {
	const [row] = await db.select({ sent: reminderMonth.sent }).from(reminderMonth).where(and(eq(reminderMonth.userId, userId), eq(reminderMonth.month, monthOf(now)))).limit(1);
	return row?.sent ?? 0;
}

export const remindersApi = new Hono<App>()
	.use(signedIn)

	// How many reminder e-mails the plan has this month, and how many went.
	.get("/", async (c) => {
		const userId = c.get("userId");
		const [plan, used] = await Promise.all([planOf(userId), usedThisMonth(userId, Date.now())]);
		return c.json({ limit: plan.limits.reminderEmails, used });
	})

	// The reminders to come, replacing the ones not sent yet.
	.put("/", validate("json", listSchema), async (c) => {
		const userId = c.get("userId");
		const now = Date.now();
		const plan = await planOf(userId);
		const limit = plan.limits.reminderEmails;
		// A plan without e-mails keeps nothing (a downgrade clears what waited).
		if (limit === 0) {
			await c.env.DB.prepare("DELETE FROM reminder WHERE user_id = ? AND sent_at IS NULL AND claim IS NULL").bind(userId).run();
			return c.json({ limit, used: 0, saved: 0 });
		}
		const list = keep(c.req.valid("json").reminders, now);
		const stmts = [c.env.DB.prepare("DELETE FROM reminder WHERE user_id = ? AND sent_at IS NULL AND claim IS NULL").bind(userId)];
		// Already sent (or skipped) stays so: the same id isn't sent twice.
		for (const r of list) stmts.push(c.env.DB.prepare("INSERT OR IGNORE INTO reminder (user_id, id, send_at) VALUES (?, ?, ?)").bind(userId, r.id, r.at));
		await c.env.DB.batch(stmts);
		return c.json({ limit, used: await usedThisMonth(userId, now), saved: list.length });
	});

// sendDue sends the reminders whose time came (every 5 minutes, from the
// Worker's cron). Over the month's limit, or more than 12 hours late, a
// reminder is skipped (sent_at 0) instead.
export async function sendDue(env:Cloudflare.Env,now=Date.now()) {
 // Release a crashed sender's hold before retrying. The attempt was counted
 // before the call, so even crashes cannot produce unbounded retries.
 const stale=await env.DB.prepare("SELECT user_id AS userId,id,claim FROM reminder WHERE sent_at IS NULL AND send_at<=? AND claim_until<=? AND claim IS NOT NULL LIMIT 50").bind(now,now).all<{userId:string;id:string;claim:string}>();
 for(const r of stale.results)await env.DB.batch([
  env.DB.prepare("UPDATE reminder SET claim=NULL,claim_until=0 WHERE user_id=? AND id=? AND claim=?").bind(r.userId,r.id,r.claim),
  env.DB.prepare("UPDATE reminder_month SET reserved=max(0,reserved-1) WHERE user_id=? AND month=? AND changes()>0").bind(r.userId,r.claim.slice(0,7)),
 ]);
 const due=await env.DB.prepare("SELECT r.user_id AS userId,r.id,r.send_at AS sendAt,r.attempts,u.email FROM reminder r JOIN user u ON u.id=r.user_id LEFT JOIN resource_quota q ON q.user_id=u.id WHERE r.sent_at IS NULL AND r.send_at<=? AND r.next_attempt_at<=? AND r.claim IS NULL AND u.banned=0 AND q.deleted_at IS NULL ORDER BY r.send_at LIMIT 100").bind(now,now).all<{userId:string;id:string;sendAt:number;attempts:number;email:string}>();
 const plans=new Map<string,number>();
 for(const r of due.results){
  let limit=plans.get(r.userId);
  if(limit===undefined){limit=(await planOf(r.userId)).limits.reminderEmails;plans.set(r.userId,limit);}
  if(!limit || now-r.sendAt>LATE || r.attempts>=3){await env.DB.prepare("UPDATE reminder SET sent_at=0 WHERE user_id=? AND id=? AND claim IS NULL").bind(r.userId,r.id).run();continue;}
  const month=monthOf(now),claim=`${month}:${crypto.randomUUID()}`;
  await env.DB.prepare("INSERT OR IGNORE INTO reminder_month(user_id,month) VALUES (?,?)").bind(r.userId,month).run();
  const claimed=await env.DB.batch([
   env.DB.prepare("UPDATE reminder SET claim=?,claim_until=?,attempts=attempts+1 WHERE user_id=? AND id=? AND sent_at IS NULL AND claim IS NULL AND attempts<3 AND next_attempt_at<=? AND EXISTS(SELECT 1 FROM reminder_month WHERE user_id=? AND month=? AND sent+reserved<?) RETURNING id").bind(claim,now+120_000,r.userId,r.id,now,r.userId,month,limit),
   env.DB.prepare("UPDATE reminder_month SET reserved=reserved+1 WHERE user_id=? AND month=? AND changes()>0").bind(r.userId,month),
  ]);
  if(!claimed[0]?.results.length)continue;
  let sent=false;
  try{sent=await sendReminder(env,r.email);}catch{sent=false;}
  await env.DB.batch([
   env.DB.prepare("UPDATE reminder SET sent_at=?,claim=NULL,claim_until=0,next_attempt_at=? WHERE user_id=? AND id=? AND claim=?").bind(sent?now:r.attempts+1>=3?0:null,now+300_000,r.userId,r.id,claim),
   env.DB.prepare("UPDATE reminder_month SET reserved=max(0,reserved-1),sent=sent+? WHERE user_id=? AND month=? AND changes()>0").bind(sent?1:0,r.userId,month),
  ]);
 }
}

// Indexed key selection and bounded deletes keep the daily cron's rows
// read/written proportional to expired data. A backlog continues next day.
export async function forgetOld(env:Cloudflare.Env,now=Date.now()) {
 const lastMonth=monthOf(new Date(now).setUTCDate(0));
 for(let i=0;i<20;i++){
  const result=await env.DB.batch([
   env.DB.prepare("DELETE FROM reminder WHERE rowid IN (SELECT rowid FROM reminder WHERE send_at<? AND claim IS NULL ORDER BY send_at LIMIT 100)").bind(now-LATE-DAY),
   env.DB.prepare("DELETE FROM reminder_month WHERE rowid IN (SELECT rowid FROM reminder_month WHERE month<? AND reserved=0 ORDER BY month LIMIT 100)").bind(lastMonth),
  ]);
  if(!result.some(r=>r.meta.changes))break;
 }
}
