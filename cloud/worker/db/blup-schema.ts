import { sql } from "drizzle-orm";
import { sqliteTable, text, integer, index, primaryKey } from "drizzle-orm/sqlite-core";

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

// Sync of a person's notes (encrypted on their computer: nothing here can
// read them). One row per account that turned sync on.
export const syncAccount = sqliteTable("sync_account", {
	userId: text("user_id").primaryKey(),
	// The master key wrapped by the notes password and by the recovery key,
	// with the Argon2id settings (JSON, as the Studio sends it).
	wrappedKey: text("wrapped_key").notNull(),
	// The wrapping before the last password change, in case it must be undone.
	previousKey: text("previous_key"),
	// Bytes stored: blocks plus manifests of every version kept. Kept here so
	// the quota check never adds up rows.
	storageBytes: integer("storage_bytes").default(0).notNull(),
	lastSyncAt: integer("last_sync_at", { mode: "timestamp_ms" }),
	// The last daily clean-up (worker/cleanup.ts).
	cleanedAt: integer("cleaned_at", { mode: "timestamp_ms" }),
	createdAt: integer("created_at", { mode: "timestamp_ms" })
		.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
		.notNull(),
	updatedAt: integer("updated_at", { mode: "timestamp_ms" })
		.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
		.notNull(),
});

// A synced folder of notes (one per site). The name is encrypted.
export const vault = sqliteTable(
	"vault",
	{
		id: text("id").primaryKey(),
		userId: text("user_id").notNull(),
		name: text("name").notNull(),
		// The current version (0 = nothing saved yet).
		head: integer("head").default(0).notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
		updatedAt: integer("updated_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [index("vault_user_idx").on(t.userId, t.id)],
);

// The versions of a vault still kept (the head, plus the history of the
// plan). Its manifest and list of blocks live in R2 under key.
export const vaultVersion = sqliteTable(
	"vault_version",
	{
		vaultId: text("vault_id").notNull(),
		n: integer("n").notNull(),
		key: text("key").notNull(),
		size: integer("size").notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [primaryKey({ columns: [t.vaultId, t.n] })],
);

// A sealed block in R2 (a piece of a file). Its id is an HMAC of the piece,
// so equal pieces are stored once per account.
export const syncBlock = sqliteTable(
	"sync_block",
	{
		userId: text("user_id").notNull(),
		id: text("id").notNull(),
		size: integer("size").notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [primaryKey({ columns: [t.userId, t.id] })],
);
