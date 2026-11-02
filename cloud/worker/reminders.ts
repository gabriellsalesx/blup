import { Hono } from "hono";
import { createMiddleware } from "hono/factory";
import { and, asc, eq, isNull, lte } from "drizzle-orm";
import { zValidator } from "@hono/zod-validator";
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
	if (!s) return c.json({ code: "UNAUTHORIZED" }, 401);
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
	.put("/", zValidator("json", listSchema), async (c) => {
		const userId = c.get("userId");
		const now = Date.now();
		const plan = await planOf(userId);
		const limit = plan.limits.reminderEmails;
		// A plan without e-mails keeps nothing (a downgrade clears what waited).
		if (limit === 0) {
			await c.env.DB.prepare("DELETE FROM reminder WHERE user_id = ? AND sent_at IS NULL").bind(userId).run();
			return c.json({ limit, used: 0, saved: 0 });
		}
		const list = keep(c.req.valid("json").reminders, now);
		const stmts = [c.env.DB.prepare("DELETE FROM reminder WHERE user_id = ? AND sent_at IS NULL").bind(userId)];
		// Already sent (or skipped) stays so: the same id isn't sent twice.
		for (const r of list) stmts.push(c.env.DB.prepare("INSERT OR IGNORE INTO reminder (user_id, id, send_at) VALUES (?, ?, ?)").bind(userId, r.id, r.at));
		await c.env.DB.batch(stmts);
		return c.json({ limit, used: await usedThisMonth(userId, now), saved: list.length });
	});

// sendDue sends the reminders whose time came (every 5 minutes, from the
// Worker's cron). Over the month's limit, or more than 12 hours late, a
// reminder is skipped (sent_at 0) instead.
export async function sendDue(env: Cloudflare.Env, now = Date.now()) {
	const due = await db
		.select({ userId: reminder.userId, id: reminder.id, sendAt: reminder.sendAt, email: user.email })
		.from(reminder)
		.innerJoin(user, eq(user.id, reminder.userId))
		.where(and(isNull(reminder.sentAt), lte(reminder.sendAt, new Date(now))))
		.orderBy(asc(reminder.sendAt))
		.limit(200);
	if (!due.length) return; // most runs: one index lookup, nothing else
	const plans = new Map<string, number>();
	for (const r of due) {
		if (!plans.has(r.userId)) plans.set(r.userId, (await planOf(r.userId)).limits.reminderEmails);
		const limit = plans.get(r.userId)!;
		const month = monthOf(now);
		let send = limit > 0 && now - r.sendAt.getTime() <= LATE;
		if (send) {
			// Counted before sending: two runs can't both pass the limit.
			const [row] = await env.DB.prepare(
				"INSERT INTO reminder_month (user_id, month, sent) VALUES (?, ?, 1) ON CONFLICT (user_id, month) DO UPDATE SET sent = sent + 1 WHERE sent < ? RETURNING sent",
			)
				.bind(r.userId, month, limit)
				.all<{ sent: number }>()
				.then((x) => x.results);
			send = !!row;
		}
		await db
			.update(reminder)
			.set({ sentAt: new Date(send ? now : 0) })
			.where(and(eq(reminder.userId, r.userId), eq(reminder.id, r.id)));
		if (send) await sendReminder(env, r.email);
	}
}

// forgetOld runs with the daily clean-up: a reminder past the window the
// Studio could send it again in (12 hours late, plus a day) goes, sent or
// not, and so do the counts of months before the last one. Nothing piles up.
export async function forgetOld(env: Cloudflare.Env, now = Date.now()) {
	const lastMonth = monthOf(new Date(now).setUTCDate(0));
	await env.DB.batch([
		env.DB.prepare("DELETE FROM reminder WHERE send_at < ?").bind(now - LATE - DAY),
		env.DB.prepare("DELETE FROM reminder_month WHERE month < ?").bind(lastMonth),
	]);
}
