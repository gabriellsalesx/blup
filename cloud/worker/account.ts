import { observePlan } from "./lifecycle";
import { env } from "cloudflare:workers";
import { and, eq, inArray } from "drizzle-orm";
import { db } from "./auth";
import { subscription, user } from "./db/schema";
import { GIFT_PLAN, ownerMatches, planNamed } from "./plans";

export { GIFT_PLAN };

// The owner of Blup (STAFF_OWNER_EMAIL, a secret) has the top plan as a
// gift: no Stripe, no subscription, the limits of GIFT_PLAN.
export const isOwner = (email: string | null | undefined, verified: boolean | null | undefined) => ownerMatches(env.STAFF_OWNER_EMAIL, email, verified);

async function ownerAccount(userId: string) {
	const [u] = await db.select({ email: user.email, verified: user.emailVerified }).from(user).where(eq(user.id, userId)).limit(1);
	return isOwner(u?.email, u?.verified);
}

// Subscriptions that give the paid plan (past_due keeps it while Stripe
// retries the card).
const PAYING = ["active", "trialing", "past_due"];

// The subscription that is paid (or on trial), if any.
export async function activeSubscription(userId: string) {
	const [sub] = await db
		.select({ plan: subscription.plan, status: subscription.status, periodEnd: subscription.periodEnd, cancelAtPeriodEnd: subscription.cancelAtPeriodEnd, cancelAt: subscription.cancelAt,billingInterval:subscription.billingInterval })
		.from(subscription)
		.where(and(eq(subscription.referenceId, userId), inArray(subscription.status, PAYING)))
		.limit(1);
	return sub;
}

// The plan an account is on now: the gift for the owner, else the paid
// one (free without a paid subscription).
export async function planOf(userId: string) {
	const [owner, sub] = await Promise.all([ownerAccount(userId), activeSubscription(userId)]);
	const plan=planNamed(owner ? GIFT_PLAN : sub?.plan);
	await observePlan(env.DB,userId,plan.name);
	return plan;
}
