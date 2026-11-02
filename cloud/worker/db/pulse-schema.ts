import {sqliteTable,text,integer,index,primaryKey} from "drizzle-orm/sqlite-core";

// D1 contains configuration/admission only. Event bodies belong to project DOs.
export const pulseProject=sqliteTable("pulse_project",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),name:text("name").notNull(),platform:text("platform").notNull(),objectName:text("object_name").notNull().unique(),active:integer("active").notNull().default(1),deletedAt:integer("deleted_at"),createdAt:integer("created_at").notNull(),cleanedAt:integer("cleaned_at").notNull().default(0),alertCheckedAt:integer("alert_checked_at").notNull().default(0),
},t=>[index("pulse_project_owner_idx").on(t.userId,t.id),index("pulse_project_deleted_idx").on(t.deletedAt),index("pulse_project_cleaned_idx").on(t.cleanedAt),index("pulse_project_alert_idx").on(t.alertCheckedAt)]);
export const pulseKey=sqliteTable("pulse_key",{
 id:text("id").primaryKey(),projectId:text("project_id").notNull(),hash:text("hash").notNull().unique(),prefix:text("prefix").notNull(),revokedAt:integer("revoked_at"),createdAt:integer("created_at").notNull(),
},t=>[index("pulse_key_project_idx").on(t.projectId)]);
export const pulseMonth=sqliteTable("pulse_month",{
 userId:text("user_id").notNull(),month:text("month").notNull(),reserved:integer("reserved").notNull().default(0),
},t=>[primaryKey({columns:[t.userId,t.month]})]);
export const pulseGrant=sqliteTable("pulse_grant",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),projectId:text("project_id").notNull(),month:text("month").notNull(),amount:integer("amount").notNull(),createdAt:integer("created_at").notNull(),
},t=>[index("pulse_grant_owner_month_idx").on(t.userId,t.month),index("pulse_grant_project_idx").on(t.projectId),index("pulse_grant_month_idx").on(t.month)]);
export const pulseAccount=sqliteTable("pulse_account",{
 userId:text("user_id").primaryKey(),lastPlan:text("last_plan").notNull(),readOnlyUntil:integer("read_only_until"),purgeStarted:integer("purge_started"),
},t=>[index("pulse_account_purge_idx").on(t.readOnlyUntil)]);
export const pulseMonitor=sqliteTable("pulse_monitor",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),projectId:text("project_id").notNull(),name:text("name").notNull(),url:text("url").notNull(),intervalMinutes:integer("interval_minutes").notNull(),active:integer("active").notNull().default(1),nextAt:integer("next_at").notNull(),claim:text("claim"),claimUntil:integer("claim_until").notNull().default(0),state:text("state").notNull().default("unknown"),revision:integer("revision").notNull().default(1),lastAt:integer("last_at"),createdAt:integer("created_at").notNull(),
},t=>[index("pulse_monitor_owner_idx").on(t.userId,t.id),index("pulse_monitor_due_idx").on(t.active,t.nextAt),index("pulse_monitor_project_idx").on(t.projectId)]);
export const pulsePage=sqliteTable("pulse_page",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),slug:text("slug").notNull().unique(),title:text("title").notNull(),published:integer("published").notNull().default(0),language:text("language").notNull().default("en"),createdAt:integer("created_at").notNull(),
},t=>[index("pulse_page_owner_idx").on(t.userId,t.id)]);
export const pulseComponent=sqliteTable("pulse_component",{
 pageId:text("page_id").notNull(),monitorId:text("monitor_id").notNull(),name:text("name").notNull(),position:integer("position").notNull(),
},t=>[primaryKey({columns:[t.pageId,t.monitorId]})]);
export const pulseIncident=sqliteTable("pulse_incident",{
 id:text("id").primaryKey(),pageId:text("page_id").notNull(),title:text("title").notNull(),state:text("state").notNull(),impact:text("impact").notNull(),createdAt:integer("created_at").notNull(),updatedAt:integer("updated_at").notNull(),expiresAt:integer("expires_at").notNull().default(0),
},t=>[index("pulse_incident_expiry_idx").on(t.expiresAt),index("pulse_incident_page_idx").on(t.pageId,t.createdAt)]);
export const pulseIncidentUpdate=sqliteTable("pulse_incident_update",{
 id:text("id").primaryKey(),incidentId:text("incident_id").notNull(),body:text("body").notNull(),state:text("state").notNull(),createdAt:integer("created_at").notNull(),expiresAt:integer("expires_at").notNull().default(0),
},t=>[index("pulse_incident_update_expiry_idx").on(t.expiresAt),index("pulse_incident_update_idx").on(t.incidentId,t.createdAt)]);
export const pulsePageDomain=sqliteTable("pulse_page_domain",{
 pageId:text("page_id").primaryKey(),hostname:text("hostname").notNull(),proof:text("proof").notNull(),providerId:text("provider_id"),verifiedAt:integer("verified_at"),status:text("status").notNull().default("pending_ownership"),ssl:text("ssl").notNull().default("pending"),records:text("records").notNull().default("[]"),checkedAt:integer("checked_at"),
},t=>[index("pulse_domain_host_idx").on(t.hostname,t.verifiedAt)]);
export const pulseChannel=sqliteTable("pulse_channel",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),type:text("type").notNull(),target:text("target").notNull(),label:text("label").notNull(),active:integer("active").notNull().default(1),lastError:text("last_error"),nextAt:integer("next_at").notNull().default(0),testAt:integer("test_at").notNull().default(0),createdAt:integer("created_at").notNull(),
},t=>[index("pulse_channel_owner_idx").on(t.userId)]);
export const pulseProjectChannel=sqliteTable("pulse_project_channel",{
 projectId:text("project_id").notNull(),channelId:text("channel_id").notNull(),type:text("type").notNull(),
},t=>[primaryKey({columns:[t.projectId,t.type]}),index("pulse_project_channel_target_idx").on(t.channelId)]);
export const pulseTelegramLink=sqliteTable("pulse_telegram_link",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),hash:text("hash").notNull().unique(),expiresAt:integer("expires_at").notNull(),state:text("state").notNull().default("pending"),chatId:text("chat_id"),chatLabel:text("chat_label"),channelId:text("channel_id"),
},t=>[index("pulse_telegram_link_owner_idx").on(t.userId),index("pulse_telegram_link_expiry_idx").on(t.expiresAt)]);
export const pulseTelegramUpdate=sqliteTable("pulse_telegram_update",{
 id:integer("id").primaryKey(),receivedAt:integer("received_at").notNull(),applied:integer("applied").notNull().default(0),
},t=>[index("pulse_telegram_update_expiry_idx").on(t.receivedAt)]);
export const pulseOutbox=sqliteTable("pulse_outbox",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),projectId:text("project_id"),channelId:text("channel_id").notNull(),refId:text("ref_id").notNull(),kind:text("kind").notNull(),createdAt:integer("created_at").notNull(),expiresAt:integer("expires_at").notNull(),nextAt:integer("next_at").notNull(),attempts:integer("attempts").notNull().default(0),state:text("state").notNull().default("pending"),claim:text("claim"),claimUntil:integer("claim_until").notNull().default(0),
},t=>[index("pulse_outbox_due_idx").on(t.state,t.nextAt),index("pulse_outbox_expiry_idx").on(t.expiresAt),index("pulse_outbox_channel_idx").on(t.channelId),index("pulse_outbox_owner_idx").on(t.userId)]);
export const pulseCooldown=sqliteTable("pulse_cooldown",{
 key:text("key").primaryKey(),until:integer("until").notNull(),
},t=>[index("pulse_cooldown_expiry_idx").on(t.until)]);
export const pulseDeliveryBudget=sqliteTable("pulse_delivery_budget",{key:text("key").primaryKey(),nextAt:integer("next_at").notNull()});
