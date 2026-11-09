import { observePlan,PAST_DUE_GRACE,PAID_SUBSCRIPTION,PLAN_STATE_COLUMNS,type PlanStateRow } from "./lifecycle";
import { ensureQuotaStatement } from "./quota";
import type { Plan } from "./plans";
import { env } from "cloudflare:workers";
import { and, eq, exists, gt, inArray, or } from "drizzle-orm";
import { db } from "./auth";
import { subscription, subscriptionGrace } from "./db/schema";
import { GIFT_PLAN, ownerMatches, planNamed } from "./plans";

export { GIFT_PLAN };

// The owner of Blup (STAFF_OWNER_EMAIL, a secret) has the top plan as a
// gift: no Stripe, no subscription, the limits of GIFT_PLAN.
export const isOwner = (email: string | null | undefined, verified: boolean | null | undefined) => ownerMatches(env.STAFF_OWNER_EMAIL, email, verified);

// Subscriptions that give the paid plan: past_due keeps it while Stripe
// retries the card, for 14 days (PAST_DUE_GRACE, the same rule as
// accountPlan in lifecycle.ts).
const paying = (now: number) => or(
	inArray(subscription.status, ["active", "trialing"]),
	and(eq(subscription.status, "past_due"), exists(db.select({ id: subscriptionGrace.subscriptionId }).from(subscriptionGrace).where(and(eq(subscriptionGrace.subscriptionId, subscription.id), gt(subscriptionGrace.pastDueSince, now - PAST_DUE_GRACE))))),
);

// The subscription that is paid (or on trial), if any.
export async function activeSubscription(userId: string, now = Date.now()) {
	const [sub] = await db
		.select({ plan: subscription.plan, status: subscription.status, periodEnd: subscription.periodEnd, cancelAtPeriodEnd: subscription.cancelAtPeriodEnd, cancelAt: subscription.cancelAt,billingInterval:subscription.billingInterval })
		.from(subscription)
		.where(and(eq(subscription.referenceId, userId), paying(now)))
		.limit(1);
	return sub;
}

type AccountRow = PlanStateRow & { email: string; verified: number; banned: number; quotaUserId: string | null; deletedAt: number | null; plan: string | null };
export type PlanState = NonNullable<Awaited<ReturnType<typeof observePlan>>>;
export type Access = { allowed: boolean; plan: Plan; state: PlanState | null };

// Whether the account may use the API, the plan it is on now (the gift for
// the owner, else the paid one, free without a paid subscription) and its
// plan state, in one trip to D1: the quota row is made if missing, then
// the account, its subscription and its plan state are read in the same
// transaction. Only a changed plan state costs more (observePlan's write).
export async function accountAccess(d1: D1Database, userId: string, now = Date.now()): Promise<Access> {
	const [, read] = await d1.batch<AccountRow>([
		ensureQuotaStatement(d1, userId),
		d1.prepare(`SELECT u.email,u.email_verified AS verified,u.banned,q.user_id AS quotaUserId,q.deleted_at AS deletedAt,(SELECT s.plan FROM subscription s WHERE s.reference_id=u.id AND ${PAID_SUBSCRIPTION} LIMIT 1) AS plan,${PLAN_STATE_COLUMNS} FROM user u LEFT JOIN resource_quota q ON q.user_id=u.id WHERE u.id=?`).bind(now - PAST_DUE_GRACE, userId),
	]);
	const row = read?.results[0];
	const plan = planNamed(row && isOwner(row.email, !!row.verified) ? GIFT_PLAN : row?.plan);
	// The same row observePlan reads: the quota row, unless it is deleted.
	const current = row && row.quotaUserId !== null && row.deletedAt === null ? row : null;
	const state = await observePlan(d1, userId, plan.name, now, current);
	return { allowed: !!row && !row.banned && row.deletedAt === null, plan, state };
}

// The plan an account is on now (see accountAccess).
export async function planOf(userId: string) {
	return (await accountAccess(env.DB, userId)).plan;
}
