import { sql } from "drizzle-orm";
import { index, SQLiteTable, type IndexBuilder } from "drizzle-orm/sqlite-core";
import { subscription, user } from "./auth-schema";

// Indexes Blup added by hand to Better Auth's tables (migrations 0001 and
// 0003). auth-schema.ts is generated (npm run db:auth-schema) and would lose
// them, so they're declared here, on top of whatever the generated table
// declares: drizzle-kit then sees the schema as the database is.
// The key is drizzle's own (Symbol.for, so the same in drizzle-kit).
type Extra = (columns: unknown) => unknown;
const extraConfig = Symbol.for("drizzle:ExtraConfigBuilder");
function addIndexes<T extends SQLiteTable>(table: T, extra: (t: T) => IndexBuilder[]): void {
	const key = extraConfig;
	const holder = table as unknown as Record<typeof key, Extra | undefined>;
	const generated = holder[key];
	holder[key] = (columns) => {
		const own = generated?.(columns);
		const list = own === undefined ? [] : Array.isArray(own) ? own.flat(1) : Object.values(own as object);
		return [...list, ...extra(table)];
	};
}

addIndexes(subscription, (t) => [index("subscription_reference_status_idx").on(t.referenceId, t.status)]);
addIndexes(user, (t) => [
	index("user_created_at_idx").on(t.createdAt),
	index("user_banned_idx").on(t.banned).where(sql`${t.banned} = 1`),
	index("user_stripe_customer_idx").on(t.stripeCustomerId),
]);

// Named so schema.ts can re-export this file (which runs the lines above).
export const authIndexes = ["subscription_reference_status_idx", "user_created_at_idx", "user_banned_idx", "user_stripe_customer_idx"] as const;
