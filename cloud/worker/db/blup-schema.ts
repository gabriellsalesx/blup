import { sql } from "drizzle-orm";
import { sqliteTable, text, integer, index, primaryKey, uniqueIndex } from "drizzle-orm/sqlite-core";

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

// Deduplication is separate from staff money events: every reconciled
// lifecycle event has a receipt, committed with the subscription state.
export const stripeWebhookEvent = sqliteTable("stripe_webhook_event", {
	id:text("id").primaryKey().notNull(), type:text("type").notNull(),
	userId:text("user_id"), processedAt:integer("processed_at").notNull(),
}, t=>[index("stripe_webhook_event_processed_idx").on(t.processedAt),index("stripe_webhook_event_user_idx").on(t.userId)]);

// Sync of a person's notes (encrypted on their computer: nothing here can
// read them). One row per account that turned sync on.
export const syncAccount = sqliteTable("sync_account", {
	userId: text("user_id").primaryKey(),
	// The master key wrapped by the notes password and by the recovery key,
	// with the Argon2id settings (JSON, as the Studio sends it).
	wrappedKey: text("wrapped_key").notNull(),
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
		headRefsReady:integer("head_refs_ready",{mode:"boolean"}).notNull().default(false),
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
		objectKey:text("object_key"),
		size: integer("size").notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [primaryKey({ columns: [t.userId, t.id] })],
);

// What each AI request cost (narration through Blup Cloud), in millionths
// of a US dollar: the plan's monthly AI allowance is checked against it.
export const aiUsage = sqliteTable(
	"ai_usage",
	{
		id: integer("id").primaryKey(),
		userId: text("user_id").notNull(),
		kind: text("kind").notNull(), // "narration", "translation" or an AI feature (ai.ts aiFeatures)
		units: integer("units").notNull(), // characters read
		costMicros: integer("cost_micros").notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [index("ai_usage_user_idx").on(t.userId, t.createdAt)],
);

// A site published through Blup Cloud (paid plans), served at
// <slug>.getblup.site (worker/serve-site.ts).
export const site = sqliteTable(
	"site",
	{
		id: text("id").primaryKey(),
		userId: text("user_id").notNull(),
		slug: text("slug").notNull().unique(),
		title: text("title").notNull(),
		// The live deploy: its manifest is s/<id>/d/<deploy> in R2 ("" before
		// the first).
		deploy: text("deploy").default("").notNull(),
		// Bytes of the files stored (as stored: text files gzipped), and how
		// many the live deploy has.
		bytes: integer("bytes").default(0).notNull(),
		files: integer("files").default(0).notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
		updatedAt: integer("updated_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [index("site_user_idx").on(t.userId)],
);

// Reservations count before R2/provider work starts. They are settled in
// the same D1 transaction as the corresponding usage row.
export const resourceQuota = sqliteTable("resource_quota", {
	userId: text("user_id").primaryKey(),
	storage: integer("storage").notNull().default(0),
	sites: integer("sites").notNull().default(0),
	vaults: integer("vaults").notNull().default(0),
	ai: integer("ai").notNull().default(0),
	devices: integer("devices").notNull().default(0),
	deletedAt: integer("deleted_at"),
	purgeAfter: integer("purge_after"),
	purgeStarted: integer("purge_started"),
	graceUntil: integer("grace_until"),
	storageGraceLimit: integer("storage_grace_limit"),
	siteCacheGrace: integer("site_cache_grace"),
	selectedSiteId: text("selected_site_id"),
	domainGraceUntil: integer("domain_grace_until"),
	domainDowngradeNotified: integer("domain_downgrade_notified"),
	downgradeNotified: integer("downgrade_notified"),
	lastPlan: text("last_plan"),
	billingCancelPending: integer("billing_cancel_pending").notNull().default(0),
}, t => [index("resource_site_cache_expiry_idx").on(t.graceUntil).where(sql`${t.siteCacheGrace} IS NOT ${t.graceUntil} AND ${t.deletedAt} IS NULL`), index("resource_purge_idx").on(t.purgeAfter), index("resource_grace_idx").on(t.graceUntil),index("resource_billing_pending_idx").on(t.billingCancelPending),index("resource_domain_notice_idx").on(t.domainGraceUntil).where(sql`${t.domainDowngradeNotified} IS NULL AND ${t.deletedAt} IS NULL`),index("resource_notice_idx").on(t.graceUntil).where(sql`${t.downgradeNotified} IS NULL AND ${t.deletedAt} IS NULL`)]);

export const resourceReservation = sqliteTable("resource_reservation", {
	id: text("id").primaryKey(), userId: text("user_id").notNull(),
	kind: text("kind").notNull(), amount: integer("amount").notNull(),
	expiresAt: integer("expires_at").notNull(),
}, t => [index("reservation_expiry_idx").on(t.expiresAt), index("reservation_user_idx").on(t.userId)]);

export const siteLease = sqliteTable("site_lease", {
	siteId: text("site_id").primaryKey(), token: text("token").notNull(), expiresAt: integer("expires_at").notNull(),
}, t => [index("site_lease_expiry_idx").on(t.expiresAt)]);

export const retiredDeploy = sqliteTable("retired_deploy", {
	siteId: text("site_id").notNull(), deploy: text("deploy").notNull(), deleteAfter: integer("delete_after").notNull(),
}, t => [primaryKey({ columns: [t.siteId,t.deploy] }), index("retired_deploy_due_idx").on(t.deleteAfter)]);

export const siteDomain = sqliteTable("site_domain", {
	siteId: text("site_id").primaryKey(), hostname: text("hostname").notNull(),
	proof: text("proof").notNull(), verifiedAt: integer("verified_at"), providerId: text("provider_id"),
	status: text("status").notNull().default("pending_ownership"), ssl: text("ssl").notNull().default("pending"),
	records: text("records").notNull().default("[]"), checkedAt: integer("checked_at"),
	disableAt: integer("disable_at"),
}, t=>[uniqueIndex("site_domain_verified_unique").on(t.hostname).where(sql`${t.verifiedAt} IS NOT NULL`),index("site_domain_disable_idx").on(t.disableAt).where(sql`${t.status} <> 'disabled' OR ${t.providerId} IS NOT NULL OR ${t.verifiedAt} IS NOT NULL`)]);

export const syncTransfer = sqliteTable("sync_transfer", {
	id:text("id").primaryKey(), userId:text("user_id").notNull(), vaultId:text("vault_id").notNull(),
	parent:integer("parent").notNull(), key:text("key").notNull(), size:integer("size").notNull(),
	reservationId:text("reservation_id").notNull(), reserved:integer("reserved").notNull(),
	lease:text("lease").notNull(), expiresAt:integer("expires_at").notNull(), createdAt:integer("created_at").notNull(),
	completedVersion:integer("completed_version"),
}, t=>[index("sync_transfer_expiry_idx").on(t.expiresAt),index("sync_transfer_user_idx").on(t.userId)]);
export const syncTransferBlock = sqliteTable("sync_transfer_block", {
	ticketId:text("ticket_id").notNull(), blockId:text("block_id").notNull(), size:integer("size").notNull(),ready:integer("ready").notNull().default(0),objectKey:text("object_key"),
}, t=>[primaryKey({columns:[t.ticketId,t.blockId]})]);
export const vaultHeadBlock = sqliteTable("vault_head_block",{
	vaultId:text("vault_id").notNull(), blockId:text("block_id").notNull(),
},t=>[primaryKey({columns:[t.vaultId,t.blockId]}),index("vault_head_block_id_idx").on(t.blockId)]);
export const syncGarbage = sqliteTable("sync_garbage",{
	id:text("id").primaryKey(),userId:text("user_id").notNull(),keys:text("keys").notNull(),bytes:integer("bytes").notNull(),createdAt:integer("created_at").notNull(),
},t=>[index("sync_garbage_user_idx").on(t.userId),index("sync_garbage_created_idx").on(t.createdAt)]);

export const deviceApproval = sqliteTable("device_approval", {
	codeHash: text("code_hash").primaryKey(), userId: text("user_id").notNull(),
	reservationId: text("reservation_id").notNull(), expiresAt: integer("expires_at").notNull(),
}, t => [index("device_approval_expiry_idx").on(t.expiresAt)]);

// A file of a site in R2 (s/<site>/f/<sha256 of what is stored>): the same
// file is stored once per site, whatever deploy or path uses it.
export const siteBlob = sqliteTable(
	"site_blob",
	{
		siteId: text("site_id").notNull(),
		hash: text("hash").notNull(),
		size: integer("size").notNull(),
		createdAt: integer("created_at", { mode: "timestamp_ms" })
			.default(sql`(cast(unixepoch('subsecond') * 1000 as integer))`)
			.notNull(),
	},
	(t) => [primaryKey({ columns: [t.siteId, t.hash] })],
);

// A calendar reminder to send by e-mail (Max and Ultra). Only when: the
// event itself (name, kind, notes) stays on the person's computer. `id` is
// an opaque hash the Studio makes, to replace the list without repeats.
// sentAt: when it went (0: skipped, over the month's limit or too late).
export const reminder = sqliteTable(
	"reminder",
	{
		userId: text("user_id").notNull(),
		id: text("id").notNull(),
		sendAt: integer("send_at", { mode: "timestamp_ms" }).notNull(),
		sentAt: integer("sent_at", { mode: "timestamp_ms" }),
		attempts: integer("attempts").notNull().default(0),
		nextAttemptAt: integer("next_attempt_at").notNull().default(0),
		claim: text("claim"),
		claimUntil: integer("claim_until").notNull().default(0),
	},
	(t) => [primaryKey({ columns: [t.userId, t.id] }), index("reminder_due_idx").on(t.sentAt, t.sendAt), index("reminder_cleanup_idx").on(t.sendAt)],
);

// Reminder e-mails sent per account and month ("2026-11"), for the plan's
// monthly limit.
export const reminderMonth = sqliteTable(
	"reminder_month",
	{
		userId: text("user_id").notNull(),
		month: text("month").notNull(),
		sent: integer("sent").default(0).notNull(),
		reserved: integer("reserved").default(0).notNull(),
	},
	(t) => [primaryKey({ columns: [t.userId, t.month] }), index("reminder_month_cleanup_idx").on(t.month)],
);

// Persistent/coalesced global cache maintenance, independent of browser
// traffic. No user content or host names in purge requests.
export const siteCachePurge=sqliteTable("site_cache_purge",{
 siteId:text("site_id").primaryKey(),generation:text("generation").notNull(),queuedAt:integer("queued_at").notNull(),
},t=>[index("site_cache_purge_queued_idx").on(t.queuedAt)]);
export const siteCachePurgeState=sqliteTable("site_cache_purge_state",{
 id:text("id").primaryKey(),nextAt:integer("next_at").notNull(),
});

// The AI models a feature uses, in order (OpenRouter's models list: the
// first answers, the next ones take over when it fails). Set by an admin
// on the staff panel; missing, the feature's default list.
export const aiModelSetting = sqliteTable("ai_model_setting", {
	feature: text("feature").primaryKey(),
	models: text("models").notNull(),
	// The model the owner marked: a voice list's recommended one, a text
	// list's main one (it answers first). None until the owner picks one.
	recommended: text("recommended"),
	updatedBy: text("updated_by").notNull(),
	updatedAt: integer("updated_at").notNull(),
});

// When a subscription went past_due (the webhook saw it first). The paid
// plan holds for 14 days from it; then the account is on Grátis even while
// Stripe keeps retrying. Cleared when the subscription is paid again.
export const subscriptionGrace=sqliteTable("subscription_grace",{
 subscriptionId:text("subscription_id").primaryKey(),userId:text("user_id").notNull(),pastDueSince:integer("past_due_since").notNull(),
},t=>[index("subscription_grace_since_idx").on(t.pastDueSince),index("subscription_grace_user_idx").on(t.userId)]);

// AppSumo codes (lifetime deal, up to 5 per account). Only an HMAC of the
// normalized code is kept (APPSUMO_CODE_PEPPER), never the code: a leak of
// the table gives no usable code. last4 is what the panel and the staff
// show. A used or refunded code stays forever (user_id goes NULL when the
// account is purged), so it can never be redeemed again.
export const appsumoCode=sqliteTable("appsumo_code",{
 codeHash:text("code_hash").primaryKey(),last4:text("last4").notNull(),
 status:text("status",{enum:["available","redeemed","revoked"]}).default("available").notNull(),
 userId:text("user_id"),importedAt:integer("imported_at").notNull(),
 redeemedAt:integer("redeemed_at"),revokedAt:integer("revoked_at"),revokeReason:text("revoke_reason"),
},t=>[index("appsumo_code_user_idx").on(t.userId,t.status)]);
// Failed redemptions of an account in the current hour (10 lock it out
// until the hour ends); a success clears it.
export const appsumoAttempt=sqliteTable("appsumo_attempt",{
 userId:text("user_id").primaryKey(),windowStart:integer("window_start").notNull(),failures:integer("failures").notNull(),
});
