import type Stripe from "stripe";
import { eq } from "drizzle-orm";
import { db } from "./auth";
import { billingEvent, user } from "./db/schema";

// Money events the staff should see on each account (Stripe's plan for
// subscriptions): paid and failed renewals, refunds, disputes and early
// fraud warnings. Better Auth already keeps the subscription itself in sync
// (checkout and customer.subscription.*); these are the rest. Each event is
// stored once (Stripe may send it again).
export const BILLING_EVENTS = ["invoice.paid", "invoice.payment_failed", "charge.refunded", "charge.dispute.created", "radar.early_fraud_warning.created"] as const;

export async function recordBillingEvent(event: Stripe.Event, stripe: Stripe) {
	if (!(BILLING_EVENTS as readonly string[]).includes(event.type)) return;
	let customer: string | null = null;
	let amount: number | null = null;
	let currency: string | null = null;
	const o = event.data.object as unknown as Record<string, unknown>;
	const id = (v: unknown) => (typeof v === "string" ? v : v && typeof v === "object" && "id" in v ? String((v as { id: string }).id) : null);
	switch (event.type) {
		case "invoice.paid":
		case "invoice.payment_failed": {
			const inv = event.data.object;
			customer = id(inv.customer);
			amount = event.type === "invoice.paid" ? inv.amount_paid : inv.amount_due;
			currency = inv.currency;
			break;
		}
		case "charge.refunded": {
			const ch = event.data.object;
			customer = id(ch.customer);
			amount = ch.amount_refunded;
			currency = ch.currency;
			break;
		}
		case "charge.dispute.created":
		case "radar.early_fraud_warning.created": {
			// These point at the charge; the charge knows the customer.
			const charge = await stripe.charges.retrieve(id(o.charge) ?? "");
			customer = id(charge.customer);
			amount = typeof o.amount === "number" ? o.amount : charge.amount;
			currency = charge.currency;
			break;
		}
	}
	const [owner] = customer ? await db.select({ id: user.id }).from(user).where(eq(user.stripeCustomerId, customer)).limit(1) : [];
	await db
		.insert(billingEvent)
		.values({ stripeEventId: event.id, userId: owner?.id ?? null, type: event.type, amount, currency })
		.onConflictDoNothing({ target: billingEvent.stripeEventId });
}
