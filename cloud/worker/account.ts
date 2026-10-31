import { and, eq, inArray } from "drizzle-orm";
import { db } from "./auth";
import { subscription } from "./db/schema";
import { planNamed } from "./plans";

// Subscriptions that give the paid plan (past_due keeps it while Stripe
// retries the card).
export const PAYING = ["active", "trialing", "past_due"];

// The subscription that is paid (or on trial), if any.
export async function activeSubscription(userId: string) {
	const [sub] = await db
		.select({ plan: subscription.plan, status: subscription.status, periodEnd: subscription.periodEnd, cancelAtPeriodEnd: subscription.cancelAtPeriodEnd, cancelAt: subscription.cancelAt })
		.from(subscription)
		.where(and(eq(subscription.referenceId, userId), inArray(subscription.status, PAYING)))
		.limit(1);
	return sub;
}

// The plan an account is on now (free without a paid subscription).
export async function planOf(userId: string) {
	return planNamed((await activeSubscription(userId))?.plan);
}
