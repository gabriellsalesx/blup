import { sql } from "drizzle-orm";
import { sqliteTable, text, integer, index } from "drizzle-orm/sqlite-core";

// Blup's own tables (Better Auth's are in auth-schema.ts).

// The staff: who may enter the staff area (besides passing Cloudflare
// Access) and with which role.
export const staff = sqliteTable("staff", {
	email: text("email").primaryKey(),
	role: text("role", { enum: ["support", "admin"] }).notNull(),
	addedBy: text("added_by").notNull(),
	createdAt: integer("created_at", { mode: "timestamp_ms" })
		.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
		.notNull(),
});

// Every staff action: who, what, on which account, when.
export const staffLog = sqliteTable(
	"staff_log",
	{
		// INTEGER PRIMARY KEY: the rowid itself (no extra index); SQLite advises
		// against AUTOINCREMENT, which adds a bookkeeping table.
		id: integer("id").primaryKey(),
		staffEmail: text("staff_email").notNull(),
		action: text("action").notNull(),
		userId: text("user_id"),
		detail: text("detail"),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [index("staff_log_created_idx").on(t.createdAt), index("staff_log_user_idx").on(t.userId)],
);

// What Stripe reported about an account's money that the staff should see:
// payments, failures, refunds, disputes and fraud warnings (from the
// webhook). One row per Stripe event (its ID keeps a resent event from
// being stored twice).
export const billingEvent = sqliteTable(
	"billing_event",
	{
		id: integer("id").primaryKey(),
		stripeEventId: text("stripe_event_id").notNull().unique(),
		userId: text("user_id"),
		type: text("type").notNull(),
		amount: integer("amount"),
		currency: text("currency"),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [index("billing_event_user_idx").on(t.userId, t.id)],
);
